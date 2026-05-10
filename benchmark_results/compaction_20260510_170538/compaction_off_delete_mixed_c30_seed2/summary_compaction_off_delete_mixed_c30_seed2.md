# Compaction Ablation Summary

- Case: compaction_off_delete_mixed_c30_seed2

| Metric | Value |
| --- | --- |
| Enable compaction | false |
| Workload | delete_mixed |
| Concurrency | 30 |
| Seed | 2 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 80.78% |
| Success QPS | 83.06 |
| Get success QPS | 33.17 |
| Get P99 latency ms | 966.868 |
| Get P99.9 latency ms | 1365.289 |
| SSTable files total | 1032 |
| Data size bytes | 14289036 |
| WAL files total | 6 |
| WAL size bytes | 5358 |
| Avg SSTable files touched per Get | 19.374 |
| Compaction count delta | 0 |
| Compaction input files delta | 0 |
| Compaction output files delta | 0 |
| Compaction total ms delta | 0.000 |
| Compaction max ms | 0.000 |
| Compaction failed delta | 0 |
