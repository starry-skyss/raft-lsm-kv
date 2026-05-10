# Compaction Ablation Summary

- Case: compaction_on_delete_mixed_c10_seed1

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | delete_mixed |
| Concurrency | 10 |
| Seed | 1 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 99.97% |
| Success QPS | 74.60 |
| Get success QPS | 29.29 |
| Get P99 latency ms | 355.744 |
| Get P99.9 latency ms | 506.236 |
| SSTable files total | 915 |
| Data size bytes | 17640098 |
| WAL files total | 6 |
| WAL size bytes | 5466 |
| Avg SSTable files touched per Get | 15.189 |
| Compaction count delta | 123 |
| Compaction input files delta | 492 |
| Compaction output files delta | 123 |
| Compaction total ms delta | 4078.870 |
| Compaction max ms | 166.033 |
| Compaction failed delta | 0 |
