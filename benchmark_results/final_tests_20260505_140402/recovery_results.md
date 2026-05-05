# Recovery Test Results

- Time: 2026-05-05T14:04:02+08:00
- Go test log: benchmark_results/final_tests_20260505_140402/recovery_go_test.log

| Test Item | Data Scale | Flush Triggered | SSTable Count | Manifest Size | Actual Result | Pass |
| --- | --- | --- | --- | --- | --- | --- |
| WAL Recovery | 2 keys | false before restart, true during recovery bootstrap | before=0, after=1 | 2 | both WAL keys readable after reopen | PASS |
| SSTable Manifest Recovery | 20 keys, value_size=256 bytes | true | before=1, after=2 | before=2, after=6 | flushed and recovered keys readable after reopen | PASS |
| Tombstone Recovery | 1 put + 1 delete | true during recovery bootstrap | before=-, after=1 | 2 | deleted key returns not found after reopen | PASS |

## Summary

- Go test exit code: 0
- Result: PASS
