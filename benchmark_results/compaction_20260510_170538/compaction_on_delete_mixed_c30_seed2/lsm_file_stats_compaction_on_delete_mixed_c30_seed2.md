# LSM File Stats

- Case: compaction_on_delete_mixed_c30_seed2
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 5.5 MiB (5814388 B) | 2 | 3.1 KiB (3195 B) | 301 | 1.8 MiB (1872091 B) | 285.9 KiB (292719 B) |
| node_1 | 5.5 MiB (5814472 B) | 2 | 3.1 KiB (3195 B) | 301 | 1.8 MiB (1872091 B) | 285.9 KiB (292803 B) |
| node_2 | 5.5 MiB (5814456 B) | 2 | 3.1 KiB (3195 B) | 301 | 1.8 MiB (1872091 B) | 285.9 KiB (292787 B) |
