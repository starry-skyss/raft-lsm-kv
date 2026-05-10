# Compaction Ablation Summary

- Case: compaction_off_mixed_50_50_c10_seed1

| Metric | Value |
| --- | --- |
| Enable compaction | false |
| Workload | mixed_50_50 |
| Concurrency | 10 |
| Seed | 1 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 80.52% |
| Success QPS | 80.75 |
| Get success QPS | 39.98 |
| Get P99 latency ms | 317.053 |
| Get P99.9 latency ms | 513.019 |
| SSTable files total | 1260 |
| Data size bytes | 16521960 |
| WAL files total | 6 |
| WAL size bytes | 3024 |
| Avg SSTable files touched per Get | 27.444 |
| Compaction count delta | 0 |
| Compaction input files delta | 0 |
| Compaction output files delta | 0 |
| Compaction total ms delta | 0.000 |
| Compaction max ms | 0.000 |
| Compaction failed delta | 0 |
