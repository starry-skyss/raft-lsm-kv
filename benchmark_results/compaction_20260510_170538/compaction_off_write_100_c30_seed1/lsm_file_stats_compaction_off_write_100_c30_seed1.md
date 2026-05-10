# LSM File Stats

- Case: compaction_off_write_100_c30_seed1
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 11.2 MiB (11717406 B) | 2 | 2.4 KiB (2447 B) | 944 | 4.1 MiB (4255022 B) | 1.6 MiB (1687095 B) |
| node_1 | 11.2 MiB (11714165 B) | 2 | 1.7 KiB (1727 B) | 944 | 4.1 MiB (4255022 B) | 1.6 MiB (1687095 B) |
| node_2 | 11.2 MiB (11714165 B) | 2 | 1.7 KiB (1727 B) | 944 | 4.1 MiB (4255022 B) | 1.6 MiB (1687095 B) |
