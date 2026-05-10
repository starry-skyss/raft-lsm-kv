# Compaction Ablation Summary

- Case: compaction_on_delete_mixed_c30_seed3

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | delete_mixed |
| Concurrency | 30 |
| Seed | 3 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 83.29% |
| Success QPS | 85.88 |
| Get success QPS | 34.65 |
| Get P99 latency ms | 677.727 |
| Get P99.9 latency ms | 1193.427 |
| SSTable files total | 795 |
| Data size bytes | 14581372 |
| WAL files total | 6 |
| WAL size bytes | 5253 |
| Avg SSTable files touched per Get | 15.778 |
| Compaction count delta | 90 |
| Compaction input files delta | 360 |
| Compaction output files delta | 90 |
| Compaction total ms delta | 2647.740 |
| Compaction max ms | 167.791 |
| Compaction failed delta | 0 |
