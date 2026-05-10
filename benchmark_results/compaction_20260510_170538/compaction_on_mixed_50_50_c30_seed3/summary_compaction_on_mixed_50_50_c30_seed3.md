# Compaction Ablation Summary

- Case: compaction_on_mixed_50_50_c30_seed3

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | mixed_50_50 |
| Concurrency | 30 |
| Seed | 3 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 65.79% |
| Success QPS | 103.28 |
| Get success QPS | 51.94 |
| Get P99 latency ms | 712.179 |
| Get P99.9 latency ms | 1214.419 |
| SSTable files total | 852 |
| Data size bytes | 13343775 |
| WAL files total | 6 |
| WAL size bytes | 2592 |
| Avg SSTable files touched per Get | 22.785 |
| Compaction count delta | 60 |
| Compaction input files delta | 240 |
| Compaction output files delta | 60 |
| Compaction total ms delta | 1687.308 |
| Compaction max ms | 81.224 |
| Compaction failed delta | 0 |
