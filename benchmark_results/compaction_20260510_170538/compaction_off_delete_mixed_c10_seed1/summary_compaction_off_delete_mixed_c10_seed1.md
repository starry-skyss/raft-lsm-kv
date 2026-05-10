# Compaction Ablation Summary

- Case: compaction_off_delete_mixed_c10_seed1

| Metric | Value |
| --- | --- |
| Enable compaction | false |
| Workload | delete_mixed |
| Concurrency | 10 |
| Seed | 1 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 86.93% |
| Success QPS | 76.45 |
| Get success QPS | 29.93 |
| Get P99 latency ms | 408.876 |
| Get P99.9 latency ms | 552.167 |
| SSTable files total | 1128 |
| Data size bytes | 15576561 |
| WAL files total | 6 |
| WAL size bytes | 13560 |
| Avg SSTable files touched per Get | 19.223 |
| Compaction count delta | 0 |
| Compaction input files delta | 0 |
| Compaction output files delta | 0 |
| Compaction total ms delta | 0.000 |
| Compaction max ms | 0.000 |
| Compaction failed delta | 0 |
