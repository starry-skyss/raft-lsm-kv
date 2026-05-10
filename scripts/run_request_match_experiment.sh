#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

API_URL="${API_URL:-http://localhost:8080}"
RESULT_ROOT="${RESULT_ROOT:-benchmark_results}"
RUN_ID="${RUN_ID:-request_match_$(date +%Y%m%d_%H%M%S)}"
RESULT_DIR="${RESULT_ROOT}/${RUN_ID}"
SERVER_WAIT_SECONDS="${SERVER_WAIT_SECONDS:-30}"
GOCACHE="${GOCACHE:-/tmp/go-build}"

TOTAL_REQUESTS="${TOTAL_REQUESTS:-5000}"
CONCURRENCY="${CONCURRENCY:-50}"
KEYSPACE="${KEYSPACE:-1000}"
VALUESIZE="${VALUESIZE:-64}"
TIMEOUT="${TIMEOUT:-5s}"
ELECTIONS="${ELECTIONS:-8}"
ELECTION_INTERVAL="${ELECTION_INTERVAL:-500ms}"
SEED="${SEED:-1}"

if [[ "${CONFIRM_RUN:-false}" != "true" ]]; then
	cat <<EOF
Request-match experiment is ready but was not started.

This script cleans data, starts the 3-node local cluster, pre-fills ${KEYSPACE} unique keys,
runs ${TOTAL_REQUESTS} verified GET requests at concurrency=${CONCURRENCY}, and forces ${ELECTIONS}
debug elections during the request run.

To run:
  CONFIRM_RUN=true GOCACHE=${GOCACHE} ./scripts/run_request_match_experiment.sh

For a quick smoke run:
  CONFIRM_RUN=true TOTAL_REQUESTS=100 CONCURRENCY=10 KEYSPACE=50 ELECTIONS=3 ./scripts/run_request_match_experiment.sh

Output directory will be:
  ${RESULT_DIR}
EOF
	exit 0
fi

if ! command -v curl >/dev/null 2>&1; then
	echo "curl is required for waiting until the API Gateway is ready." >&2
	exit 1
fi
if ! command -v setsid >/dev/null 2>&1; then
	echo "setsid is required for starting and stopping the server process group safely." >&2
	exit 1
fi

mkdir -p "${RESULT_DIR}"

SERVER_LOG="${RESULT_DIR}/server.log"
COMMAND_FILE="${RESULT_DIR}/command.txt"
EXPERIMENT_LOG="${RESULT_DIR}/request_match.log"

cat >"${COMMAND_FILE}" <<EOF
repo: ${REPO_ROOT}
run_id: ${RUN_ID}
result_dir: ${RESULT_DIR}
api_url: ${API_URL}
total_requests: ${TOTAL_REQUESTS}
concurrency: ${CONCURRENCY}
keyspace: ${KEYSPACE}
valuesize: ${VALUESIZE}
timeout: ${TIMEOUT}
elections: ${ELECTIONS}
election_interval: ${ELECTION_INTERVAL}
seed: ${SEED}
cleaned_data_dir: ${REPO_ROOT}/data
server_command: GOCACHE=${GOCACHE} go run ./cmd/kvnode
experiment_command: GOCACHE=${GOCACHE} go run ./tests/requestmatch -runid ${RUN_ID} -u ${API_URL} -n ${TOTAL_REQUESTS} -c ${CONCURRENCY} -keyspace ${KEYSPACE} -valuesize ${VALUESIZE} -timeout ${TIMEOUT} -elections ${ELECTIONS} -election-interval ${ELECTION_INTERVAL} -seed ${SEED} -outdir ${RESULT_DIR}
EOF

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

echo "Cleaning project runtime data directory: ${REPO_ROOT}/data"
rm -rf data

echo "Starting server..."
setsid env GOCACHE="${GOCACHE}" go run ./cmd/kvnode >"${SERVER_LOG}" 2>&1 &
SERVER_PID=$!

ready=false
for ((i = 1; i <= SERVER_WAIT_SECONDS; i++)); do
	if ! kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
		echo "Server exited before becoming ready. See ${SERVER_LOG}" >&2
		exit 1
	fi
	status="$(curl -s -o /dev/null -w "%{http_code}" "${API_URL}/debug/metrics" || true)"
	if [[ "${status}" == "200" ]]; then
		ready=true
		break
	fi
	sleep 1
done

if [[ "${ready}" != "true" ]]; then
	echo "API Gateway did not become ready within ${SERVER_WAIT_SECONDS}s. See ${SERVER_LOG}" >&2
	exit 1
fi

echo "Server is ready."
echo "Running request-match experiment..."

GOCACHE="${GOCACHE}" go run ./tests/requestmatch \
	-runid "${RUN_ID}" \
	-u "${API_URL}" \
	-n "${TOTAL_REQUESTS}" \
	-c "${CONCURRENCY}" \
	-keyspace "${KEYSPACE}" \
	-valuesize "${VALUESIZE}" \
	-timeout "${TIMEOUT}" \
	-elections "${ELECTIONS}" \
	-election-interval "${ELECTION_INTERVAL}" \
	-seed "${SEED}" \
	-outdir "${RESULT_DIR}" 2>&1 | tee "${EXPERIMENT_LOG}"

echo
echo "Request-match experiment finished."
echo "Result directory: ${RESULT_DIR}"
echo "CSV: ${RESULT_DIR}/request_match_results.csv"
echo "Markdown: ${RESULT_DIR}/request_match_results.md"
