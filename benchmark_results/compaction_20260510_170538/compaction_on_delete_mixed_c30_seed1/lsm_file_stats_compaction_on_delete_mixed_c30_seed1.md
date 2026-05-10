# LSM File Stats

- Case: compaction_on_delete_mixed_c30_seed1
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 5.0 MiB (5237036 B) | 2 | 2.2 KiB (2206 B) | 278 | 1.6 MiB (1710241 B) | 236.9 KiB (242610 B) |
| node_1 | 5.0 MiB (5236484 B) | 2 | 2.2 KiB (2206 B) | 278 | 1.6 MiB (1709701 B) | 236.9 KiB (242598 B) |
| node_2 | 5.0 MiB (5237171 B) | 2 | 2.2 KiB (2206 B) | 278 | 1.6 MiB (1709573 B) | 236.9 KiB (242574 B) |
