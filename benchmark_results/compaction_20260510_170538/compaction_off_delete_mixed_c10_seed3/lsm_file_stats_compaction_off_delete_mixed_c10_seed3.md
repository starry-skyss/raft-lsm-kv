# LSM File Stats

- Case: compaction_off_delete_mixed_c10_seed3
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 4.8 MiB (5051065 B) | 2 | 4.0 KiB (4055 B) | 364 | 1.6 MiB (1668255 B) | 225.9 KiB (231295 B) |
| node_1 | 4.8 MiB (5051065 B) | 2 | 4.0 KiB (4055 B) | 364 | 1.6 MiB (1668255 B) | 225.9 KiB (231295 B) |
| node_2 | 4.8 MiB (5051065 B) | 2 | 4.0 KiB (4055 B) | 364 | 1.6 MiB (1668255 B) | 225.9 KiB (231295 B) |
