# Compaction Ablation Summary

- Case: compaction_on_mixed_50_50_c10_seed2

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | mixed_50_50 |
| Concurrency | 10 |
| Seed | 2 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 100.00% |
| Success QPS | 73.37 |
| Get success QPS | 36.70 |
| Get P99 latency ms | 291.381 |
| Get P99.9 latency ms | 487.806 |
| SSTable files total | 1143 |
| Data size bytes | 20143791 |
| WAL files total | 6 |
| WAL size bytes | 10779 |
| Avg SSTable files touched per Get | 21.863 |
| Compaction count delta | 126 |
| Compaction input files delta | 504 |
| Compaction output files delta | 126 |
| Compaction total ms delta | 3565.715 |
| Compaction max ms | 179.978 |
| Compaction failed delta | 0 |
