# Compaction Ablation Summary

- Case: compaction_on_write_100_c10_seed1

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | write_100 |
| Concurrency | 10 |
| Seed | 1 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 84.76% |
| Success QPS | 68.26 |
| Get success QPS | 0.00 |
| Get P99 latency ms | 0.000 |
| Get P99.9 latency ms | 0.000 |
| SSTable files total | 2088 |
| Data size bytes | 29151361 |
| WAL files total | 6 |
| WAL size bytes | 429 |
| Avg SSTable files touched per Get | 0.000 |
| Compaction count delta | 111 |
| Compaction input files delta | 444 |
| Compaction output files delta | 111 |
| Compaction total ms delta | 2399.609 |
| Compaction max ms | 41.778 |
| Compaction failed delta | 0 |
