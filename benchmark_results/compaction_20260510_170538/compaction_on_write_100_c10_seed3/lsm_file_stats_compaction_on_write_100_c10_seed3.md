# LSM File Stats

- Case: compaction_on_write_100_c10_seed3
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 11.2 MiB (11707538 B) | 2 | 3.8 KiB (3884 B) | 802 | 4.0 MiB (4232174 B) | 1.6 MiB (1654612 B) |
| node_1 | 11.2 MiB (11707526 B) | 2 | 3.8 KiB (3884 B) | 802 | 4.0 MiB (4232174 B) | 1.6 MiB (1654600 B) |
| node_2 | 11.2 MiB (11707543 B) | 2 | 3.8 KiB (3884 B) | 802 | 4.0 MiB (4232174 B) | 1.6 MiB (1654617 B) |
