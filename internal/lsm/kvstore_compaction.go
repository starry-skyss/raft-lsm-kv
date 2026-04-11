package lsm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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

		kv.immTable = kv.memTable
		kv.memTable = NewMemTable()

		go kv.flush(kv.immTable)
	}
}

// flush 是后台写盘任务的骨架
func (kv *KVStore) flush(imm *MemTable) {
	if imm == nil || len(imm.pairs) == 0 {
		return
	}

	fileID := kv.reserveNextFileID()

	filename := filepath.Join(kv.sstDir, fmt.Sprintf("%06d.sst", fileID))

	size, index, err := writeSSTable(filename, imm)
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

	kv.installFlushedTable(meta)

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

func (kv *KVStore) installFlushedTable(meta *SSTableMeta) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	kv.sstables = append(kv.sstables, meta)
	kv.immTable = nil
}

// ================= 后台引擎：Recovery =================

// RecoverFromWAL 负责在系统启动时，调用 wal 的重放功能
func (kv *KVStore) RecoverFromWAL() error {
	err := kv.wal.Replay(
		func(key, val string) {
			kv.memTable.Put([]byte(key), []byte(val))
		},
		func(key string) {
			kv.memTable.Put([]byte(key), nil)
		},
	)
	return err
}

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
			panic(fmt.Sprintf("SSTable %s is corrupted: bad magic number", f.Name()))
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
		file.Close()

		if len(index) == 0 {
			continue
		}

		meta := &SSTableMeta{
			FileID: fileID,
			MinKey: index[0].MaxKey,
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

	kv.sstables = loadedMetas
	return nil
}

func (kv *KVStore) bumpNextFileID(fileID uint64) {
	if fileID >= kv.nextFileID {
		kv.nextFileID = fileID + 1
	}
}

// ================= 后台引擎：Compaction (预留) =================

// doCompaction 预留给 Week 4 的后台合并流程。
func (kv *KVStore) doCompaction() error {
	return nil
}
