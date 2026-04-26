# raft-lsm-kv 项目整体架构说明（基于当前代码）

## 1. 项目定位

这是一个“分布式复制 + 本地 LSM 存储”的原型系统：

- 上层：3 节点 Raft 集群，负责写请求复制与提交。
- 中层：store 层把 Raft Apply 日志转成状态机操作。
- 下层：每个节点本地一个 LSM 引擎（MemTable + WAL + SSTable + 后台 Flush + 基础 Compaction）。
- 对外：一个 HTTP 网关（Gin）统一接收 Put/Get/Delete。

当前代码已经不是单机 CLI KV 的形态，而是“单机 LSM 内核 + 分布式外壳”的组合架构。

## 2. 分层与目录职责

### 2.1 集群入口与网络

- cmd/kvnode/main.go
	- 启动 3 个 Raft 节点和 3 个 store.KVStore。
	- 通过 labrpc 组网并注册 Raft RPC 服务。
	- 最后启动 HTTP 网关。

### 2.2 网关层

- internal/api/server.go
	- POST /put：轮询节点，找到 leader 执行写。
	- DELETE /delete/:key：同样通过 leader 执行删。
	- GET /get/:key：随机读任一节点（最终一致性读，不保证线性一致）。

### 2.3 状态机适配层

- internal/store/store.go
	- 定义 Op（PUT/DELETE）并提交到 Raft。
	- 用 waitChans 按 log index 等待 Apply 回执。
	- applyLoop 从 applyCh 读取已提交日志，真正调用 lsm.DB 执行落地。

### 2.4 存储引擎层（LSM）

- internal/lsm/kvstore.go
	- 定义核心引擎 DB。
	- 维护 memTable / immTable / sstables / WAL 切分状态。
	- 提供 Put/Get/Delete/Len。

- internal/lsm/kvstore_compaction.go
	- Flush、WAL 恢复、SST 加载、Manifest 重排、定时 Compaction 控制。

- internal/lsm/memtable.go
	- 有序切片 + 二分查找实现内存表。

- internal/lsm/sstable_writer.go
	- 将 KV 对写成 SST（Data blocks + Index block + Footer）。

- internal/lsm/sstable_reader.go
	- 基于内存索引二分定位 block，再在 block 内线性扫描 key。

- internal/lsm/compaction_*.go
	- 迭代器、K 路归并堆、builder 与 compaction 主流程。

### 2.5 WAL 与 Raft 持久化

- internal/wal/wal.go
	- WAL append/replay/delete/clear。

- internal/raft/persister.go
	- Raft 状态持久化到磁盘文件（raft_state.bin）。

## 3. 核心数据结构

### 3.1 store 层

- Op
	- Operation: PUT 或 DELETE
	- Key / Value
	- OpId: 请求唯一标识，用于识别领导权变更造成的“同 index 异命令”场景

- OpResult
	- AppliedOp
	- Err

- KVStore
	- db *lsm.DB
	- rf raftapi.Raft
	- applyCh
	- waitChans map[int]chan OpResult

### 3.2 LSM 层

- DB
	- memTable: 当前可写内存表
	- immTable: 正在刷盘的只读内存表
	- sstables: 当前逻辑顺序下的 SST 元数据列表
	- nextFileID: SST 文件号分配器
	- wal / nextwal / walFileID / walNotifyCh: WAL 分段切分与预分配
	- sstDir / walDir / manifestPath
	- compactionThreshold / compactionInterval

- MemTable
	- pairs []KVPair（按 key 有序）
	- size（估算内存大小，触发 flush）

- KVPair
	- Key []byte
	- Value []byte（nil 表示 tombstone）

- SSTableMeta
	- FileID / MinKey / MaxKey / Size / Index

- IndexEntry
	- MaxKey
	- Handle{Offset, Size}

### 3.3 WAL 层

- WAL
	- mu + file

- 记录类型
	- OpPut = 0
	- OpDelete = 1

## 4. 关键流程（当前实现）

### 4.1 启动流程

main
-> 创建 3 个 Raft 节点
-> 每节点 NewKVStore
-> NewKVStore 内 NewDB
-> NewDB: loadSSTables + RecoverFromWAL + 必要时强制 flush 回放数据 + 清理旧 WAL + 打开新活跃 WAL + 启动 compaction loop + 启动 WAL 预分配
-> StartGateway

### 4.2 写入流程（PUT/DELETE）

HTTP
-> api 轮询节点找 leader
-> store.KVStore.Put/Delete 调用 Raft.Start(op)
-> 等待 waitChans[index] 回执
-> applyLoop 收到已提交日志后调用 db.Put/Delete
-> db.Put/Delete 先写当前 WAL，再改 memTable
-> 达到阈值后切换 memTable->immTable、切分 WAL、异步 flush
-> flush 写 SST，更新 sstables，落 manifest，删除旧 WAL 文件

### 4.3 读取流程（GET）

HTTP
-> api 随机选一个节点
-> store.KVStore.Get
-> db.Get
-> 查 memTable
-> 查 immTable
-> 拷贝 sstables 快照后释放读锁
-> 倒序查 SST（新到旧），先范围过滤，再索引定位 block，再块内扫描

### 4.4 崩溃恢复流程

DB 启动时：

1. 扫描并加载 SST 文件，校验 Footer magic。
2. 读取 manifest 最后一条有效记录恢复逻辑顺序。
3. 扫描 walDir 下所有 .log，按 wal id 从小到大 replay 到 memTable。
4. 回放后若 memTable 非空，立即 flush 成 SST。
5. 清理旧 WAL，创建新活跃 WAL。

## 5. 文件格式与持久化语义

### 5.1 WAL 编码

- Put: [op(1)][keyLen(4)][key][valLen(4)][val]
- Delete: [op(1)][keyLen(4)][key]
- 大端编码长度字段。

### 5.2 SST 编码

- Data Record: keyLen(u32 LE) + key + valLen(u32 LE) + val
- tombstone: valLen = 0xFFFFFFFF
- Index Entry: maxKeyLen + maxKey + offset(u64 LE) + size(u64 LE)
- Footer(24B): indexOffset(u64 LE) + indexSize(u64 LE) + magic(u64 LE)

### 5.3 Manifest

- 逐行 append 文件 id 序列（逗号分隔）。
- 恢复时取最后一条有效行重排 sstables 逻辑顺序。

## 6. 并发模型与一致性边界

### 6.1 并发控制

- DB 使用 RWMutex 保护内存状态。
- WAL 自带互斥锁，append 串行。
- Get 采用“锁内拷贝元数据 + 锁外读盘”降低写阻塞。
- flush/compaction 的元数据替换在写锁内完成，文件删除在锁外执行。

### 6.2 一致性语义

- 写请求：通过 Raft 提交后才应用到状态机，具备复制日志意义上的一致性。
- 读请求：当前是随机节点本地读，未通过 Raft ReadIndex/lease read，因此不是线性一致读，只能视为最终一致读。

## 7. Compaction 当前状态

已经具备：

- 定时触发框架
- 选文件策略（默认最旧 N 个）
- SSTableIterator
- doMerge（最小堆 K 路归并）
- 写新 SST + installCompactedTable + 删除旧文件

当前风险点：

- doMerge 中 fileID 优先级基于迭代器输入顺序的临时编号，不是 SST 真正 FileID，版本优先级语义可能与“新文件覆盖旧文件”预期不完全一致。

## 8. 与早期设计相比的关键变化

1. LSM 核心结构名为 DB，不再是早期文档中的 KVStore。
2. WAL 从“单文件 + clear”演进为“分段文件 + 切换 + 旧文件删除”。
3. 入口从 CLI 命令循环变为 3 节点 Raft + HTTP Gateway。
4. Compaction 从纯 TODO 演进到可执行主流程（仍有优化空间）。

## 9. 测试现状

- internal/lsm/kvstore_test.go 覆盖了基础读写、flush、恢复、并发压力。
- internal/lsm/memtable_test.go 当前仍按旧构造函数签名调用 NewDB，直接导致 go test ./... 构建失败。
- tests/e2e_test.go 仍按旧 CLI 交互假设设计，和当前 HTTP 网关入口形态不一致。

结论：测试体系处于“新旧架构并存”状态，需要统一。

## 10. 已知问题与下一步建议

1. 读一致性：引入 leader read 或 ReadIndex，避免随机 follower 读旧数据。
2. Manifest 健壮性：增加 checksum/版本号，避免部分写入引发重排异常。
3. TableCache：减少 read path 的重复 os.Open 开销。
4. Compaction 去重优先级：用真实 file generation 或 sequence number 判定新旧。
5. 测试重构：清理旧 API 假设，补齐 Raft + HTTP 集成测试。

## 11. 推荐阅读顺序（按当前代码）

1. cmd/kvnode/main.go
2. internal/api/server.go
3. internal/store/store.go
4. internal/lsm/kvstore.go
5. internal/lsm/kvstore_compaction.go
6. internal/lsm/sstable_writer.go + internal/lsm/sstable_reader.go
7. internal/wal/wal.go
8. internal/lsm/compaction_iterator.go + internal/lsm/compaction_merge.go
