# Compaction Ablation Summary

- Case: compaction_on_delete_mixed_c30_seed1

| Metric | Value |
| --- | --- |
| Enable compaction | true |
| Workload | delete_mixed |
| Concurrency | 30 |
| Seed | 1 |
| Total requests | 30000 |
| Keyspace | 1000 |
| Value size | 128 |
| Success rate | 88.71% |
| Success QPS | 77.92 |
| Get success QPS | 30.52 |
| Get P99 latency ms | 743.284 |
| Get P99.9 latency ms | 5148.642 |
| SSTable files total | 834 |
| Data size bytes | 15710691 |
| WAL files total | 6 |
| WAL size bytes | 6618 |
| Avg SSTable files touched per Get | 15.346 |
| Compaction count delta | 105 |
| Compaction input files delta | 420 |
| Compaction output files delta | 105 |
| Compaction total ms delta | 2537.031 |
| Compaction max ms | 53.861 |
| Compaction failed delta | 0 |
