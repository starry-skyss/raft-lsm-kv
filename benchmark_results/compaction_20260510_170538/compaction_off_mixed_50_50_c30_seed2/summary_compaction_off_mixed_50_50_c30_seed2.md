# Compaction Ablation Summary

- Case: compaction_off_mixed_50_50_c30_seed2

| Metric | Value |
| --- | --- |
| Enable compaction | false |
| Workload | mixed_50_50 |
| Concurrency | 30 |
| Seed | 2 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 72.24% |
| Success QPS | 94.77 |
| Get success QPS | 47.75 |
| Get P99 latency ms | 790.137 |
| Get P99.9 latency ms | 1225.827 |
| SSTable files total | 1119 |
| Data size bytes | 14646509 |
| WAL files total | 6 |
| WAL size bytes | 2592 |
| Avg SSTable files touched per Get | 27.712 |
| Compaction count delta | 0 |
| Compaction input files delta | 0 |
| Compaction output files delta | 0 |
| Compaction total ms delta | 0.000 |
| Compaction max ms | 0.000 |
| Compaction failed delta | 0 |
