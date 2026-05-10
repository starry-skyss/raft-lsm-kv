#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

API_URL="${API_URL:-http://localhost:8080}"
RESULT_ROOT="${RESULT_ROOT:-benchmark_results}"
SUITE_ID="${SUITE_ID:-compaction_$(date +%Y%m%d_%H%M%S)}"
SUITE_ROOT="${RESULT_ROOT}/${SUITE_ID}"
SERVER_WAIT_SECONDS="${SERVER_WAIT_SECONDS:-30}"
GOCACHE="${GOCACHE:-/tmp/go-build}"

TOTAL_REQUESTS="${TOTAL_REQUESTS:-30000}"
KEYSPACE="${KEYSPACE:-1000}"
PREFILLN="${PREFILLN:-${KEYSPACE}}"
VALUESIZE="${VALUESIZE:-128}"
WARMUP="${WARMUP:-0}"
TIMEOUT="${TIMEOUT:-10s}"
READ_MODE="${READ_MODE:-raft}"

COMPACTION_MODES="${COMPACTION_MODES:-true false}"
WORKLOADS="${WORKLOADS:-write_100 mixed_50_50 delete_mixed}"
CONCURRENCIES="${CONCURRENCIES:-10 30}"
SEEDS="${SEEDS:-1 2 3}"

if [[ "${CONFIRM_RUN:-false}" != "true" ]]; then
	cat <<EOF
Compaction ablation suite is ready but was not started.

This script runs:
  compaction_modes: ${COMPACTION_MODES}
  workloads: ${WORKLOADS}
  concurrencies: ${CONCURRENCIES}
  seeds: ${SEEDS}
  n=${TOTAL_REQUESTS}, keyspace=${KEYSPACE}, prefilln=${PREFILLN}, valuesize=${VALUESIZE}, timeout=${TIMEOUT}

The default formal matrix is 36 clean-data runs and may take a long time.

To run the full suite:
  CONFIRM_RUN=true GOCACHE=${GOCACHE} ./scripts/run_compaction_ablation_suite.sh

For a quick smoke run:
  CONFIRM_RUN=true TOTAL_REQUESTS=60 KEYSPACE=20 PREFILLN=20 VALUESIZE=32 CONCURRENCIES=1 SEEDS=1 WORKLOADS="write_100 delete_mixed" COMPACTION_MODES="true false" ./scripts/run_compaction_ablation_suite.sh

Output directory will be:
  ${SUITE_ROOT}
EOF
	exit 0
fi

if ! command -v curl >/dev/null 2>&1; then
	echo "curl is required for compaction metrics collection." >&2
	exit 1
fi
if ! command -v setsid >/dev/null 2>&1; then
	echo "setsid is required for starting and stopping the server process group safely." >&2
	exit 1
fi

mkdir -p "${SUITE_ROOT}"

MANIFEST="${SUITE_ROOT}/compaction_ablation_manifest.tsv"
SUITE_COMMAND="${SUITE_ROOT}/compaction_ablation_command.txt"
SUITE_CSV="${SUITE_ROOT}/compaction_ablation_summary.csv"
SUITE_MD="${SUITE_ROOT}/compaction_ablation_summary.md"

cat >"${SUITE_COMMAND}" <<EOF
repo: ${REPO_ROOT}
suite_id: ${SUITE_ID}
suite_root: ${SUITE_ROOT}
api_url: ${API_URL}
compaction_modes: ${COMPACTION_MODES}
workloads: ${WORKLOADS}
concurrencies: ${CONCURRENCIES}
seeds: ${SEEDS}
total_requests: ${TOTAL_REQUESTS}
keyspace: ${KEYSPACE}
prefilln: ${PREFILLN}
valuesize: ${VALUESIZE}
warmup: ${WARMUP}
timeout: ${TIMEOUT}
read_mode: ${READ_MODE}
cleaned_data_dir_per_case: ${REPO_ROOT}/data
EOF

printf "enable_compaction\tworkload\tconcurrency\tseed\twrite_ratio\tdelete_ratio\tprefill\tn\tkeyspace\tprefilln\tvaluesize\ttimeout\tcase_dir\n" >"${MANIFEST}"
printf "enable_compaction,workload,concurrency,seed,total_requests,keyspace,value_size,success_rate,success_qps,get_success_qps,get_p99_ms,get_p999_ms,sstable_files_total,data_size_bytes,wal_files_total,wal_size_bytes,get_sstable_files_touched_avg,compaction_total_delta,compaction_input_files_delta,compaction_output_files_delta,compaction_total_ms_delta,compaction_max_ms\n" >"${SUITE_CSV}"

read -r -a compaction_mode_list <<<"${COMPACTION_MODES}"
read -r -a workload_list <<<"${WORKLOADS}"
read -r -a concurrency_list <<<"${CONCURRENCIES}"
read -r -a seed_list <<<"${SEEDS}"

workload_write_ratio() {
	case "$1" in
	write_100)
		printf "1.0"
		;;
	mixed_50_50)
		printf "0.5"
		;;
	delete_mixed)
		printf "0.4"
		;;
	*)
		echo "unknown workload: $1" >&2
		return 1
		;;
	esac
}

workload_delete_ratio() {
	case "$1" in
	write_100 | mixed_50_50)
		printf "0.0"
		;;
	delete_mixed)
		printf "0.2"
		;;
	*)
		echo "unknown workload: $1" >&2
		return 1
		;;
	esac
}

workload_prefill() {
	case "$1" in
	write_100)
		printf "false"
		;;
	mixed_50_50 | delete_mixed)
		printf "true"
		;;
	*)
		echo "unknown workload: $1" >&2
		return 1
		;;
	esac
}

compaction_label() {
	case "$1" in
	true)
		printf "compaction_on"
		;;
	false)
		printf "compaction_off"
		;;
	*)
		echo "unknown compaction mode: $1" >&2
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
		printf "# Compaction Ablation Summary\n\n"
		printf -- "- Suite: %s\n" "${SUITE_ID}"
		printf -- "- Compaction modes: %s\n" "${COMPACTION_MODES}"
		printf -- "- Workloads: %s\n" "${WORKLOADS}"
		printf -- "- Concurrencies: %s\n" "${CONCURRENCIES}"
		printf -- "- Seeds: %s\n\n" "${SEEDS}"
		printf "| Enable Compaction | Workload | Concurrency | Seed | Success Rate | Success QPS | Get Success QPS | Get P99 ms | Get P99.9 ms | SSTables | Data Bytes | WAL Files | WAL Bytes | Avg SSTables/Get | Compactions | Input Files | Output Files | Compaction Total ms | Compaction Max ms |\n"
		printf "| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n"
		tail -n +2 "${SUITE_CSV}" | while IFS=',' read -r enable_compaction workload concurrency seed total_requests keyspace value_size success_rate success_qps get_success_qps get_p99 get_p999 sstables data_bytes wal_files wal_bytes touched_avg compactions input_files output_files compaction_total_ms compaction_max_ms; do
			printf "| %s | %s | %s | %s | %s%% | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n" \
				"${enable_compaction}" "${workload}" "${concurrency}" "${seed}" "${success_rate}" "${success_qps}" \
				"${get_success_qps}" "${get_p99}" "${get_p999}" "${sstables}" "${data_bytes}" "${wal_files}" \
				"${wal_bytes}" "${touched_avg}" "${compactions}" "${input_files}" "${output_files}" \
				"${compaction_total_ms}" "${compaction_max_ms}"
		done
	} >"${SUITE_MD}"
}

run_case() {
	local enable_compaction="$1"
	local workload="$2"
	local concurrency="$3"
	local seed="$4"
	local write_ratio
	local delete_ratio
	local prefill
	local label
	local case_name
	local case_dir

	label="$(compaction_label "${enable_compaction}")"
	write_ratio="$(workload_write_ratio "${workload}")"
	delete_ratio="$(workload_delete_ratio "${workload}")"
	prefill="$(workload_prefill "${workload}")"
	case_name="${label}_${workload}_c${concurrency}_seed${seed}"
	case_dir="${SUITE_ROOT}/${case_name}"

	local result_json="${case_dir}/result_${case_name}.json"
	local benchmark_log="${case_dir}/benchmark_${case_name}.log"
	local server_log="${case_dir}/server_${case_name}.log"
	local command_file="${case_dir}/command_${case_name}.txt"
	local metrics_before="${case_dir}/metrics_before_${case_name}.json"
	local metrics_after="${case_dir}/metrics_after_${case_name}.json"
	local metrics_summary="${case_dir}/summary_${case_name}.md"
	local csv_row="${case_dir}/summary_${case_name}.csv"
	local lsm_file_stats="${case_dir}/lsm_file_stats_${case_name}.md"

	mkdir -p "${case_dir}"
	printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n" \
		"${enable_compaction}" "${workload}" "${concurrency}" "${seed}" "${write_ratio}" "${delete_ratio}" "${prefill}" \
		"${TOTAL_REQUESTS}" "${KEYSPACE}" "${PREFILLN}" "${VALUESIZE}" "${TIMEOUT}" "${case_dir}" >>"${MANIFEST}"

	echo
	echo "============================================================"
	echo "Running ${case_name}: enable_compaction=${enable_compaction}, workload=${workload}, concurrency=${concurrency}, seed=${seed}"
	echo "Result directory: ${case_dir}"
	echo "============================================================"

	echo "Cleaning project runtime data directory: ${REPO_ROOT}/data"
	rm -rf data

	echo "Starting server..."
	setsid env GOCACHE="${GOCACHE}" ENABLE_COMPACTION="${enable_compaction}" go run ./cmd/kvnode >"${server_log}" 2>&1 &
	SERVER_PID=$!
	wait_for_server "${server_log}"

	curl -sS "${API_URL}/debug/metrics" >"${metrics_before}"

	benchmark_cmd=(
		go run ./tests/benchmark/main.go
		-runid "${case_name}"
		-workload "${workload}"
		-readmode "${READ_MODE}"
		-enablecompaction="${enable_compaction}"
		-c "${concurrency}"
		-n "${TOTAL_REQUESTS}"
		-w "${write_ratio}"
		-deleteratio "${delete_ratio}"
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
		printf "enable_compaction: %s\n" "${enable_compaction}"
		printf "cleaned_data_dir: %s/data\n" "${REPO_ROOT}"
		printf "server_command: ENABLE_COMPACTION=%s GOCACHE=%s go run ./cmd/kvnode\n" "${enable_compaction}" "${GOCACHE}"
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
			-mode compaction-summary \
			-case "${case_name}" \
			-result "${result_json}" \
			-before "${metrics_before}" \
			-after "${metrics_after}" \
			-data data \
			-out "${metrics_summary}"
		GOCACHE="${GOCACHE}" go run ./tests/diagnostic \
			-mode compaction-csv-row \
			-result "${result_json}" \
			-before "${metrics_before}" \
			-after "${metrics_after}" \
			-data data \
			-out "${csv_row}"
		cat "${csv_row}" >>"${SUITE_CSV}"
		write_suite_markdown
	else
		cat >"${metrics_summary}" <<EOF
# Compaction Ablation Summary

Benchmark did not produce all required inputs.

- benchmark_status: ${benchmark_status}
- result_json_exists: $([[ -f "${result_json}" ]] && printf true || printf false)
- metrics_before_size: $(wc -c <"${metrics_before}" 2>/dev/null || printf 0)
- metrics_after_size: $(wc -c <"${metrics_after}" 2>/dev/null || printf 0)
EOF
	fi

	GOCACHE="${GOCACHE}" go run ./tests/diagnostic \
		-mode lsm-stats \
		-case "${case_name}" \
		-data data \
		-server-log "${server_log}" \
		-out "${lsm_file_stats}"

	cleanup_server

	if [[ "${benchmark_status}" != "0" ]]; then
		echo "Benchmark failed for ${case_name}. See ${benchmark_log}" >&2
		return "${benchmark_status}"
	fi
}

echo "Compaction ablation suite starting"
echo "Suite root: ${SUITE_ROOT}"
echo "Manifest: ${MANIFEST}"

for enable_compaction in "${compaction_mode_list[@]}"; do
	for workload in "${workload_list[@]}"; do
		for concurrency in "${concurrency_list[@]}"; do
			for seed in "${seed_list[@]}"; do
				run_case "${enable_compaction}" "${workload}" "${concurrency}" "${seed}"
			done
		done
	done
done

write_suite_markdown

echo
echo "Compaction ablation suite finished."
echo "Suite root: ${SUITE_ROOT}"
echo "Manifest: ${MANIFEST}"
echo "Summary CSV: ${SUITE_CSV}"
echo "Summary Markdown: ${SUITE_MD}"
