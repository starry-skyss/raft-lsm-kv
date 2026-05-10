# LSM File Stats

- Case: compaction_on_delete_mixed_c30_seed3
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 4.6 MiB (4860229 B) | 2 | 1.7 KiB (1747 B) | 265 | 1.5 MiB (1589057 B) | 202.3 KiB (207133 B) |
| node_1 | 4.6 MiB (4860217 B) | 2 | 1.7 KiB (1747 B) | 265 | 1.5 MiB (1589057 B) | 202.3 KiB (207121 B) |
| node_2 | 4.6 MiB (4860926 B) | 2 | 1.7 KiB (1759 B) | 265 | 1.5 MiB (1589057 B) | 202.3 KiB (207109 B) |
