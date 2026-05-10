# Recovery Quantitative Test Results

- Time: 2026-05-10T16:25:19+08:00
- Go test log: benchmark_results/recovery_quant_20260510_162356/recovery_quant_go_test.log
- CSV: benchmark_results/recovery_quant_20260510_162356/recovery_quant_results.csv
- Go test exit code: 0

| scenario | run_id | write_key_count | overwrite_key_count | delete_key_count | verify_key_count | verify_success_count | recovery_ms | wal_file_count_before | sstable_file_count_before | sstable_file_count_after | manifest_loaded | pass |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| wal | 1 | 1000 | 0 | 0 | 1000 | 1000 | 17.126 | 2 | 0 | 1 | true | true |
| wal | 2 | 1000 | 0 | 0 | 1000 | 1000 | 26.692 | 2 | 0 | 1 | true | true |
| wal | 3 | 1000 | 0 | 0 | 1000 | 1000 | 20.053 | 2 | 0 | 1 | true | true |
| sstable_manifest | 1 | 1000 | 0 | 0 | 1000 | 1000 | 20.958 | 2 | 17 | 18 | true | true |
| sstable_manifest | 2 | 1000 | 0 | 0 | 1000 | 1000 | 18.562 | 2 | 17 | 18 | true | true |
| sstable_manifest | 3 | 1000 | 0 | 0 | 1000 | 1000 | 18.943 | 2 | 17 | 18 | true | true |
| tombstone | 1 | 1000 | 0 | 1000 | 1000 | 1000 | 19.802 | 2 | 1 | 2 | true | true |
| tombstone | 2 | 1000 | 0 | 1000 | 1000 | 1000 | 18.255 | 2 | 1 | 2 | true | true |
| tombstone | 3 | 1000 | 0 | 1000 | 1000 | 1000 | 18.328 | 2 | 1 | 2 | true | true |
| mixed | 1 | 1000 | 500 | 250 | 1000 | 1000 | 16.667 | 2 | 4 | 5 | true | true |
| mixed | 2 | 1000 | 500 | 250 | 1000 | 1000 | 20.156 | 2 | 4 | 5 | true | true |
| mixed | 3 | 1000 | 500 | 250 | 1000 | 1000 | 18.407 | 2 | 4 | 5 | true | true |

## Average Recovery Time By Scenario

| scenario | run_count | avg_recovery_ms |
| --- | --- | --- |
| mixed | 3 | 18.410 |
| sstable_manifest | 3 | 19.488 |
| tombstone | 3 | 18.795 |
| wal | 3 | 21.290 |

## Summary

- Data rows: 12
- Pass rows: 12
- Fail rows: 0
- Result: PASS
