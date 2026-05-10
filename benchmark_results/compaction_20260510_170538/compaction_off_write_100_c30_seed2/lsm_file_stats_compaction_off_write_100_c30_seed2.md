# LSM File Stats

- Case: compaction_off_write_100_c30_seed2
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 11.3 MiB (11815757 B) | 2 | 1.3 KiB (1295 B) | 951 | 4.1 MiB (4286508 B) | 1.6 MiB (1712883 B) |
| node_1 | 11.3 MiB (11815757 B) | 2 | 1.3 KiB (1295 B) | 951 | 4.1 MiB (4286508 B) | 1.6 MiB (1712883 B) |
| node_2 | 11.3 MiB (11815757 B) | 2 | 1.3 KiB (1295 B) | 951 | 4.1 MiB (4286508 B) | 1.6 MiB (1712883 B) |
