# Compaction Ablation Summary

- Case: compaction_off_mixed_50_50_c10_seed2

| Metric | Value |
| --- | --- |
| Enable compaction | false |
| Workload | mixed_50_50 |
| Concurrency | 10 |
| Seed | 2 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 53.32% |
| Success QPS | 99.21 |
| Get success QPS | 50.05 |
| Get P99 latency ms | 255.917 |
| Get P99.9 latency ms | 358.477 |
| SSTable files total | 849 |
| Data size bytes | 10927214 |
| WAL files total | 6 |
| WAL size bytes | 4026 |
| Avg SSTable files touched per Get | 26.962 |
| Compaction count delta | 0 |
| Compaction input files delta | 0 |
| Compaction output files delta | 0 |
| Compaction total ms delta | 0.000 |
| Compaction max ms | 0.000 |
| Compaction failed delta | 0 |
