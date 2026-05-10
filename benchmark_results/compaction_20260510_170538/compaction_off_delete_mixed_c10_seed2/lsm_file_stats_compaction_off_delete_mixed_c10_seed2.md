# LSM File Stats

- Case: compaction_off_delete_mixed_c10_seed2
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 5.6 MiB (5904955 B) | 2 | 144 B | 422 | 1.8 MiB (1934495 B) | 308.9 KiB (316323 B) |
| node_1 | 5.6 MiB (5904955 B) | 2 | 144 B | 422 | 1.8 MiB (1934495 B) | 308.9 KiB (316323 B) |
| node_2 | 5.6 MiB (5904955 B) | 2 | 144 B | 422 | 1.8 MiB (1934495 B) | 308.9 KiB (316323 B) |
