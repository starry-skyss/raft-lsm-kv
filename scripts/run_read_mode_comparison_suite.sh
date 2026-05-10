#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

API_URL="${API_URL:-http://localhost:8080}"
RESULT_ROOT="${RESULT_ROOT:-benchmark_results}"
SUITE_ID="${SUITE_ID:-readmode_$(date +%Y%m%d_%H%M%S)}"
SUITE_ROOT="${RESULT_ROOT}/${SUITE_ID}"
SERVER_WAIT_SECONDS="${SERVER_WAIT_SECONDS:-30}"
GOCACHE="${GOCACHE:-/tmp/go-build}"

TOTAL_REQUESTS="${TOTAL_REQUESTS:-10000}"
KEYSPACE="${KEYSPACE:-10000}"
PREFILLN="${PREFILLN:-10000}"
VALUESIZE="${VALUESIZE:-128}"
WARMUP="${WARMUP:-0}"
TIMEOUT="${TIMEOUT:-10s}"

READ_MODES="${READ_MODES:-raft local}"
WORKLOADS="${WORKLOADS:-read_100 read95_write5 mixed_50_50}"
CONCURRENCIES="${CONCURRENCIES:-1 10 30 50}"
SEEDS="${SEEDS:-1 2 3}"

if [[ "${CONFIRM_RUN:-false}" != "true" ]]; then
	cat <<EOF
Read-mode comparison suite is ready but was not started.

This script runs:
  read_modes: ${READ_MODES}
  workloads: ${WORKLOADS}
  concurrencies: ${CONCURRENCIES}
  seeds: ${SEEDS}
  n=${TOTAL_REQUESTS}, prefilln=${PREFILLN}, keyspace=${KEYSPACE}, valuesize=${VALUESIZE}, timeout=${TIMEOUT}

The default formal matrix is 72 clean-data runs and may take a long time.

To run the full suite:
  CONFIRM_RUN=true GOCACHE=${GOCACHE} ./scripts/run_read_mode_comparison_suite.sh

For a quick smoke run:
  CONFIRM_RUN=true TOTAL_REQUESTS=20 PREFILLN=20 KEYSPACE=20 VALUESIZE=32 CONCURRENCIES=1 SEEDS=1 WORKLOADS=read_100 READ_MODES="raft local" ./scripts/run_read_mode_comparison_suite.sh

Output directory will be:
  ${SUITE_ROOT}
EOF
	exit 0
fi

if ! command -v curl >/dev/null 2>&1; then
	echo "curl is required for read-mode metrics collection." >&2
	exit 1
fi
if ! command -v setsid >/dev/null 2>&1; then
	echo "setsid is required for starting and stopping the server process group safely." >&2
	exit 1
fi

mkdir -p "${SUITE_ROOT}"

MANIFEST="${SUITE_ROOT}/read_mode_comparison_manifest.tsv"
SUITE_COMMAND="${SUITE_ROOT}/read_mode_comparison_command.txt"
SUITE_CSV="${SUITE_ROOT}/read_mode_comparison_summary.csv"
SUITE_MD="${SUITE_ROOT}/read_mode_comparison_summary.md"

cat >"${SUITE_COMMAND}" <<EOF
repo: ${REPO_ROOT}
suite_id: ${SUITE_ID}
suite_root: ${SUITE_ROOT}
api_url: ${API_URL}
read_modes: ${READ_MODES}
workloads: ${WORKLOADS}
concurrencies: ${CONCURRENCIES}
seeds: ${SEEDS}
total_requests: ${TOTAL_REQUESTS}
keyspace: ${KEYSPACE}
prefilln: ${PREFILLN}
valuesize: ${VALUESIZE}
warmup: ${WARMUP}
timeout: ${TIMEOUT}
cleaned_data_dir_per_case: ${REPO_ROOT}/data
EOF

printf "read_mode\tworkload\tconcurrency\tseed\twrite_ratio\tprefill\tn\tkeyspace\tprefilln\tvaluesize\ttimeout\tcase_dir\n" >"${MANIFEST}"
printf "read_mode,workload,concurrency,seed,success_rate,success_qps,p50_ms,p90_ms,p99_ms,p999_ms,http_500_total,http_503_total,request_timeout_total,leader_change_delta,election_started_delta,term_change_delta\n" >"${SUITE_CSV}"

read -r -a read_mode_list <<<"${READ_MODES}"
read -r -a workload_list <<<"${WORKLOADS}"
read -r -a concurrency_list <<<"${CONCURRENCIES}"
read -r -a seed_list <<<"${SEEDS}"

workload_write_ratio() {
	case "$1" in
	read_100)
		printf "0.0"
		;;
	read95_write5)
		printf "0.05"
		;;
	mixed_50_50)
		printf "0.5"
		;;
	*)
		echo "unknown workload: $1" >&2
		return 1
		;;
	esac
}

workload_prefill() {
	case "$1" in
	read_100 | read95_write5 | mixed_50_50)
		printf "true"
		;;
	*)
		echo "unknown workload: $1" >&2
		return 1
		;;
	esac
}

validate_read_mode() {
	case "$1" in
	raft | local)
		return 0
		;;
	*)
		echo "unknown read mode: $1" >&2
		return 1
		;;
	esac
}

SERVER_PID=""

cleanup_server() {
	if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
		kill -TERM "-${SERVER_PID}" >/dev/null 2>&1 || true
		for _ in {1..30}; do
			if ! kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
				break
			fi
			sleep 0.1
		done
		if kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
			kill -KILL "-${SERVER_PID}" >/dev/null 2>&1 || true
		fi
		wait "${SERVER_PID}" >/dev/null 2>&1 || true
	fi
	SERVER_PID=""
}
trap cleanup_server EXIT

wait_for_server() {
	local server_log="$1"
	local ready=false
	for ((i = 1; i <= SERVER_WAIT_SECONDS; i++)); do
		if ! kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
			echo "Server exited before becoming ready. See ${server_log}" >&2
			return 1
		fi
		status="$(curl -s -o /dev/null -w "%{http_code}" "${API_URL}/debug/metrics" || true)"
		if [[ "${status}" == "200" ]]; then
			ready=true
			break
		fi
		sleep 1
	done
	if [[ "${ready}" != "true" ]]; then
		echo "API Gateway did not become ready within ${SERVER_WAIT_SECONDS}s. See ${server_log}" >&2
		return 1
	fi
}

write_suite_markdown() {
	{
		printf "# Read Mode Comparison Summary\n\n"
		printf -- "- Suite: %s\n" "${SUITE_ID}"
		printf -- "- Read modes: %s\n" "${READ_MODES}"
		printf -- "- Workloads: %s\n" "${WORKLOADS}"
		printf -- "- Concurrencies: %s\n" "${CONCURRENCIES}"
		printf -- "- Seeds: %s\n\n" "${SEEDS}"
		printf "| Read Mode | Workload | Concurrency | Seed | Success Rate | Success QPS | P50 ms | P90 ms | P99 ms | P99.9 ms | HTTP 500 | HTTP 503 | Request Timeout | Leader Changes | Elections | Term Changes |\n"
		printf "| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n"
		tail -n +2 "${SUITE_CSV}" | while IFS=',' read -r read_mode workload concurrency seed success_rate success_qps p50 p90 p99 p999 http500 http503 timeout_count leader_delta election_delta term_delta; do
			printf "| %s | %s | %s | %s | %s%% | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n" \
				"${read_mode}" "${workload}" "${concurrency}" "${seed}" "${success_rate}" "${success_qps}" \
				"${p50}" "${p90}" "${p99}" "${p999}" "${http500}" "${http503}" "${timeout_count}" \
				"${leader_delta}" "${election_delta}" "${term_delta}"
		done
	} >"${SUITE_MD}"
}

run_case() {
	local read_mode="$1"
	local workload="$2"
	local concurrency="$3"
	local seed="$4"
	local write_ratio
	local prefill
	local case_name
	local case_dir

	validate_read_mode "${read_mode}"
	write_ratio="$(workload_write_ratio "${workload}")"
	prefill="$(workload_prefill "${workload}")"
	case_name="${read_mode}_${workload}_c${concurrency}_seed${seed}"
	case_dir="${SUITE_ROOT}/${case_name}"

	local result_json="${case_dir}/result_${case_name}.json"
	local benchmark_log="${case_dir}/benchmark_${case_name}.log"
	local server_log="${case_dir}/server_${case_name}.log"
	local command_file="${case_dir}/command_${case_name}.txt"
	local metrics_before="${case_dir}/metrics_before_${case_name}.json"
	local metrics_after="${case_dir}/metrics_after_${case_name}.json"
	local metrics_summary="${case_dir}/summary_${case_name}.md"
	local csv_row="${case_dir}/summary_${case_name}.csv"

	mkdir -p "${case_dir}"
	printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n" \
		"${read_mode}" "${workload}" "${concurrency}" "${seed}" "${write_ratio}" "${prefill}" \
		"${TOTAL_REQUESTS}" "${KEYSPACE}" "${PREFILLN}" "${VALUESIZE}" "${TIMEOUT}" "${case_dir}" >>"${MANIFEST}"

	echo
	echo "============================================================"
	echo "Running ${case_name}: read_mode=${read_mode}, workload=${workload}, concurrency=${concurrency}, seed=${seed}"
	echo "Result directory: ${case_dir}"
	echo "============================================================"

	echo "Cleaning project runtime data directory: ${REPO_ROOT}/data"
	rm -rf data

	echo "Starting server..."
	setsid env GOCACHE="${GOCACHE}" go run ./cmd/kvnode >"${server_log}" 2>&1 &
	SERVER_PID=$!
	wait_for_server "${server_log}"

	curl -sS "${API_URL}/debug/metrics" >"${metrics_before}"

	benchmark_cmd=(
		go run ./tests/benchmark/main.go
		-runid "${case_name}"
		-workload "${workload}"
		-readmode "${read_mode}"
		-c "${concurrency}"
		-n "${TOTAL_REQUESTS}"
		-w "${write_ratio}"
		-keyspace "${KEYSPACE}"
		-valuesize "${VALUESIZE}"
		-seed "${seed}"
		-prefill="${prefill}"
		-prefilln "${PREFILLN}"
		-warmup "${WARMUP}"
		-timeout "${TIMEOUT}"
		-u "${API_URL}"
		-out "${result_json}"
	)

	{
		printf "repo: %s\n" "${REPO_ROOT}"
		printf "case_name: %s\n" "${case_name}"
		printf "result_dir: %s\n" "${case_dir}"
		printf "cleaned_data_dir: %s/data\n" "${REPO_ROOT}"
		printf "server_command: GOCACHE=%s go run ./cmd/kvnode\n" "${GOCACHE}"
		printf "benchmark_command: GOCACHE=%s" "${GOCACHE}"
		printf " %q" "${benchmark_cmd[@]}"
		printf "\n"
	} >"${command_file}"

	set +e
	env GOCACHE="${GOCACHE}" "${benchmark_cmd[@]}" 2>&1 | tee "${benchmark_log}"
	benchmark_status="${PIPESTATUS[0]}"
	set -e

	curl -sS "${API_URL}/debug/metrics" >"${metrics_after}" || true

	if [[ -f "${result_json}" && -s "${metrics_before}" && -s "${metrics_after}" ]]; then
		GOCACHE="${GOCACHE}" go run ./tests/diagnostic \
			-mode summary \
			-case "${case_name}" \
			-result "${result_json}" \
			-before "${metrics_before}" \
			-after "${metrics_after}" \
			-out "${metrics_summary}"
		GOCACHE="${GOCACHE}" go run ./tests/diagnostic \
			-mode summary-csv-row \
			-result "${result_json}" \
			-before "${metrics_before}" \
			-after "${metrics_after}" \
			-out "${csv_row}"
		cat "${csv_row}" >>"${SUITE_CSV}"
		write_suite_markdown
	else
		cat >"${metrics_summary}" <<EOF
# Metrics Summary

Benchmark did not produce all required inputs.

- benchmark_status: ${benchmark_status}
- result_json_exists: $([[ -f "${result_json}" ]] && printf true || printf false)
- metrics_before_size: $(wc -c <"${metrics_before}" 2>/dev/null || printf 0)
- metrics_after_size: $(wc -c <"${metrics_after}" 2>/dev/null || printf 0)
EOF
	fi

	cleanup_server

	if [[ "${benchmark_status}" != "0" ]]; then
		echo "Benchmark failed for ${case_name}. See ${benchmark_log}" >&2
		return "${benchmark_status}"
	fi
}

echo "Read-mode comparison suite starting"
echo "Suite root: ${SUITE_ROOT}"
echo "Manifest: ${MANIFEST}"

for read_mode in "${read_mode_list[@]}"; do
	for workload in "${workload_list[@]}"; do
		for concurrency in "${concurrency_list[@]}"; do
			for seed in "${seed_list[@]}"; do
				run_case "${read_mode}" "${workload}" "${concurrency}" "${seed}"
			done
		done
	done
done

write_suite_markdown

echo
echo "Read-mode comparison suite finished."
echo "Suite root: ${SUITE_ROOT}"
echo "Manifest: ${MANIFEST}"
echo "Summary CSV: ${SUITE_CSV}"
echo "Summary Markdown: ${SUITE_MD}"
