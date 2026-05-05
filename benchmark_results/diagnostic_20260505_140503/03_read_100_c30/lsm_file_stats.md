# LSM File Stats

- Case: 03_read_100_c30
- Data directory: data
- Compaction log entries in server.log: 0
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 4.0 MiB (4207352 B) | 2 | 2.5 KiB (2610 B) | 226 | 1.4 MiB (1457023 B) | 178.0 KiB (182246 B) |
| node_1 | 4.0 MiB (4207352 B) | 2 | 2.5 KiB (2610 B) | 226 | 1.4 MiB (1457023 B) | 178.0 KiB (182246 B) |
| node_2 | 4.0 MiB (4207364 B) | 2 | 2.5 KiB (2610 B) | 226 | 1.4 MiB (1457023 B) | 178.0 KiB (182258 B) |
