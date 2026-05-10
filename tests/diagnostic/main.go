package main

import (
	"encoding/csv"
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
	RunID            string         `json:"run_id"`
	WorkloadName     string         `json:"workload_name"`
	ReadMode         string         `json:"read_mode"`
	EnableCompaction bool           `json:"enable_compaction"`
	Concurrency      int            `json:"concurrency"`
	Seed             int64          `json:"seed"`
	TotalRequests    int            `json:"total_requests"`
	Keyspace         int            `json:"keyspace"`
	ValueSize        int            `json:"valuesize"`
	SuccessRatePct   float64        `json:"success_rate_percent"`
	SuccessQPS       float64        `json:"success_qps"`
	DurationSeconds  float64        `json:"total_duration_seconds"`
	GetSuccess       int            `json:"get_success"`
	GetSuccessQPS    float64        `json:"get_success_qps"`
	GetP99MS         float64        `json:"get_p99_ms"`
	GetP999MS        float64        `json:"get_p999_ms"`
	P50Latency       string         `json:"p50_latency"`
	P50LatencyMS     float64        `json:"p50_latency_ms"`
	P90Latency       string         `json:"p90_latency"`
	P90LatencyMS     float64        `json:"p90_latency_ms"`
	P99Latency       string         `json:"p99_latency"`
	P99LatencyMS     float64        `json:"p99_latency_ms"`
	P999Latency      string         `json:"p999_latency"`
	P999LatencyMS    float64        `json:"p999_latency_ms"`
	ErrorCounts      map[string]int `json:"error_counts"`
	Config           struct {
		ReadMode         string `json:"read_mode"`
		EnableCompaction bool   `json:"enable_compaction"`
	} `json:"config"`
}

type lsmDebugMetrics struct {
	CompactionEnabled           bool    `json:"compaction_enabled"`
	GetTotal                    uint64  `json:"get_total"`
	GetSSTableFilesTouchedTotal uint64  `json:"get_sstable_files_touched_total"`
	GetSSTableFilesTouchedAvg   float64 `json:"get_sstable_files_touched_avg"`
	GetSSTableFilesTouchedMax   uint64  `json:"get_sstable_files_touched_max"`
	CompactionTotal             uint64  `json:"compaction_total"`
	CompactionInputFilesTotal   uint64  `json:"compaction_input_files_total"`
	CompactionOutputFilesTotal  uint64  `json:"compaction_output_files_total"`
	CompactionTotalMS           float64 `json:"compaction_total_ms"`
	CompactionMaxMS             float64 `json:"compaction_max_ms"`
	CompactionLastMS            float64 `json:"compaction_last_ms"`
	CompactionFailedTotal       uint64  `json:"compaction_failed_total"`
	SSTableCount                int     `json:"sstable_count"`
}

type debugMetrics struct {
	CurrentLeader                      int             `json:"current_leader"`
	CurrentTerm                        int             `json:"current_term"`
	LeaderChangeTotal                  uint64          `json:"leader_change_total"`
	ElectionStartedTotal               uint64          `json:"election_started_total"`
	TermChangeTotal                    uint64          `json:"term_change_total"`
	AppendEntriesFailedTotal           uint64          `json:"append_entries_failed_total"`
	RaftPersistCount                   uint64          `json:"raft_persist_count"`
	RaftPersistTotalMS                 float64         `json:"raft_persist_total_ms"`
	RaftPersistMaxMS                   float64         `json:"raft_persist_max_ms"`
	RequestTotal                       uint64          `json:"request_total"`
	HTTP500Total                       uint64          `json:"http_500_total"`
	HTTP503Total                       uint64          `json:"http_503_total"`
	RequestTimeoutTotal                uint64          `json:"request_timeout_total"`
	LeadershipChangedBeforeCommitTotal uint64          `json:"leadership_changed_before_commit_total"`
	NotLeaderTotal                     uint64          `json:"not_leader_total"`
	ActiveWaitChans                    int             `json:"active_wait_chans"`
	GoroutineNum                       int             `json:"goroutine_num"`
	HeapAlloc                          uint64          `json:"heap_alloc"`
	NumGC                              uint32          `json:"num_gc"`
	PauseTotalNS                       uint64          `json:"pause_total_ns"`
	LastGCPauseNS                      uint64          `json:"last_gc_pause_ns"`
	LSM                                lsmDebugMetrics `json:"lsm"`
}

type fileStats struct {
	Count int
	Bytes int64
}

type clusterFileStats struct {
	DataBytes    int64
	WALFiles     fileStats
	SSTableFiles fileStats
}

func main() {
	mode := flag.String("mode", "", "summary, summary-csv-row, compaction-summary, compaction-csv-row, or lsm-stats")
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
	case "summary-csv-row":
		err = writeSummaryCSVRow(*resultPath, *beforePath, *afterPath, *outPath)
	case "compaction-summary":
		err = writeCompactionSummary(*caseName, *resultPath, *beforePath, *afterPath, *dataDir, *outPath)
	case "compaction-csv-row":
		err = writeCompactionCSVRow(*resultPath, *beforePath, *afterPath, *dataDir, *outPath)
	case "lsm-stats":
		err = writeLSMFileStats(*caseName, *dataDir, *serverLog, *outPath)
	default:
		err = fmt.Errorf("-mode must be summary, summary-csv-row, compaction-summary, compaction-csv-row, or lsm-stats")
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
	fmt.Fprintf(&b, "| Read mode | %s |\n", report.readMode())
	fmt.Fprintf(&b, "| Workload | %s |\n", report.WorkloadName)
	fmt.Fprintf(&b, "| Concurrency | %d |\n", report.Concurrency)
	fmt.Fprintf(&b, "| Seed | %d |\n", report.Seed)
	fmt.Fprintf(&b, "| Success rate | %.2f%% |\n", report.SuccessRatePct)
	fmt.Fprintf(&b, "| Success QPS | %.2f |\n", report.SuccessQPS)
	fmt.Fprintf(&b, "| P50 latency | %s (%.3f ms) |\n", report.P50Latency, report.P50LatencyMS)
	fmt.Fprintf(&b, "| P90 latency | %s (%.3f ms) |\n", report.P90Latency, report.P90LatencyMS)
	fmt.Fprintf(&b, "| P99 latency | %s (%.3f ms) |\n", report.P99Latency, report.P99LatencyMS)
	fmt.Fprintf(&b, "| P99.9 latency | %s (%.3f ms) |\n", report.P999Latency, report.P999LatencyMS)
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
	fmt.Fprintf(&b, "| LSM compaction enabled | %t |\n", after.LSM.CompactionEnabled)
	fmt.Fprintf(&b, "| LSM SSTable count | %d -> %d |\n", before.LSM.SSTableCount, after.LSM.SSTableCount)
	fmt.Fprintf(&b, "| LSM Get SSTable touched avg | %.3f |\n", deltaLSMTouchedAvg(before.LSM, after.LSM))
	fmt.Fprintf(&b, "| LSM compactions | %d |\n", deltaUint64(before.LSM.CompactionTotal, after.LSM.CompactionTotal))
	fmt.Fprintf(&b, "| LSM compaction total ms | %.3f |\n", deltaFloat64(before.LSM.CompactionTotalMS, after.LSM.CompactionTotalMS))

	return os.WriteFile(outPath, []byte(b.String()), 0644)
}

func writeSummaryCSVRow(resultPath, beforePath, afterPath, outPath string) error {
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

	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	row := []string{
		report.readMode(),
		report.WorkloadName,
		fmt.Sprintf("%d", report.Concurrency),
		fmt.Sprintf("%d", report.Seed),
		fmt.Sprintf("%.4f", report.SuccessRatePct),
		fmt.Sprintf("%.4f", report.SuccessQPS),
		fmt.Sprintf("%.4f", report.P50LatencyMS),
		fmt.Sprintf("%.4f", report.P90LatencyMS),
		fmt.Sprintf("%.4f", report.P99LatencyMS),
		fmt.Sprintf("%.4f", report.P999LatencyMS),
		fmt.Sprintf("%d", report.ErrorCounts["http_500"]),
		fmt.Sprintf("%d", report.ErrorCounts["http_503"]),
		fmt.Sprintf("%d", report.ErrorCounts["request_timeout"]),
		fmt.Sprintf("%d", deltaUint64(before.LeaderChangeTotal, after.LeaderChangeTotal)),
		fmt.Sprintf("%d", deltaUint64(before.ElectionStartedTotal, after.ElectionStartedTotal)),
		fmt.Sprintf("%d", deltaUint64(before.TermChangeTotal, after.TermChangeTotal)),
	}

	writer := csv.NewWriter(file)
	if err := writer.Write(row); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

func writeCompactionSummary(caseName, resultPath, beforePath, afterPath, dataDir, outPath string) error {
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
	files := collectClusterFileStats(dataDir)

	var b strings.Builder
	fmt.Fprintf(&b, "# Compaction Ablation Summary\n\n")
	fmt.Fprintf(&b, "- Case: %s\n\n", caseName)
	fmt.Fprintf(&b, "| Metric | Value |\n")
	fmt.Fprintf(&b, "| --- | --- |\n")
	fmt.Fprintf(&b, "| Enable compaction | %t |\n", report.enableCompaction())
	fmt.Fprintf(&b, "| Workload | %s |\n", report.WorkloadName)
	fmt.Fprintf(&b, "| Concurrency | %d |\n", report.Concurrency)
	fmt.Fprintf(&b, "| Seed | %d |\n", report.Seed)
	fmt.Fprintf(&b, "| Total requests | %d |\n", report.TotalRequests)
	fmt.Fprintf(&b, "| Keyspace | %d |\n", report.Keyspace)
	fmt.Fprintf(&b, "| Value size | %d |\n", report.ValueSize)
	fmt.Fprintf(&b, "| Success rate | %.2f%% |\n", report.SuccessRatePct)
	fmt.Fprintf(&b, "| Success QPS | %.2f |\n", report.SuccessQPS)
	fmt.Fprintf(&b, "| Get success QPS | %.2f |\n", report.getSuccessQPS())
	fmt.Fprintf(&b, "| Get P99 latency ms | %.3f |\n", report.GetP99MS)
	fmt.Fprintf(&b, "| Get P99.9 latency ms | %.3f |\n", report.GetP999MS)
	fmt.Fprintf(&b, "| SSTable files total | %d |\n", files.SSTableFiles.Count)
	fmt.Fprintf(&b, "| Data size bytes | %d |\n", files.DataBytes)
	fmt.Fprintf(&b, "| WAL files total | %d |\n", files.WALFiles.Count)
	fmt.Fprintf(&b, "| WAL size bytes | %d |\n", files.WALFiles.Bytes)
	fmt.Fprintf(&b, "| Avg SSTable files touched per Get | %.3f |\n", deltaLSMTouchedAvg(before.LSM, after.LSM))
	fmt.Fprintf(&b, "| Compaction count delta | %d |\n", deltaUint64(before.LSM.CompactionTotal, after.LSM.CompactionTotal))
	fmt.Fprintf(&b, "| Compaction input files delta | %d |\n", deltaUint64(before.LSM.CompactionInputFilesTotal, after.LSM.CompactionInputFilesTotal))
	fmt.Fprintf(&b, "| Compaction output files delta | %d |\n", deltaUint64(before.LSM.CompactionOutputFilesTotal, after.LSM.CompactionOutputFilesTotal))
	fmt.Fprintf(&b, "| Compaction total ms delta | %.3f |\n", deltaFloat64(before.LSM.CompactionTotalMS, after.LSM.CompactionTotalMS))
	fmt.Fprintf(&b, "| Compaction max ms | %.3f |\n", after.LSM.CompactionMaxMS)
	fmt.Fprintf(&b, "| Compaction failed delta | %d |\n", deltaUint64(before.LSM.CompactionFailedTotal, after.LSM.CompactionFailedTotal))

	return os.WriteFile(outPath, []byte(b.String()), 0644)
}

func writeCompactionCSVRow(resultPath, beforePath, afterPath, dataDir, outPath string) error {
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
	files := collectClusterFileStats(dataDir)

	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	row := []string{
		fmt.Sprintf("%t", report.enableCompaction()),
		report.WorkloadName,
		fmt.Sprintf("%d", report.Concurrency),
		fmt.Sprintf("%d", report.Seed),
		fmt.Sprintf("%d", report.TotalRequests),
		fmt.Sprintf("%d", report.Keyspace),
		fmt.Sprintf("%d", report.ValueSize),
		fmt.Sprintf("%.4f", report.SuccessRatePct),
		fmt.Sprintf("%.4f", report.SuccessQPS),
		fmt.Sprintf("%.4f", report.getSuccessQPS()),
		fmt.Sprintf("%.4f", report.GetP99MS),
		fmt.Sprintf("%.4f", report.GetP999MS),
		fmt.Sprintf("%d", files.SSTableFiles.Count),
		fmt.Sprintf("%d", files.DataBytes),
		fmt.Sprintf("%d", files.WALFiles.Count),
		fmt.Sprintf("%d", files.WALFiles.Bytes),
		fmt.Sprintf("%.4f", deltaLSMTouchedAvg(before.LSM, after.LSM)),
		fmt.Sprintf("%d", deltaUint64(before.LSM.CompactionTotal, after.LSM.CompactionTotal)),
		fmt.Sprintf("%d", deltaUint64(before.LSM.CompactionInputFilesTotal, after.LSM.CompactionInputFilesTotal)),
		fmt.Sprintf("%d", deltaUint64(before.LSM.CompactionOutputFilesTotal, after.LSM.CompactionOutputFilesTotal)),
		fmt.Sprintf("%.4f", deltaFloat64(before.LSM.CompactionTotalMS, after.LSM.CompactionTotalMS)),
		fmt.Sprintf("%.4f", after.LSM.CompactionMaxMS),
	}

	writer := csv.NewWriter(file)
	if err := writer.Write(row); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

func (r benchmarkReport) readMode() string {
	if r.ReadMode != "" {
		return r.ReadMode
	}
	if r.Config.ReadMode != "" {
		return r.Config.ReadMode
	}
	return "unknown"
}

func (r benchmarkReport) enableCompaction() bool {
	return r.EnableCompaction || r.Config.EnableCompaction
}

func (r benchmarkReport) getSuccessQPS() float64 {
	if r.GetSuccessQPS > 0 {
		return r.GetSuccessQPS
	}
	if r.DurationSeconds <= 0 {
		return 0
	}
	return float64(r.GetSuccess) / r.DurationSeconds
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

func deltaLSMTouchedAvg(before, after lsmDebugMetrics) float64 {
	getDelta := deltaUint64(before.GetTotal, after.GetTotal)
	if getDelta == 0 {
		return 0
	}
	touchedDelta := deltaUint64(before.GetSSTableFilesTouchedTotal, after.GetSSTableFilesTouchedTotal)
	return float64(touchedDelta) / float64(getDelta)
}

func collectClusterFileStats(dataDir string) clusterFileStats {
	var stats clusterFileStats
	nodes, err := os.ReadDir(dataDir)
	if err != nil {
		return stats
	}

	for _, node := range nodes {
		if !node.IsDir() || !strings.HasPrefix(node.Name(), "node_") {
			continue
		}
		nodeDir := filepath.Join(dataDir, node.Name())
		walStats := collectGlobStats(filepath.Join(nodeDir, "data", "wal", "*.log"))
		sstStats := collectGlobStats(filepath.Join(nodeDir, "data", "sst", "*.sst"))
		stats.DataBytes += dirSize(nodeDir)
		stats.WALFiles.Count += walStats.Count
		stats.WALFiles.Bytes += walStats.Bytes
		stats.SSTableFiles.Count += sstStats.Count
		stats.SSTableFiles.Bytes += sstStats.Bytes
	}
	return stats
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
