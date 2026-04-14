package lsm

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"raft-lsm-kv/internal/wal"
	"sync"
	"time"
)

type KVStore struct {
	mu                  sync.RWMutex
	memTable            *MemTable      // 接收写入
	immTable            *MemTable      // flush 只读表
	sstables            []*SSTableMeta // 内存中维护的磁盘文件列表（逻辑顺序）
	nextFileID          uint64         // 全局单调递增文件号
	wal                 *wal.WAL       // 当前 WAL 文件（活跃的写日志）
	nextwal             *wal.WAL       // 下一个 WAL 文件（切分后）
	walNotifyCh         chan struct{}  // 通知后台 compaction 有新文件需要处理
	walFileID           uint64         // 当前 WAL 文件号
	rootDir             string         //根目录
	dataDir             string         //数据目录
	sstDir              string         //SSTable目录
	walDir              string         //WAL目录
	manifestPath        string         //Manifest文件路径
	compactionThreshold int            //触发 compaction 的 SSTable 数量阈值
	compactionInterval  time.Duration  //定时 compaction 的时间间隔
}

func NewKVStore(rootDir string) *KVStore {
	dataDir := filepath.Join(rootDir, "data")
	sstDir := filepath.Join(dataDir, "sst")
	walDir := filepath.Join(dataDir, "wal")
	manifestPath := filepath.Join(dataDir, "manifest.log")

	if err := os.MkdirAll(sstDir, 0755); err != nil {
		fmt.Printf("ERROR: Failed to create sst directory: %v\n", err)
	}
	if err := os.MkdirAll(walDir, 0755); err != nil {
		fmt.Printf("ERROR: Failed to create wal directory: %v\n", err)
	}

	kv := &KVStore{
		memTable:            NewMemTable(),
		immTable:            nil,
		sstables:            make([]*SSTableMeta, 0),
		nextFileID:          1,
		nextwal:             nil,
		walNotifyCh:         make(chan struct{}, 1),
		walFileID:           1,
		rootDir:             rootDir,
		dataDir:             dataDir,
		sstDir:              sstDir,
		walDir:              walDir,
		manifestPath:        manifestPath,
		compactionThreshold: 4,
		compactionInterval:  10 * time.Second,
	}
	// 启动时加载现有的 SSTable 文件，并从 WAL 恢复数据
	if err := kv.loadSSTables(); err != nil {
		fmt.Printf("ERROR: Failed to load existing SSTables: %v\n", err)
	}
	if err := kv.RecoverFromWAL(); err != nil {
		fmt.Printf("ERROR: Failed to recover from WAL during initialization: %v\n", err)
	}
	//强行落盘
	if kv.memTable.size > 0 {
		// 把装满回放数据的 memTable 变成 immTable
		kv.immTable = kv.memTable
		kv.memTable = NewMemTable()

		// 直接调用现成的 flush！
		// 💡 盲点突破：这里传 nil 作为 oldWal。
		// 因为你的 installFlushedTable 里面有 `if oldWal != nil` 的防御性检查，
		// 传 nil 就意味着“只落盘、只改 Manifest，先别删 WAL”。
		kv.flush(kv.immTable, nil)
	}

	// 3. 💣 大洗牌：一把火烧掉所有旧账
	// 既然数据已经安全变成了 SSTable，直接把 walDir 目录下所有 .log 后缀的文件全部物理删除！
	kv.removeAllOldWalFiles() // 你需要自己写这个不到 10 行的辅助函数，它会删除 walDir 目录下所有 .log 文件（包括之前的回放文件和切分前的旧 WAL 文件）

	//回放完毕！现在可以用确定好的 kv.walFileID 创建全新的活跃 WAL 了！
	activeWal, _ := wal.OpenWAL(filepath.Join(walDir, fmt.Sprintf("%06dwal.log", kv.walFileID)))
	kv.wal = activeWal
	kv.walFileID++ // 当前的被占用了，全局 ID 往后挪一位给老黄牛用

	// 启动后台 compaction 触发器（架构骨架）
	kv.StartCompactionLoop()

	// 启动 WAL 预分配器
	go kv.walPreAllocator()
	kv.walNotifyCh <- struct{}{} // 启动时预分配第一个 WAL 文件
	return kv
}

func (kv *KVStore) removeAllOldWalFiles() error {
	files, err := os.ReadDir(kv.walDir)
	if err != nil {
		return fmt.Errorf("failed to read wal directory: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".log" {
			fullPath := filepath.Join(kv.walDir, file.Name())
			if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
				fmt.Printf("ERROR: Failed to remove old WAL file %s: %v\n", fullPath, err)
			} else {
				fmt.Printf("Removed old WAL file: %s\n", fullPath)
			}
		}
	}
	return nil
}

func (kv *KVStore) Put(key, val string) error {
	// 1. 独立写 WAL（磁盘 I/O）
	// WAL 内部自带 w.mu 锁，并发安全，不要把它包在 kv.mu 里面
	if err := kv.wal.AppendPut(key, val); err != nil {
		return err
	}

	kv.mu.Lock()
	defer kv.mu.Unlock()

	kv.memTable.Put([]byte(key), []byte(val))
	kv.checkAndFlush()
	return nil
}

func (kv *KVStore) Get(key string) (string, bool) {
	kv.mu.RLock()
	kBytes := []byte(key)

	// 1. 先查 MemTable
	if val, ok := kv.memTable.Get(kBytes); ok {
		kv.mu.RUnlock() // 查到就解锁返回
		if val == nil {
			return "", false
		}
		return string(val), true
	}

	// 2. 查正在落盘的 immTable
	if kv.immTable != nil {
		if val, ok := kv.immTable.Get(kBytes); ok {
			kv.mu.RUnlock() // 查到就解锁返回
			if val == nil {
				return "", false
			}
			return string(val), true
		}
	}
	// ================== 添加点（重要架构注释） ==================
	// 💡 Trade-off (架构权衡):
	// 目前我们在持有 RLock 的情况下进行慢速的磁盘 I/O (readValFromSSTable)。
	// 这会导致在读盘期间，所有的 Put 写请求被阻塞。
	// 工业界的优化方案：先在 RLock 保护下，将需要查询的 []*SSTableMeta 拷贝到一个局部变量中，
	// 然后释放 RLock，再拿着局部的 meta 列表去无锁地进行底层磁盘 I/O。
	// 这样读磁盘就不会阻塞前台的新写入了。
	// (MVP 阶段为保证实现正确性，暂用粗粒度锁)
	// ============================================================

	// 3. 🌟 核心优化：提取 SSTableMeta 列表的快照！
	// 我们把当前的文件列表拷贝一份到局部变量中。
	var sstablesSnapshot []*SSTableMeta
	sstablesSnapshot = append(sstablesSnapshot, kv.sstables...)

	// 列表拷贝完了，立刻释放 RLock！
	// 从此刻起，后台的 Flush 和前台的 Put 再也不会被慢速磁盘 I/O 阻塞了！
	kv.mu.RUnlock()

	// 4. 在快照上倒序遍历（以下全部是无锁执行）
	for i := len(sstablesSnapshot) - 1; i >= 0; i-- {
		meta := sstablesSnapshot[i]
		if bytes.Compare(kBytes, meta.MinKey) < 0 || bytes.Compare(kBytes, meta.MaxKey) > 0 {
			continue
		}

		val, found := readValFromSSTable(meta, kv.sstDir, kBytes)
		if found {
			if val == nil {
				return "", false
			}
			return string(val), true
		}
	}

	/*
		for i := len(kv.sstables) - 1; i >= 0; i-- {
			meta := kv.sstables[i]
			if bytes.Compare(kBytes, meta.MinKey) < 0 || bytes.Compare(kBytes, meta.MaxKey) > 0 {
				continue
			}

			val, found := readValFromSSTable(meta, kv.sstDir, kBytes)
			if found {
				if val == nil {
					return "", false
				}
				return string(val), true
			}
		}
	*/

	return "", false
}

func (kv *KVStore) Delete(key string) error {

	// 同put，磁盘 I/O 移出大锁
	if err := kv.wal.AppendDelete(key); err != nil {
		return err
	}
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.memTable.Put([]byte(key), nil)
	kv.checkAndFlush()
	return nil
}

func (kv *KVStore) Len() int {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	return len(kv.memTable.pairs)
}

func (kv *KVStore) walPreAllocator() {
	for {
		<-kv.walNotifyCh // 等待通知
		//创建wal文件
		newWal, err := wal.OpenWAL(filepath.Join(kv.walDir, fmt.Sprintf("%06dwal.log", kv.walFileID)))
		if err != nil {
			fmt.Printf("ERROR: Failed to pre-allocate WAL file: %v\n", err)
			time.Sleep(100 * time.Millisecond) // 避免频繁失败时的忙等待
			continue
		}
		kv.mu.Lock()
		kv.nextwal = newWal
		kv.walFileID++
		kv.mu.Unlock()

	}
}
