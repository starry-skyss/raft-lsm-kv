# LSM File Stats

- Case: compaction_off_mixed_50_50_c10_seed2
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 3.5 MiB (3642148 B) | 2 | 1.1 KiB (1150 B) | 283 | 1.2 MiB (1275582 B) | 131.9 KiB (135067 B) |
| node_1 | 3.5 MiB (3642148 B) | 2 | 1.1 KiB (1150 B) | 283 | 1.2 MiB (1275582 B) | 131.9 KiB (135067 B) |
| node_2 | 3.5 MiB (3642918 B) | 2 | 1.7 KiB (1726 B) | 283 | 1.2 MiB (1275582 B) | 131.9 KiB (135067 B) |
