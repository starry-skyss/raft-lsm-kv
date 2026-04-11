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
		fmt.Printf("ERROR: Failed to create wal directory: %v\n", err)
	}
	return &KVStore{
		memTable:   NewMemTable(),
		immTable:   nil,
		sstables:   make([]*SSTableMeta, 0),
		nextFileID: 1,
		wal:        w,
		sstDir:     sstDir,
	}
}

func (kv *KVStore) Put(key, val string) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	if err := kv.wal.AppendPut(key, val); err != nil {
		return err
	}

	kv.memTable.Put([]byte(key), []byte(val))
	kv.checkAndFlush()
	return nil
}

func (kv *KVStore) Get(key string) (string, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	kBytes := []byte(key)

	if val, ok := kv.memTable.Get(kBytes); ok {
		if val == nil {
			return "", false
		}
		return string(val), true
	}

	if kv.immTable != nil {
		if val, ok := kv.immTable.Get(kBytes); ok {
			if val == nil {
				return "", false
			}
			return string(val), true
		}
	}

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

	return "", false
}

func (kv *KVStore) Delete(key string) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	if err := kv.wal.AppendDelete(key); err != nil {
		return err
	}

	kv.memTable.Put([]byte(key), nil)
	kv.checkAndFlush()
	return nil
}

func (kv *KVStore) Len() int {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	return len(kv.memTable.pairs)
}
