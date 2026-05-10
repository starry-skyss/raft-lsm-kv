# LSM File Stats

- Case: compaction_off_mixed_50_50_c10_seed1
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 5.3 MiB (5507320 B) | 2 | 1008 B | 420 | 1.8 MiB (1893120 B) | 305.8 KiB (313167 B) |
| node_1 | 5.3 MiB (5507320 B) | 2 | 1008 B | 420 | 1.8 MiB (1893120 B) | 305.8 KiB (313167 B) |
| node_2 | 5.3 MiB (5507320 B) | 2 | 1008 B | 420 | 1.8 MiB (1893120 B) | 305.8 KiB (313167 B) |
