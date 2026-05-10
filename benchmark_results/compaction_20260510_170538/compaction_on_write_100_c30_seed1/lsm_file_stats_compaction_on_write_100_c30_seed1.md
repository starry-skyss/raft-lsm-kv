# LSM File Stats

- Case: compaction_on_write_100_c30_seed1
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 9.2 MiB (9597768 B) | 2 | 1.5 KiB (1583 B) | 677 | 3.4 MiB (3557066 B) | 1.1 MiB (1150214 B) |
| node_1 | 9.2 MiB (9597924 B) | 2 | 1.5 KiB (1583 B) | 677 | 3.4 MiB (3557066 B) | 1.1 MiB (1150370 B) |
| node_2 | 9.2 MiB (9598754 B) | 2 | 1.8 KiB (1871 B) | 677 | 3.4 MiB (3557066 B) | 1.1 MiB (1150330 B) |
