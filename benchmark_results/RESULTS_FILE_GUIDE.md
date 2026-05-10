# 结果文件说明

本文档说明 `benchmark_results/` 下各类结果目录和文件的内容、命名规则与格式。本文档只解释文件结构和字段用途，不解释实验结果数值。

## 顶层目录

`benchmark_results/` 是实验结果根目录。当前主要包含以下几类结果：

| 目录 | 内容 |
| --- | --- |
| `formal_20260427_160431/` | 早期正式 benchmark 结果，按 workload 和并发度/seed 组织。 |
| `final_tests_20260505_140402/` | LSM 持久化恢复功能测试结果。 |
| `final_tests_20260505_140415/` | HTTP API 功能正确性测试结果。 |
| `diagnostic_20260505_140503/` | 高并发 Leader 稳定性诊断压测结果。 |
| `recovery_quant_20260510_162356/` | 本地恢复量化测试结果，用于统计恢复耗时和校验数量。 |
| `readmode_20260510_163937/` | 强一致读 `raft` 与 Leader 本地直接读 `local` 的对比实验结果。 |
| `compaction_20260510_170538/` | Compaction 开启/关闭消融实验结果。 |
| `request_match_formal_20260510_214916/` | 请求结果匹配机制正式实验的一次运行结果。 |
| `request_match_formal_20260510_215023/` | 请求结果匹配机制正式实验的一次运行结果。 |

目录名中的时间戳格式通常为：

```text
YYYYMMDD_HHMMSS
```

例如 `compaction_20260510_170538` 表示该结果目录创建于 `2026-05-10 17:05:38`。

## 通用文件类型

多数组合实验的每一组 case 都会保存以下文件：

| 文件 | 格式 | 内容 |
| --- | --- | --- |
| `result*.json` | JSON | Benchmark 客户端输出的结构化结果，适合程序读取和制表。 |
| `benchmark*.log` | 文本日志 | Benchmark 客户端终端输出。 |
| `server*.log` | 文本日志 | 本组实验期间服务端输出。 |
| `command*.txt` | 文本 | 本组实验实际执行的命令和关键环境变量。 |
| `metrics_before*.json` | JSON | 压测前采集的 `/debug/metrics`。 |
| `metrics_after*.json` | JSON | 压测后采集的 `/debug/metrics`。 |
| `summary*.md` | Markdown | 当前 case 的摘要表，便于人工查看。 |
| `summary*.csv` | CSV | 当前 case 的摘要行，便于表格汇总。 |
| `lsm_file_stats*.md` | Markdown | 当前 case 结束后的 LSM 文件级统计。 |

文件名中的 `*` 通常包含实验条件，例如：

```text
result_compaction_on_write_100_c10_seed1.json
summary_local_read_100_c1_seed1.md
metrics_after_compaction_off_delete_mixed_c30_seed2.json
```

## 功能测试结果

### `final_tests_20260505_140415/`

该目录保存 HTTP API 功能正确性测试结果。

| 文件 | 内容 |
| --- | --- |
| `functional_results.md` | Markdown 表格，记录每个功能测试项、测试步骤、预期结果、实际结果和是否通过。 |
| `functional_command.txt` | 运行功能测试脚本时的命令记录。 |
| `functional_server.log` | 功能测试期间启动服务产生的日志。 |

## 持久化恢复测试结果

### `final_tests_20260505_140402/`

该目录保存 LSM 恢复功能测试结果。

| 文件 | 内容 |
| --- | --- |
| `recovery_results.md` | WAL、SSTable/Manifest、Tombstone 恢复测试的 Markdown 汇总。 |
| `recovery_go_test.log` | Go test 的完整输出。 |
| `recovery_command.txt` | 运行恢复测试脚本时的命令记录。 |

### `recovery_quant_20260510_162356/`

该目录保存本地恢复量化测试结果。

| 文件 | 内容 |
| --- | --- |
| `recovery_quant_results.csv` | 量化测试主结果，字段固定，适合导入表格。 |
| `recovery_quant_results.md` | 与 CSV 对应的 Markdown 版本，并包含场景级汇总。 |
| `recovery_quant_go_test.log` | Go test 详细日志，包含原始 `RECOVERY_QUANT` 输出行。 |
| `recovery_quant_command.txt` | 运行该测试的命令记录。 |

`recovery_quant_results.csv` 字段如下：

```text
scenario,run_id,write_key_count,overwrite_key_count,delete_key_count,verify_key_count,verify_success_count,recovery_ms,wal_file_count_before,sstable_file_count_before,sstable_file_count_after,manifest_loaded,pass
```

## Leader 稳定性诊断结果

### `diagnostic_20260505_140503/`

该目录保存高并发 Leader 稳定性诊断压测结果。

根目录文件：

| 文件 | 内容 |
| --- | --- |
| `diagnostic_command.txt` | 整个诊断套件的启动命令和配置。 |
| `diagnostic_manifest.tsv` | 诊断 case 索引表，记录 case 名称和结果目录。 |

每个 case 子目录命名为：

```text
<序号>_<workload>_c<concurrency>/
```

例如：

```text
01_write_100_c50/
02_read_100_c10/
```

每个 case 子目录包含：

| 文件 | 内容 |
| --- | --- |
| `result.json` | Benchmark 结构化结果。 |
| `benchmark.log` | Benchmark 客户端输出。 |
| `server.log` | 服务端日志。 |
| `command.txt` | 当前 case 的具体命令。 |
| `metrics_before.json` | 压测前 debug metrics。 |
| `metrics_after.json` | 压测后 debug metrics。 |
| `metrics_summary.md` | 当前 case 的指标差值摘要。 |
| `lsm_file_stats.md` | 当前 case 结束后的 LSM 文件统计。 |

## 读模式对比结果

### `readmode_20260510_163937/`

该目录保存强一致读与本地直接读的对比实验结果。

根目录文件：

| 文件 | 内容 |
| --- | --- |
| `read_mode_comparison_command.txt` | 对比实验套件启动命令和配置。 |
| `read_mode_comparison_manifest.tsv` | case 索引表。 |
| `read_mode_comparison_summary.csv` | 全部 case 的 CSV 汇总。 |
| `read_mode_comparison_summary.md` | 全部 case 的 Markdown 汇总。 |

case 子目录命名为：

```text
<read_mode>_<workload>_c<concurrency>_seed<seed>/
```

例如：

```text
raft_read_100_c1_seed1/
local_read_100_c1_seed1/
```

`read_mode_comparison_summary.csv` 字段如下：

```text
read_mode,workload,concurrency,seed,success_rate,success_qps,p50_ms,p90_ms,p99_ms,p999_ms,http_500_total,http_503_total,request_timeout_total,leader_change_delta,election_started_delta,term_change_delta
```

## Compaction 消融实验结果

### `compaction_20260510_170538/`

该目录保存 Compaction 开启/关闭消融实验结果。

根目录文件：

| 文件 | 内容 |
| --- | --- |
| `compaction_ablation_command.txt` | 消融实验套件启动命令和全局参数。 |
| `compaction_ablation_manifest.tsv` | case 索引表。 |
| `compaction_ablation_summary.csv` | 全部 case 的 CSV 汇总。 |
| `compaction_ablation_summary.md` | 全部 case 的 Markdown 汇总。 |

case 子目录命名为：

```text
compaction_<on|off>_<workload>_c<concurrency>_seed<seed>/
```

例如：

```text
compaction_on_write_100_c10_seed1/
compaction_off_delete_mixed_c30_seed2/
```

每个 case 子目录中的文件名也会带上相同的 case 名称，例如：

```text
result_compaction_on_write_100_c10_seed1.json
metrics_before_compaction_on_write_100_c10_seed1.json
metrics_after_compaction_on_write_100_c10_seed1.json
lsm_file_stats_compaction_on_write_100_c10_seed1.md
summary_compaction_on_write_100_c10_seed1.csv
summary_compaction_on_write_100_c10_seed1.md
```

`compaction_ablation_summary.csv` 字段如下：

```text
enable_compaction,workload,concurrency,seed,total_requests,keyspace,value_size,success_rate,success_qps,get_success_qps,get_p99_ms,get_p999_ms,sstable_files_total,data_size_bytes,wal_files_total,wal_size_bytes,get_sstable_files_touched_avg,compaction_total_delta,compaction_input_files_delta,compaction_output_files_delta,compaction_total_ms_delta,compaction_max_ms
```

## 请求结果匹配实验结果

### `request_match_formal_20260510_214916/`
### `request_match_formal_20260510_215023/`

这两个目录保存请求结果匹配机制验证实验的正式运行结果。目录结构相同：

| 文件 | 内容 |
| --- | --- |
| `request_match_results.csv` | 主结果 CSV，一行记录本次实验的请求分类统计。 |
| `request_match_results.md` | 与 CSV 对应的 Markdown 结果表。 |
| `request_match_raw.json` | 请求结果匹配实验的完整结构化输出。 |
| `metrics_before.json` | 实验前 debug metrics。 |
| `metrics_after.json` | 实验后 debug metrics。 |
| `force_election.log` | 实验期间触发选举请求的日志。 |
| `request_match.log` | 请求匹配测试客户端输出。 |
| `server.log` | 服务端日志。 |
| `command.txt` | 本次实验实际执行的命令。 |

`request_match_results.csv` 字段如下：

```text
run_id,total_requests,success_requests,not_leader_total,request_timeout_total,leadership_changed_before_commit_total,http_500_total,http_503_total,error_result_total,other_error_total,leader_change_delta,election_started_delta,term_change_delta,force_election_attempts,duration_seconds
```

其中 `error_result_total` 表示成功返回但值与请求 key 的预期值不匹配的数量。

## 早期正式 Benchmark 结果

### `formal_20260427_160431/`

该目录是早期正式 benchmark 结果。目录内已有独立说明文件：

```text
formal_20260427_160431/RESULTS_README.md
```

该目录按 workload 分组：

```text
write_100/
read_100/
mixed_50_50/
read95_write5/
```

每个 workload 下按并发度和 seed 组织：

```text
c<concurrency>_seed<seed>/
```

每个 run 目录通常包含：

```text
result.json
benchmark.log
server.log
command.txt
```

## JSON 文件格式说明

`result*.json` 主要来自 benchmark 客户端，通常包含：

| 字段类型 | 示例字段 |
| --- | --- |
| 实验配置 | `run_id`, `workload_name`, `concurrency`, `write_ratio`, `delete_ratio`, `read_mode`, `enable_compaction`, `keyspace`, `valuesize`, `seed` |
| 总体结果 | `total_requests`, `success_requests`, `failed_requests`, `success_rate_percent`, `success_qps` |
| 延迟指标 | `avg_latency_ms`, `p50_latency_ms`, `p90_latency_ms`, `p99_latency_ms`, `p999_latency_ms`, `max_latency_ms` |
| 操作分类 | `put_total`, `get_total`, `delete_total`, `put_success`, `get_success`, `delete_success` |
| 错误统计 | `error_counts.timeout`, `error_counts.http_500`, `error_counts.http_503`, `error_counts.request_timeout`, `error_counts.not_leader` |

`metrics_before*.json` 和 `metrics_after*.json` 来自 `GET /debug/metrics`，通常包含：

| 指标类别 | 内容 |
| --- | --- |
| Raft | leader、term、leader change、election、persist、commit index、log length 等。 |
| Store/API | 请求总数、Put/Get/Delete 数量、HTTP 500/503、timeout、not leader、等待提交请求数量等。 |
| Runtime | goroutine、heap、GC 次数、GC pause 等。 |
| LSM | SSTable 数量、Get 访问 SSTable 文件数、Compaction 次数/耗时等。 |
| Nodes | 每个节点的 Raft 与 LSM 指标快照。 |

## Markdown 与 CSV 文件使用建议

| 文件类型 | 适合用途 |
| --- | --- |
| `.csv` | 导入 Excel、WPS、Python、R 或论文制表工具。 |
| `.md` | 直接阅读、截图、复制到论文草稿。 |
| `.json` | 程序化复算指标或保留完整原始结构。 |
| `.log` | 排查异常、确认服务端或客户端运行过程。 |
| `.txt` | 复现实验命令和参数。 |

## Manifest 文件

`*_manifest.tsv` 是实验索引文件，使用 Tab 分隔。它用于从套件级结果目录定位每个 case 的子目录。常见字段包括：

```text
case_id
workload
concurrency
seed
result_dir
```

不同实验套件的 manifest 字段可能略有差异，但最后通常会包含当前 case 的结果目录路径。

