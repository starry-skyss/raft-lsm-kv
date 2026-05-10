# LSM File Stats

- Case: compaction_on_mixed_50_50_c10_seed3
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 6.2 MiB (6485553 B) | 2 | 4.1 KiB (4172 B) | 367 | 2.1 MiB (2170501 B) | 404.9 KiB (414649 B) |
| node_1 | 6.2 MiB (6485553 B) | 2 | 4.1 KiB (4172 B) | 367 | 2.1 MiB (2170501 B) | 404.9 KiB (414649 B) |
| node_2 | 6.2 MiB (6485565 B) | 2 | 4.1 KiB (4172 B) | 367 | 2.1 MiB (2170501 B) | 404.9 KiB (414661 B) |
