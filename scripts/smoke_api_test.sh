#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"

if ! command -v curl >/dev/null 2>&1; then
	echo "ERROR: curl is required." >&2
	exit 1
fi

passed=0
failed=0
case_no=0

HTTP_STATUS=""
HTTP_BODY=""

separator() {
	printf '%s\n' '------------------------------------------------------------'
}

request() {
	local method="$1"
	local path="$2"
	local body="${3:-}"
	local response

	if [[ -n "${body}" ]]; then
		response="$(curl -sS -X "${method}" -H "Content-Type: application/json" -d "${body}" -w $'\n%{http_code}' "${API_URL}${path}" || true)"
	else
		response="$(curl -sS -X "${method}" -w $'\n%{http_code}' "${API_URL}${path}" || true)"
	fi

	HTTP_STATUS="${response##*$'\n'}"
	HTTP_BODY="${response%$'\n'*}"
}

print_request_result() {
	local method="$1"
	local path="$2"
	local body="$3"
	local status="$4"
	local response_body="$5"

	printf 'Request Method : %s\n' "${method}"
	printf 'Request Path   : %s\n' "${path}"
	printf 'Request Body   : %s\n' "${body:-<empty>}"
	printf 'HTTP Status    : %s\n' "${status}"
	printf 'Response Body  : %s\n' "${response_body}"
}

begin_case() {
	local title="$1"
	((case_no += 1))
	separator
	printf 'Test %02d: %s\n' "${case_no}" "${title}"
	separator
}

end_case() {
	local ok="$1"
	local message="$2"

	if [[ "${ok}" == "true" ]]; then
		((passed += 1))
		printf 'Check Result   : PASS - %s\n' "${message}"
	else
		((failed += 1))
		printf 'Check Result   : FAIL - %s\n' "${message}"
	fi
	printf '\n'
}

run_single_request_case() {
	local title="$1"
	local method="$2"
	local path="$3"
	local body="$4"
	local expected_message="$5"
	local check_expr="$6"

	begin_case "${title}"
	request "${method}" "${path}" "${body}"
	print_request_result "${method}" "${path}" "${body}" "${HTTP_STATUS}" "${HTTP_BODY}"

	local ok=false
	if eval "${check_expr}"; then
		ok=true
	fi
	end_case "${ok}" "${expected_message}"
}

echo "Raft-LSM-KV Basic HTTP API Smoke Test"
echo "Target API Gateway: ${API_URL}"
echo

run_single_request_case \
	"Put(k1, v1) writes successfully" \
	"POST" \
	"/put" \
	'{"key":"k1","value":"v1"}' \
	"Put(k1, v1) returned HTTP 200 and success body" \
	'[[ "${HTTP_STATUS}" == "200" && "${HTTP_BODY}" == *"success"* ]]'

run_single_request_case \
	"Get(k1) returns v1" \
	"GET" \
	"/get/k1" \
	"" \
	"Get(k1) returned v1" \
	'[[ "${HTTP_STATUS}" == "200" && "${HTTP_BODY}" == *"\"value\":\"v1\""* ]]'

begin_case "Put(k1, v2), then Get(k1) returns v2"
request "POST" "/put" '{"key":"k1","value":"v2"}'
put_status="${HTTP_STATUS}"
put_body="${HTTP_BODY}"
print_request_result "POST" "/put" '{"key":"k1","value":"v2"}' "${put_status}" "${put_body}"
printf '\n'
request "GET" "/get/k1" ""
get_status="${HTTP_STATUS}"
get_body="${HTTP_BODY}"
print_request_result "GET" "/get/k1" "" "${get_status}" "${get_body}"
ok=false
if [[ "${put_status}" == "200" && "${get_status}" == "200" && "${get_body}" == *'"value":"v2"'* ]]; then
	ok=true
fi
end_case "${ok}" "Overwrite write is visible and Get(k1) returned v2"

begin_case "Put(k2, v1), Delete(k2), then Get(k2) is not found"
request "POST" "/put" '{"key":"k2","value":"v1"}'
put_status="${HTTP_STATUS}"
put_body="${HTTP_BODY}"
print_request_result "POST" "/put" '{"key":"k2","value":"v1"}' "${put_status}" "${put_body}"
printf '\n'
request "DELETE" "/delete/k2" ""
delete_status="${HTTP_STATUS}"
delete_body="${HTTP_BODY}"
print_request_result "DELETE" "/delete/k2" "" "${delete_status}" "${delete_body}"
printf '\n'
request "GET" "/get/k2" ""
get_status="${HTTP_STATUS}"
get_body="${HTTP_BODY}"
print_request_result "GET" "/get/k2" "" "${get_status}" "${get_body}"
ok=false
if [[ "${put_status}" == "200" && "${delete_status}" == "200" && "${get_status}" == "404" && "${get_body}" == *"key not found"* ]]; then
	ok=true
fi
end_case "${ok}" "Deleted key returned key not found / HTTP 404"

run_single_request_case \
	"Get(k_unknown) returns not found, not HTTP 500" \
	"GET" \
	"/get/k_unknown" \
	"" \
	"Unknown key returned key not found / HTTP 404 and did not return 500" \
	'[[ "${HTTP_STATUS}" != "500" && "${HTTP_STATUS}" == "404" && "${HTTP_BODY}" == *"key not found"* ]]'

begin_case "Put(k3, v1), then Get(k3) demonstrates committed strong read"
request "POST" "/put" '{"key":"k3","value":"v1"}'
put_status="${HTTP_STATUS}"
put_body="${HTTP_BODY}"
print_request_result "POST" "/put" '{"key":"k3","value":"v1"}' "${put_status}" "${put_body}"
printf '\n'
request "GET" "/get/k3" ""
get_status="${HTTP_STATUS}"
get_body="${HTTP_BODY}"
print_request_result "GET" "/get/k3" "" "${get_status}" "${get_body}"
ok=false
if [[ "${put_status}" == "200" && "${get_status}" == "200" && "${get_body}" == *'"value":"v1"'* ]]; then
	ok=true
fi
end_case "${ok}" "Get(k3) read the committed value v1"

separator
printf 'Summary: PASS=%d FAIL=%d TOTAL=%d\n' "${passed}" "${failed}" "$((passed + failed))"
separator

if [[ "${failed}" != "0" ]]; then
	exit 1
fi
