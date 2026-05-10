#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

RESULT_ROOT="${RESULT_ROOT:-benchmark_results}"
RUN_ID="${RUN_ID:-recovery_quant_$(date +%Y%m%d_%H%M%S)}"
RESULT_DIR="${RESULT_ROOT}/${RUN_ID}"
GOCACHE="${GOCACHE:-/tmp/go-build}"
TEST_PATTERN='TestQuantitativeRecoveryScenarios'

mkdir -p "${RESULT_DIR}"

RESULT_CSV="${RESULT_DIR}/recovery_quant_results.csv"
RESULT_MD="${RESULT_DIR}/recovery_quant_results.md"
TEST_LOG="${RESULT_DIR}/recovery_quant_go_test.log"
COMMAND_FILE="${RESULT_DIR}/recovery_quant_command.txt"

cat >"${COMMAND_FILE}" <<EOF
repo: ${REPO_ROOT}
result_dir: ${RESULT_DIR}
cleaned_data_dir: ${REPO_ROOT}/data
test_command: GOCACHE=${GOCACHE} go test -v ./internal/lsm -run '${TEST_PATTERN}' -count=1
EOF

echo "Cleaning project runtime data directory: ${REPO_ROOT}/data"
rm -rf data

set +e
GOCACHE="${GOCACHE}" go test -v ./internal/lsm -run "${TEST_PATTERN}" -count=1 2>&1 | tee "${TEST_LOG}"
test_status="${PIPESTATUS[0]}"
set -e

lookup_field() {
	local key="$1"
	shift
	local part
	for part in "$@"; do
		if [[ "${part}" == "${key}="* ]]; then
			printf "%s" "${part#*=}"
			return
		fi
	done
	printf ""
}

csv_escape() {
	local value="$1"
	value="${value//\"/\"\"}"
	printf '"%s"' "${value}"
}

{
	printf 'scenario,run_id,write_key_count,overwrite_key_count,delete_key_count,verify_key_count,verify_success_count,recovery_ms,wal_file_count_before,sstable_file_count_before,sstable_file_count_after,manifest_loaded,pass\n'

	result_lines="$(grep 'RECOVERY_QUANT|' "${TEST_LOG}" || true)"
	if [[ -n "${result_lines}" ]]; then
		while IFS= read -r line; do
			line="${line#*RECOVERY_QUANT|}"
			IFS='|' read -r -a parts <<<"${line}"

			scenario="$(lookup_field "scenario" "${parts[@]}")"
			run_id="$(lookup_field "run_id" "${parts[@]}")"
			write_key_count="$(lookup_field "write_key_count" "${parts[@]}")"
			overwrite_key_count="$(lookup_field "overwrite_key_count" "${parts[@]}")"
			delete_key_count="$(lookup_field "delete_key_count" "${parts[@]}")"
			verify_key_count="$(lookup_field "verify_key_count" "${parts[@]}")"
			verify_success_count="$(lookup_field "verify_success_count" "${parts[@]}")"
			recovery_ms="$(lookup_field "recovery_ms" "${parts[@]}")"
			wal_file_count_before="$(lookup_field "wal_file_count_before" "${parts[@]}")"
			sstable_file_count_before="$(lookup_field "sstable_file_count_before" "${parts[@]}")"
			sstable_file_count_after="$(lookup_field "sstable_file_count_after" "${parts[@]}")"
			manifest_loaded="$(lookup_field "manifest_loaded" "${parts[@]}")"
			pass="$(lookup_field "pass" "${parts[@]}")"

			printf '%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n' \
				"$(csv_escape "${scenario}")" \
				"${run_id}" \
				"${write_key_count}" \
				"${overwrite_key_count}" \
				"${delete_key_count}" \
				"${verify_key_count}" \
				"${verify_success_count}" \
				"${recovery_ms}" \
				"${wal_file_count_before}" \
				"${sstable_file_count_before}" \
				"${sstable_file_count_after}" \
				"${manifest_loaded}" \
				"${pass}"
		done <<<"${result_lines}"
	fi
} >"${RESULT_CSV}"

data_rows="$(tail -n +2 "${RESULT_CSV}" | wc -l | tr -d ' ')"
pass_rows="$(awk -F, 'NR > 1 && $13 == "true" { count++ } END { print count + 0 }' "${RESULT_CSV}")"
fail_rows="$((data_rows - pass_rows))"

cat >"${RESULT_MD}" <<EOF
# Recovery Quantitative Test Results

- Time: $(date --iso-8601=seconds)
- Go test log: ${TEST_LOG}
- CSV: ${RESULT_CSV}
- Go test exit code: ${test_status}

| scenario | run_id | write_key_count | overwrite_key_count | delete_key_count | verify_key_count | verify_success_count | recovery_ms | wal_file_count_before | sstable_file_count_before | sstable_file_count_after | manifest_loaded | pass |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
EOF

tail -n +2 "${RESULT_CSV}" | while IFS=, read -r scenario run_id write_key_count overwrite_key_count delete_key_count verify_key_count verify_success_count recovery_ms wal_file_count_before sstable_file_count_before sstable_file_count_after manifest_loaded pass; do
	scenario="${scenario%\"}"
	scenario="${scenario#\"}"
	printf '| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n' \
		"${scenario}" \
		"${run_id}" \
		"${write_key_count}" \
		"${overwrite_key_count}" \
		"${delete_key_count}" \
		"${verify_key_count}" \
		"${verify_success_count}" \
		"${recovery_ms}" \
		"${wal_file_count_before}" \
		"${sstable_file_count_before}" \
		"${sstable_file_count_after}" \
		"${manifest_loaded}" \
		"${pass}" >>"${RESULT_MD}"
done

cat >>"${RESULT_MD}" <<EOF

## Average Recovery Time By Scenario

| scenario | run_count | avg_recovery_ms |
| --- | --- | --- |
EOF

awk -F, 'NR > 1 {
	gsub(/"/, "", $1)
	sum[$1] += $8
	count[$1]++
}
END {
	for (scenario in count) {
		printf("| %s | %d | %.3f |\n", scenario, count[scenario], sum[scenario] / count[scenario])
	}
}' "${RESULT_CSV}" | sort >>"${RESULT_MD}"

cat >>"${RESULT_MD}" <<EOF

## Summary

- Data rows: ${data_rows}
- Pass rows: ${pass_rows}
- Fail rows: ${fail_rows}
- Result: $([[ "${test_status}" == "0" && "${data_rows}" == "12" && "${fail_rows}" == "0" ]] && printf PASS || printf FAIL)
EOF

echo "Recovery quantitative CSV written to ${RESULT_CSV}"
echo "Recovery quantitative Markdown written to ${RESULT_MD}"

if [[ "${test_status}" != "0" || "${data_rows}" != "12" || "${fail_rows}" != "0" ]]; then
	exit 1
fi
