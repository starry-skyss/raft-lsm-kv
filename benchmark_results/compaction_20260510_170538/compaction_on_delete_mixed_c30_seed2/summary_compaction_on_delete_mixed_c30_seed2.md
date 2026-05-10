# Compaction Ablation Summary

- Case: compaction_on_delete_mixed_c30_seed2

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | delete_mixed |
| Concurrency | 30 |
| Seed | 2 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 99.78% |
| Success QPS | 76.43 |
| Get success QPS | 30.45 |
| Get P99 latency ms | 1018.119 |
| Get P99.9 latency ms | 1637.428 |
| SSTable files total | 903 |
| Data size bytes | 17443316 |
| WAL files total | 6 |
| WAL size bytes | 9585 |
| Avg SSTable files touched per Get | 15.464 |
| Compaction count delta | 120 |
| Compaction input files delta | 480 |
| Compaction output files delta | 120 |
| Compaction total ms delta | 3321.838 |
| Compaction max ms | 117.388 |
| Compaction failed delta | 0 |
