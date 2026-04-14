package lsm

import (
	"bytes"
	"container/heap"
)

type mergeItem struct {
	iter   Iterator
	fileID uint64
}

type mergeHeap []mergeItem

func (h mergeHeap) Len() int { return len(h) }

func (h mergeHeap) Less(i, j int) bool {
	cmp := bytes.Compare(h[i].iter.Key(), h[j].iter.Key())
	if cmp != 0 {
		return cmp < 0
	}
	// Key 相同时，FileID 越大代表越新，优先输出。
	return h[i].fileID > h[j].fileID
}

func (h mergeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *mergeHeap) Push(x any) {
	*h = append(*h, x.(mergeItem))
}

func (h *mergeHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

// doMerge 执行 K 路归并的主流程。
// 输入：多个已排序的迭代器（来自待合并的 SSTable）（按逻辑顺序传入，FileID 从大到小）。
// 输出：一个包含合并结果的 SSTableBuilder，供后续刷盘使用。
func doMerge(iters []Iterator) *SSTableBuilder {
	builder := NewSSTableBuilder()

	h := &mergeHeap{}
	heap.Init(h)

	for idx, it := range iters {
		if it == nil || !it.Valid() {
			continue
		}
		heap.Push(h, mergeItem{iter: it, fileID: uint64(idx + 1)})
	}

	var lastKey []byte
	for h.Len() > 0 {
		item := heap.Pop(h).(mergeItem)
		k := append([]byte(nil), item.iter.Key()...)
		v := append([]byte(nil), item.iter.Value()...)

		if len(lastKey) == 0 || !bytes.Equal(lastKey, k) {
			builder.Append(k, v)
			lastKey = k
		}

		item.iter.Next()
		if item.iter.Valid() {
			heap.Push(h, item)
		}
	}

	for _, it := range iters {
		if it != nil {
			it.Close()
		}
	}

	return builder
}
