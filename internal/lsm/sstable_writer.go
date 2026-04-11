package lsm

import (
	"bytes"
	"encoding/binary"
	"os"
)

// 写盘核心逻辑
func writeSSTable(filename string, imm *MemTable) (uint64, []IndexEntry, error) {
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

	for i, pair := range imm.pairs {
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

		if dataBuf.Len() >= 4096 || i == len(imm.pairs)-1 {
			// 写满了或者是最后一个了，先把数据块写到文件里
			if _, err := file.Write(dataBuf.Bytes()); err != nil {
				return writtenBytes, index, err
			}
			writtenBytes += uint64(dataBuf.Len())

			// 记录索引信息：Key 和对应的数据块位置
			index = append(index, IndexEntry{
				MaxKey: pair.Key,
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
