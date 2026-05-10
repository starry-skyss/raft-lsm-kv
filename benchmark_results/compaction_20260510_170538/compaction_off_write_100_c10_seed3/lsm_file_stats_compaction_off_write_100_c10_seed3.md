# LSM File Stats

- Case: compaction_off_write_100_c10_seed3
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 10.1 MiB (10617566 B) | 2 | 864 B | 867 | 3.7 MiB (3907948 B) | 1.4 MiB (1416363 B) |
| node_1 | 10.1 MiB (10617566 B) | 2 | 864 B | 867 | 3.7 MiB (3907948 B) | 1.4 MiB (1416363 B) |
| node_2 | 10.1 MiB (10617566 B) | 2 | 864 B | 867 | 3.7 MiB (3907948 B) | 1.4 MiB (1416363 B) |
