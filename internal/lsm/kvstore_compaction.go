package lsm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"raft-lsm-kv/internal/wal"
)

// ================= 后台引擎：Flush =================

// checkAndFlush 检查容量并在需要时触发后台落盘
func (kv *KVStore) checkAndFlush() {
	// 假设阈值是 4096 字节 (4KB)
	if kv.memTable.size >= 4096 {
		if kv.immTable != nil {
			// MVP 阶段简单处理：如果上一波还没落盘完，新的一波又满了，直接阻塞等待（反压）
			fmt.Println("Warning: Write too fast, waiting for previous flush...")
			return
		}
		if kv.nextwal== nil {
			fmt.Println("Warning: Write too fast, waiting for previous wal split...")
			return
		}
		//切换memtable
		kv.immTable = kv.memTable
		kv.memTable = NewMemTable()
		//切分WAL
		oldWal := kv.wal
		kv.wal = kv.nextwal
		kv.nextwal = nil

		// 💡 架构闭环: 切分 WAL 是为了让新的写入继续记录到新的日志文件中，旧的日志文件则交给后台线程处理，避免写入被落盘阻塞。
		select {
		case kv.walNotifyCh <- struct{}{}:
		default:
		}

		go kv.flush(kv.immTable, oldWal)
	}
}

// flush 是后台写盘任务的骨架
func (kv *KVStore) flush(imm *MemTable, oldWal *wal.WAL) {
	if imm == nil || len(imm.pairs) == 0 {
		return
	}

	fileID := kv.reserveNextFileID()

	filename := filepath.Join(kv.sstDir, fmt.Sprintf("%06d.sst", fileID))

	size, index, err := writeSSTableFromMem(filename, imm)
	if err != nil {
		fmt.Printf("ERROR: Failed to flush sstable %s: %v\n", filename, err)
		return
	}

	minKey := imm.pairs[0].Key
	maxKey := imm.pairs[len(imm.pairs)-1].Key

	meta := &SSTableMeta{
		FileID: fileID,
		MinKey: minKey,
		MaxKey: maxKey,
		Size:   size,
		Index:  index,
	}

	kv.installFlushedTable(meta, oldWal)

	fmt.Printf("Flush completed! Created %s (Size: %d bytes, Range: [%s] - [%s])\n",
		filename, size, string(minKey), string(maxKey))
}

func (kv *KVStore) reserveNextFileID() uint64 {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	fileID := kv.nextFileID
	kv.nextFileID++
	return fileID
}

func (kv *KVStore) installFlushedTable(meta *SSTableMeta, oldWal *wal.WAL) {
	kv.mu.Lock()

	kv.sstables = append(kv.sstables, meta)
	kv.immTable = nil
	snapshot := append([]*SSTableMeta(nil), kv.sstables...) //当前逻辑顺序快照
	kv.mu.Unlock()

	if err := kv.persistManifest(snapshot); err != nil {
		fmt.Printf("ERROR: Failed to persist manifest after flush: %v\n", err)
	}

	// ================== 添加点 ==================
	// 💡 架构闭环: 数据已经安全固化到 SSTable，此时哪怕断电也不会丢。
	// 旧 WAL 已不再参与写入，直接删除其对应文件，避免无限膨胀和重复回放。
	if oldWal != nil {
		if err := oldWal.Delete(); err != nil {
			fmt.Printf("ERROR: Failed to delete old WAL after flush: %v\n", err)
		}
	}
	// ============================================
}

// ================= 后台引擎：Recovery =================

// RecoverFromWAL 负责在系统启动时，调用 wal 的重放功能
func (kv *KVStore) RecoverFromWAL() error {
// 1. 扫盘：读取 wal 目录下的所有文件
    files, err := os.ReadDir(kv.walDir)
    if err != nil {
        return err
    }

    var walIDs []uint64
    for _, f := range files {
        if filepath.Ext(f.Name()) == ".log" {
            var id uint64
            // 假设文件名是 000001wal.log
            fmt.Sscanf(f.Name(), "%06dwal.log", &id)
            walIDs = append(walIDs, id)
        }
    }

    // 2. 排序：确保从小到大顺序回放
    sort.Slice(walIDs, func(i, j int) bool {
        return walIDs[i] < walIDs[j]
    })

    // 3. 核心逻辑：挨个回放
    for _, id := range walIDs {
        walPath := filepath.Join(kv.walDir, fmt.Sprintf("%06dwal.log", id))
        
        // TODO: (你来主笔)
        // 3.1 你需要写代码去临时打开这个 walPath 文件进行读取
        // 3.2 调用这个旧 wal 的 Replay 方法把数据塞进 kv.memTable
        // 3.3 回放完了之后，如何处理这个已经没用的旧文件？(提示：数据已经在 MemTable 里了)
		curwal,err:=wal.OpenWAL(walPath)
		if err!=nil{
			fmt.Printf("ERROR: Failed to open WAL file %s for recovery: %v\n", walPath, err)
			continue
		}
		err=curwal.Replay(func(key, val string) {
			kv.memTable.Put([]byte(key), []byte(val))
		}, func(key string) {
			kv.memTable.Put([]byte(key), nil)
		})
		if err!=nil{
			fmt.Printf("ERROR: Failed to replay WAL file %s: %v\n", walPath, err)
		}
		curwal.Close()
    }

    // 4. TODO: (你来主笔) 确定下一个新 WAL 的 ID
    // 遍历结束后，你需要根据最大的 walID，来更新 kv.walFileID。
    // 如果没有任何旧文件（第一次启动），kv.walFileID 应该设为什么？
	if len(walIDs)>0{
		kv.walFileID=walIDs[len(walIDs)-1]+1
	}else{
		kv.walFileID=1
	}
    return nil
}

// loadSSTables 扫描数据目录下的 .sst 文件，校验 Magic Number，
// 读取文件尾部的 Index Block 构建出内存索引结构（SSTableMeta），最后根据 Manifest 恢复正确的逻辑文件层级顺序
func (kv *KVStore) loadSSTables() error {
	files, err := os.ReadDir(kv.sstDir)
	if err != nil {
		return err
	}

	var loadedMetas []*SSTableMeta

	for _, f := range files {
		if filepath.Ext(f.Name()) != ".sst" {
			continue
		}

		var fileID uint64
		fmt.Sscanf(f.Name(), "%06d.sst", &fileID)

		filePath := filepath.Join(kv.sstDir, f.Name())
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}

		stat, _ := file.Stat()
		fileSize := stat.Size()
		if fileSize < FooterSize {
			file.Close()
			continue
		}

		footerBuf := make([]byte, FooterSize)
		if _, err := file.ReadAt(footerBuf, fileSize-FooterSize); err != nil {
			file.Close()
			return err
		}

		magic := binary.LittleEndian.Uint64(footerBuf[16:24])
		if magic != SSTableMagicNumber {
			file.Close()
			return fmt.Errorf("sstable %s is corrupted: bad magic number", f.Name())
		}

		indexOffset := binary.LittleEndian.Uint64(footerBuf[0:8])
		indexSize := binary.LittleEndian.Uint64(footerBuf[8:16])
		indexBuf := make([]byte, indexSize)
		if _, err := file.ReadAt(indexBuf, int64(indexOffset)); err != nil {
			file.Close()
			return err
		}

		var index []IndexEntry
		reader := bytes.NewReader(indexBuf)
		lenBuf := make([]byte, 8)

		for reader.Len() > 0 {
			if _, err := io.ReadFull(reader, lenBuf[:4]); err != nil {
				break
			}
			keyLen := binary.LittleEndian.Uint32(lenBuf[:4])

			key := make([]byte, keyLen)
			if _, err := io.ReadFull(reader, key); err != nil {
				break
			}

			if _, err := io.ReadFull(reader, lenBuf); err != nil {
				break
			}
			offset := binary.LittleEndian.Uint64(lenBuf)

			if _, err := io.ReadFull(reader, lenBuf); err != nil {
				break
			}
			size := binary.LittleEndian.Uint64(lenBuf)

			index = append(index, IndexEntry{
				MaxKey: key,
				Handle: BlockHandle{Offset: offset, Size: size},
			})
		}
		if len(index) == 0 {
			file.Close()
			continue
		}

		minKey := make([]byte, 0)
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			file.Close()
			return err
		}
		minKeyBuf := make([]byte, 4)
		if _, err := io.ReadFull(file, minKeyBuf); err == nil {
			keyLen := binary.LittleEndian.Uint32(minKeyBuf)
			minKey = make([]byte, keyLen)
			if _, err := io.ReadFull(file, minKey); err != nil {
				file.Close()
				return err
			}
		}
		file.Close()

		meta := &SSTableMeta{
			FileID: fileID,
			MinKey: minKey,
			MaxKey: index[len(index)-1].MaxKey,
			Size:   uint64(fileSize),
			Index:  index,
		}
		loadedMetas = append(loadedMetas, meta)

		kv.bumpNextFileID(fileID)
	}

	sort.Slice(loadedMetas, func(i, j int) bool {
		return loadedMetas[i].FileID < loadedMetas[j].FileID
	})

	reordered, err := kv.applyManifestOrder(loadedMetas)
	if err != nil {
		return err
	}

	kv.sstables = reordered
	return nil
}

func (kv *KVStore) bumpNextFileID(fileID uint64) {
	if fileID >= kv.nextFileID {
		kv.nextFileID = fileID + 1
	}
}

// 提取文件 ID 列表并以逗号分隔追加写入 manifest.log。重启时只需读取最后一行即可恢复表顺序
func (kv *KVStore) persistManifest(snapshot []*SSTableMeta) error {
	ids := make([]string, 0, len(snapshot))
	for _, meta := range snapshot {
		ids = append(ids, strconv.FormatUint(meta.FileID, 10))
	}

	line := strings.Join(ids, ",") + "\n"
	manifestDir := filepath.Dir(kv.manifestPath)
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(kv.manifestPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(line); err != nil {
		return err
	}
	return f.Sync()
}

// 读取 manifest.log 最后一行有效记录，对物理文件列表进行重排，确保 Get 查询时能够按从新到旧的正确顺序遍历
func (kv *KVStore) applyManifestOrder(loadedMetas []*SSTableMeta) ([]*SSTableMeta, error) {
	data, err := os.ReadFile(kv.manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return loadedMetas, nil
		}
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	last := ""
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			last = line
			break
		}
	}
	if last == "" {
		return loadedMetas, nil
	}

	metaByID := make(map[uint64]*SSTableMeta, len(loadedMetas))
	for _, meta := range loadedMetas {
		metaByID[meta.FileID] = meta
	}

	ordered := make([]*SSTableMeta, 0, len(loadedMetas))
	seen := make(map[uint64]struct{}, len(loadedMetas))
	for _, idStr := range strings.Split(last, ",") {
		idStr = strings.TrimSpace(idStr)
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			continue
		}
		meta, ok := metaByID[id]
		if !ok {
			continue
		}
		ordered = append(ordered, meta)
		seen[id] = struct{}{}
	}

	for _, meta := range loadedMetas {
		if _, ok := seen[meta.FileID]; ok {
			continue
		}
		ordered = append(ordered, meta)
	}

	return ordered, nil
}

// ================= 后台引擎：Compaction =================

// StartCompactionLoop 启动一个后台 goroutine，定期检查是否需要合并 SSTable 文件，并执行合并
func (kv *KVStore) StartCompactionLoop() {
	if kv.compactionInterval <= 0 {
		kv.compactionInterval = 10 * time.Second
	}

	go func() {
		ticker := time.NewTicker(kv.compactionInterval)
		defer ticker.Stop()

		for range ticker.C {
			if !kv.needCompaction() {
				continue
			}
			if err := kv.doCompaction(); err != nil {
				fmt.Printf("ERROR: compaction failed: %v\n", err)
			}
		}
	}()
}

func (kv *KVStore) needCompaction() bool {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	return len(kv.sstables) >= kv.compactionThreshold
}

// pickCompactionFiles 根据 compaction 策略选择需要合并的 SSTable 文件列表。
// 这里我们简单实现了 tiered 策略：选最旧的 4 个文件进行合并。
func (kv *KVStore) pickCompactionFiles() []*SSTableMeta {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	if len(kv.sstables) < kv.compactionThreshold {
		return nil
	}

	pickN := kv.compactionThreshold

	picked := make([]*SSTableMeta, pickN)
	copy(picked, kv.sstables[:pickN]) // 简化 tiered: 选最旧的 4 个
	return picked
}

// installCompactedTable 将新生成的 SSTable 元信息插入内存列表，并持久化 Manifest 以记录新的文件层级顺序。
// 旧文件的物理删除留给锁外的 removeObsoleteFiles 来执行，避免长时间持锁。
func (kv *KVStore) installCompactedTable(newSST *SSTableMeta, oldSSTs []*SSTableMeta) {
	kv.mu.Lock()

	obsolete := make(map[uint64]struct{}, len(oldSSTs))
	for _, meta := range oldSSTs {
		obsolete[meta.FileID] = struct{}{}
	}

	// 容量按“是否有新表”计算，避免预估偏差
	capHint := len(kv.sstables) - len(oldSSTs)
	if capHint < 0 {
		capHint = 0
	}
	if newSST != nil {
		capHint++
	}
	kept := make([]*SSTableMeta, 0, capHint)
	for _, meta := range kv.sstables {
		if _, ok := obsolete[meta.FileID]; ok {
			continue
		}
		kept = append(kept, meta)
	}
	if newSST != nil {
		kept = append(kept, newSST)
	}
	kv.sstables = kept

	snapshot := append([]*SSTableMeta(nil), kv.sstables...)
	kv.mu.Unlock()

	if err := kv.persistManifest(snapshot); err != nil {
		fmt.Printf("ERROR: Failed to persist manifest after compaction: %v\n", err)
	}
}

func (kv *KVStore) removeObsoleteFiles(oldSSTs []*SSTableMeta) {
	for _, meta := range oldSSTs {
		name := filepath.Join(kv.sstDir, fmt.Sprintf("%06d.sst", meta.FileID))
		if err := os.Remove(name); err != nil && !os.IsNotExist(err) {
			fmt.Printf("WARN: Failed to remove obsolete sstable %s: %v\n", name, err)
		}
	}
}

// doCompaction 预留给 Week 4 的后台合并流程。
func (kv *KVStore) doCompaction() error {
	oldSSTs := kv.pickCompactionFiles()
	if len(oldSSTs) == 0 {
		return nil
	}

	// TODO(week4):
	// 1. 基于 oldSSTs 构造 SSTable 迭代器
	// 2. 使用 merge iterator 执行 k-way merge，生成新 SSTable
	// 3. 通过 installCompactedTable 做原子替换
	// 4. 锁外调用 removeObsoleteFiles 清理旧文件
	// 2. 为每个旧表构造迭代器
	var iters []Iterator
	for _, meta := range oldSSTs {
		it, err := NewSSTableIterator(meta, kv.sstDir)
		if err != nil {
			// 防御性编程：如果中途打开某个文件失败，必须把前面已经打开的文件句柄关掉，防止 FD 泄露
			for _, opened := range iters {
				opened.Close()
			}
			return fmt.Errorf("failed to create iterator for sst %d: %v", meta.FileID, err)
		}
		iters = append(iters, it)
	}

	// 3. 执行 K 路归并，提取最新且去重后的数据
	builder := doMerge(iters)

	// 极端情况：如果合并后所有数据都被相互抵消（比如全是墓碑），可以直接跳过写新表
	if builder.Len() == 0 {
		kv.installCompactedTable(nil, oldSSTs) // 从视图中抹除旧表
		kv.removeObsoleteFiles(oldSSTs)        // 物理删除旧表
		return nil
	}

	// 4. 为新表分配 ID 并落盘
	newFileID := kv.reserveNextFileID()
	filename := filepath.Join(kv.sstDir, fmt.Sprintf("%06d.sst", newFileID))

	// 💡 工程提示：这里假设我们有 writeSSTableFromPairs 函数，
	// 它和之前的 writeSSTable 逻辑完全一样，只是输入参数从 *MemTable 变成了 []KVPair。
	size, index, err := writeSSTableFromPairs(filename, builder.pairs)
	if err != nil {
		return fmt.Errorf("failed to write compacted sstable %s: %v", filename, err)
	}

	newMeta := &SSTableMeta{
		FileID: newFileID,
		MinKey: builder.pairs[0].Key,
		MaxKey: builder.pairs[len(builder.pairs)-1].Key,
		Size:   size,
		Index:  index,
	}

	// 5. 元数据视图原子替换并持久化 Manifest
	kv.installCompactedTable(newMeta, oldSSTs)

	// 6. 锁外进行安全的物理文件清理
	kv.removeObsoleteFiles(oldSSTs)

	fmt.Printf("Compaction completed! Replaced %d old files with %s (Size: %d bytes)\n",
		len(oldSSTs), filename, size)

	return nil
}
