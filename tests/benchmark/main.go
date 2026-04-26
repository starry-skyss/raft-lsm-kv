package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"
)

// ============================================================================
// 1. 真实的 HTTP KV 客户端
// ============================================================================

type KVClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewKVClient(baseURL string) *KVClient {
	// 配置高并发场景下的 HTTP 传输层 (非常重要：复用 TCP 连接)
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 200
	t.MaxConnsPerHost = 200
	t.MaxIdleConnsPerHost = 200

	return &KVClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout:   5 * time.Second, // 设置合理的超时，防止卡死
			Transport: t,
		},
	}
}

// Put 发起真实的 HTTP POST 请求
func (c *KVClient) Put(key, value string) error {
	payload := map[string]string{
		"key":   key,
		"value": value,
	}
	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", c.baseURL+"/put", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http error %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

// Get 发起真实的 HTTP GET 请求
func (c *KVClient) Get(key string) (string, error) {
	url := fmt.Sprintf("%s/get/%s", c.baseURL, key)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 注意：压测时可能会随机读取到不存在的 key，404 算作正常的业务响应，不算压测失败
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("http error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return "", nil
}

// ============================================================================
// 2. 核心压测逻辑
// ============================================================================

type ReqResult struct {
	Latency time.Duration
	Err     error
}

func main() {
	// 命令行参数
	concurrency := flag.Int("c", 50, "并发客户端数量 (Concurrency)")
	totalReqs := flag.Int("n", 10000, "总请求数量 (Total Requests)")
	writeRatio := flag.Float64("w", 0.5, "写请求比例 (Write Ratio, 0.0 - 1.0)")
	targetURL := flag.String("u", "http://localhost:8080", "API Gateway 地址")
	flag.Parse()

	fmt.Printf("🎯 开始压测...\n目标地址: %s\n并发数: %d\n总请求数: %d\n写比例: %.2f\n\n", 
		*targetURL, *concurrency, *totalReqs, *writeRatio)

	results := make(chan ReqResult, *totalReqs)
	tasks := make(chan int, *totalReqs)
	
	// 预填任务队列
	for i := 0; i < *totalReqs; i++ {
		tasks <- i
	}
	close(tasks)

	var wg sync.WaitGroup
	startTime := time.Now()

	// 启动 Worker 池
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 每个 Worker 共享或独占 Client 都可以，这里我们每个 Worker 建一个
			client := NewKVClient(*targetURL) 

			for range tasks {
				isWrite := rand.Float64() < *writeRatio
				// 控制 Key 的范围以产生一定的冲突和覆盖写
				key := fmt.Sprintf("key_%d", rand.Intn(5000))
				val := fmt.Sprintf("val_%d", time.Now().UnixNano())

				reqStart := time.Now()
				var err error

				if isWrite {
					err = client.Put(key, val)
				} else {
					_, err = client.Get(key)
				}

				results <- ReqResult{
					Latency: time.Since(reqStart),
					Err:     err,
				}
			}
		}()
	}

	wg.Wait()
	close(results)
	totalTime := time.Since(startTime)

	// ============================================================================
	// 3. 统计与分析指标
	// ============================================================================
	analyzeResults(results, totalTime, *totalReqs)
}

func analyzeResults(results <-chan ReqResult, totalTime time.Duration, totalReqs int) {
	var latencies []time.Duration
	var successCount, errCount int

	for res := range results {
		if res.Err != nil {
			errCount++
			// 取消注释下方代码可以打印具体错误，但高并发下会刷屏
			// log.Printf("Request error: %v", res.Err) 
		} else {
			successCount++
			latencies = append(latencies, res.Latency)
		}
	}

	qps := float64(successCount) / totalTime.Seconds()

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	if len(latencies) == 0 {
		log.Fatal("❌ 所有请求均失败，请检查服务端状态！")
	}

	getPercentile := func(p float64) time.Duration {
		idx := int(float64(len(latencies)) * p)
		if idx >= len(latencies) {
			idx = len(latencies) - 1
		}
		return latencies[idx]
	}

	fmt.Println("================ 📊 压测结果 (Benchmark Results) ================")
	fmt.Printf("总耗时 (Total Time):     %v\n", totalTime)
	fmt.Printf("总请求数 (Total Reqs):   %d\n", totalReqs)
	fmt.Printf("成功数 (Success):        %d\n", successCount)
	fmt.Printf("失败数 (Errors):         %d\n", errCount)
	fmt.Printf("吞吐量 (Throughput):     %.2f QPS\n", qps)
	fmt.Println("---------------------------------------------------------------")
	fmt.Println("⏱️ 延迟分布 (Latency Distribution):")
	fmt.Printf("  Min:    %v\n", latencies[0])
	fmt.Printf("  Avg:    %v\n", calculateAverage(latencies))
	fmt.Printf("  P50:    %v (中位数)\n", getPercentile(0.50))
	fmt.Printf("  P90:    %v\n", getPercentile(0.90))
	fmt.Printf("  P99:    %v (关键尾延迟)\n", getPercentile(0.99))
	fmt.Printf("  P99.9:  %v\n", getPercentile(0.999))
	fmt.Printf("  Max:    %v\n", latencies[len(latencies)-1])
	fmt.Println("===============================================================")
}

func calculateAverage(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	var total int64
	for _, l := range latencies {
		total += int64(l)
	}
	return time.Duration(total / int64(len(latencies)))
}