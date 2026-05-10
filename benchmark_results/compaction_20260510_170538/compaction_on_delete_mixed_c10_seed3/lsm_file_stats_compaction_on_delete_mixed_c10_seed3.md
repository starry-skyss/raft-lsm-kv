# LSM File Stats

- Case: compaction_on_delete_mixed_c10_seed3
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 4.6 MiB (4805290 B) | 2 | 3.6 KiB (3707 B) | 252 | 1.5 MiB (1568065 B) | 196.8 KiB (201532 B) |
| node_1 | 4.6 MiB (4805355 B) | 2 | 3.6 KiB (3707 B) | 252 | 1.5 MiB (1568065 B) | 196.8 KiB (201532 B) |
| node_2 | 4.6 MiB (4805290 B) | 2 | 3.6 KiB (3707 B) | 252 | 1.5 MiB (1568065 B) | 196.8 KiB (201532 B) |
