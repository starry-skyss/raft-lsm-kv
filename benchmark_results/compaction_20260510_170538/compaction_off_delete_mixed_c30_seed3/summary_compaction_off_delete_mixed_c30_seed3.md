# Compaction Ablation Summary

- Case: compaction_off_delete_mixed_c30_seed3

| Metric | Value |
| --- | --- |
| Enable compaction | false |
| Workload | delete_mixed |
| Concurrency | 30 |
| Seed | 3 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 82.55% |
| Success QPS | 79.98 |
| Get success QPS | 32.23 |
| Get P99 latency ms | 1040.068 |
| Get P99.9 latency ms | 4771.264 |
| SSTable files total | 1053 |
| Data size bytes | 14598989 |
| WAL files total | 6 |
| WAL size bytes | 12915 |
| Avg SSTable files touched per Get | 19.605 |
| Compaction count delta | 0 |
| Compaction input files delta | 0 |
| Compaction output files delta | 0 |
| Compaction total ms delta | 0.000 |
| Compaction max ms | 0.000 |
| Compaction failed delta | 0 |
