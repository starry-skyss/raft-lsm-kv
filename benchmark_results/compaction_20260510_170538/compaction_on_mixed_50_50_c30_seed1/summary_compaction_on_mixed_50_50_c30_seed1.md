# Compaction Ablation Summary

- Case: compaction_on_mixed_50_50_c30_seed1

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | mixed_50_50 |
| Concurrency | 30 |
| Seed | 1 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 85.89% |
| Success QPS | 84.98 |
| Get success QPS | 42.05 |
| Get P99 latency ms | 967.245 |
| Get P99.9 latency ms | 5186.605 |
| SSTable files total | 1059 |
| Data size bytes | 17501152 |
| WAL files total | 6 |
| WAL size bytes | 432 |
| Avg SSTable files touched per Get | 21.965 |
| Compaction count delta | 93 |
| Compaction input files delta | 372 |
| Compaction output files delta | 93 |
| Compaction total ms delta | 2585.895 |
| Compaction max ms | 107.461 |
| Compaction failed delta | 0 |
