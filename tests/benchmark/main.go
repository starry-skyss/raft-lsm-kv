package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	opPut = "put"
	opGet = "get"

	defaultRunIDTimeFormat = "20060102_150405"
)

var knownErrorTypes = []string{
	"timeout",
	"connection_refused",
	"http_500",
	"http_503",
	"request_timeout",
	"not_leader",
	"other",
}

// ============================================================================
// 1. HTTP KV Client
// ============================================================================

type KVClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewKVClient(baseURL string, timeout time.Duration) *KVClient {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 200
	t.MaxConnsPerHost = 200
	t.MaxIdleConnsPerHost = 200

	return &KVClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: t,
		},
	}
}

func (c *KVClient) Put(key, value string) (int, string, error) {
	payload := map[string]string{
		"key":   key,
		"value": value,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return 0, "", err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/put", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), nil
}

func (c *KVClient) Get(key string) (int, string, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/get/" + url.PathEscape(key))
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), nil
}

// ============================================================================
// 2. Benchmark Model
// ============================================================================

type BenchmarkTask struct {
	Operation string
	Key       string
	Value     string
}

type RequestResult struct {
	Operation  string
	Latency    time.Duration
	HTTPStatus int
	Success    bool
	Found      bool
	NotFound   bool
	ErrorType  string
}

type BenchmarkConfig struct {
	RunID         string  `json:"run_id"`
	WorkloadName  string  `json:"workload_name"`
	Concurrency   int     `json:"concurrency"`
	TotalRequests int     `json:"total_requests"`
	WriteRatio    float64 `json:"write_ratio"`
	TargetURL     string  `json:"target_url"`
	Keyspace      int     `json:"keyspace"`
	ValueSize     int     `json:"value_size"`
	Seed          int64   `json:"seed"`
	Prefill       bool    `json:"prefill"`
	PrefillN      int     `json:"prefilln"`
	Warmup        int     `json:"warmup"`
	Timeout       string  `json:"timeout"`
}

type StageSummary struct {
	Enabled         bool           `json:"enabled"`
	Requested       int            `json:"requested"`
	Success         int            `json:"success"`
	Failed          int            `json:"failed"`
	Duration        string         `json:"duration"`
	DurationSeconds float64        `json:"duration_seconds"`
	ErrorCounts     map[string]int `json:"error_counts,omitempty"`
}

type BenchmarkReport struct {
	Config       BenchmarkConfig `json:"config"`
	PrefillStage StageSummary    `json:"prefill_stage"`
	WarmupStage  StageSummary    `json:"warmup_stage"`

	RunID        string  `json:"run_id"`
	WorkloadName string  `json:"workload_name"`
	Concurrency  int     `json:"concurrency"`
	WriteRatio   float64 `json:"write_ratio"`
	Keyspace     int     `json:"keyspace"`
	ValueSize    int     `json:"valuesize"`
	Seed         int64   `json:"seed"`
	Prefill      bool    `json:"prefill"`

	TotalRequests   int     `json:"total_requests"`
	SuccessRequests int     `json:"success_requests"`
	FailedRequests  int     `json:"failed_requests"`
	SuccessRate     float64 `json:"success_rate"`
	SuccessRatePct  float64 `json:"success_rate_percent"`
	TotalDuration   string  `json:"total_duration"`
	DurationSeconds float64 `json:"total_duration_seconds"`
	TotalQPS        float64 `json:"total_qps"`
	SuccessQPS      float64 `json:"success_qps"`

	PutTotal        int     `json:"put_total"`
	PutSuccess      int     `json:"put_success"`
	PutFailed       int     `json:"put_failed"`
	PutAvgLatency   string  `json:"put_avg_latency"`
	PutAvgLatencyMs float64 `json:"put_avg_latency_ms"`
	PutP50          string  `json:"put_p50"`
	PutP50Ms        float64 `json:"put_p50_ms"`
	PutP90          string  `json:"put_p90"`
	PutP90Ms        float64 `json:"put_p90_ms"`
	PutP99          string  `json:"put_p99"`
	PutP99Ms        float64 `json:"put_p99_ms"`
	PutP999         string  `json:"put_p999"`
	PutP999Ms       float64 `json:"put_p999_ms"`
	PutMax          string  `json:"put_max"`
	PutMaxMs        float64 `json:"put_max_ms"`

	GetTotal        int     `json:"get_total"`
	GetSuccess      int     `json:"get_success"`
	GetFailed       int     `json:"get_failed"`
	GetFound        int     `json:"get_found"`
	GetNotFound     int     `json:"get_not_found"`
	GetAvgLatency   string  `json:"get_avg_latency"`
	GetAvgLatencyMs float64 `json:"get_avg_latency_ms"`
	GetP50          string  `json:"get_p50"`
	GetP50Ms        float64 `json:"get_p50_ms"`
	GetP90          string  `json:"get_p90"`
	GetP90Ms        float64 `json:"get_p90_ms"`
	GetP99          string  `json:"get_p99"`
	GetP99Ms        float64 `json:"get_p99_ms"`
	GetP999         string  `json:"get_p999"`
	GetP999Ms       float64 `json:"get_p999_ms"`
	GetMax          string  `json:"get_max"`
	GetMaxMs        float64 `json:"get_max_ms"`

	AvgLatency    string  `json:"avg_latency"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	P50Latency    string  `json:"p50_latency"`
	P50LatencyMs  float64 `json:"p50_latency_ms"`
	P90Latency    string  `json:"p90_latency"`
	P90LatencyMs  float64 `json:"p90_latency_ms"`
	P99Latency    string  `json:"p99_latency"`
	P99LatencyMs  float64 `json:"p99_latency_ms"`
	P999Latency   string  `json:"p999_latency"`
	P999LatencyMs float64 `json:"p999_latency_ms"`
	MaxLatency    string  `json:"max_latency"`
	MaxLatencyMs  float64 `json:"max_latency_ms"`

	ErrorCounts map[string]int `json:"error_counts"`
}

type latencySummary struct {
	Avg  time.Duration
	P50  time.Duration
	P90  time.Duration
	P99  time.Duration
	P999 time.Duration
	Max  time.Duration
}

// ============================================================================
// 3. Main Flow
// ============================================================================

func main() {
	runID := flag.String("runid", time.Now().Format(defaultRunIDTimeFormat), "本次实验运行 ID")
	workloadName := flag.String("workload", "", "负载名称，空表示根据 -w 和 -prefill 自动推断")
	concurrency := flag.Int("c", 50, "并发客户端数量")
	totalReqs := flag.Int("n", 10000, "总请求数量")
	writeRatio := flag.Float64("w", 0.5, "写请求比例，范围 0.0 到 1.0")
	targetURL := flag.String("u", "http://localhost:8080", "API Gateway 地址")

	keyspace := flag.Int("keyspace", 10000, "key 空间大小")
	valueSize := flag.Int("valuesize", 128, "value 字节大小")
	seed := flag.Int64("seed", 1, "随机种子")
	prefill := flag.Bool("prefill", false, "是否在正式压测前预填充数据")
	prefillN := flag.Int("prefilln", 0, "预填充 key 数量，默认等于 keyspace")
	warmup := flag.Int("warmup", 100, "正式压测前的预热请求数")
	out := flag.String("out", "", "JSON 结果输出文件路径，空表示不写文件")
	timeout := flag.Duration("timeout", 10*time.Second, "HTTP 客户端超时时间")
	flag.Parse()

	if *prefillN == 0 {
		*prefillN = *keyspace
	}
	if err := validateConfig(*concurrency, *totalReqs, *writeRatio, *keyspace, *valueSize, *prefillN, *warmup, *timeout); err != nil {
		fmt.Fprintf(os.Stderr, "invalid arguments: %v\n", err)
		os.Exit(2)
	}
	if *workloadName == "" {
		*workloadName = inferWorkloadName(*writeRatio, *prefill)
	}

	config := BenchmarkConfig{
		RunID:         *runID,
		WorkloadName:  *workloadName,
		Concurrency:   *concurrency,
		TotalRequests: *totalReqs,
		WriteRatio:    *writeRatio,
		TargetURL:     *targetURL,
		Keyspace:      *keyspace,
		ValueSize:     *valueSize,
		Seed:          *seed,
		Prefill:       *prefill,
		PrefillN:      *prefillN,
		Warmup:        *warmup,
		Timeout:       timeout.String(),
	}

	fmt.Println("Benchmark starting")
	fmt.Printf("Run ID: %s\n", *runID)
	fmt.Printf("Workload: %s\n", *workloadName)
	fmt.Printf("Target URL: %s\n", *targetURL)
	fmt.Printf("Concurrency: %d, Requests: %d, Write ratio: %.4f\n", *concurrency, *totalReqs, *writeRatio)
	fmt.Printf("Keyspace: %d, Value size: %d bytes, Seed: %d, Timeout: %s\n", *keyspace, *valueSize, *seed, timeout.String())
	printDataPreparationReminder(*writeRatio, *prefill)
	fmt.Println()

	value := makeFixedValue(*valueSize)
	prefillSummary := StageSummary{
		Enabled:   *prefill,
		Requested: *prefillN,
		Duration:  "0s",

		ErrorCounts: newErrorCounts(),
	}
	if *prefill {
		fmt.Printf("Prefilling %d keys...\n", *prefillN)
		prefillSummary = runPrefill(*targetURL, *timeout, *prefillN, value)
		fmt.Printf("Prefill completed: success=%d failed=%d duration=%s\n\n",
			prefillSummary.Success, prefillSummary.Failed, prefillSummary.Duration)
	}

	warmupSummary := StageSummary{
		Enabled:   *warmup > 0,
		Requested: *warmup,
		Duration:  "0s",

		ErrorCounts: newErrorCounts(),
	}
	if *warmup > 0 {
		fmt.Printf("Running warmup: %d requests...\n", *warmup)
		warmupTasks := generateTasks(*warmup, *writeRatio, *keyspace, value, *seed+1)
		warmupResults, warmupDuration := runTasks(warmupTasks, *concurrency, *targetURL, *timeout)
		warmupSummary = summarizeStage(true, *warmup, warmupResults, warmupDuration)
		fmt.Printf("Warmup completed: success=%d failed=%d duration=%s\n\n",
			warmupSummary.Success, warmupSummary.Failed, warmupSummary.Duration)
	}

	tasks := generateTasks(*totalReqs, *writeRatio, *keyspace, value, *seed)
	results, totalDuration := runTasks(tasks, *concurrency, *targetURL, *timeout)
	report := buildReport(config, prefillSummary, warmupSummary, results, totalDuration)

	printReport(report)

	if *out != "" {
		if err := writeJSONReport(*out, report); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write JSON report: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("JSON report written to %s\n", *out)
	}
}

// ============================================================================
// 4. Request Execution
// ============================================================================

func validateConfig(concurrency, totalReqs int, writeRatio float64, keyspace, valueSize, prefillN, warmup int, timeout time.Duration) error {
	if concurrency <= 0 {
		return errors.New("-c must be greater than 0")
	}
	if totalReqs < 0 {
		return errors.New("-n must be greater than or equal to 0")
	}
	if writeRatio < 0 || writeRatio > 1 {
		return errors.New("-w must be between 0.0 and 1.0")
	}
	if keyspace <= 0 {
		return errors.New("-keyspace must be greater than 0")
	}
	if valueSize <= 0 {
		return errors.New("-valuesize must be greater than 0")
	}
	if prefillN < 0 {
		return errors.New("-prefilln must be greater than or equal to 0")
	}
	if warmup < 0 {
		return errors.New("-warmup must be greater than or equal to 0")
	}
	if timeout <= 0 {
		return errors.New("-timeout must be greater than 0")
	}
	return nil
}

func inferWorkloadName(writeRatio float64, prefill bool) string {
	switch {
	case almostEqual(writeRatio, 1.0) && !prefill:
		return "write_only"
	case almostEqual(writeRatio, 0.0) && prefill:
		return "read_only"
	case almostEqual(writeRatio, 0.5) && prefill:
		return "read_write_50_50"
	case almostEqual(writeRatio, 0.05) && prefill:
		return "read_heavy_95_5"
	default:
		return "custom"
	}
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.0000001
}

func printDataPreparationReminder(writeRatio float64, prefill bool) {
	fmt.Println("Data preparation reminder:")
	if almostEqual(writeRatio, 1.0) && !prefill {
		fmt.Println("- For write-only tests, clear the server data directory before starting the server if you need a clean baseline.")
	}
	if !almostEqual(writeRatio, 1.0) && !prefill {
		fmt.Println("- Read-heavy or mixed tests should usually run with -prefill=true.")
	}
	if prefill {
		fmt.Println("- Prefill is enabled. Keep the same initial data scale across repeated runs for comparable results.")
	}
}

func makeFixedValue(size int) string {
	return strings.Repeat("v", size)
}

func generateTasks(total int, writeRatio float64, keyspace int, value string, seed int64) []BenchmarkTask {
	rng := rand.New(rand.NewSource(seed))
	tasks := make([]BenchmarkTask, 0, total)

	for i := 0; i < total; i++ {
		op := opGet
		if rng.Float64() < writeRatio {
			op = opPut
		}
		key := fmt.Sprintf("key_%d", rng.Intn(keyspace))
		tasks = append(tasks, BenchmarkTask{
			Operation: op,
			Key:       key,
			Value:     value,
		})
	}

	return tasks
}

func runPrefill(targetURL string, timeout time.Duration, total int, value string) StageSummary {
	client := NewKVClient(targetURL, timeout)
	results := make([]RequestResult, 0, total)
	start := time.Now()

	for i := 0; i < total; i++ {
		task := BenchmarkTask{
			Operation: opPut,
			Key:       fmt.Sprintf("key_%d", i),
			Value:     value,
		}
		results = append(results, executeTask(client, task))
	}

	return summarizeStage(true, total, results, time.Since(start))
}

func runTasks(tasks []BenchmarkTask, concurrency int, targetURL string, timeout time.Duration) ([]RequestResult, time.Duration) {
	if len(tasks) == 0 {
		return nil, 0
	}

	workerCount := concurrency
	if workerCount > len(tasks) {
		workerCount = len(tasks)
	}

	taskCh := make(chan BenchmarkTask)
	resultCh := make(chan RequestResult, len(tasks))

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := NewKVClient(targetURL, timeout)
			for task := range taskCh {
				resultCh <- executeTask(client, task)
			}
		}()
	}

	for _, task := range tasks {
		taskCh <- task
	}
	close(taskCh)

	wg.Wait()
	close(resultCh)

	results := make([]RequestResult, 0, len(tasks))
	for result := range resultCh {
		results = append(results, result)
	}

	return results, time.Since(start)
}

func executeTask(client *KVClient, task BenchmarkTask) RequestResult {
	start := time.Now()
	result := RequestResult{
		Operation: task.Operation,
	}

	var status int
	var body string
	var err error

	switch task.Operation {
	case opPut:
		status, body, err = client.Put(task.Key, task.Value)
	case opGet:
		status, body, err = client.Get(task.Key)
	default:
		err = fmt.Errorf("unknown operation: %s", task.Operation)
	}

	result.Latency = time.Since(start)
	result.HTTPStatus = status

	if err != nil {
		result.Success = false
		result.ErrorType = classifyNetworkError(err)
		return result
	}

	switch task.Operation {
	case opPut:
		result.Success = status == http.StatusOK
		if !result.Success {
			result.ErrorType = classifyHTTPFailure(status, body)
		}
	case opGet:
		switch status {
		case http.StatusOK:
			result.Success = true
			result.Found = true
		case http.StatusNotFound:
			result.Success = true
			result.NotFound = true
		default:
			result.Success = false
			result.ErrorType = classifyHTTPFailure(status, body)
		}
	}

	return result
}

func classifyNetworkError(err error) string {
	if err == nil {
		return ""
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	if os.IsTimeout(err) {
		return "timeout"
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "client.timeout"):
		return "timeout"
	case strings.Contains(msg, "deadline exceeded"):
		return "timeout"
	case strings.Contains(msg, "connection refused"):
		return "connection_refused"
	case strings.Contains(msg, "not leader"):
		return "not_leader"
	case strings.Contains(msg, "request timeout"):
		return "request_timeout"
	default:
		return "other"
	}
}

func classifyHTTPFailure(status int, body string) string {
	body = strings.ToLower(body)
	switch {
	case strings.Contains(body, "not leader"):
		return "not_leader"
	case strings.Contains(body, "request timeout"):
		return "request_timeout"
	case status > 0:
		return fmt.Sprintf("http_%d", status)
	default:
		return "other"
	}
}

func newErrorCounts() map[string]int {
	counts := make(map[string]int, len(knownErrorTypes))
	for _, errorType := range knownErrorTypes {
		counts[errorType] = 0
	}
	return counts
}

func hasAnyError(counts map[string]int) bool {
	for _, count := range counts {
		if count > 0 {
			return true
		}
	}
	return false
}

// ============================================================================
// 5. Statistics and Output
// ============================================================================

func summarizeStage(enabled bool, requested int, results []RequestResult, duration time.Duration) StageSummary {
	summary := StageSummary{
		Enabled:         enabled,
		Requested:       requested,
		Duration:        duration.String(),
		DurationSeconds: duration.Seconds(),
		ErrorCounts:     newErrorCounts(),
	}

	for _, result := range results {
		if result.Success {
			summary.Success++
			continue
		}
		summary.Failed++
		if result.ErrorType != "" {
			summary.ErrorCounts[result.ErrorType]++
		}
	}

	return summary
}

func buildReport(config BenchmarkConfig, prefill StageSummary, warmup StageSummary, results []RequestResult, totalDuration time.Duration) BenchmarkReport {
	var allLatencies []time.Duration
	var putLatencies []time.Duration
	var getLatencies []time.Duration

	report := BenchmarkReport{
		Config:          config,
		PrefillStage:    prefill,
		WarmupStage:     warmup,
		RunID:           config.RunID,
		WorkloadName:    config.WorkloadName,
		Concurrency:     config.Concurrency,
		WriteRatio:      config.WriteRatio,
		Keyspace:        config.Keyspace,
		ValueSize:       config.ValueSize,
		Seed:            config.Seed,
		Prefill:         config.Prefill,
		TotalRequests:   len(results),
		TotalDuration:   totalDuration.String(),
		DurationSeconds: totalDuration.Seconds(),
		ErrorCounts:     newErrorCounts(),
	}

	for _, result := range results {
		allLatencies = append(allLatencies, result.Latency)
		if result.Success {
			report.SuccessRequests++
		} else {
			report.FailedRequests++
			if result.ErrorType != "" {
				report.ErrorCounts[result.ErrorType]++
			}
		}

		switch result.Operation {
		case opPut:
			report.PutTotal++
			putLatencies = append(putLatencies, result.Latency)
			if result.Success {
				report.PutSuccess++
			} else {
				report.PutFailed++
			}
		case opGet:
			report.GetTotal++
			getLatencies = append(getLatencies, result.Latency)
			if result.Success {
				report.GetSuccess++
			} else {
				report.GetFailed++
			}
			if result.Found {
				report.GetFound++
			}
			if result.NotFound {
				report.GetNotFound++
			}
		}
	}

	if report.TotalRequests > 0 {
		report.SuccessRate = float64(report.SuccessRequests) / float64(report.TotalRequests)
		report.SuccessRatePct = report.SuccessRate * 100
	}
	if totalDuration > 0 {
		report.TotalQPS = float64(report.TotalRequests) / totalDuration.Seconds()
		report.SuccessQPS = float64(report.SuccessRequests) / totalDuration.Seconds()
	}

	allStats := summarizeLatencies(allLatencies)
	putStats := summarizeLatencies(putLatencies)
	getStats := summarizeLatencies(getLatencies)

	fillOverallLatencyFields(&report, allStats)
	fillPutLatencyFields(&report, putStats)
	fillGetLatencyFields(&report, getStats)

	return report
}

func summarizeLatencies(latencies []time.Duration) latencySummary {
	if len(latencies) == 0 {
		return latencySummary{}
	}

	sorted := append([]time.Duration(nil), latencies...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	var total time.Duration
	for _, latency := range sorted {
		total += latency
	}

	return latencySummary{
		Avg:  total / time.Duration(len(sorted)),
		P50:  percentile(sorted, 0.50),
		P90:  percentile(sorted, 0.90),
		P99:  percentile(sorted, 0.99),
		P999: percentile(sorted, 0.999),
		Max:  sorted[len(sorted)-1],
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(len(sorted))*p)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func durationMs(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / float64(time.Millisecond)
}

func fillPutLatencyFields(report *BenchmarkReport, stats latencySummary) {
	report.PutAvgLatency = stats.Avg.String()
	report.PutAvgLatencyMs = durationMs(stats.Avg)
	report.PutP50 = stats.P50.String()
	report.PutP50Ms = durationMs(stats.P50)
	report.PutP90 = stats.P90.String()
	report.PutP90Ms = durationMs(stats.P90)
	report.PutP99 = stats.P99.String()
	report.PutP99Ms = durationMs(stats.P99)
	report.PutP999 = stats.P999.String()
	report.PutP999Ms = durationMs(stats.P999)
	report.PutMax = stats.Max.String()
	report.PutMaxMs = durationMs(stats.Max)
}

func fillGetLatencyFields(report *BenchmarkReport, stats latencySummary) {
	report.GetAvgLatency = stats.Avg.String()
	report.GetAvgLatencyMs = durationMs(stats.Avg)
	report.GetP50 = stats.P50.String()
	report.GetP50Ms = durationMs(stats.P50)
	report.GetP90 = stats.P90.String()
	report.GetP90Ms = durationMs(stats.P90)
	report.GetP99 = stats.P99.String()
	report.GetP99Ms = durationMs(stats.P99)
	report.GetP999 = stats.P999.String()
	report.GetP999Ms = durationMs(stats.P999)
	report.GetMax = stats.Max.String()
	report.GetMaxMs = durationMs(stats.Max)
}

func fillOverallLatencyFields(report *BenchmarkReport, stats latencySummary) {
	report.AvgLatency = stats.Avg.String()
	report.AvgLatencyMs = durationMs(stats.Avg)
	report.P50Latency = stats.P50.String()
	report.P50LatencyMs = durationMs(stats.P50)
	report.P90Latency = stats.P90.String()
	report.P90LatencyMs = durationMs(stats.P90)
	report.P99Latency = stats.P99.String()
	report.P99LatencyMs = durationMs(stats.P99)
	report.P999Latency = stats.P999.String()
	report.P999LatencyMs = durationMs(stats.P999)
	report.MaxLatency = stats.Max.String()
	report.MaxLatencyMs = durationMs(stats.Max)
}

func printReport(report BenchmarkReport) {
	fmt.Println("================ Benchmark Results ================")
	fmt.Printf("run_id:           %s\n", report.RunID)
	fmt.Printf("workload_name:    %s\n", report.WorkloadName)
	fmt.Printf("concurrency:      %d\n", report.Concurrency)
	fmt.Printf("write_ratio:      %.4f\n", report.WriteRatio)
	fmt.Printf("keyspace:         %d\n", report.Keyspace)
	fmt.Printf("valuesize:        %d\n", report.ValueSize)
	fmt.Printf("seed:             %d\n", report.Seed)
	fmt.Printf("prefill:          %v\n", report.Prefill)
	fmt.Printf("total_requests:   %d\n", report.TotalRequests)
	fmt.Printf("success_requests: %d\n", report.SuccessRequests)
	fmt.Printf("failed_requests:  %d\n", report.FailedRequests)
	fmt.Printf("success_rate:     %.2f%%\n", report.SuccessRatePct)
	fmt.Printf("total_duration:   %s\n", report.TotalDuration)
	fmt.Printf("total_qps:        %.2f\n", report.TotalQPS)
	fmt.Printf("success_qps:      %.2f\n", report.SuccessQPS)

	fmt.Println("---------------------------------------------------")
	fmt.Println("Latency (all requests)")
	fmt.Printf("avg_latency:      %s\n", report.AvgLatency)
	fmt.Printf("p50_latency:      %s\n", report.P50Latency)
	fmt.Printf("p90_latency:      %s\n", report.P90Latency)
	fmt.Printf("p99_latency:      %s\n", report.P99Latency)
	fmt.Printf("p999_latency:     %s\n", report.P999Latency)
	fmt.Printf("max_latency:      %s\n", report.MaxLatency)

	fmt.Println("---------------------------------------------------")
	fmt.Println("PUT")
	fmt.Printf("put_total:        %d\n", report.PutTotal)
	fmt.Printf("put_success:      %d\n", report.PutSuccess)
	fmt.Printf("put_failed:       %d\n", report.PutFailed)
	fmt.Printf("put_avg_latency:  %s\n", report.PutAvgLatency)
	fmt.Printf("put_p50:          %s\n", report.PutP50)
	fmt.Printf("put_p90:          %s\n", report.PutP90)
	fmt.Printf("put_p99:          %s\n", report.PutP99)
	fmt.Printf("put_p999:         %s\n", report.PutP999)
	fmt.Printf("put_max:          %s\n", report.PutMax)

	fmt.Println("---------------------------------------------------")
	fmt.Println("GET")
	fmt.Printf("get_total:        %d\n", report.GetTotal)
	fmt.Printf("get_success:      %d\n", report.GetSuccess)
	fmt.Printf("get_failed:       %d\n", report.GetFailed)
	fmt.Printf("get_found:        %d\n", report.GetFound)
	fmt.Printf("get_not_found:    %d\n", report.GetNotFound)
	fmt.Printf("get_avg_latency:  %s\n", report.GetAvgLatency)
	fmt.Printf("get_p50:          %s\n", report.GetP50)
	fmt.Printf("get_p90:          %s\n", report.GetP90)
	fmt.Printf("get_p99:          %s\n", report.GetP99)
	fmt.Printf("get_p999:         %s\n", report.GetP999)
	fmt.Printf("get_max:          %s\n", report.GetMax)

	fmt.Println("---------------------------------------------------")
	fmt.Println("Errors")
	if !hasAnyError(report.ErrorCounts) {
		fmt.Println("none")
	} else {
		keys := make([]string, 0, len(report.ErrorCounts))
		for key := range report.ErrorCounts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Printf("%s: %d\n", key, report.ErrorCounts[key])
		}
	}
	fmt.Println("===================================================")
}

func writeJSONReport(path string, report BenchmarkReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
