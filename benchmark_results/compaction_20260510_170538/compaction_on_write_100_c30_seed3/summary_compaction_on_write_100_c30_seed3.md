# Compaction Ablation Summary

- Case: compaction_on_write_100_c30_seed3

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | write_100 |
| Concurrency | 30 |
| Seed | 3 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 64.06% |
| Success QPS | 69.37 |
| Get success QPS | 0.00 |
| Get P99 latency ms | 0.000 |
| Get P99.9 latency ms | 0.000 |
| SSTable files total | 1578 |
| Data size bytes | 21329958 |
| WAL files total | 6 |
| WAL size bytes | 9057 |
| Avg SSTable files touched per Get | 0.000 |
| Compaction count delta | 84 |
| Compaction input files delta | 336 |
| Compaction output files delta | 84 |
| Compaction total ms delta | 2557.609 |
| Compaction max ms | 186.419 |
| Compaction failed delta | 0 |
