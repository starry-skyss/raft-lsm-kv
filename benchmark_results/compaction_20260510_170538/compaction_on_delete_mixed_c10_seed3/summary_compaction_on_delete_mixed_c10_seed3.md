# Compaction Ablation Summary

- Case: compaction_on_delete_mixed_c10_seed3

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | delete_mixed |
| Concurrency | 10 |
| Seed | 3 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 82.45% |
| Success QPS | 77.22 |
| Get success QPS | 31.16 |
| Get P99 latency ms | 300.520 |
| Get P99.9 latency ms | 463.337 |
| SSTable files total | 756 |
| Data size bytes | 14415935 |
| WAL files total | 6 |
| WAL size bytes | 11121 |
| Avg SSTable files touched per Get | 15.485 |
| Compaction count delta | 99 |
| Compaction input files delta | 396 |
| Compaction output files delta | 99 |
| Compaction total ms delta | 2927.526 |
| Compaction max ms | 178.872 |
| Compaction failed delta | 0 |
