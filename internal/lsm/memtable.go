package lsm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"raft-lsm-kv/internal/wal"
	"sort"
	"sync"
)

// ================= 1. MemTable 定义与实现 =================

// KVPair 表示一个键值对
type KVPair struct {
	Key   []byte
	Value []byte // 如果 Value 为 nil，代表这是一个被删除的 Key（墓碑）
}

// MemTable 使用有序 Slice 作为底层存储
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

// ================= 2. KVStore (LSM 引擎入口) =================

type BlockHandle struct {
	Offset uint64 // 数据块在文件中的偏移位置
	Size   uint64 // 数据块的大小
}

type Footer struct {
	IndexHandle BlockHandle // 指向 MetaIndex Block 的位置和大小
	MagicNumber uint64      // 文件格式标识，方便未来版本升级
}

const FooterSize = 24 // IndexHandle 的 16 字节 + MagicNumber 的 8 字节

type SSTableMeta struct {
	FileID uint64 // 比如 1 代表 000001.sst
	MinKey []byte // 该文件的最小 Key
	MaxKey []byte // 该文件的最大 Key
	Size   uint64 // 文件大小
	Index  []IndexEntry
	// Level  int // 等 Week 4 做 Compaction 时，我们再给它分层
}

type KVStore struct {
	mu         sync.RWMutex
	memTable   *MemTable      //接收写入
	immTable   *MemTable      //flush只读表
	sstables   []*SSTableMeta // 新增：内存中维护的磁盘文件列表
	nextFileID uint64         // 用于生成下一个文件名，比如 1, 2, 3...
	wal        *wal.WAL
	sstDir     string //保存 sst 文件的具体目录
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

// checkAndFlush 检查容量并在需要时触发后台落盘
func (kv *KVStore) checkAndFlush() {
	// 假设阈值是 4096 字节 (4KB)
	if kv.memTable.size >= 4096 {
		if kv.immTable != nil {
			// MVP 阶段简单处理：如果上一波还没落盘完，新的一波又满了，直接阻塞等待（反压）
			// 实际工程中这里可以用 condition variable 等待
			fmt.Println("Warning: Write too fast, waiting for previous flush...")
			return
		}

		// 切换表
		kv.immTable = kv.memTable
		kv.memTable = NewMemTable()

		// 开启后台 Goroutine 落盘，不要阻塞当前的 Put/Delete
		go kv.flush(kv.immTable)
	}
}

func (kv *KVStore) Put(key, val string) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	// 1. 写 WAL
	if err := kv.wal.AppendPut(key, val); err != nil {
		return err
	}

	// 2. 写 MemTable (这里我们把 string 转成 []byte 存入底层)
	kv.memTable.Put([]byte(key), []byte(val))

	// 3. 检查是否需要 Flush
	kv.checkAndFlush()
	return nil
}

func (kv *KVStore) Get(key string) (string, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	kBytes := []byte(key)

	// 1. 先查活跃的 MemTable
	if val, ok := kv.memTable.Get(kBytes); ok {
		if val == nil { // 检查是否是墓碑
			return "", false
		}
		return string(val), true
	}

	// 2. 如果 active 里没有，再查正在落盘的 immTable
	if kv.immTable != nil {
		if val, ok := kv.immTable.Get(kBytes); ok {
			if val == nil { // 检查是否是墓碑
				return "", false
			}
			return string(val), true
		}
	}

	// 3. 倒序遍历磁盘上的 SSTable (从最新的往旧的查)
	for i := len(kv.sstables) - 1; i >= 0; i-- {
		meta := kv.sstables[i]

		// O(1) 拦截：如果目标 key 不在这个文件的 [MinKey, MaxKey] 范围内，直接跳过！
		if bytes.Compare(kBytes, meta.MinKey) < 0 || bytes.Compare(kBytes, meta.MaxKey) > 0 {
			continue
		}

		// 拼出文件名去磁盘里线性扫描
		val, found := readValFromSSTable(meta, kv.sstDir, kBytes)

		if found {
			if val == nil { // 碰到墓碑了
				return "", false
			}
			return string(val), true
		}
	}

	// 内存和所有磁盘文件里都找不到
	return "", false
}

// 传入 meta，利用它内存中的 Index 进行二分查找
func readValFromSSTable(meta *SSTableMeta, dir string, targetKey []byte) ([]byte, bool) {
	// 1. 在内存中的 Index 数组里进行二分查找
	idx := sort.Search(len(meta.Index), func(i int) bool {
		return bytes.Compare(meta.Index[i].MaxKey, targetKey) >= 0
	})

	// 如果所有的 MaxKey 都比 targetKey 小，说明不在这个文件里
	if idx == len(meta.Index) {
		return nil, false
	}

	handle := meta.Index[idx].Handle

	// 2. 精准打开文件，Seek 到对应的数据块
	filename := filepath.Join(dir, fmt.Sprintf("%06d.sst", meta.FileID))
	file, err := os.Open(filename)
	if err != nil {
		return nil, false
	}
	defer file.Close()

	// 3. 把这 4KB (或更大) 的 Data Block 一次性读进内存
	blockData := make([]byte, handle.Size)
	if _, err := file.ReadAt(blockData, int64(handle.Offset)); err != nil {
		return nil, false
	}

	// 4. 在这 4KB 的内存 buffer 里，寻找真实的 KV（就像你之前的线性扫描，不过是在内存里扫 4KB）
	// 你可以使用 bytes.Reader 来解析这块内存：
	reader := bytes.NewReader(blockData)
	lenBuf := make([]byte, 4)

	for reader.Len() > 0 {
		// ... (读 KeyLen, 读 Key, 读 ValLen, 读 Val)
		// ... 如果 bytes.Equal(keyBuf, targetKey) 则返回
		// 读 KeyLen
		if _, err := io.ReadFull(reader, lenBuf); err != nil {
			break
		}
		keyLen := binary.LittleEndian.Uint32(lenBuf)

		// 读 Key
		keyBuf := make([]byte, keyLen)
		if _, err := io.ReadFull(reader, keyBuf); err != nil {
			break
		}

		// 读 ValLen
		if _, err := io.ReadFull(reader, lenBuf); err != nil {
			break
		}
		valLen := binary.LittleEndian.Uint32(lenBuf)

		var valBuf []byte
		if valLen == 0xFFFFFFFF { // 墓碑标记
			valBuf = nil
		} else {
			valBuf = make([]byte, valLen)
			if _, err := io.ReadFull(reader, valBuf); err != nil {
				break
			}
		}

		if bytes.Equal(keyBuf, targetKey) {
			return valBuf, true // 找到了，返回 Value 和 true
		}

	}

	return nil, false // 数据块里没找到
}

func (kv *KVStore) Delete(key string) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	if err := kv.wal.AppendDelete(key); err != nil {
		return err
	}

	// LSM 的 Delete 就是插入一个 value 为 nil 的墓碑！
	kv.memTable.Put([]byte(key), nil)

	kv.checkAndFlush()
	return nil
}

// RecoverFromWAL 负责在系统启动时，调用 wal 的重放功能
func (kv *KVStore) RecoverFromWAL() error {
	// 系统刚刚启动，不用加锁
	err := kv.wal.Replay(
		func(key, val string) {
			kv.memTable.Put([]byte(key), []byte(val))
		},
		func(key string) {
			kv.memTable.Put([]byte(key), nil) // 恢复墓碑
		},
	)
	return err
}

func (kv *KVStore) Len() int {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	return len(kv.memTable.pairs)
}

// flush 是后台写盘任务的骨架
func (kv *KVStore) flush(imm *MemTable) {
	if imm == nil || len(imm.pairs) == 0 {
		return
	}
	// TODO: 1. 将 imm.pairs 序列化写入 SSTable 文件
	// TODO: 2. 写入完成后，清空 WAL（因为数据已经安全落盘了）

	kv.mu.Lock()
	fileID := kv.nextFileID
	kv.nextFileID++
	kv.mu.Unlock()

	filename := filepath.Join(kv.sstDir, fmt.Sprintf("%06d.sst", fileID))

	// ============ 阶段 2：执行沉重的写盘操作 ============
	// 此时没有持有任何锁！前台可以继续疯狂 Put 到新的 memTable
	size, index, err := writeSSTable(filename, imm)
	if err != nil {
		fmt.Printf("ERROR: Failed to flush sstable %s: %v\n", filename, err)
		// 实际工程中这里需要报警、重试或者 Panic 宕机
		return
	}

	// ============ 阶段 3：提取元信息 ============
	// 因为 immTable 现在是只读的，所以安全，不需要锁
	minKey := imm.pairs[0].Key
	maxKey := imm.pairs[len(imm.pairs)-1].Key

	meta := &SSTableMeta{
		FileID: fileID,
		MinKey: minKey, // 有序切片的第一个元素
		MaxKey: maxKey, // 有序切片的最后一个元素
		Size:   size,
		Index:  index,
	}

	// ============ 阶段 4：注册新文件，清理旧状态 ============
	kv.mu.Lock()
	// 把新生成的 SSTable 元数据加到内存列表中
	kv.sstables = append(kv.sstables, meta)

	// 接力完成，清空 immTable，留给 Go 的 GC 去回收内存
	kv.immTable = nil

	// TODO: 清空或轮转 WAL 日志！
	// 因为 immTable 里的数据已经持久化到 SSTable 了，此时即便机器断电，
	// 数据也不会丢。所以我们可以安全地把 WAL 里对应的数据删掉，防止 WAL 无限膨胀。
	// 如果你之前在 wal 包里实现了 Clear() 或 Rotate() 方法，在这里调用：
	// kv.wal.Clear()

	kv.mu.Unlock()

	fmt.Printf("Flush completed! Created %s (Size: %d bytes, Range: [%s] - [%s])\n",
		filename, size, string(minKey), string(maxKey))

}

type IndexEntry struct {
	MaxKey []byte
	Handle BlockHandle
}

// 写盘核心逻辑
func writeSSTable(filename string, imm *MemTable) (uint64, []IndexEntry, error) {
	file, err := os.Create(filename)
	if err != nil {
		return 0, nil, err
	}
	defer file.Close()

	var writtenBytes uint64 = 0 //计数器
	var dataBuf bytes.Buffer
	var index []IndexEntry
	var currentOffset uint64 = 0

	// 用一个 8 字节的 buffer 复用，避免频繁内存分配
	lenBuf := make([]byte, 8)

	for i, pair := range imm.pairs {
		// 1. 写入 Key 的长度 (uint32)
		binary.LittleEndian.PutUint32(lenBuf[0:4], uint32(len(pair.Key)))
		dataBuf.Write(lenBuf[0:4])

		// 2. 写入 Key 内容
		dataBuf.Write(pair.Key)

		// 3. 处理 Value 长度和内容
		if pair.Value == nil {
			// 墓碑标记：用最大值或特定值表示，这里我们如果转成 int32，就是 -1
			binary.LittleEndian.PutUint32(lenBuf[0:4], 0xFFFFFFFF)
			dataBuf.Write(lenBuf[0:4])
		} else {
			// 正常写入 Value 长度 (uint32)
			binary.LittleEndian.PutUint32(lenBuf[0:4], uint32(len(pair.Value)))
			dataBuf.Write(lenBuf[0:4])

			// 4. 写入 Value 内容
			dataBuf.Write(pair.Value)
		}

		if dataBuf.Len() >= 4096 || i == len(imm.pairs)-1 {
			// 写满了或者是最后一个了，先把数据块写到文件里
			_, err := file.Write(dataBuf.Bytes())
			if err != nil {
				return writtenBytes, index, err
			}
			writtenBytes += uint64(dataBuf.Len())

			// 记录索引信息：Key 和对应的数据块位置
			index = append(index, IndexEntry{
				MaxKey: pair.Key,
				Handle: BlockHandle{Offset: uint64(currentOffset), Size: uint64(dataBuf.Len())},
			})

			currentOffset += uint64(dataBuf.Len())
			dataBuf.Reset() // 清空 buffer，准备写下一批数据
		}
	}
	// 3. 构建并写入 Index Block
	// 把上面的 index 数组也序列化，写进 file
	var indexBuf bytes.Buffer
	for _, entry := range index {
		// 写 MaxKey 长度
		binary.LittleEndian.PutUint32(lenBuf[0:4], uint32(len(entry.MaxKey)))
		indexBuf.Write(lenBuf[0:4])

		// 写 MaxKey 内容
		indexBuf.Write(entry.MaxKey)

		// 写 BlockHandle (Offset 和 Size)
		binary.LittleEndian.PutUint64(lenBuf[0:8], entry.Handle.Offset)
		indexBuf.Write(lenBuf)
		binary.LittleEndian.PutUint64(lenBuf[0:8], entry.Handle.Size)
		indexBuf.Write(lenBuf)
	}

	indexOffset := currentOffset
	indexSize := uint64(indexBuf.Len())
	if _, err := file.Write(indexBuf.Bytes()); err != nil {
		return writtenBytes, index, err
	}
	writtenBytes += indexSize

	// 4. 构建并写入 Footer
	var footerBuf bytes.Buffer

	// 写 Index Block 的 BlockHandle
	binary.LittleEndian.PutUint64(lenBuf[0:8], indexOffset)
	footerBuf.Write(lenBuf)
	binary.LittleEndian.PutUint64(lenBuf[0:8], indexSize)
	footerBuf.Write(lenBuf)

	// 写 Magic Number (比如 0x12345678ABCDEF00，未来版本升级可以改这个值来区分格式)
	magicNumber := uint64(0x12345678ABCDEF00)
	binary.LittleEndian.PutUint64(lenBuf[0:8], magicNumber)
	footerBuf.Write(lenBuf)

	if _, err := file.Write(footerBuf.Bytes()); err != nil {
		return writtenBytes, index, err
	}
	writtenBytes += uint64(footerBuf.Len())

	if err := file.Sync(); err != nil {
		return writtenBytes, index, err
	}
	return writtenBytes, index, nil
}

func (kv *KVStore) loadSSTables() error {
	// 1. 读取 sstDir 目录下所有的文件
	files, err := os.ReadDir(kv.sstDir)
	if err != nil {
		return err
	}

	var loadedMetas []*SSTableMeta

	for _, f := range files {
		if filepath.Ext(f.Name()) != ".sst" {
			continue
		}

		// 解析文件名，获取 FileID (例如 "000001.sst" -> 1)
		var fileID uint64
		fmt.Sscanf(f.Name(), "%06d.sst", &fileID)

		filePath := filepath.Join(kv.sstDir, f.Name())
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}

		stat, _ := file.Stat()
		fileSize := stat.Size()

		// 2. 读 Footer (最后 24 字节)
		if fileSize < 24 {
			file.Close()
			continue // 文件损坏或过小
		}
		footerBuf := make([]byte, 24)
		file.ReadAt(footerBuf, fileSize-24)

		magic := binary.LittleEndian.Uint64(footerBuf[16:24])
		if magic != 0x12345678ABCDEF00 {
			file.Close()
			panic(fmt.Sprintf("SSTable %s is corrupted: bad magic number", f.Name()))
		}

		indexOffset := binary.LittleEndian.Uint64(footerBuf[0:8])
		indexSize := binary.LittleEndian.Uint64(footerBuf[8:16])

		// 3. 读 Index Block
		indexBuf := make([]byte, indexSize)
		file.ReadAt(indexBuf, int64(indexOffset))

		// 4. 反序列化 Index Block 恢复到内存
		var index []IndexEntry
		reader := bytes.NewReader(indexBuf)
		lenBuf := make([]byte, 8) // 用来读 4字节或8字节

		for reader.Len() > 0 {
			io.ReadFull(reader, lenBuf[:4])
			keyLen := binary.LittleEndian.Uint32(lenBuf[:4])

			key := make([]byte, keyLen)
			io.ReadFull(reader, key)

			io.ReadFull(reader, lenBuf) // 读 8 字节 offset
			offset := binary.LittleEndian.Uint64(lenBuf)

			io.ReadFull(reader, lenBuf) // 读 8 字节 size
			size := binary.LittleEndian.Uint64(lenBuf)

			index = append(index, IndexEntry{
				MaxKey: key,
				Handle: BlockHandle{Offset: offset, Size: size},
			})
		}
		file.Close()

		if len(index) == 0 {
			continue
		}

		// 5. 组装 Meta
		meta := &SSTableMeta{
			FileID: fileID,
			MinKey: index[0].MaxKey, // 注：严格来说应该是该文件首个KV的Key，为了简化，MVP阶段用第一个Block的MaxKey代替也可
			MaxKey: index[len(index)-1].MaxKey,
			Size:   uint64(fileSize),
			Index:  index,
		}
		loadedMetas = append(loadedMetas, meta)

		// 更新 nextFileID 以防覆盖老文件
		if fileID >= kv.nextFileID {
			kv.nextFileID = fileID + 1
		}
	}

	// 6. 按照 FileID 从小到大排序 (FileID 越大，数据越新)
	sort.Slice(loadedMetas, func(i, j int) bool {
		return loadedMetas[i].FileID < loadedMetas[j].FileID
	})

	kv.sstables = loadedMetas
	return nil
}
