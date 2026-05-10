# Compaction Ablation Summary

- Case: compaction_on_write_100_c10_seed2

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | write_100 |
| Concurrency | 10 |
| Seed | 2 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 100.00% |
| Success QPS | 59.41 |
| Get success QPS | 0.00 |
| Get P99 latency ms | 0.000 |
| Get P99.9 latency ms | 0.000 |
| SSTable files total | 2406 |
| Data size bytes | 35057060 |
| WAL files total | 6 |
| WAL size bytes | 432 |
| Avg SSTable files touched per Get | 0.000 |
| Compaction count delta | 150 |
| Compaction input files delta | 600 |
| Compaction output files delta | 150 |
| Compaction total ms delta | 4117.457 |
| Compaction max ms | 56.705 |
| Compaction failed delta | 0 |
