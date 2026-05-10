# LSM File Stats

- Case: compaction_off_delete_mixed_c30_seed2
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 4.5 MiB (4763012 B) | 2 | 1.7 KiB (1786 B) | 344 | 1.5 MiB (1577528 B) | 200.3 KiB (205095 B) |
| node_1 | 4.5 MiB (4763012 B) | 2 | 1.7 KiB (1786 B) | 344 | 1.5 MiB (1577528 B) | 200.3 KiB (205095 B) |
| node_2 | 4.5 MiB (4763012 B) | 2 | 1.7 KiB (1786 B) | 344 | 1.5 MiB (1577528 B) | 200.3 KiB (205095 B) |
