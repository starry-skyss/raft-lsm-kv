package lsm

import (
	"bytes"
	"sort"
)

// ================= 1. MemTable 定义与实现 =================

// KVPair 表示一个键值对
type KVPair struct {
	Key   []byte
	Value []byte // 如果 Value 为 nil，代表这是一个被删除的 Key（墓碑）
}

// MemTable 使用有序 Slice 作为底层存储
// 💡 Trade-off (架构权衡):
// 真实的工业界实现 (如 LevelDB) 内存表通常使用 SkipList (跳表)。
// 本系统在 MVP 阶段为了降低心智负担并保证绝对的正确性，采用了有序 Slice。
// 优点：二分查找极其简单，CPU 缓存命中率极高 (Cache-friendly)。
// 缺点：每次插入新 Key 需要 O(N) 的数据挪动。
// 前提支撑：因为我们的 Flush 阈值极小 (如 4KB)，O(N) 的挪动成本微乎其微。如果阈值上升到 64MB，必须重构为跳表。
type MemTable struct {
	pairs []KVPair // 核心存储：保持按 Key 字母序排列
	size  int      // 跟踪当前估算的字节数，用于触发 Flush 阈值
}

func NewMemTable() *MemTable {
	return &MemTable{
		pairs: make([]KVPair, 0, 256),
		size:  0,
	}
}

// Put 写入数据或墓碑
func (m *MemTable) Put(key, val []byte) {
	idx := sort.Search(len(m.pairs), func(i int) bool {
		return bytes.Compare(m.pairs[i].Key, key) >= 0
	})

	// 存在则更新
	if idx < len(m.pairs) && bytes.Equal(m.pairs[idx].Key, key) {
		m.size += len(val) - len(m.pairs[idx].Value)
		m.pairs[idx].Value = val
		return
	}

	// 不存在则插入并挪动数据
	m.pairs = append(m.pairs, KVPair{})
	copy(m.pairs[idx+1:], m.pairs[idx:])
	m.pairs[idx] = KVPair{Key: key, Value: val}

	m.size += len(key) + len(val)
}

// Get 查询数据
func (m *MemTable) Get(key []byte) ([]byte, bool) {
	idx := sort.Search(len(m.pairs), func(i int) bool {
		return bytes.Compare(m.pairs[i].Key, key) >= 0
	})

	if idx < len(m.pairs) && bytes.Equal(m.pairs[idx].Key, key) {
		return m.pairs[idx].Value, true
	}
	return nil, false
}
