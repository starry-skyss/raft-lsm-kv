# LSM File Stats

- Case: compaction_on_mixed_50_50_c30_seed3
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 4.2 MiB (4447929 B) | 2 | 864 B | 284 | 1.5 MiB (1535825 B) | 194.4 KiB (199036 B) |
| node_1 | 4.2 MiB (4447941 B) | 2 | 864 B | 284 | 1.5 MiB (1535825 B) | 194.4 KiB (199048 B) |
| node_2 | 4.2 MiB (4447905 B) | 2 | 864 B | 284 | 1.5 MiB (1535825 B) | 194.3 KiB (199012 B) |
