# LSM File Stats

- Case: compaction_off_delete_mixed_c10_seed1
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 5.0 MiB (5192187 B) | 2 | 4.4 KiB (4520 B) | 376 | 1.6 MiB (1723137 B) | 242.0 KiB (247783 B) |
| node_1 | 5.0 MiB (5192187 B) | 2 | 4.4 KiB (4520 B) | 376 | 1.6 MiB (1723137 B) | 242.0 KiB (247783 B) |
| node_2 | 5.0 MiB (5192187 B) | 2 | 4.4 KiB (4520 B) | 376 | 1.6 MiB (1723137 B) | 242.0 KiB (247783 B) |
