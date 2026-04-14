# raft-lsm-kv 项目整体架构说明

## 1. 项目目标与当前实现边界

这是一个基于 LSM-Tree 思路的 Key-Value 存储原型，当前重点是：

- 前台读写路径（Put/Get/Delete）
- WAL 持久化与崩溃恢复
- MemTable 到 SSTable 的异步 Flush
- SSTable 索引读取
- Compaction 的整体架构骨架（触发、选文件、迭代器/merge 接口、原子替换接口）

当前不完整部分：


## 2. 目录与职责

- `cmd/kvnode/main.go`
- 职责：CLI 入口，初始化目录/WAL/KVStore，提供交互命令循环。

- `internal/wal/wal.go`
- 职责：WAL 二进制日志写入、重放、清空。

- `internal/lsm/memtable.go`
- 职责：内存有序表（有序切片 + 二分查找）。

- `internal/lsm/sstable_format.go`
- 职责：SSTable 相关格式常量和元数据结构体定义。

- `internal/lsm/sstable_writer.go`
- 职责：将 MemTable 刷到 .sst 文件（Data Block + Index Block + Footer）。

- `internal/lsm/sstable_reader.go`
- 职责：基于内存索引定位并读取目标 Key。

- `internal/lsm/kvstore.go`
- 职责：KVStore 主结构、前台 API（Put/Get/Delete/Len）。

- `internal/lsm/kvstore_compaction.go`
- 职责：后台相关流程（Flush/Recover/Load/Manifest/Compaction 控制骨架）。

- `internal/lsm/compaction_iterator.go`
- 职责：Compaction 迭代器接口与单表迭代器骨架。

- `internal/lsm/compaction_merge.go`
- 职责：K 路归并最小堆框架。

- `internal/lsm/compaction_builder.go`
- 职责：Compaction 输出 builder 骨架。

- `internal/lsm/*_test.go` 与 `tests/e2e_test.go`
- 职责：单元/并发/恢复/E2E 测试。

## 3. 核心数据结构说明

## 3.1 WAL 层

### `type WAL struct`

- `mu sync.Mutex`
- 作用：保护 WAL 文件写入串行化，避免多协程写错乱。

- `file *os.File`
- 作用：底层日志文件句柄。

### `const OpPut, OpDelete`

- `OpPut byte = 0`
- `OpDelete byte = 1`
- 作用：WAL 记录类型标识。

## 3.2 MemTable 层

### `type KVPair struct`

- `Key []byte`
- `Value []byte`
- 语义：`Value == nil` 表示 tombstone（删除标记）。

### `type MemTable struct`

- `pairs []KVPair`
- 作用：按 Key 有序存储；通过二分查找定位。

- `size int`
- 作用：粗略跟踪内存占用，用于触发 flush 阈值。

## 3.3 SSTable 格式层

### `type BlockHandle struct`

- `Offset uint64`
- `Size uint64`
- 作用：描述文件内一个连续块位置（Data/Index 都可）。

### `type Footer struct`

- `IndexHandle BlockHandle`
- `MagicNumber uint64`
- 作用：文件尾元数据，定位 Index Block 并校验文件格式。

### `type IndexEntry struct`

- `MaxKey []byte`
- `Handle BlockHandle`
- 作用：每个 Data Block 的上界键与块句柄，支持二分定位。

### `type SSTableMeta struct`

- `FileID uint64`
- `MinKey []byte`
- `MaxKey []byte`
- `Size uint64`
- `Index []IndexEntry`
- 作用：SSTable 常驻内存元数据，Get 时先用它做范围过滤和索引定位。

## 3.4 KVStore 层

### `type KVStore struct`

- 并发控制：`mu sync.RWMutex`
- 内存态：`memTable`, `immTable`
- 磁盘态：`sstables []*SSTableMeta`
- 文件编号：`nextFileID uint64`
- 组件依赖：`wal *wal.WAL`
- 路径配置：`rootDir`, `dataDir`, `sstDir`, `walDir`, `manifestPath`
- 压实配置：`compactionThreshold int`, `compactionInterval time.Duration`

其中 `sstables` 是“逻辑查询顺序”。重启后会通过 manifest 最后一条记录恢复该顺序。

## 3.5 Compaction 骨架层

### `type Iterator interface`

- `Next()`：推进游标
- `Valid() bool`：是否可读
- `Key() []byte`：当前 key
- `Value() []byte`：当前 value
- `Close()`：释放资源

### `type SSTableIterator struct`（未完整实现）

- `meta *SSTableMeta`
- `dir string`
- `key []byte`
- `value []byte`
- `valid bool`

### `type SSTableBuilder struct`

- `pairs []KVPair`
- 作用：归并输出缓冲。

### `type mergeItem` / `type mergeHeap`

- `mergeItem{iter Iterator, fileID uint64}`
- `mergeHeap` 实现最小堆，按 key 排序；key 相同按“更新优先”（当前用 fileID 大者优先）。

## 4. 关键函数与参数说明（按模块）

## 4.1 入口层

### `func main()`

流程：
1. 创建 `./data` 与 `./data/wal`
2. `wal.OpenWAL(data/wal/wal.log)`
3. `lsm.NewKVStore(w, ".")`
4. 启动 CLI 循环：`put/get/del/exit`

## 4.2 WAL 模块

### `func OpenWAL(filePath string) (*WAL, error)`

- `filePath`：WAL 文件路径。
- 返回：WAL 实例与错误。
- 关键点：以 `O_CREATE|O_APPEND|O_RDWR` 打开。

### `func (w *WAL) AppendPut(key, val string) error`

- 参数：业务键值。
- 行为：编码并追加 `[OpPut, KeyLen, Key, ValLen, Val]`，随后 `Sync`。

### `func (w *WAL) AppendDelete(key string) error`

- 参数：删除键。
- 行为：编码并追加 `[OpDelete, KeyLen, Key]`，随后 `Sync`。

### `func (w *WAL) Replay(onPut func(key, val string), onDelete func(key string)) error`

- 参数：回放回调。
- 行为：从文件头顺序解码日志并回调；结束后 seek 到末尾供后续 append。

### `func (w *WAL) Clear() error`

- 行为：`Truncate(0)` + `Seek(0)` + `Sync`。
- 用途：Flush 成功后清 WAL，防止无限增长与重复回放。

### `func (w *WAL) Close() error`

- 行为：`Sync` 后关闭文件句柄。

## 4.3 MemTable 模块

### `func NewMemTable() *MemTable`

- 返回：空 MemTable，默认容量 256。

### `func (m *MemTable) Put(key, val []byte)`

- 参数：二进制 key/value。
- 行为：二分定位后更新或插入；维护 `size`。
- 语义：`val == nil` 代表 tombstone。

### `func (m *MemTable) Get(key []byte) ([]byte, bool)`

- 返回：命中值与是否命中。
- 注意：命中 tombstone 时会返回 `(nil, true)`。

## 4.4 KVStore 前台 API

### `func NewKVStore(w *wal.WAL, rootDir string) *KVStore`

- `w`：注入 WAL 组件。
- `rootDir`：项目根路径（内部构造 `rootDir/data/...`）。
- 行为：
1. 创建 `data/sst`、`data/wal`
2. 初始化默认参数（threshold=4, interval=10s）
3. `loadSSTables()`
4. `RecoverFromWAL()`
5. `StartCompactionLoop()`

### `func (kv *KVStore) Put(key, val string) error`

- 参数：字符串键值。
- 写路径：
1. 先 `wal.AppendPut`
2. 再加锁写 `memTable.Put`
3. `checkAndFlush` 决定是否异步 flush

### `func (kv *KVStore) Get(key string) (string, bool)`

- 读路径优先级：
1. `memTable`
2. `immTable`
3. `sstables`（倒序，越新越先查）

- 并发优化：
- 先在读锁内复制 `sstables` 快照
- 释放读锁后进行慢速磁盘 I/O，降低对写入阻塞

### `func (kv *KVStore) Delete(key string) error`

- 写路径：
1. 先 `wal.AppendDelete`
2. 加锁写入 tombstone：`memTable.Put(key,nil)`
3. `checkAndFlush`

### `func (kv *KVStore) Len() int`

- 返回当前 memTable 中的 pair 数，不含 sstable 数量。

## 4.5 Flush / Recovery / Load / Manifest / Compaction 控制

### `func (kv *KVStore) checkAndFlush()`

- 条件：`memTable.size >= 4096`
- 行为：
- 若 `immTable != nil`，打印 backpressure 警告并返回
- 否则切换 `immTable = memTable`，新建 memTable，后台 goroutine 调 `flush`

### `func (kv *KVStore) flush(imm *MemTable)`

- 参数：只读 immTable。
- 行为：
1. `reserveNextFileID`
2. 调 `writeSSTable`
3. 组装 `SSTableMeta`
4. `installFlushedTable`

### `func (kv *KVStore) reserveNextFileID() uint64`

- 作用：在锁内分配全局递增 file id。

### `func (kv *KVStore) installFlushedTable(meta *SSTableMeta)`

- 作用：
1. 锁内 append 到 `sstables`，清空 `immTable`
2. 锁外 `persistManifest`
3. 调 `wal.Clear`

### `func (kv *KVStore) RecoverFromWAL() error`

- 行为：调用 `wal.Replay`，将 put/delete 回放到 `memTable`。

### `func (kv *KVStore) loadSSTables() error`

- 行为：
1. 扫描 `sstDir` 下 `.sst`
2. 读取 footer 校验 magic
3. 加载 index block 到内存
4. 读取最小 key 组装 `SSTableMeta`
5. `bumpNextFileID`
6. 按 `FileID` 排序
7. `applyManifestOrder` 恢复逻辑顺序

### `func (kv *KVStore) bumpNextFileID(fileID uint64)`

- 作用：保证 `nextFileID = max(nextFileID, fileID+1)`。

### `func (kv *KVStore) persistManifest(snapshot []*SSTableMeta) error`

- 参数：当前逻辑顺序快照。
- 行为：提取 file id 列表并 append 一行到 `manifest.log`。

### `func (kv *KVStore) applyManifestOrder(loadedMetas []*SSTableMeta) ([]*SSTableMeta, error)`

- 参数：扫描加载出的 metas。
- 行为：读取 manifest 最后一条有效行重排；没出现的 id 追加到尾部。

### `func (kv *KVStore) StartCompactionLoop()`

- 行为：起后台 ticker（默认 10s）循环触发压实检查。

### `func (kv *KVStore) needCompaction() bool`

- 策略：`len(sstables) >= compactionThreshold`。

### `func (kv *KVStore) pickCompactionFiles() []*SSTableMeta`

- 简化 tiered 策略：选最旧的 N 个（当前 N=4）。

### `func (kv *KVStore) installCompactedTable(newSST *SSTableMeta, oldSSTs []*SSTableMeta)`

- 参数：新表 meta + 被替换旧表列表。
- 行为：
1. 写锁内从 `sstables` 删除 old 并追加 new
2. 锁外 `persistManifest`

### `func (kv *KVStore) removeObsoleteFiles(oldSSTs []*SSTableMeta)`

- 行为：遍历 old tables，`os.Remove` 物理删除旧文件。

### `func (kv *KVStore) doCompaction() error`

- 当前：仅完成 pick 文件，merge/写新表/替换/删除仍是 TODO。

## 4.6 SSTable 写入模块

### `func writeSSTable(filename string, imm *MemTable) (uint64, []IndexEntry, error)`

- `filename`：输出文件名（如 `000001.sst`）
- `imm`：输入有序数据
- 返回：写入字节数、内存索引、错误

输出文件布局：

1. Data Blocks
- 记录编码：`keyLen(u32) + key + valLen(u32) + val`
- tombstone：`valLen = 0xFFFFFFFF`

2. Index Block
- 条目编码：`maxKeyLen(u32) + maxKey + offset(u64) + size(u64)`

3. Footer (24 bytes)
- `indexOffset(u64) + indexSize(u64) + magic(u64)`

## 4.7 SSTable 读取模块

### `func readValFromSSTable(meta *SSTableMeta, dir string, targetKey []byte) ([]byte, bool)`

- 参数：
- `meta`：内存索引元信息
- `dir`：sst 目录
- `targetKey`：查询 key

- 行为：
1. 在 `meta.Index` 中二分找候选 Data Block
2. 打开对应文件并 `ReadAt` 读整个 block
3. 在 block 内顺序扫描记录
4. 命中返回 value（tombstone 返回 `nil,true`）

## 4.8 Compaction 迭代器 / merge / builder（架构骨架）

### `func NewSSTableIterator(meta *SSTableMeta, dir string) (*SSTableIterator, error)`

- 当前仅做参数校验，未实际打开文件并定位第一条。

### `func (it *SSTableIterator) Next()/Valid()/Key()/Value()/Close()`

- 接口已就位，具体扫描逻辑待实现。

### `func doMerge(iters []Iterator) *SSTableBuilder`

- 行为：
1. 初始化最小堆
2. 逐个弹出最小 key 记录写入 builder
3. 推进来源迭代器并回堆
4. 去重处理：同 key 只保留优先记录
5. 关闭全部迭代器

### `func (b *SSTableBuilder) Append(key, value []byte)`

- 行为：追加输出；如果连续同 key，覆盖最后一条。

## 5. 端到端调用链

## 5.1 启动恢复链

`main -> OpenWAL -> NewKVStore -> loadSSTables -> applyManifestOrder -> RecoverFromWAL -> StartCompactionLoop`

## 5.2 Put 链

`Put -> WAL.AppendPut -> MemTable.Put -> checkAndFlush -> (异步) flush -> writeSSTable -> installFlushedTable -> persistManifest -> WAL.Clear`

## 5.3 Delete 链

`Delete -> WAL.AppendDelete -> MemTable.Put(tombstone) -> checkAndFlush ...`

## 5.4 Get 链

`Get -> MemTable -> immTable -> sstables(倒序) -> readValFromSSTable`

## 5.5 计划中的 Compaction 链（目标态）

`StartCompactionLoop -> needCompaction -> pickCompactionFiles -> NewSSTableIterator* -> doMerge -> write new sst -> installCompactedTable -> persistManifest -> removeObsoleteFiles`

## 6. 并发与一致性设计要点

- WAL 写在 kv 大锁外执行，降低临界区时长。
- Get 的磁盘读取在释放 RLock 后执行，减少写阻塞。
- `nextFileID` 分配有锁保护，避免并发重复文件名。
- 元数据替换（flush/compaction）通过写锁做原子更新。
- Manifest 在元数据更新后写入，重启可恢复逻辑顺序。

## 7. 测试覆盖说明

- `internal/lsm/memtable_test.go`
- 基础读写删、并发读写、恢复链路。

- `internal/lsm/kvstore_test.go`
- 基础操作、大量 flush、高并发竞态、崩溃重启恢复。

- `tests/e2e_test.go`
- CLI 端到端验证（构建二进制并执行 put/get/exit）。

## 8. 已知风险与优化方向

- WAL 清空时机在每次 flush 后立即执行，若未来支持多个 imm/并行 flush，需要引入更严格的 WAL segment 管理。
- Manifest 目前是 append 日志，无 checksum/版本号，可考虑加入事务标记与校验。
- `readValFromSSTable` 每次查询都 `os.Open`，可引入 TableCache 降低 syscall 开销。
- Compaction 尚未真正落地，当前只完成控制面与接口面。

## 9. 快速阅读建议

建议按以下顺序读代码：

1. `cmd/kvnode/main.go`
2. `internal/lsm/kvstore.go`
3. `internal/lsm/kvstore_compaction.go`
4. `internal/wal/wal.go`
5. `internal/lsm/sstable_writer.go` + `internal/lsm/sstable_reader.go`
6. `internal/lsm/compaction_*.go`
7. `internal/lsm/kvstore_test.go` + `tests/e2e_test.go`

这样可以先建立全局调用图，再看细节格式与边界条件。
