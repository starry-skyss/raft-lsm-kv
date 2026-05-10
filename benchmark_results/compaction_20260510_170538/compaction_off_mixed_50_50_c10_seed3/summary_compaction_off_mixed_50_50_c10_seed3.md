# Compaction Ablation Summary

- Case: compaction_off_mixed_50_50_c10_seed3

| Metric | Value |
| --- | --- |
| Enable compaction | false |
| Workload | mixed_50_50 |
| Concurrency | 10 |
| Seed | 3 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 96.10% |
| Success QPS | 70.97 |
| Get success QPS | 35.85 |
| Get P99 latency ms | 444.792 |
| Get P99.9 latency ms | 595.125 |
| SSTable files total | 1455 |
| Data size bytes | 19420785 |
| WAL files total | 6 |
| WAL size bytes | 6480 |
| Avg SSTable files touched per Get | 27.873 |
| Compaction count delta | 0 |
| Compaction input files delta | 0 |
| Compaction output files delta | 0 |
| Compaction total ms delta | 0.000 |
| Compaction max ms | 0.000 |
| Compaction failed delta | 0 |
