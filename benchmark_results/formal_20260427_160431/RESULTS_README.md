# Benchmark Results Layout

This directory contains the complete formal benchmark result set.

## Root Files

- `suite_manifest.tsv`
  - Index of all benchmark runs.
  - Columns: `workload`, `concurrency`, `seed`, `write_ratio`, `prefill`, `run_id`, `result_dir`.
  - Use this file to locate each run's result directory.

- `suite_command.txt`
  - Records the global suite configuration used for this run.
  - Includes API URL, request count, keyspace, value size, warmup count, timeout, workloads, concurrency list, and seed list.

- `RESULTS_README.md`
  - This file.

## Workload Directories

Each workload has its own directory:

- `write_100/`
  - Write-only workload.
  - `write_ratio = 1.0`, `prefill = false`.

- `read_100/`
  - Read-only workload.
  - `write_ratio = 0.0`, `prefill = true`.

- `mixed_50_50/`
  - Mixed read/write workload.
  - `write_ratio = 0.5`, `prefill = true`.

- `read95_write5/`
  - Read-heavy workload.
  - `write_ratio = 0.05`, `prefill = true`.

Inside each workload directory, each run is stored as:

```text
c<concurrency>_seed<seed>/
```

Example:

```text
read_100/c10_seed2/
```

## Per-Run Files

Each per-run directory contains:

- `result.json`
  - Structured benchmark result for this run.
  - This is the main file for table extraction.

- `benchmark.log`
  - Human-readable console output from the benchmark client.

- `server.log`
  - Server-side log captured while this run was executing.

- `command.txt`
  - Exact command and environment used for this run.

## Main JSON Fields

In each `result.json`, common experiment settings are available at the top level:

- `run_id`
- `workload_name`
- `concurrency`
- `write_ratio`
- `keyspace`
- `valuesize`
- `seed`
- `prefill`

Overall throughput and success metrics:

- `total_requests`
- `success_requests`
- `failed_requests`
- `success_rate`
- `success_rate_percent`
- `total_duration`
- `total_duration_seconds`
- `total_qps`
- `success_qps`

Overall latency metrics:

- `avg_latency`
- `avg_latency_ms`
- `p50_latency`
- `p50_latency_ms`
- `p90_latency`
- `p90_latency_ms`
- `p99_latency`
- `p99_latency_ms`
- `p999_latency`
- `p999_latency_ms`
- `max_latency`
- `max_latency_ms`

Put-specific metrics:

- `put_total`
- `put_success`
- `put_failed`
- `put_avg_latency`
- `put_avg_latency_ms`
- `put_p50`
- `put_p50_ms`
- `put_p90`
- `put_p90_ms`
- `put_p99`
- `put_p99_ms`
- `put_p999`
- `put_p999_ms`
- `put_max`
- `put_max_ms`

Get-specific metrics:

- `get_total`
- `get_success`
- `get_failed`
- `get_found`
- `get_not_found`
- `get_avg_latency`
- `get_avg_latency_ms`
- `get_p50`
- `get_p50_ms`
- `get_p90`
- `get_p90_ms`
- `get_p99`
- `get_p99_ms`
- `get_p999`
- `get_p999_ms`
- `get_max`
- `get_max_ms`

Error counts are stored under:

```json
"error_counts": {
  "timeout": 0,
  "connection_refused": 0,
  "http_500": 0,
  "http_503": 0,
  "request_timeout": 0,
  "not_leader": 0,
  "other": 0
}
```

## Prefill And Warmup Sections

Each `result.json` also contains:

- `prefill_stage`
  - Records whether prefill was enabled and how many prefill requests succeeded or failed.
  - Prefill results are not counted in the formal benchmark metrics.

- `warmup_stage`
  - Records warmup request success and failure counts.
  - Warmup results are not counted in the formal benchmark metrics.

## Result Count

This formal result set contains:

- 4 workloads
- 4 concurrency levels per workload
- 3 seeds per concurrency level
- 48 total `result.json` files
