# LSM File Stats

- Case: compaction_on_write_100_c10_seed2
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 11.1 MiB (11685664 B) | 2 | 144 B | 802 | 4.0 MiB (4233997 B) | 1.6 MiB (1634707 B) |
| node_1 | 11.1 MiB (11685700 B) | 2 | 144 B | 802 | 4.0 MiB (4233997 B) | 1.6 MiB (1634743 B) |
| node_2 | 11.1 MiB (11685696 B) | 2 | 144 B | 802 | 4.0 MiB (4233997 B) | 1.6 MiB (1634739 B) |
