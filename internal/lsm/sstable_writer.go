package lsm

import (
	"bytes"
	"encoding/binary"
	"os"
)

// writeSSTable (供 Flush 调用)
// 适配层：仅仅是把 MemTable 拆解成 KVPair 切片，传给底层核心函数
func writeSSTableFromMem(filename string, imm *MemTable) (uint64, []IndexEntry, error) {
	return writeSSTable(filename, imm.pairs)
}

// writeSSTableFromPairs (供 Compaction 调用)
// 适配层：同样只是一个皮包函数，方便不同上下文调用
func writeSSTableFromPairs(filename string, pairs []KVPair) (uint64, []IndexEntry, error) {
	return writeSSTable(filename, pairs)
}

// 写盘核心逻辑
func writeSSTable(filename string, pairs []KVPair) (uint64, []IndexEntry, error) {
	// 💡 Trade-off (架构权衡 - 磁盘 I/O 模式):
	// 此处使用了标准的 os.Create 和 file.Write，依赖操作系统的 Page Cache 来做写缓冲。
	// 工业界极致优化 (如 RocksDB)：会开启 Direct I/O (O_DIRECT) 绕过内核缓存，
	// 配合完全由应用层管理的 Block Cache，以避免内核缓存与应用缓存的“双重缓存”问题。
	// (MVP 阶段利用 OS Cache 是最稳定、代码量最少的做法)
	file, err := os.Create(filename)
	if err != nil {
		return 0, nil, err
	}
	defer file.Close()

	var writtenBytes uint64 = 0 // 计数器
	var dataBuf bytes.Buffer
	var index []IndexEntry
	var currentOffset uint64 = 0

	// 用一个 8 字节的 buffer 复用，避免频繁内存分配
	lenBuf := make([]byte, 8)

	for i, pair := range pairs {
		// 1. 写入 Key 的长度 (uint32)
		binary.LittleEndian.PutUint32(lenBuf[0:4], uint32(len(pair.Key)))
		dataBuf.Write(lenBuf[0:4])

		// 2. 写入 Key 内容
		dataBuf.Write(pair.Key)

		// 3. 处理 Value 长度和内容
		if pair.Value == nil {
			// 墓碑标记：0xFFFFFFFF
			binary.LittleEndian.PutUint32(lenBuf[0:4], 0xFFFFFFFF)
			dataBuf.Write(lenBuf[0:4])
		} else {
			// 正常写入 Value 长度 (uint32)
			binary.LittleEndian.PutUint32(lenBuf[0:4], uint32(len(pair.Value)))
			dataBuf.Write(lenBuf[0:4])

			// 4. 写入 Value 内容
			dataBuf.Write(pair.Value)
		}

		if dataBuf.Len() >= 4096 || i == len(pairs)-1 {
			// 💡 Trade-off (架构权衡 - 数据压缩 Compression):
			// 真实的 LevelDB 在落盘 file.Write 之前，会对 dataBuf 使用 Snappy 等极速算法进行压缩。
			// 因为磁盘 I/O 带宽通常是瓶颈，用少量的 CPU 时间换取磁盘写入量的减小，能极大提升 Flush 吞吐量。
			// MVP 阶段我们存入裸数据 (Raw Bytes)，方便通过 Hex 编辑器直接查看文件进行 Debug。
			// 写满了或者是最后一个了，先把数据块写到文件里
			if _, err := file.Write(dataBuf.Bytes()); err != nil {
				return writtenBytes, index, err
			}
			writtenBytes += uint64(dataBuf.Len())

			// 记录索引信息：Key 和对应的数据块位置
			index = append(index, IndexEntry{
				MaxKey: append([]byte(nil), pair.Key...), // 复制一份 Key，避免后续修改
				Handle: BlockHandle{Offset: currentOffset, Size: uint64(dataBuf.Len())},
			})

			currentOffset += uint64(dataBuf.Len())
			dataBuf.Reset() // 清空 buffer，准备写下一批数据
		}
	}

	// 3. 构建并写入 Index Block
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

	binary.LittleEndian.PutUint64(lenBuf[0:8], SSTableMagicNumber)
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
