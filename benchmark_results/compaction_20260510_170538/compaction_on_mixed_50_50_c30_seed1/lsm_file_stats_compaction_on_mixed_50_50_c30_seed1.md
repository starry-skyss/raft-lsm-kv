# LSM File Stats

- Case: compaction_on_mixed_50_50_c30_seed1
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 5.6 MiB (5833722 B) | 2 | 144 B | 353 | 1.9 MiB (1981427 B) | 334.4 KiB (342408 B) |
| node_1 | 5.6 MiB (5833687 B) | 2 | 144 B | 353 | 1.9 MiB (1981427 B) | 334.3 KiB (342373 B) |
| node_2 | 5.6 MiB (5833743 B) | 2 | 144 B | 353 | 1.9 MiB (1981427 B) | 334.4 KiB (342429 B) |
