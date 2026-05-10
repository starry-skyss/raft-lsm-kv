# LSM File Stats

- Case: compaction_off_mixed_50_50_c30_seed3
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 5.7 MiB (6012255 B) | 2 | 3.4 KiB (3453 B) | 453 | 1.9 MiB (2041879 B) | 358.7 KiB (367287 B) |
| node_1 | 5.7 MiB (6012379 B) | 2 | 3.4 KiB (3453 B) | 453 | 1.9 MiB (2041879 B) | 358.7 KiB (367287 B) |
| node_2 | 5.7 MiB (6012255 B) | 2 | 3.4 KiB (3453 B) | 453 | 1.9 MiB (2041879 B) | 358.7 KiB (367287 B) |
