# Compaction Ablation Summary

- Case: compaction_off_delete_mixed_c10_seed3

| Metric | Value |
| --- | --- |
| Enable compaction | false |
| Workload | delete_mixed |
| Concurrency | 10 |
| Seed | 3 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 85.87% |
| Success QPS | 72.00 |
| Get success QPS | 29.09 |
| Get P99 latency ms | 350.341 |
| Get P99.9 latency ms | 509.738 |
| SSTable files total | 1092 |
| Data size bytes | 15153195 |
| WAL files total | 6 |
| WAL size bytes | 12165 |
| Avg SSTable files touched per Get | 19.686 |
| Compaction count delta | 0 |
| Compaction input files delta | 0 |
| Compaction output files delta | 0 |
| Compaction total ms delta | 0.000 |
| Compaction max ms | 0.000 |
| Compaction failed delta | 0 |
