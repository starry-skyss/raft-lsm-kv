package lsm

import "bytes"

// SSTableBuilder 是 compaction 输出阶段的缓冲构建器（骨架）。
// 在后台执行多路归并（K-way Merge）时，负责暂存清洗后、按序输出的键值对记录。
// 当缓冲满或合并结束时，再将这些记录统一刷盘生成新的 SSTable。
type SSTableBuilder struct {
	pairs []KVPair
}

func NewSSTableBuilder() *SSTableBuilder {
	return &SSTableBuilder{pairs: make([]KVPair, 0, 256)}
}

// Append 接收一个键值对，追加到构建器的缓冲区中。
// 如果新追加的键与缓冲区最后一个键相同，则覆盖最后一个键的值（实现同一键的合并）。
// 注意：这里的覆盖逻辑仅适用于 compaction 过程中同一键的合并，外部调用时请确保按键顺序追加。
func (b *SSTableBuilder) Append(key, value []byte) {
	if len(b.pairs) > 0 && bytes.Equal(b.pairs[len(b.pairs)-1].Key, key) {
		b.pairs[len(b.pairs)-1].Value = append([]byte(nil), value...)
		return
	}

	entry := KVPair{Key: append([]byte(nil), key...)}
	if value != nil {
		entry.Value = append([]byte(nil), value...)
	}
	b.pairs = append(b.pairs, entry)
}

func (b *SSTableBuilder) Len() int {
	return len(b.pairs)
}
