# LSM File Stats

- Case: compaction_off_mixed_50_50_c30_seed2
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 4.7 MiB (4882299 B) | 2 | 864 B | 373 | 1.6 MiB (1681250 B) | 237.9 KiB (243607 B) |
| node_1 | 4.7 MiB (4882105 B) | 2 | 864 B | 373 | 1.6 MiB (1681250 B) | 237.9 KiB (243607 B) |
| node_2 | 4.7 MiB (4882105 B) | 2 | 864 B | 373 | 1.6 MiB (1681250 B) | 237.9 KiB (243607 B) |
