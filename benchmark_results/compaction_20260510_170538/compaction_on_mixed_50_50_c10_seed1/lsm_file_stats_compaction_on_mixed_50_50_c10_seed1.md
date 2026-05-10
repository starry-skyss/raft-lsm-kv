# LSM File Stats

- Case: compaction_on_mixed_50_50_c10_seed1
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 3.5 MiB (3692320 B) | 2 | 2.0 KiB (2014 B) | 242 | 1.2 MiB (1296075 B) | 134.2 KiB (137394 B) |
| node_1 | 3.5 MiB (3692308 B) | 2 | 2.0 KiB (2014 B) | 242 | 1.2 MiB (1296075 B) | 134.2 KiB (137382 B) |
| node_2 | 3.5 MiB (3692308 B) | 2 | 2.0 KiB (2014 B) | 242 | 1.2 MiB (1296075 B) | 134.2 KiB (137382 B) |
