# LSM File Stats

- Case: compaction_off_mixed_50_50_c10_seed3
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 6.2 MiB (6473595 B) | 2 | 2.1 KiB (2160 B) | 485 | 2.1 MiB (2186104 B) | 414.0 KiB (423927 B) |
| node_1 | 6.2 MiB (6473595 B) | 2 | 2.1 KiB (2160 B) | 485 | 2.1 MiB (2186104 B) | 414.0 KiB (423927 B) |
| node_2 | 6.2 MiB (6473595 B) | 2 | 2.1 KiB (2160 B) | 485 | 2.1 MiB (2186104 B) | 414.0 KiB (423927 B) |
