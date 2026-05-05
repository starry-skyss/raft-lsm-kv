// internal/api/server.go
package api

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"raft-lsm-kv/internal/raft/raftapi"
	"raft-lsm-kv/internal/store"
)

type gatewayMetrics struct {
	requestTotal                       atomic.Uint64
	putTotal                           atomic.Uint64
	getTotal                           atomic.Uint64
	deleteTotal                        atomic.Uint64
	http500Total                       atomic.Uint64
	http503Total                       atomic.Uint64
	requestTimeoutTotal                atomic.Uint64
	leadershipChangedBeforeCommitTotal atomic.Uint64
	notLeaderTotal                     atomic.Uint64
}

type storeAPIMetricsSnapshot struct {
	RequestTotal                       uint64 `json:"request_total"`
	PutTotal                           uint64 `json:"put_total"`
	GetTotal                           uint64 `json:"get_total"`
	DeleteTotal                        uint64 `json:"delete_total"`
	HTTP500Total                       uint64 `json:"http_500_total"`
	HTTP503Total                       uint64 `json:"http_503_total"`
	RequestTimeoutTotal                uint64 `json:"request_timeout_total"`
	LeadershipChangedBeforeCommitTotal uint64 `json:"leadership_changed_before_commit_total"`
	NotLeaderTotal                     uint64 `json:"not_leader_total"`
	ActiveWaitChans                    int    `json:"active_wait_chans"`
}

type runtimeMetricsSnapshot struct {
	GoroutineNum  int    `json:"goroutine_num"`
	HeapAlloc     uint64 `json:"heap_alloc"`
	NumGC         uint32 `json:"num_gc"`
	PauseTotalNS  uint64 `json:"pause_total_ns"`
	LastGCPauseNS uint64 `json:"last_gc_pause_ns"`
}

type nodeDebugMetrics struct {
	NodeID          int                 `json:"node_id"`
	ActiveWaitChans int                 `json:"active_wait_chans"`
	Raft            raftapi.RaftMetrics `json:"raft"`
}

type debugMetricsResponse struct {
	Timestamp                          string                  `json:"timestamp"`
	CurrentLeader                      int                     `json:"current_leader"`
	CurrentTerm                        int                     `json:"current_term"`
	LeaderChangeTotal                  uint64                  `json:"leader_change_total"`
	ElectionStartedTotal               uint64                  `json:"election_started_total"`
	TermChangeTotal                    uint64                  `json:"term_change_total"`
	AppendEntriesFailedTotal           uint64                  `json:"append_entries_failed_total"`
	RaftPersistCount                   uint64                  `json:"raft_persist_count"`
	RaftPersistTotalMS                 float64                 `json:"raft_persist_total_ms"`
	RaftPersistMaxMS                   float64                 `json:"raft_persist_max_ms"`
	CommitIndex                        int                     `json:"commit_index"`
	LastApplied                        int                     `json:"last_applied"`
	LogLength                          int                     `json:"log_length"`
	RequestTotal                       uint64                  `json:"request_total"`
	PutTotal                           uint64                  `json:"put_total"`
	GetTotal                           uint64                  `json:"get_total"`
	DeleteTotal                        uint64                  `json:"delete_total"`
	HTTP500Total                       uint64                  `json:"http_500_total"`
	HTTP503Total                       uint64                  `json:"http_503_total"`
	RequestTimeoutTotal                uint64                  `json:"request_timeout_total"`
	LeadershipChangedBeforeCommitTotal uint64                  `json:"leadership_changed_before_commit_total"`
	NotLeaderTotal                     uint64                  `json:"not_leader_total"`
	ActiveWaitChans                    int                     `json:"active_wait_chans"`
	GoroutineNum                       int                     `json:"goroutine_num"`
	HeapAlloc                          uint64                  `json:"heap_alloc"`
	NumGC                              uint32                  `json:"num_gc"`
	PauseTotalNS                       uint64                  `json:"pause_total_ns"`
	LastGCPauseNS                      uint64                  `json:"last_gc_pause_ns"`
	StoreAPI                           storeAPIMetricsSnapshot `json:"store_api"`
	Runtime                            runtimeMetricsSnapshot  `json:"runtime"`
	Nodes                              []nodeDebugMetrics      `json:"nodes"`
}

// StartGateway 启动对外的统一 API 网关
func StartGateway(kvStores []*store.KVStore, port string) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()        // 创建一个没有任何默认中间件的裸引擎
	r.Use(gin.Recovery()) // 仅挂载 Panic 恢复中间件
	metrics := &gatewayMetrics{}

	// 自定义仅记录错误的日志中间件
	r.Use(func(c *gin.Context) {
		c.Next() // 先执行具体的请求处理逻辑

		status := c.Writer.Status()
		if status == http.StatusInternalServerError {
			metrics.http500Total.Add(1)
		}
		if status == http.StatusServiceUnavailable {
			metrics.http503Total.Add(1)
		}
		// 忽略 404 (Not Found)，只打印真正的服务端故障 (如 500) 或 400 坏请求
		if len(c.Errors) > 0 || (status >= 400 && status != http.StatusNotFound) {
			fmt.Printf("[Error] %s | %d | %s | %s\n",
				time.Now().Format("15:04:05"),
				status,
				c.Request.Method,
				c.Request.URL.Path,
			)
		}
	})

	// 1. 处理 PUT 请求
	r.POST("/put", func(c *gin.Context) {
		metrics.requestTotal.Add(1)
		metrics.putTotal.Add(1)
		var req struct {
			Key   string `json:"key" binding:"required"`
			Value string `json:"value" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 轮询 3 个节点，寻找 Leader
		for i, kv := range kvStores {
			err := kv.Put(req.Key, req.Value)
			if err == nil {
				c.JSON(http.StatusOK, gin.H{"status": "success", "leader": i})
				return
			}
			if strings.Contains(err.Error(), "not leader") {
				observeStoreError(metrics, err)
				continue // 换下一个试试
			}
			observeStoreError(metrics, err)
			fmt.Printf("❌ [DEBUG] Put 失败，报错内容: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cluster has no leader right now"})
	})

	// 2. 处理 GET 请求
	r.GET("/get/:key", func(c *gin.Context) {
		metrics.requestTotal.Add(1)
		metrics.getTotal.Add(1)
		key := c.Param("key")

		// 轮询寻找 Leader
		for i, kv := range kvStores {
			val, exists, err := kv.Get(key) // 注意这里 Get 的签名需要修改，加上 error

			if err == nil {
				if !exists {
					c.JSON(http.StatusNotFound, gin.H{"error": "key not found", "leader": i})
					return
				}
				c.JSON(http.StatusOK, gin.H{"value": val, "leader": i})
				return
			}

			if strings.Contains(err.Error(), "not leader") {
				observeStoreError(metrics, err)
				continue // 不是 Leader，换下一个
			}
			observeStoreError(metrics, err)
			fmt.Printf("❌ [DEBUG] Get 失败，报错内容: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cluster has no leader right now"})
	})

	// 3. 处理 DELETE 请求
	r.DELETE("/delete/:key", func(c *gin.Context) {
		metrics.requestTotal.Add(1)
		metrics.deleteTotal.Add(1)
		key := c.Param("key")

		for i, kv := range kvStores {
			err := kv.Delete(key)
			if err == nil {
				c.JSON(http.StatusOK, gin.H{"status": "deleted", "leader": i})
				return
			}
			if strings.Contains(err.Error(), "not leader") {
				observeStoreError(metrics, err)
				continue
			}
			observeStoreError(metrics, err)
			fmt.Printf("❌ [DEBUG] DELETE 失败，报错内容: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cluster has no leader right now"})
	})

	r.GET("/debug/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, buildDebugMetrics(kvStores, metrics))
	})

	// 启动 HTTP 服务
	fmt.Printf("API Gateway running on http://localhost%s\n", port)
	if err := r.Run(port); err != nil {
		panic("Failed to start API Gateway: " + err.Error())
	}
}

func observeStoreError(metrics *gatewayMetrics, err error) {
	if err == nil {
		return
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "not leader") {
		metrics.notLeaderTotal.Add(1)
	}
	if strings.Contains(msg, "request timeout") {
		metrics.requestTimeoutTotal.Add(1)
	}
	if strings.Contains(msg, "leadership changed before commit") {
		metrics.leadershipChangedBeforeCommitTotal.Add(1)
	}
}

func buildDebugMetrics(kvStores []*store.KVStore, metrics *gatewayMetrics) debugMetricsResponse {
	currentLeader := -1
	currentTerm := 0
	commitIndex := 0
	lastApplied := 0
	logLength := 0
	activeWaitChans := 0
	var leaderChangeTotal uint64
	var electionStartedTotal uint64
	var termChangeTotal uint64
	var appendEntriesFailedTotal uint64
	var raftPersistCount uint64
	var raftPersistTotalMS float64
	var raftPersistMaxMS float64

	nodes := make([]nodeDebugMetrics, 0, len(kvStores))
	for i, kv := range kvStores {
		raftMetrics := kv.RaftMetrics()
		waitChans := kv.ActiveWaitChans()
		nodes = append(nodes, nodeDebugMetrics{
			NodeID:          i,
			ActiveWaitChans: waitChans,
			Raft:            raftMetrics,
		})

		activeWaitChans += waitChans
		leaderChangeTotal += raftMetrics.LeaderChangeTotal
		electionStartedTotal += raftMetrics.ElectionStartedTotal
		termChangeTotal += raftMetrics.TermChangeTotal
		appendEntriesFailedTotal += raftMetrics.AppendEntriesFailedTotal
		raftPersistCount += raftMetrics.RaftPersistCount
		raftPersistTotalMS += raftMetrics.RaftPersistTotalMS
		if raftMetrics.RaftPersistMaxMS > raftPersistMaxMS {
			raftPersistMaxMS = raftMetrics.RaftPersistMaxMS
		}
		if raftMetrics.CurrentTerm > currentTerm {
			currentTerm = raftMetrics.CurrentTerm
		}
		if raftMetrics.CommitIndex > commitIndex {
			commitIndex = raftMetrics.CommitIndex
		}
		if raftMetrics.LastApplied > lastApplied {
			lastApplied = raftMetrics.LastApplied
		}
		if raftMetrics.LogLength > logLength {
			logLength = raftMetrics.LogLength
		}
		if raftMetrics.IsLeader {
			currentLeader = i
			currentTerm = raftMetrics.CurrentTerm
		}
	}

	storeAPI := storeAPIMetricsSnapshot{
		RequestTotal:                       metrics.requestTotal.Load(),
		PutTotal:                           metrics.putTotal.Load(),
		GetTotal:                           metrics.getTotal.Load(),
		DeleteTotal:                        metrics.deleteTotal.Load(),
		HTTP500Total:                       metrics.http500Total.Load(),
		HTTP503Total:                       metrics.http503Total.Load(),
		RequestTimeoutTotal:                metrics.requestTimeoutTotal.Load(),
		LeadershipChangedBeforeCommitTotal: metrics.leadershipChangedBeforeCommitTotal.Load(),
		NotLeaderTotal:                     metrics.notLeaderTotal.Load(),
		ActiveWaitChans:                    activeWaitChans,
	}
	runtimeMetrics := readRuntimeMetrics()

	return debugMetricsResponse{
		Timestamp:                          time.Now().Format(time.RFC3339Nano),
		CurrentLeader:                      currentLeader,
		CurrentTerm:                        currentTerm,
		LeaderChangeTotal:                  leaderChangeTotal,
		ElectionStartedTotal:               electionStartedTotal,
		TermChangeTotal:                    termChangeTotal,
		AppendEntriesFailedTotal:           appendEntriesFailedTotal,
		RaftPersistCount:                   raftPersistCount,
		RaftPersistTotalMS:                 raftPersistTotalMS,
		RaftPersistMaxMS:                   raftPersistMaxMS,
		CommitIndex:                        commitIndex,
		LastApplied:                        lastApplied,
		LogLength:                          logLength,
		RequestTotal:                       storeAPI.RequestTotal,
		PutTotal:                           storeAPI.PutTotal,
		GetTotal:                           storeAPI.GetTotal,
		DeleteTotal:                        storeAPI.DeleteTotal,
		HTTP500Total:                       storeAPI.HTTP500Total,
		HTTP503Total:                       storeAPI.HTTP503Total,
		RequestTimeoutTotal:                storeAPI.RequestTimeoutTotal,
		LeadershipChangedBeforeCommitTotal: storeAPI.LeadershipChangedBeforeCommitTotal,
		NotLeaderTotal:                     storeAPI.NotLeaderTotal,
		ActiveWaitChans:                    activeWaitChans,
		GoroutineNum:                       runtimeMetrics.GoroutineNum,
		HeapAlloc:                          runtimeMetrics.HeapAlloc,
		NumGC:                              runtimeMetrics.NumGC,
		PauseTotalNS:                       runtimeMetrics.PauseTotalNS,
		LastGCPauseNS:                      runtimeMetrics.LastGCPauseNS,
		StoreAPI:                           storeAPI,
		Runtime:                            runtimeMetrics,
		Nodes:                              nodes,
	}
}

func readRuntimeMetrics() runtimeMetricsSnapshot {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	var lastPause uint64
	if mem.NumGC > 0 {
		lastPause = mem.PauseNs[int((mem.NumGC+255)%256)]
	}

	return runtimeMetricsSnapshot{
		GoroutineNum:  runtime.NumGoroutine(),
		HeapAlloc:     mem.HeapAlloc,
		NumGC:         mem.NumGC,
		PauseTotalNS:  mem.PauseTotalNs,
		LastGCPauseNS: lastPause,
	}
}
