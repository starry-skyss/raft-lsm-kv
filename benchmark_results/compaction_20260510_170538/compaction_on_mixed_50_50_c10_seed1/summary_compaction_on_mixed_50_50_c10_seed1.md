# Compaction Ablation Summary

- Case: compaction_on_mixed_50_50_c10_seed1

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | mixed_50_50 |
| Concurrency | 10 |
| Seed | 1 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 53.36% |
| Success QPS | 105.64 |
| Get success QPS | 52.04 |
| Get P99 latency ms | 242.284 |
| Get P99.9 latency ms | 405.290 |
| SSTable files total | 726 |
| Data size bytes | 11076936 |
| WAL files total | 6 |
| WAL size bytes | 6042 |
| Avg SSTable files touched per Get | 21.614 |
| Compaction count delta | 48 |
| Compaction input files delta | 192 |
| Compaction output files delta | 48 |
| Compaction total ms delta | 1872.883 |
| Compaction max ms | 174.605 |
| Compaction failed delta | 0 |
