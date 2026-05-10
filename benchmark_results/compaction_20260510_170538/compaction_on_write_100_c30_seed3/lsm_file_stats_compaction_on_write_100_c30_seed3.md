# LSM File Stats

- Case: compaction_on_write_100_c30_seed3
- Data directory: data
- Compaction log entries in server.log: 1
- Note: current compaction logs are not tagged by node, so compaction log count is reported at cluster log level.

| Node | Data Size | WAL Files | WAL Size | SSTable Files | SSTable Size | Manifest Size |
| --- | --- | --- | --- | --- | --- | --- |
| node_0 | 6.8 MiB (7109974 B) | 2 | 2.9 KiB (3019 B) | 526 | 2.6 MiB (2715877 B) | 649.0 KiB (664564 B) |
| node_1 | 6.8 MiB (7110010 B) | 2 | 2.9 KiB (3019 B) | 526 | 2.6 MiB (2715877 B) | 649.0 KiB (664600 B) |
| node_2 | 6.8 MiB (7109974 B) | 2 | 2.9 KiB (3019 B) | 526 | 2.6 MiB (2715877 B) | 649.0 KiB (664564 B) |
