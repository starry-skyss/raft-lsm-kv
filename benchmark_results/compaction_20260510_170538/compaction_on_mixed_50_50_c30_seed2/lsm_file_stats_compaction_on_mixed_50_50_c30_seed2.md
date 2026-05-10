# LSM File Stats

- Case: compaction_on_mixed_50_50_c30_seed2
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 4.5 MiB (4747630 B) | 2 | 2.0 KiB (2016 B) | 296 | 1.6 MiB (1627555 B) | 219.9 KiB (225133 B) |
| node_1 | 4.5 MiB (4747618 B) | 2 | 2.0 KiB (2016 B) | 296 | 1.6 MiB (1627555 B) | 219.8 KiB (225121 B) |
| node_2 | 4.5 MiB (4747938 B) | 2 | 2.0 KiB (2016 B) | 296 | 1.6 MiB (1627555 B) | 219.8 KiB (225061 B) |
