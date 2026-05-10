# Compaction Ablation Summary

- Case: compaction_off_delete_mixed_c30_seed1

| Metric | Value |
| --- | --- |
| Enable compaction | false |
| Workload | delete_mixed |
| Concurrency | 30 |
| Seed | 1 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 85.21% |
| Success QPS | 82.32 |
| Get success QPS | 32.19 |
| Get P99 latency ms | 778.450 |
| Get P99.9 latency ms | 1183.540 |
| SSTable files total | 1107 |
| Data size bytes | 15270492 |
| WAL files total | 6 |
| WAL size bytes | 8385 |
| Avg SSTable files touched per Get | 19.146 |
| Compaction count delta | 0 |
| Compaction input files delta | 0 |
| Compaction output files delta | 0 |
| Compaction total ms delta | 0.000 |
| Compaction max ms | 0.000 |
| Compaction failed delta | 0 |
