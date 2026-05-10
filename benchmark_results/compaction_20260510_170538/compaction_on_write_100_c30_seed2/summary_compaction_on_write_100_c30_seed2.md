# Compaction Ablation Summary

- Case: compaction_on_write_100_c30_seed2

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | write_100 |
| Concurrency | 30 |
| Seed | 2 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 99.95% |
| Success QPS | 60.10 |
| Get success QPS | 0.00 |
| Get P99 latency ms | 0.000 |
| Get P99.9 latency ms | 0.000 |
| SSTable files total | 2403 |
| Data size bytes | 35046444 |
| WAL files total | 6 |
| WAL size bytes | 6906 |
| Avg SSTable files touched per Get | 0.000 |
| Compaction count delta | 150 |
| Compaction input files delta | 600 |
| Compaction output files delta | 150 |
| Compaction total ms delta | 3807.873 |
| Compaction max ms | 58.855 |
| Compaction failed delta | 0 |
