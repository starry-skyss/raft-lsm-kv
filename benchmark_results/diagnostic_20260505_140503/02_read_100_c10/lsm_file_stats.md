# LSM File Stats

- Case: 02_read_100_c10
- Data directory: data
- Compaction log entries in server.log: 0
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 4.0 MiB (4218454 B) | 2 | 2.5 KiB (2610 B) | 232 | 1.4 MiB (1457335 B) | 176.4 KiB (180594 B) |
| node_1 | 4.0 MiB (4218466 B) | 2 | 2.5 KiB (2610 B) | 232 | 1.4 MiB (1457335 B) | 176.4 KiB (180606 B) |
| node_2 | 4.0 MiB (4218466 B) | 2 | 2.5 KiB (2610 B) | 232 | 1.4 MiB (1457335 B) | 176.4 KiB (180606 B) |
