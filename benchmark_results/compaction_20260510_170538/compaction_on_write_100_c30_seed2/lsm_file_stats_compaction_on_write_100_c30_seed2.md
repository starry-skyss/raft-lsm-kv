# LSM File Stats

- Case: compaction_on_write_100_c30_seed2
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 11.1 MiB (11681840 B) | 2 | 2.2 KiB (2302 B) | 801 | 4.0 MiB (4232399 B) | 1.6 MiB (1633426 B) |
| node_1 | 11.1 MiB (11682308 B) | 2 | 2.2 KiB (2302 B) | 801 | 4.0 MiB (4232571 B) | 1.6 MiB (1633722 B) |
| node_2 | 11.1 MiB (11682296 B) | 2 | 2.2 KiB (2302 B) | 801 | 4.0 MiB (4232571 B) | 1.6 MiB (1633710 B) |
