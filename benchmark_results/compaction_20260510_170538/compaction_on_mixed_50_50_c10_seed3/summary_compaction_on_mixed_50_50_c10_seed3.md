# Compaction Ablation Summary

- Case: compaction_on_mixed_50_50_c10_seed3

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | mixed_50_50 |
| Concurrency | 10 |
| Seed | 3 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 96.85% |
| Success QPS | 73.80 |
| Get success QPS | 37.16 |
| Get P99 latency ms | 397.938 |
| Get P99.9 latency ms | 546.675 |
| SSTable files total | 1113 |
| Data size bytes | 19546509 |
| WAL files total | 6 |
| WAL size bytes | 12516 |
| Avg SSTable files touched per Get | 22.013 |
| Compaction count delta | 120 |
| Compaction input files delta | 480 |
| Compaction output files delta | 120 |
| Compaction total ms delta | 2797.545 |
| Compaction max ms | 35.318 |
| Compaction failed delta | 0 |
