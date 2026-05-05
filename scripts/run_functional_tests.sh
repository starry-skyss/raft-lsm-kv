#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

API_URL="${API_URL:-http://localhost:8080}"
RESULT_ROOT="${RESULT_ROOT:-benchmark_results}"
RUN_ID="${RUN_ID:-final_tests_$(date +%Y%m%d_%H%M%S)}"
RESULT_DIR="${RESULT_ROOT}/${RUN_ID}"
SERVER_WAIT_SECONDS="${SERVER_WAIT_SECONDS:-30}"
GOCACHE="${GOCACHE:-/tmp/go-build}"

mkdir -p "${RESULT_DIR}"

SERVER_LOG="${RESULT_DIR}/functional_server.log"
RESULT_MD="${RESULT_DIR}/functional_results.md"
COMMAND_FILE="${RESULT_DIR}/functional_command.txt"

if ! command -v curl >/dev/null 2>&1; then
	echo "curl is required for functional tests." >&2
	exit 1
fi
if ! command -v setsid >/dev/null 2>&1; then
	echo "setsid is required for starting and stopping the server process group safely." >&2
	exit 1
fi

cat >"${COMMAND_FILE}" <<EOF
repo: ${REPO_ROOT}
api_url: ${API_URL}
result_dir: ${RESULT_DIR}
cleaned_data_dir: ${REPO_ROOT}/data
server_command: GOCACHE=${GOCACHE} go run ./cmd/kvnode
test_command: ${BASH_SOURCE[0]}
EOF

echo "Cleaning project runtime data directory: ${REPO_ROOT}/data"
rm -rf data

echo "Starting server for functional tests..."
setsid env GOCACHE="${GOCACHE}" go run ./cmd/kvnode >"${SERVER_LOG}" 2>&1 &
SERVER_PID=$!

cleanup() {
	if kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
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
}
trap cleanup EXIT

echo "Waiting for API Gateway at ${API_URL}..."
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

HTTP_STATUS=""
HTTP_BODY=""

http_request() {
	local method="$1"
	local path="$2"
	local payload="${3:-}"
	local response

	if [[ -n "${payload}" ]]; then
		response="$(curl -sS -X "${method}" -H "Content-Type: application/json" -d "${payload}" -w $'\n%{http_code}' "${API_URL}${path}" || true)"
	else
		response="$(curl -sS -X "${method}" -w $'\n%{http_code}' "${API_URL}${path}" || true)"
	fi

	HTTP_STATUS="${response##*$'\n'}"
	HTTP_BODY="${response%$'\n'*}"
}

md_escape() {
	local value="$1"
	value="${value//$'\n'/<br>}"
	value="${value//|/\\|}"
	printf "%s" "${value}"
}

passed=0
failed=0

init_report() {
	cat >"${RESULT_MD}" <<EOF
# Functional Test Results

- Time: $(date --iso-8601=seconds)
- API URL: ${API_URL}
- Data cleanup: ${REPO_ROOT}/data
- Server log: ${SERVER_LOG}

| Test Item | Steps | Expected Result | Actual Result | Pass |
| --- | --- | --- | --- | --- |
EOF
}

record_result() {
	local name="$1"
	local steps="$2"
	local expected="$3"
	local actual="$4"
	local ok="$5"

	if [[ "${ok}" == "true" ]]; then
		((passed += 1))
	else
		((failed += 1))
	fi

	printf '| %s | %s | %s | %s | %s |\n' \
		"$(md_escape "${name}")" \
		"$(md_escape "${steps}")" \
		"$(md_escape "${expected}")" \
		"$(md_escape "${actual}")" \
		"$(md_escape "$([[ "${ok}" == "true" ]] && printf PASS || printf FAIL)")" >>"${RESULT_MD}"
}

init_report

http_request POST /put '{"key":"functional_put_get","value":"value_1"}'
put_status="${HTTP_STATUS}"
put_body="${HTTP_BODY}"
http_request GET /get/functional_put_get
get_status="${HTTP_STATUS}"
get_body="${HTTP_BODY}"
ok=false
if [[ "${put_status}" == "200" && "${get_status}" == "200" && "${get_body}" == *'"value":"value_1"'* ]]; then
	ok=true
fi
record_result "Put then Get" "POST /put value_1; GET same key" "GET returns value_1" "PUT ${put_status} ${put_body}; GET ${get_status} ${get_body}" "${ok}"

http_request POST /put '{"key":"functional_overwrite","value":"old_value"}'
put1_status="${HTTP_STATUS}"
put1_body="${HTTP_BODY}"
http_request POST /put '{"key":"functional_overwrite","value":"new_value"}'
put2_status="${HTTP_STATUS}"
put2_body="${HTTP_BODY}"
http_request GET /get/functional_overwrite
get_status="${HTTP_STATUS}"
get_body="${HTTP_BODY}"
ok=false
if [[ "${put1_status}" == "200" && "${put2_status}" == "200" && "${get_status}" == "200" && "${get_body}" == *'"value":"new_value"'* && "${get_body}" != *'"value":"old_value"'* ]]; then
	ok=true
fi
record_result "Overwrite Put" "POST old_value; POST new_value; GET same key" "GET returns latest value new_value" "PUT1 ${put1_status} ${put1_body}; PUT2 ${put2_status} ${put2_body}; GET ${get_status} ${get_body}" "${ok}"

http_request POST /put '{"key":"functional_delete","value":"delete_me"}'
put_status="${HTTP_STATUS}"
put_body="${HTTP_BODY}"
http_request DELETE /delete/functional_delete
delete_status="${HTTP_STATUS}"
delete_body="${HTTP_BODY}"
http_request GET /get/functional_delete
get_status="${HTTP_STATUS}"
get_body="${HTTP_BODY}"
ok=false
if [[ "${put_status}" == "200" && "${delete_status}" == "200" && "${get_status}" == "404" && "${get_body}" == *"key not found"* ]]; then
	ok=true
fi
record_result "Delete then Get" "POST key; DELETE key; GET same key" "GET returns key not found" "PUT ${put_status} ${put_body}; DELETE ${delete_status} ${delete_body}; GET ${get_status} ${get_body}" "${ok}"

http_request GET /get/functional_missing_key
get_status="${HTTP_STATUS}"
get_body="${HTTP_BODY}"
ok=false
if [[ "${get_status}" == "404" && "${get_body}" == *"key not found"* ]]; then
	ok=true
fi
record_result "Missing Key Get" "GET nonexistent key" "Returns not found without HTTP 500 or 503" "GET ${get_status} ${get_body}" "${ok}"

http_request POST /put '{"key":"functional_read_after_write","value":"committed_value"}'
put_status="${HTTP_STATUS}"
put_body="${HTTP_BODY}"
http_request GET /get/functional_read_after_write
get_status="${HTTP_STATUS}"
get_body="${HTTP_BODY}"
ok=false
if [[ "${put_status}" == "200" && "${get_status}" == "200" && "${get_body}" == *'"value":"committed_value"'* ]]; then
	ok=true
fi
record_result "Read After Write" "POST committed_value; immediately GET same key" "Raft-backed GET sees committed write" "PUT ${put_status} ${put_body}; GET ${get_status} ${get_body}" "${ok}"

http_request POST /put '{"key":"functional_read_after_overwrite","value":"version_1"}'
put1_status="${HTTP_STATUS}"
put1_body="${HTTP_BODY}"
http_request POST /put '{"key":"functional_read_after_overwrite","value":"version_2"}'
put2_status="${HTTP_STATUS}"
put2_body="${HTTP_BODY}"
http_request GET /get/functional_read_after_overwrite
get_status="${HTTP_STATUS}"
get_body="${HTTP_BODY}"
ok=false
if [[ "${put1_status}" == "200" && "${put2_status}" == "200" && "${get_status}" == "200" && "${get_body}" == *'"value":"version_2"'* && "${get_body}" != *'"value":"version_1"'* ]]; then
	ok=true
fi
record_result "Read After Overwrite" "POST version_1; POST version_2; immediately GET same key" "GET sees version_2, not version_1" "PUT1 ${put1_status} ${put1_body}; PUT2 ${put2_status} ${put2_body}; GET ${get_status} ${get_body}" "${ok}"

http_request POST /put '{"key":"functional_read_after_delete","value":"will_be_deleted"}'
put_status="${HTTP_STATUS}"
put_body="${HTTP_BODY}"
http_request DELETE /delete/functional_read_after_delete
delete_status="${HTTP_STATUS}"
delete_body="${HTTP_BODY}"
http_request GET /get/functional_read_after_delete
get_status="${HTTP_STATUS}"
get_body="${HTTP_BODY}"
ok=false
if [[ "${put_status}" == "200" && "${delete_status}" == "200" && "${get_status}" == "404" && "${get_body}" == *"key not found"* ]]; then
	ok=true
fi
record_result "Read After Delete" "POST key; DELETE key; immediately GET same key" "Delete is visible and GET returns key not found" "PUT ${put_status} ${put_body}; DELETE ${delete_status} ${delete_body}; GET ${get_status} ${get_body}" "${ok}"

cat >>"${RESULT_MD}" <<EOF

## Summary

- Passed: ${passed}
- Failed: ${failed}
- Result: $([[ "${failed}" == "0" ]] && printf PASS || printf FAIL)
EOF

echo "Functional results written to ${RESULT_MD}"

if [[ "${failed}" != "0" ]]; then
	exit 1
fi
