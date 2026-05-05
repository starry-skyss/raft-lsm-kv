# LSM File Stats

- Case: 01_write_100_c50
- Data directory: data
- Compaction log entries in server.log: 0
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 3.4 MiB (3580568 B) | 2 | 4.1 KiB (4201 B) | 291 | 1.4 MiB (1454465 B) | 168.8 KiB (172877 B) |
| node_1 | 3.4 MiB (3580604 B) | 2 | 4.1 KiB (4201 B) | 291 | 1.4 MiB (1454465 B) | 168.9 KiB (172913 B) |
| node_2 | 3.4 MiB (3580580 B) | 2 | 4.1 KiB (4201 B) | 291 | 1.4 MiB (1454465 B) | 168.8 KiB (172889 B) |
