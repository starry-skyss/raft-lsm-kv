package lsm

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"raft-lsm-kv/internal/wal"
	"sync"
)

type KVStore struct {
	mu         sync.RWMutex
	memTable   *MemTable      // 接收写入
	immTable   *MemTable      // flush 只读表
	sstables   []*SSTableMeta // 内存中维护的磁盘文件列表
	nextFileID uint64         // 用于生成下一个文件名，比如 1, 2, 3...
	wal        *wal.WAL
	sstDir     string // 保存 sst 文件的具体目录
}

func NewKVStore(w *wal.WAL, dir string) *KVStore {
	sstDir := filepath.Join(dir, "sst")
	if err := os.MkdirAll(sstDir, 0755); err != nil {
		fmt.Printf("ERROR: Failed to create sst directory: %v\n", err)
	}
	kv := &KVStore{
		memTable:   NewMemTable(),
		immTable:   nil,
		sstables:   make([]*SSTableMeta, 0),
		nextFileID: 1,
		wal:        w,
		sstDir:     sstDir,
	}
	// 启动时加载现有的 SSTable 文件，并从 WAL 恢复数据
	if err := kv.loadSSTables(); err != nil {
		fmt.Printf("ERROR: Failed to load existing SSTables: %v\n", err)
	}
	if err := kv.RecoverFromWAL(); err != nil {
		fmt.Printf("ERROR: Failed to recover from WAL during initialization: %v\n", err)
	}
	return kv
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
