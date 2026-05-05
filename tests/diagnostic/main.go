package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type benchmarkReport struct {
	SuccessRatePct float64        `json:"success_rate_percent"`
	SuccessQPS     float64        `json:"success_qps"`
	P99Latency     string         `json:"p99_latency"`
	P99LatencyMS   float64        `json:"p99_latency_ms"`
	ErrorCounts    map[string]int `json:"error_counts"`
}

type debugMetrics struct {
	CurrentLeader                      int     `json:"current_leader"`
	CurrentTerm                        int     `json:"current_term"`
	LeaderChangeTotal                  uint64  `json:"leader_change_total"`
	ElectionStartedTotal               uint64  `json:"election_started_total"`
	TermChangeTotal                    uint64  `json:"term_change_total"`
	AppendEntriesFailedTotal           uint64  `json:"append_entries_failed_total"`
	RaftPersistCount                   uint64  `json:"raft_persist_count"`
	RaftPersistTotalMS                 float64 `json:"raft_persist_total_ms"`
	RaftPersistMaxMS                   float64 `json:"raft_persist_max_ms"`
	RequestTotal                       uint64  `json:"request_total"`
	HTTP500Total                       uint64  `json:"http_500_total"`
	HTTP503Total                       uint64  `json:"http_503_total"`
	RequestTimeoutTotal                uint64  `json:"request_timeout_total"`
	LeadershipChangedBeforeCommitTotal uint64  `json:"leadership_changed_before_commit_total"`
	NotLeaderTotal                     uint64  `json:"not_leader_total"`
	ActiveWaitChans                    int     `json:"active_wait_chans"`
	GoroutineNum                       int     `json:"goroutine_num"`
	HeapAlloc                          uint64  `json:"heap_alloc"`
	NumGC                              uint32  `json:"num_gc"`
	PauseTotalNS                       uint64  `json:"pause_total_ns"`
	LastGCPauseNS                      uint64  `json:"last_gc_pause_ns"`
}

type fileStats struct {
	Count int
	Bytes int64
}

func main() {
	mode := flag.String("mode", "", "summary or lsm-stats")
	caseName := flag.String("case", "", "diagnostic case name")
	resultPath := flag.String("result", "", "benchmark result.json path")
	beforePath := flag.String("before", "", "metrics_before.json path")
	afterPath := flag.String("after", "", "metrics_after.json path")
	dataDir := flag.String("data", "data", "cluster data directory")
	serverLog := flag.String("server-log", "", "server log path")
	outPath := flag.String("out", "", "output markdown path")
	flag.Parse()

	var err error
	switch *mode {
	case "summary":
		err = writeMetricsSummary(*caseName, *resultPath, *beforePath, *afterPath, *outPath)
	case "lsm-stats":
		err = writeLSMFileStats(*caseName, *dataDir, *serverLog, *outPath)
	default:
		err = fmt.Errorf("-mode must be summary or lsm-stats")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "diagnostic: %v\n", err)
		os.Exit(1)
	}
}

func writeMetricsSummary(caseName, resultPath, beforePath, afterPath, outPath string) error {
	var report benchmarkReport
	if err := readJSONFile(resultPath, &report); err != nil {
		return err
	}
	var before debugMetrics
	if err := readJSONFile(beforePath, &before); err != nil {
		return err
	}
	var after debugMetrics
	if err := readJSONFile(afterPath, &after); err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Metrics Summary\n\n")
	fmt.Fprintf(&b, "- Case: %s\n\n", caseName)
	fmt.Fprintf(&b, "| Metric | Value |\n")
	fmt.Fprintf(&b, "| --- | --- |\n")
	fmt.Fprintf(&b, "| Success rate | %.2f%% |\n", report.SuccessRatePct)
	fmt.Fprintf(&b, "| Success QPS | %.2f |\n", report.SuccessQPS)
	fmt.Fprintf(&b, "| P99 latency | %s (%.3f ms) |\n", report.P99Latency, report.P99LatencyMS)
	fmt.Fprintf(&b, "| HTTP 503 count | %d |\n", report.ErrorCounts["http_503"])
	fmt.Fprintf(&b, "| HTTP 500 count | %d |\n", report.ErrorCounts["http_500"])
	fmt.Fprintf(&b, "| Request timeout count | %d benchmark errors, %d metrics delta |\n",
		report.ErrorCounts["request_timeout"], deltaUint64(before.RequestTimeoutTotal, after.RequestTimeoutTotal))
	fmt.Fprintf(&b, "| Leader changes | %d |\n", deltaUint64(before.LeaderChangeTotal, after.LeaderChangeTotal))
	fmt.Fprintf(&b, "| Elections started | %d |\n", deltaUint64(before.ElectionStartedTotal, after.ElectionStartedTotal))
	fmt.Fprintf(&b, "| Term changes | %d |\n", deltaUint64(before.TermChangeTotal, after.TermChangeTotal))
	fmt.Fprintf(&b, "| AppendEntries failed | %d |\n", deltaUint64(before.AppendEntriesFailedTotal, after.AppendEntriesFailedTotal))
	fmt.Fprintf(&b, "| Not leader attempts | %d |\n", deltaUint64(before.NotLeaderTotal, after.NotLeaderTotal))
	fmt.Fprintf(&b, "| Leadership changed before commit | %d |\n", deltaUint64(before.LeadershipChangedBeforeCommitTotal, after.LeadershipChangedBeforeCommitTotal))
	fmt.Fprintf(&b, "| Raft persist count | %d |\n", deltaUint64(before.RaftPersistCount, after.RaftPersistCount))
	fmt.Fprintf(&b, "| Raft persist total ms | %.3f -> %.3f (delta %.3f) |\n",
		before.RaftPersistTotalMS, after.RaftPersistTotalMS, deltaFloat64(before.RaftPersistTotalMS, after.RaftPersistTotalMS))
	fmt.Fprintf(&b, "| Raft persist max ms | %.3f -> %.3f |\n", before.RaftPersistMaxMS, after.RaftPersistMaxMS)
	fmt.Fprintf(&b, "| Current leader | %d -> %d |\n", before.CurrentLeader, after.CurrentLeader)
	fmt.Fprintf(&b, "| Current term | %d -> %d |\n", before.CurrentTerm, after.CurrentTerm)
	fmt.Fprintf(&b, "| Active wait channels | %d -> %d |\n", before.ActiveWaitChans, after.ActiveWaitChans)
	fmt.Fprintf(&b, "| Goroutine count | %d -> %d (delta %+d) |\n",
		before.GoroutineNum, after.GoroutineNum, after.GoroutineNum-before.GoroutineNum)
	fmt.Fprintf(&b, "| Heap alloc bytes | %d -> %d (delta %+d) |\n",
		before.HeapAlloc, after.HeapAlloc, int64(after.HeapAlloc)-int64(before.HeapAlloc))
	fmt.Fprintf(&b, "| GC count | %d -> %d (delta %d) |\n",
		before.NumGC, after.NumGC, deltaUint32(before.NumGC, after.NumGC))
	fmt.Fprintf(&b, "| GC pause total ns | %d -> %d (delta %d) |\n",
		before.PauseTotalNS, after.PauseTotalNS, deltaUint64(before.PauseTotalNS, after.PauseTotalNS))
	fmt.Fprintf(&b, "| Last GC pause ns | %d -> %d |\n", before.LastGCPauseNS, after.LastGCPauseNS)

	return os.WriteFile(outPath, []byte(b.String()), 0644)
}

func writeLSMFileStats(caseName, dataDir, serverLog, outPath string) error {
	nodes, err := os.ReadDir(dataDir)
	if err != nil {
		return err
	}

	var nodeNames []string
	for _, node := range nodes {
		if node.IsDir() && strings.HasPrefix(node.Name(), "node_") {
			nodeNames = append(nodeNames, node.Name())
		}
	}
	sort.Strings(nodeNames)

	compactionLogs := countCompactionLogEntries(serverLog)

	var b strings.Builder
	fmt.Fprintf(&b, "# LSM File Stats\n\n")
	fmt.Fprintf(&b, "- Case: %s\n", caseName)
	fmt.Fprintf(&b, "- Data directory: %s\n", dataDir)
	fmt.Fprintf(&b, "- Compaction log entries in server.log: %d\n", compactionLogs)
	fmt.Fprintf(&b, "- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.\n\n")
	fmt.Fprintf(&b, "| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |\n")
	fmt.Fprintf(&b, "| --- | --- | --- | --- | --- | --- | --- |\n")

	for _, name := range nodeNames {
		nodeDir := filepath.Join(dataDir, name)
		walStats := collectGlobStats(filepath.Join(nodeDir, "data", "wal", "*.log"))
		sstStats := collectGlobStats(filepath.Join(nodeDir, "data", "sst", "*.sst"))
		manifestSize := fileSize(filepath.Join(nodeDir, "data", "manifest.log"))
		totalSize := dirSize(nodeDir)

		fmt.Fprintf(&b, "| %s | %s | %d | %s | %d | %s | %s |\n",
			name,
			formatBytes(totalSize),
			walStats.Count,
			formatBytes(walStats.Bytes),
			sstStats.Count,
			formatBytes(sstStats.Bytes),
			formatBytes(manifestSize))
	}

	if len(nodeNames) == 0 {
		fmt.Fprintf(&b, "| none | 0 B | 0 | 0 B | 0 | 0 B | 0 B |\n")
	}

	return os.WriteFile(outPath, []byte(b.String()), 0644)
}

func readJSONFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func deltaUint64(before, after uint64) uint64 {
	if after < before {
		return 0
	}
	return after - before
}

func deltaUint32(before, after uint32) uint32 {
	if after < before {
		return 0
	}
	return after - before
}

func deltaFloat64(before, after float64) float64 {
	if after < before {
		return 0
	}
	return after - before
}

func collectGlobStats(pattern string) fileStats {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fileStats{}
	}

	var stats fileStats
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || info.IsDir() {
			continue
		}
		stats.Count++
		stats.Bytes += info.Size()
	}
	return stats
}

func dirSize(path string) int64 {
	var total int64
	_ = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return 0
	}
	return info.Size()
}

func countCompactionLogEntries(path string) int {
	if path == "" {
		return 0
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return strings.Count(strings.ToLower(string(data)), "compaction")
}

func formatBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div := int64(unit)
	exp := 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB (%d B)", float64(size)/float64(div), "KMGTPE"[exp], size)
}
