#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

RESULT_ROOT="${RESULT_ROOT:-benchmark_results}"
RUN_ID="${RUN_ID:-final_tests_$(date +%Y%m%d_%H%M%S)}"
RESULT_DIR="${RESULT_ROOT}/${RUN_ID}"
GOCACHE="${GOCACHE:-/tmp/go-build}"

mkdir -p "${RESULT_DIR}"

RESULT_MD="${RESULT_DIR}/recovery_results.md"
TEST_LOG="${RESULT_DIR}/recovery_go_test.log"
COMMAND_FILE="${RESULT_DIR}/recovery_command.txt"
TEST_PATTERN='Test(WALRecoveryPreservesUnflushedWrites|SSTableManifestRecoveryLoadsFlushedData|TombstoneRecoveryHidesDeletedValue)'

cat >"${COMMAND_FILE}" <<EOF
repo: ${REPO_ROOT}
result_dir: ${RESULT_DIR}
test_command: GOCACHE=${GOCACHE} go test -v ./internal/lsm -run '${TEST_PATTERN}' -count=1
EOF

set +e
GOCACHE="${GOCACHE}" go test -v ./internal/lsm -run "${TEST_PATTERN}" -count=1 2>&1 | tee "${TEST_LOG}"
test_status="${PIPESTATUS[0]}"
set -e

md_escape() {
	local value="$1"
	value="${value//$'\n'/<br>}"
	value="${value//|/\\|}"
	printf "%s" "${value}"
}

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
	printf "-"
}

cat >"${RESULT_MD}" <<EOF
# Recovery Test Results

- Time: $(date --iso-8601=seconds)
- Go test log: ${TEST_LOG}

| Test Item | Data Scale | Flush Triggered | SSTable Count | Manifest Size | Actual Result | Pass |
| --- | --- | --- | --- | --- | --- | --- |
EOF

result_lines="$(grep 'RECOVERY_RESULT|' "${TEST_LOG}" || true)"
if [[ -n "${result_lines}" ]]; then
	while IFS= read -r line; do
		line="${line#*RECOVERY_RESULT|}"
		IFS='|' read -r -a parts <<<"${line}"
		name="${parts[0]}"
		fields=("${parts[@]:1}")
		data_scale="$(lookup_field "data_scale" "${fields[@]}")"
		flush_triggered="$(lookup_field "flush_triggered" "${fields[@]}")"
		actual="$(lookup_field "actual" "${fields[@]}")"
		pass="$(lookup_field "pass" "${fields[@]}")"

		sstable_count="$(lookup_field "sstable_count" "${fields[@]}")"
		if [[ "${sstable_count}" == "-" ]]; then
			before="$(lookup_field "sstable_count_before" "${fields[@]}")"
			after="$(lookup_field "sstable_count_after" "${fields[@]}")"
			sstable_count="before=${before}, after=${after}"
		fi

		manifest_size="$(lookup_field "manifest_size" "${fields[@]}")"
		if [[ "${manifest_size}" == "-" ]]; then
			before="$(lookup_field "manifest_size_before" "${fields[@]}")"
			after="$(lookup_field "manifest_size_after" "${fields[@]}")"
			manifest_size="before=${before}, after=${after}"
		fi

		printf '| %s | %s | %s | %s | %s | %s | %s |\n' \
			"$(md_escape "${name}")" \
			"$(md_escape "${data_scale}")" \
			"$(md_escape "${flush_triggered}")" \
			"$(md_escape "${sstable_count}")" \
			"$(md_escape "${manifest_size}")" \
			"$(md_escape "${actual}")" \
			"$(md_escape "$([[ "${pass}" == "true" ]] && printf PASS || printf FAIL)")" >>"${RESULT_MD}"
	done <<<"${result_lines}"
else
	printf '| %s | %s | %s | %s | %s | %s | %s |\n' \
		"No RECOVERY_RESULT lines found" "-" "-" "-" "-" "See ${TEST_LOG}" "FAIL" >>"${RESULT_MD}"
fi

cat >>"${RESULT_MD}" <<EOF

## Summary

- Go test exit code: ${test_status}
- Result: $([[ "${test_status}" == "0" ]] && printf PASS || printf FAIL)
EOF

echo "Recovery results written to ${RESULT_MD}"

exit "${test_status}"
