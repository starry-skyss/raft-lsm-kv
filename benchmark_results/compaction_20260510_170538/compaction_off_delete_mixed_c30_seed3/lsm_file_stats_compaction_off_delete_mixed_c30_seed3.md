# LSM File Stats

- Case: compaction_off_delete_mixed_c30_seed3
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 4.6 MiB (4866265 B) | 2 | 4.2 KiB (4305 B) | 351 | 1.5 MiB (1608874 B) | 209.1 KiB (214083 B) |
| node_1 | 4.6 MiB (4866265 B) | 2 | 4.2 KiB (4305 B) | 351 | 1.5 MiB (1608874 B) | 209.1 KiB (214083 B) |
| node_2 | 4.6 MiB (4866459 B) | 2 | 4.2 KiB (4305 B) | 351 | 1.5 MiB (1608874 B) | 209.1 KiB (214083 B) |
