# LSM File Stats

- Case: compaction_on_delete_mixed_c10_seed1
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 5.6 MiB (5879945 B) | 2 | 1.8 KiB (1822 B) | 305 | 1.8 MiB (1906024 B) | 294.8 KiB (301908 B) |
| node_1 | 5.6 MiB (5880208 B) | 2 | 1.8 KiB (1822 B) | 305 | 1.8 MiB (1906279 B) | 294.8 KiB (301916 B) |
| node_2 | 5.6 MiB (5879945 B) | 2 | 1.8 KiB (1822 B) | 305 | 1.8 MiB (1906024 B) | 294.8 KiB (301908 B) |
