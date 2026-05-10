# LSM File Stats

- Case: compaction_on_write_100_c10_seed1
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 9.3 MiB (9717119 B) | 2 | 143 B | 696 | 3.4 MiB (3593563 B) | 1.1 MiB (1192826 B) |
| node_1 | 9.3 MiB (9717127 B) | 2 | 143 B | 696 | 3.4 MiB (3593563 B) | 1.1 MiB (1192834 B) |
| node_2 | 9.3 MiB (9717115 B) | 2 | 143 B | 696 | 3.4 MiB (3593563 B) | 1.1 MiB (1192822 B) |
