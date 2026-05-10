# LSM File Stats

- Case: compaction_off_write_100_c10_seed1
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 11.3 MiB (11827543 B) | 2 | 2.9 KiB (3020 B) | 952 | 4.1 MiB (4291083 B) | 1.6 MiB (1716583 B) |
| node_1 | 11.3 MiB (11827543 B) | 2 | 2.9 KiB (3020 B) | 952 | 4.1 MiB (4291083 B) | 1.6 MiB (1716583 B) |
| node_2 | 11.3 MiB (11827543 B) | 2 | 2.9 KiB (3020 B) | 952 | 4.1 MiB (4291083 B) | 1.6 MiB (1716583 B) |
