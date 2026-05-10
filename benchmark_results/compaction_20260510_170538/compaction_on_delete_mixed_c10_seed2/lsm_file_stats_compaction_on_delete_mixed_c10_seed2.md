# LSM File Stats

- Case: compaction_on_delete_mixed_c10_seed2
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 5.4 MiB (5644337 B) | 2 | 4.0 KiB (4104 B) | 278 | 1.7 MiB (1811852 B) | 267.5 KiB (273878 B) |
| node_1 | 5.4 MiB (5644729 B) | 2 | 4.0 KiB (4104 B) | 278 | 1.7 MiB (1812236 B) | 267.5 KiB (273886 B) |
| node_2 | 5.4 MiB (5644729 B) | 2 | 4.0 KiB (4104 B) | 278 | 1.7 MiB (1812236 B) | 267.5 KiB (273886 B) |
