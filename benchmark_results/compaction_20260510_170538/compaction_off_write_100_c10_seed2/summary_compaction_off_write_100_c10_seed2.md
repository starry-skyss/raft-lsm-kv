# Compaction Ablation Summary

- Case: compaction_off_write_100_c10_seed2

| Metric | Value |
| --- | --- |
| Enable compaction | false |
| Workload | write_100 |
| Concurrency | 10 |
| Seed | 2 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 62.01% |
| Success QPS | 67.31 |
| Get success QPS | 0.00 |
| Get P99 latency ms | 0.000 |
| Get P99.9 latency ms | 0.000 |
| SSTable files total | 1767 |
| Data size bytes | 20706945 |
| WAL files total | 6 |
| WAL size bytes | 11658 |
| Avg SSTable files touched per Get | 0.000 |
| Compaction count delta | 0 |
| Compaction input files delta | 0 |
| Compaction output files delta | 0 |
| Compaction total ms delta | 0.000 |
| Compaction max ms | 0.000 |
| Compaction failed delta | 0 |
