# Compaction Ablation Summary

- Case: compaction_on_mixed_50_50_c30_seed2

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | mixed_50_50 |
| Concurrency | 30 |
| Seed | 2 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 70.55% |
| Success QPS | 96.64 |
| Get success QPS | 48.66 |
| Get P99 latency ms | 792.494 |
| Get P99.9 latency ms | 1318.486 |
| SSTable files total | 888 |
| Data size bytes | 14243186 |
| WAL files total | 6 |
| WAL size bytes | 6048 |
| Avg SSTable files touched per Get | 22.516 |
| Compaction count delta | 69 |
| Compaction input files delta | 276 |
| Compaction output files delta | 69 |
| Compaction total ms delta | 1605.702 |
| Compaction max ms | 32.280 |
| Compaction failed delta | 0 |
