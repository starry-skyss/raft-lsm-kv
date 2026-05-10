# Compaction Ablation Summary

- Case: compaction_off_mixed_50_50_c30_seed1

| Metric | Value |
| --- | --- |
| Enable compaction | false |
| Workload | mixed_50_50 |
| Concurrency | 30 |
| Seed | 1 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 56.70% |
| Success QPS | 107.49 |
| Get success QPS | 52.81 |
| Get P99 latency ms | 733.691 |
| Get P99.9 latency ms | 1002.921 |
| SSTable files total | 921 |
| Data size bytes | 11807106 |
| WAL files total | 6 |
| WAL size bytes | 2154 |
| Avg SSTable files touched per Get | 26.221 |
| Compaction count delta | 0 |
| Compaction input files delta | 0 |
| Compaction output files delta | 0 |
| Compaction total ms delta | 0.000 |
| Compaction max ms | 0.000 |
| Compaction failed delta | 0 |
