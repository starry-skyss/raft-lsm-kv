# LSM File Stats

- Case: compaction_off_write_100_c10_seed2
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 6.6 MiB (6902315 B) | 2 | 3.8 KiB (3886 B) | 589 | 2.5 MiB (2654864 B) | 621.4 KiB (636295 B) |
| node_1 | 6.6 MiB (6902315 B) | 2 | 3.8 KiB (3886 B) | 589 | 2.5 MiB (2654864 B) | 621.4 KiB (636295 B) |
| node_2 | 6.6 MiB (6902315 B) | 2 | 3.8 KiB (3886 B) | 589 | 2.5 MiB (2654864 B) | 621.4 KiB (636295 B) |
