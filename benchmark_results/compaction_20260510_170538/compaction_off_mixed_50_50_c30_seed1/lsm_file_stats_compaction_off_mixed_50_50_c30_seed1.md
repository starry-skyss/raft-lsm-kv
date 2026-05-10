# LSM File Stats

- Case: compaction_off_mixed_50_50_c30_seed1
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 3.8 MiB (3935423 B) | 2 | 718 B | 307 | 1.3 MiB (1383768 B) | 157.1 KiB (160843 B) |
| node_1 | 3.8 MiB (3935423 B) | 2 | 718 B | 307 | 1.3 MiB (1383768 B) | 157.1 KiB (160843 B) |
| node_2 | 3.8 MiB (3936260 B) | 2 | 718 B | 307 | 1.3 MiB (1383768 B) | 157.1 KiB (160843 B) |
