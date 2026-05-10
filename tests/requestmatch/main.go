package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type client struct {
	baseURL string
	http    *http.Client
}

type debugMetrics struct {
	CurrentLeader        int    `json:"current_leader"`
	CurrentTerm          int    `json:"current_term"`
	LeaderChangeTotal    uint64 `json:"leader_change_total"`
	ElectionStartedTotal uint64 `json:"election_started_total"`
	TermChangeTotal      uint64 `json:"term_change_total"`
}

type requestResult struct {
	Success                       bool
	ResultMismatch                bool
	NotLeader                     bool
	RequestTimeout                bool
	LeadershipChangedBeforeCommit bool
	HTTP500                       bool
	HTTP503                       bool
	OtherError                    bool
}

type report struct {
	RunID                              string  `json:"run_id"`
	TotalRequests                      int     `json:"total_requests"`
	Concurrency                        int     `json:"concurrency"`
	Keyspace                           int     `json:"keyspace"`
	ValueSize                          int     `json:"value_size"`
	SuccessRequests                    int     `json:"success_requests"`
	NotLeaderTotal                     int     `json:"not_leader_total"`
	RequestTimeoutTotal                int     `json:"request_timeout_total"`
	LeadershipChangedBeforeCommitTotal int     `json:"leadership_changed_before_commit_total"`
	HTTP500Total                       int     `json:"http_500_total"`
	HTTP503Total                       int     `json:"http_503_total"`
	OtherErrorTotal                    int     `json:"other_error_total"`
	ErrorResultTotal                   int     `json:"error_result_total"`
	LeaderChangeDelta                  uint64  `json:"leader_change_delta"`
	ElectionStartedDelta               uint64  `json:"election_started_delta"`
	TermChangeDelta                    uint64  `json:"term_change_delta"`
	DurationSeconds                    float64 `json:"duration_seconds"`
	ForceElectionAttempts              int     `json:"force_election_attempts"`
}

func main() {
	runID := flag.String("runid", time.Now().Format("20060102_150405"), "run id")
	baseURL := flag.String("u", "http://localhost:8080", "API Gateway URL")
	totalRequests := flag.Int("n", 5000, "total verified GET requests")
	concurrency := flag.Int("c", 50, "concurrent clients")
	keyspace := flag.Int("keyspace", 1000, "number of keys to prefill and verify")
	valueSize := flag.Int("valuesize", 64, "value size in bytes")
	timeout := flag.Duration("timeout", 5*time.Second, "HTTP client timeout")
	electionInterval := flag.Duration("election-interval", 500*time.Millisecond, "forced election interval during request run")
	electionAttempts := flag.Int("elections", 8, "forced election attempts during request run")
	outDir := flag.String("outdir", "", "output directory")
	seed := flag.Int64("seed", 1, "random seed")
	flag.Parse()

	if err := validate(*totalRequests, *concurrency, *keyspace, *valueSize, *timeout, *electionInterval, *electionAttempts, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "invalid arguments: %v\n", err)
		os.Exit(2)
	}
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "create output directory: %v\n", err)
		os.Exit(1)
	}

	c := newClient(*baseURL, *timeout)
	expected := make(map[string]string, *keyspace)
	for i := 0; i < *keyspace; i++ {
		key := keyFor(i)
		value := valueFor(i, *valueSize)
		if err := c.put(key, value); err != nil {
			fmt.Fprintf(os.Stderr, "prefill %s: %v\n", key, err)
			os.Exit(1)
		}
		expected[key] = value
	}

	before, err := c.metrics()
	if err != nil {
		fmt.Fprintf(os.Stderr, "read metrics before: %v\n", err)
		os.Exit(1)
	}
	writeJSON(filepath.Join(*outDir, "metrics_before.json"), before)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	forceLog := filepath.Join(*outDir, "force_election.log")
	forceDone := make(chan int, 1)
	go func() {
		forceDone <- runForcedElections(ctx, c, *electionAttempts, *electionInterval, forceLog)
	}()

	start := time.Now()
	results := runRequests(c, expected, *totalRequests, *concurrency, *keyspace, *seed)
	cancel()
	forceAttempts := <-forceDone
	duration := time.Since(start)

	after, err := c.metrics()
	if err != nil {
		fmt.Fprintf(os.Stderr, "read metrics after: %v\n", err)
		os.Exit(1)
	}
	writeJSON(filepath.Join(*outDir, "metrics_after.json"), after)

	rep := summarize(*runID, *totalRequests, *concurrency, *keyspace, *valueSize, duration, forceAttempts, before, after, results)
	if err := writeJSON(filepath.Join(*outDir, "request_match_raw.json"), rep); err != nil {
		fmt.Fprintf(os.Stderr, "write raw report: %v\n", err)
		os.Exit(1)
	}
	if err := writeCSV(filepath.Join(*outDir, "request_match_results.csv"), rep); err != nil {
		fmt.Fprintf(os.Stderr, "write csv report: %v\n", err)
		os.Exit(1)
	}
	if err := writeMarkdown(filepath.Join(*outDir, "request_match_results.md"), rep); err != nil {
		fmt.Fprintf(os.Stderr, "write markdown report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Request match experiment finished: success=%d/%d, mismatches=%d, leader_changes=%d, term_changes=%d\n",
		rep.SuccessRequests, rep.TotalRequests, rep.ErrorResultTotal, rep.LeaderChangeDelta, rep.TermChangeDelta)
}

func validate(totalRequests, concurrency, keyspace, valueSize int, timeout, electionInterval time.Duration, elections int, outDir string) error {
	if totalRequests <= 0 {
		return fmt.Errorf("-n must be greater than 0")
	}
	if concurrency <= 0 {
		return fmt.Errorf("-c must be greater than 0")
	}
	if keyspace <= 0 {
		return fmt.Errorf("-keyspace must be greater than 0")
	}
	if valueSize <= 0 {
		return fmt.Errorf("-valuesize must be greater than 0")
	}
	if timeout <= 0 {
		return fmt.Errorf("-timeout must be greater than 0")
	}
	if electionInterval <= 0 {
		return fmt.Errorf("-election-interval must be greater than 0")
	}
	if elections < 0 {
		return fmt.Errorf("-elections must be greater than or equal to 0")
	}
	if outDir == "" {
		return fmt.Errorf("-outdir is required")
	}
	return nil
}

func newClient(baseURL string, timeout time.Duration) *client {
	return &client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *client) put(key, value string) error {
	payload, _ := json.Marshal(map[string]string{"key": key, "value": value})
	resp, err := c.http.Post(c.baseURL+"/put", "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *client) get(key string) (int, string, error) {
	resp, err := c.http.Get(c.baseURL + "/get/" + url.PathEscape(key))
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), nil
}

func (c *client) metrics() (debugMetrics, error) {
	var metrics debugMetrics
	resp, err := c.http.Get(c.baseURL + "/debug/metrics")
	if err != nil {
		return metrics, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return metrics, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		return metrics, err
	}
	return metrics, nil
}

func (c *client) forceElection() (int, string, error) {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/debug/force-election", nil)
	if err != nil {
		return 0, "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), nil
}

func runForcedElections(ctx context.Context, c *client, attempts int, interval time.Duration, logPath string) int {
	var log strings.Builder
	completed := 0
	for i := 0; i < attempts; i++ {
		select {
		case <-ctx.Done():
			_ = os.WriteFile(logPath, []byte(log.String()), 0644)
			return completed
		case <-time.After(interval):
		}
		status, body, err := c.forceElection()
		completed++
		if err != nil {
			fmt.Fprintf(&log, "attempt=%d error=%v\n", completed, err)
			continue
		}
		fmt.Fprintf(&log, "attempt=%d status=%d body=%s\n", completed, status, strings.TrimSpace(body))
	}
	_ = os.WriteFile(logPath, []byte(log.String()), 0644)
	return completed
}

func runRequests(c *client, expected map[string]string, total, concurrency, keyspace int, seed int64) []requestResult {
	taskCh := make(chan int)
	resultCh := make(chan requestResult, total)
	workerCount := concurrency
	if workerCount > total {
		workerCount = total
	}

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		workerID := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(seed + int64(workerID)*1000003))
			for range taskCh {
				key := keyFor(rng.Intn(keyspace))
				status, body, err := c.get(key)
				resultCh <- classifyResult(status, body, err, expected[key])
			}
		}()
	}

	for i := 0; i < total; i++ {
		taskCh <- i
	}
	close(taskCh)
	wg.Wait()
	close(resultCh)

	results := make([]requestResult, 0, total)
	for result := range resultCh {
		results = append(results, result)
	}
	return results
}

func classifyResult(status int, body string, err error, expected string) requestResult {
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded") {
			return requestResult{RequestTimeout: true}
		}
		return requestResult{OtherError: true}
	}

	lowerBody := strings.ToLower(body)
	if strings.Contains(lowerBody, "leadership changed before commit") {
		return requestResult{LeadershipChangedBeforeCommit: true, HTTP500: status == http.StatusInternalServerError}
	}
	if strings.Contains(lowerBody, "request timeout") {
		return requestResult{RequestTimeout: true, HTTP500: status == http.StatusInternalServerError}
	}
	if strings.Contains(lowerBody, "not leader") {
		return requestResult{NotLeader: true}
	}
	if status == http.StatusServiceUnavailable {
		return requestResult{HTTP503: true}
	}
	if status == http.StatusInternalServerError {
		return requestResult{HTTP500: true}
	}
	if status != http.StatusOK {
		return requestResult{OtherError: true}
	}

	var payload struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return requestResult{ResultMismatch: true}
	}
	if payload.Value != expected {
		return requestResult{ResultMismatch: true}
	}
	return requestResult{Success: true}
}

func summarize(runID string, totalRequests, concurrency, keyspace, valueSize int, duration time.Duration, forceAttempts int, before, after debugMetrics, results []requestResult) report {
	rep := report{
		RunID:                 runID,
		TotalRequests:         totalRequests,
		Concurrency:           concurrency,
		Keyspace:              keyspace,
		ValueSize:             valueSize,
		DurationSeconds:       duration.Seconds(),
		ForceElectionAttempts: forceAttempts,
		LeaderChangeDelta:     delta(before.LeaderChangeTotal, after.LeaderChangeTotal),
		ElectionStartedDelta:  delta(before.ElectionStartedTotal, after.ElectionStartedTotal),
		TermChangeDelta:       delta(before.TermChangeTotal, after.TermChangeTotal),
	}
	for _, result := range results {
		if result.Success {
			rep.SuccessRequests++
		}
		if result.ResultMismatch {
			rep.ErrorResultTotal++
		}
		if result.NotLeader {
			rep.NotLeaderTotal++
		}
		if result.RequestTimeout {
			rep.RequestTimeoutTotal++
		}
		if result.LeadershipChangedBeforeCommit {
			rep.LeadershipChangedBeforeCommitTotal++
		}
		if result.HTTP500 {
			rep.HTTP500Total++
		}
		if result.HTTP503 {
			rep.HTTP503Total++
		}
		if result.OtherError {
			rep.OtherErrorTotal++
		}
	}
	return rep
}

func writeCSV(path string, rep report) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.Write([]string{
		"run_id",
		"total_requests",
		"success_requests",
		"not_leader_total",
		"request_timeout_total",
		"leadership_changed_before_commit_total",
		"http_500_total",
		"http_503_total",
		"error_result_total",
		"other_error_total",
		"leader_change_delta",
		"election_started_delta",
		"term_change_delta",
		"force_election_attempts",
		"duration_seconds",
	}); err != nil {
		return err
	}
	if err := writer.Write([]string{
		rep.RunID,
		fmt.Sprintf("%d", rep.TotalRequests),
		fmt.Sprintf("%d", rep.SuccessRequests),
		fmt.Sprintf("%d", rep.NotLeaderTotal),
		fmt.Sprintf("%d", rep.RequestTimeoutTotal),
		fmt.Sprintf("%d", rep.LeadershipChangedBeforeCommitTotal),
		fmt.Sprintf("%d", rep.HTTP500Total),
		fmt.Sprintf("%d", rep.HTTP503Total),
		fmt.Sprintf("%d", rep.ErrorResultTotal),
		fmt.Sprintf("%d", rep.OtherErrorTotal),
		fmt.Sprintf("%d", rep.LeaderChangeDelta),
		fmt.Sprintf("%d", rep.ElectionStartedDelta),
		fmt.Sprintf("%d", rep.TermChangeDelta),
		fmt.Sprintf("%d", rep.ForceElectionAttempts),
		fmt.Sprintf("%.3f", rep.DurationSeconds),
	}); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

func writeMarkdown(path string, rep report) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Request Match Experiment\n\n")
	fmt.Fprintf(&b, "| Metric | Value |\n")
	fmt.Fprintf(&b, "| --- | --- |\n")
	fmt.Fprintf(&b, "| Run ID | %s |\n", rep.RunID)
	fmt.Fprintf(&b, "| Total requests | %d |\n", rep.TotalRequests)
	fmt.Fprintf(&b, "| Success requests | %d |\n", rep.SuccessRequests)
	fmt.Fprintf(&b, "| Not leader | %d |\n", rep.NotLeaderTotal)
	fmt.Fprintf(&b, "| Request timeout | %d |\n", rep.RequestTimeoutTotal)
	fmt.Fprintf(&b, "| Leadership changed before commit | %d |\n", rep.LeadershipChangedBeforeCommitTotal)
	fmt.Fprintf(&b, "| HTTP 500 | %d |\n", rep.HTTP500Total)
	fmt.Fprintf(&b, "| HTTP 503 | %d |\n", rep.HTTP503Total)
	fmt.Fprintf(&b, "| Error result count | %d |\n", rep.ErrorResultTotal)
	fmt.Fprintf(&b, "| Other errors | %d |\n", rep.OtherErrorTotal)
	fmt.Fprintf(&b, "| Leader changes | %d |\n", rep.LeaderChangeDelta)
	fmt.Fprintf(&b, "| Elections started | %d |\n", rep.ElectionStartedDelta)
	fmt.Fprintf(&b, "| Term changes | %d |\n", rep.TermChangeDelta)
	fmt.Fprintf(&b, "| Force election attempts | %d |\n", rep.ForceElectionAttempts)
	fmt.Fprintf(&b, "| Duration seconds | %.3f |\n", rep.DurationSeconds)
	fmt.Fprintf(&b, "\nError result count is the number of HTTP 200 responses whose returned value did not match the requested key's expected value. The ideal value is 0.\n")
	return os.WriteFile(path, []byte(b.String()), 0644)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func keyFor(i int) string {
	return fmt.Sprintf("match_key_%06d", i)
}

func valueFor(i, size int) string {
	prefix := fmt.Sprintf("match_value_%06d_", i)
	if len(prefix) >= size {
		return prefix[:size]
	}
	return prefix + strings.Repeat("x", size-len(prefix))
}

func delta(before, after uint64) uint64 {
	if after < before {
		return 0
	}
	return after - before
}
