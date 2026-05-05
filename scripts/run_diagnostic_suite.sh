#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

API_URL="${API_URL:-http://localhost:8080}"
RESULT_ROOT="${RESULT_ROOT:-benchmark_results}"
SUITE_ID="${SUITE_ID:-diagnostic_$(date +%Y%m%d_%H%M%S)}"
SUITE_ROOT="${RESULT_ROOT}/${SUITE_ID}"
SERVER_WAIT_SECONDS="${SERVER_WAIT_SECONDS:-30}"
GOCACHE="${GOCACHE:-/tmp/go-build}"
SEED="${SEED:-1}"
WARMUP="${WARMUP:-0}"

TOTAL_REQUESTS="${TOTAL_REQUESTS:-10000}"
KEYSPACE="${KEYSPACE:-10000}"
PREFILLN="${PREFILLN:-10000}"
VALUESIZE="${VALUESIZE:-128}"
TIMEOUT="${TIMEOUT:-10s}"

CASES=(
	"write_100|50|1.0|false|${TOTAL_REQUESTS}|${KEYSPACE}|0|${VALUESIZE}|${TIMEOUT}"
	"read_100|10|0.0|true|${TOTAL_REQUESTS}|${KEYSPACE}|${PREFILLN}|${VALUESIZE}|${TIMEOUT}"
	"read_100|30|0.0|true|${TOTAL_REQUESTS}|${KEYSPACE}|${PREFILLN}|${VALUESIZE}|${TIMEOUT}"
	"read_100|50|0.0|true|${TOTAL_REQUESTS}|${KEYSPACE}|${PREFILLN}|${VALUESIZE}|${TIMEOUT}"
	"mixed_50_50|30|0.5|true|${TOTAL_REQUESTS}|${KEYSPACE}|${PREFILLN}|${VALUESIZE}|${TIMEOUT}"
)

if [[ "${CONFIRM_RUN:-false}" != "true" ]]; then
	cat <<EOF
Diagnostic suite is ready but was not started.

This script runs 5 clean-data benchmark cases with n=${TOTAL_REQUESTS}, keyspace=${KEYSPACE}, prefilln=${PREFILLN}, valuesize=${VALUESIZE}, timeout=${TIMEOUT}.
Read-heavy cases include a 10000-key prefill and may take several minutes each on the current implementation.

To run the full diagnostic suite:
  CONFIRM_RUN=true ./scripts/run_diagnostic_suite.sh

Output directory will be:
  ${SUITE_ROOT}
EOF
	exit 0
fi

if ! command -v curl >/dev/null 2>&1; then
	echo "curl is required for diagnostic metrics collection." >&2
	exit 1
fi
if ! command -v setsid >/dev/null 2>&1; then
	echo "setsid is required for starting and stopping the server process group safely." >&2
	exit 1
fi

mkdir -p "${SUITE_ROOT}"

MANIFEST="${SUITE_ROOT}/diagnostic_manifest.tsv"
SUITE_COMMAND="${SUITE_ROOT}/diagnostic_command.txt"

cat >"${SUITE_COMMAND}" <<EOF
repo: ${REPO_ROOT}
suite_id: ${SUITE_ID}
suite_root: ${SUITE_ROOT}
api_url: ${API_URL}
total_requests: ${TOTAL_REQUESTS}
keyspace: ${KEYSPACE}
prefilln: ${PREFILLN}
valuesize: ${VALUESIZE}
timeout: ${TIMEOUT}
warmup: ${WARMUP}
seed: ${SEED}
cleaned_data_dir_per_case: ${REPO_ROOT}/data
EOF

printf "case_id\tworkload\tconcurrency\twrite_ratio\tprefill\tn\tkeyspace\tprefilln\tvaluesize\ttimeout\tresult_dir\n" >"${MANIFEST}"

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

run_case() {
	local case_id="$1"
	local workload="$2"
	local concurrency="$3"
	local write_ratio="$4"
	local prefill="$5"
	local total_requests="$6"
	local keyspace="$7"
	local prefilln="$8"
	local valuesize="$9"
	local timeout="${10}"

	local case_dir="${SUITE_ROOT}/${case_id}_${workload}_c${concurrency}"
	local result_json="${case_dir}/result.json"
	local benchmark_log="${case_dir}/benchmark.log"
	local server_log="${case_dir}/server.log"
	local command_file="${case_dir}/command.txt"
	local metrics_before="${case_dir}/metrics_before.json"
	local metrics_after="${case_dir}/metrics_after.json"
	local metrics_summary="${case_dir}/metrics_summary.md"
	local lsm_file_stats="${case_dir}/lsm_file_stats.md"

	mkdir -p "${case_dir}"
	printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n" \
		"${case_id}" "${workload}" "${concurrency}" "${write_ratio}" "${prefill}" \
		"${total_requests}" "${keyspace}" "${prefilln}" "${valuesize}" "${timeout}" "${case_dir}" >>"${MANIFEST}"

	echo
	echo "============================================================"
	echo "Running ${case_id}: workload=${workload}, concurrency=${concurrency}, n=${total_requests}, prefill=${prefill}"
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
		-runid "${case_id}_${workload}_c${concurrency}"
		-workload "${workload}"
		-c "${concurrency}"
		-n "${total_requests}"
		-w "${write_ratio}"
		-keyspace "${keyspace}"
		-valuesize "${valuesize}"
		-seed "${SEED}"
		-prefill="${prefill}"
		-prefilln "${prefilln}"
		-warmup "${WARMUP}"
		-timeout "${timeout}"
		-u "${API_URL}"
		-out "${result_json}"
	)

	{
		printf "repo: %s\n" "${REPO_ROOT}"
		printf "case_id: %s\n" "${case_id}"
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
			-case "${case_id}_${workload}_c${concurrency}" \
			-result "${result_json}" \
			-before "${metrics_before}" \
			-after "${metrics_after}" \
			-out "${metrics_summary}"
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

	GOCACHE="${GOCACHE}" go run ./tests/diagnostic \
		-mode lsm-stats \
		-case "${case_id}_${workload}_c${concurrency}" \
		-data data \
		-server-log "${server_log}" \
		-out "${lsm_file_stats}"

	cleanup_server

	if [[ "${benchmark_status}" != "0" ]]; then
		echo "Benchmark failed for ${case_id}. See ${benchmark_log}" >&2
		return "${benchmark_status}"
	fi
}

case_index=1
for case_def in "${CASES[@]}"; do
	IFS='|' read -r workload concurrency write_ratio prefill total_requests keyspace prefilln valuesize timeout <<<"${case_def}"
	case_id="$(printf "%02d" "${case_index}")"
	run_case "${case_id}" "${workload}" "${concurrency}" "${write_ratio}" "${prefill}" "${total_requests}" "${keyspace}" "${prefilln}" "${valuesize}" "${timeout}"
	((case_index += 1))
done

echo
echo "Diagnostic suite finished."
echo "Suite root: ${SUITE_ROOT}"
echo "Manifest: ${MANIFEST}"
