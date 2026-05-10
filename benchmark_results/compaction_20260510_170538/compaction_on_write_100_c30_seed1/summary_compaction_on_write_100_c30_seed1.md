# Compaction Ablation Summary

- Case: compaction_on_write_100_c30_seed1

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | write_100 |
| Concurrency | 30 |
| Seed | 1 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 84.04% |
| Success QPS | 61.60 |
| Get success QPS | 0.00 |
| Get P99 latency ms | 0.000 |
| Get P99.9 latency ms | 0.000 |
| SSTable files total | 2031 |
| Data size bytes | 28794446 |
| WAL files total | 6 |
| WAL size bytes | 5037 |
| Avg SSTable files touched per Get | 0.000 |
| Compaction count delta | 123 |
| Compaction input files delta | 492 |
| Compaction output files delta | 123 |
| Compaction total ms delta | 3943.045 |
| Compaction max ms | 128.170 |
| Compaction failed delta | 0 |
