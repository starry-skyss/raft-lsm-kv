# LSM File Stats

- Case: compaction_off_delete_mixed_c30_seed1
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 4.9 MiB (5089844 B) | 2 | 2.7 KiB (2795 B) | 369 | 1.6 MiB (1690839 B) | 232.5 KiB (238095 B) |
| node_1 | 4.9 MiB (5090804 B) | 2 | 2.7 KiB (2795 B) | 369 | 1.6 MiB (1690839 B) | 232.5 KiB (238095 B) |
| node_2 | 4.9 MiB (5089844 B) | 2 | 2.7 KiB (2795 B) | 369 | 1.6 MiB (1690839 B) | 232.5 KiB (238095 B) |
