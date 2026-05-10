# Compaction Ablation Summary

- Case: compaction_off_delete_mixed_c10_seed2

| Metric | Value |
| --- | --- |
| Enable compaction | false |
| Workload | delete_mixed |
| Concurrency | 10 |
| Seed | 2 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 100.00% |
| Success QPS | 69.53 |
| Get success QPS | 27.71 |
| Get P99 latency ms | 331.532 |
| Get P99.9 latency ms | 505.215 |
| SSTable files total | 1266 |
| Data size bytes | 17714865 |
| WAL files total | 6 |
| WAL size bytes | 432 |
| Avg SSTable files touched per Get | 19.734 |
| Compaction count delta | 0 |
| Compaction input files delta | 0 |
| Compaction output files delta | 0 |
| Compaction total ms delta | 0.000 |
| Compaction max ms | 0.000 |
| Compaction failed delta | 0 |
