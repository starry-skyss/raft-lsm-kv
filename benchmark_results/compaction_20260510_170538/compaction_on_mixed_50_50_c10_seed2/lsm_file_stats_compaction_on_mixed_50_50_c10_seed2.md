# LSM File Stats

- Case: compaction_on_mixed_50_50_c10_seed2
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 6.4 MiB (6714593 B) | 2 | 3.5 KiB (3593 B) | 381 | 2.1 MiB (2242571 B) | 428.6 KiB (438911 B) |
| node_1 | 6.4 MiB (6714593 B) | 2 | 3.5 KiB (3593 B) | 381 | 2.1 MiB (2242571 B) | 428.6 KiB (438911 B) |
| node_2 | 6.4 MiB (6714605 B) | 2 | 3.5 KiB (3593 B) | 381 | 2.1 MiB (2242571 B) | 428.6 KiB (438923 B) |
