package lsm

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Iterator 是 compaction 多路归并的统一抽象。
type Iterator interface {
	Next()
	Valid() bool
	Key() []byte
	Value() []byte
	Close()
}

// SSTableIterator 提供对单个 SSTable 的顺序扫描能力（骨架）。
type SSTableIterator struct {
	meta  *SSTableMeta
	dir   string
	key   []byte
	value []byte
	valid bool

	// --- 补全的底层流控制字段 ---
	file   *os.File
	reader io.Reader // 包装了 LimitReader 和 bufio.Reader
}

func NewSSTableIterator(meta *SSTableMeta, dir string) (*SSTableIterator, error) {
	if meta == nil {
		return nil, errors.New("nil sstable meta")
	}

	// TODO(week4): 打开文件并初始化到第一条 KV。
	filename := filepath.Join(dir, fmt.Sprintf("%06d.sst", meta.FileID))
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	// 🌟 核心技巧：计算数据区的准确结束位置
	var dataEndOffset uint64 = 0
	if len(meta.Index) > 0 {
		lastBlock := meta.Index[len(meta.Index)-1]
		dataEndOffset = lastBlock.Handle.Offset + lastBlock.Handle.Size
	}

	// 🌟 核心技巧：LimitReader 防止读穿到 Index Block，bufio 提供极致性能
	limitReader := io.LimitReader(file, int64(dataEndOffset))
	bufferedReader := bufio.NewReader(limitReader)

	it := &SSTableIterator{
		meta:   meta,
		dir:    dir,
		file:   file,
		reader: bufferedReader,
		valid:  false,
	}

	// 立即读取第一条数据，让游标处于 Ready 状态
	it.Next()

	return it, nil
}

func (it *SSTableIterator) Next() {
	lenBuf := make([]byte, 4)

	// 1. 尝试读取 Key 的长度
	// 如果这里碰到 EOF，说明 LimitReader 已经把数据区读完了，正常结束。
	if _, err := io.ReadFull(it.reader, lenBuf); err != nil {
		it.valid = false
		return
	}
	keyLen := binary.LittleEndian.Uint32(lenBuf)

	// 2. 读取 Key 的内容
	it.key = make([]byte, keyLen)
	if _, err := io.ReadFull(it.reader, it.key); err != nil {
		it.valid = false
		return
	}

	// 3. 尝试读取 Value 的长度
	if _, err := io.ReadFull(it.reader, lenBuf); err != nil {
		it.valid = false
		return
	}
	valLen := binary.LittleEndian.Uint32(lenBuf)

	// 4. 读取 Value 的内容或识别墓碑
	if valLen == 0xFFFFFFFF {
		it.value = nil // 这是一个墓碑（删除操作）
	} else {
		it.value = make([]byte, valLen)
		if _, err := io.ReadFull(it.reader, it.value); err != nil {
			it.valid = false
			return
		}
	}

	it.valid = true // 成功读出一条完整数据
}

func (it *SSTableIterator) Valid() bool {
	return it.valid
}

func (it *SSTableIterator) Key() []byte {
	return it.key
}

func (it *SSTableIterator) Value() []byte {
	return it.value
}

func (it *SSTableIterator) Close() {
	// TODO(week4): 关闭文件句柄和关联资源。
	if it.file != nil {
		it.file.Close()
		it.file = nil
	}
	it.valid = false
}
