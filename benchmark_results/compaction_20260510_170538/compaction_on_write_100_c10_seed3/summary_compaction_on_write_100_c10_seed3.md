# Compaction Ablation Summary

- Case: compaction_on_write_100_c10_seed3

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | write_100 |
| Concurrency | 10 |
| Seed | 3 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 100.00% |
| Success QPS | 60.34 |
| Get success QPS | 0.00 |
| Get P99 latency ms | 0.000 |
| Get P99.9 latency ms | 0.000 |
| SSTable files total | 2406 |
| Data size bytes | 35122607 |
| WAL files total | 6 |
| WAL size bytes | 11652 |
| Avg SSTable files touched per Get | 0.000 |
| Compaction count delta | 150 |
| Compaction input files delta | 600 |
| Compaction output files delta | 150 |
| Compaction total ms delta | 4724.344 |
| Compaction max ms | 215.878 |
| Compaction failed delta | 0 |
