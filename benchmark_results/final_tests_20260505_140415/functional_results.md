# Functional Test Results

- Time: 2026-05-05T14:04:19+08:00
- API URL: http://localhost:8080
- Data cleanup: /home/star/projects/kvstore/raft-lsm-kv/data
- Server log: benchmark_results/final_tests_20260505_140415/functional_server.log

| Test Item | Steps | Expected Result | Actual Result | Pass |
| --- | --- | --- | --- | --- |
| Put then Get | POST /put value_1; GET same key | GET returns value_1 | PUT 200 {"leader":1,"status":"success"}; GET 200 {"leader":1,"value":"value_1"} | PASS |
| Overwrite Put | POST old_value; POST new_value; GET same key | GET returns latest value new_value | PUT1 200 {"leader":1,"status":"success"}; PUT2 200 {"leader":1,"status":"success"}; GET 200 {"leader":1,"value":"new_value"} | PASS |
| Delete then Get | POST key; DELETE key; GET same key | GET returns key not found | PUT 200 {"leader":1,"status":"success"}; DELETE 200 {"leader":1,"status":"deleted"}; GET 404 {"error":"key not found","leader":1} | PASS |
| Missing Key Get | GET nonexistent key | Returns not found without HTTP 500 or 503 | GET 404 {"error":"key not found","leader":1} | PASS |
| Read After Write | POST committed_value; immediately GET same key | Raft-backed GET sees committed write | PUT 200 {"leader":1,"status":"success"}; GET 200 {"leader":1,"value":"committed_value"} | PASS |
| Read After Overwrite | POST version_1; POST version_2; immediately GET same key | GET sees version_2, not version_1 | PUT1 200 {"leader":1,"status":"success"}; PUT2 200 {"leader":1,"status":"success"}; GET 200 {"leader":1,"value":"version_2"} | PASS |
| Read After Delete | POST key; DELETE key; immediately GET same key | Delete is visible and GET returns key not found | PUT 200 {"leader":1,"status":"success"}; DELETE 200 {"leader":1,"status":"deleted"}; GET 404 {"error":"key not found","leader":1} | PASS |

## Summary

- Passed: 7
- Failed: 0
- Result: PASS
