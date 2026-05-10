# Compaction Ablation Summary

- Case: compaction_off_mixed_50_50_c30_seed3

| Metric | Value |
| --- | --- |
| Enable compaction | false |
| Workload | mixed_50_50 |
| Concurrency | 30 |
| Seed | 3 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 89.03% |
| Success QPS | 84.06 |
| Get success QPS | 42.33 |
| Get P99 latency ms | 956.762 |
| Get P99.9 latency ms | 1537.743 |
| SSTable files total | 1359 |
| Data size bytes | 18036889 |
| WAL files total | 6 |
| WAL size bytes | 10359 |
| Avg SSTable files touched per Get | 27.838 |
| Compaction count delta | 0 |
| Compaction input files delta | 0 |
| Compaction output files delta | 0 |
| Compaction total ms delta | 0.000 |
| Compaction max ms | 0.000 |
| Compaction failed delta | 0 |
