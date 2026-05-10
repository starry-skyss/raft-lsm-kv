# Read Mode Comparison Summary

- Suite: readmode_20260510_163937
- Read modes: raft local
- Workloads: read_100
- Concurrencies: 1
- Seeds: 1

| Read Mode | Workload | Concurrency | Seed | Success Rate | Success QPS | P50 ms | P90 ms | P99 ms | P99.9 ms | HTTP 500 | HTTP 503 | Request Timeout | Leader Changes | Elections | Term Changes |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| raft | read_100 | 1 | 1 | 100.0000% | 95.9887 | 10.8994 | 11.0361 | 11.3771 | 11.3771 | 0 | 0 | 0 | 0 | 0 | 0 |
| local | read_100 | 1 | 1 | 100.0000% | 4737.3944 | 0.1692 | 0.3414 | 0.4794 | 0.4794 | 0 | 0 | 0 | 0 | 0 | 0 |
