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
	// 💡 Trade-off (架构权衡 - 文件句柄管理):
	// MVP 阶段每次查询都动态 os.Open 文件，这会产生高昂的 Syscall 开销。
	// 优化方向：在真实的 LSM 引擎中，这里会接入一个 TableCache (LRU 缓存)，
	// 缓存频繁访问的 *os.File 句柄，避免反复打开/关闭文件。
	filename := filepath.Join(dir, fmt.Sprintf("%06d.sst", meta.FileID))
	file, err := os.Open(filename)
	if err != nil {
		return nil, false
	}
	defer file.Close()

	// 3. 把 Data Block 一次性读进内存
	// !!! 亮点：这里使用 ReadAt 而不是 Seek+Read，保证了底层并发读的线程安全性!!!!
	// 💡 Trade-off (架构权衡 - 读盘方式)	:
	// 目前我们直接使用 ReadAt 进行随机访问，简单且线程安全，但每次都要进行一次系统调用。
	// 优化方向：如果访问模式显示出局部性（比如热点数据），可以考虑将整个 SSTable 文件映射到内存 (mmap)，
	// 这样就可以在用户态直接访问数据，极大降低访问延迟。
	blockData := make([]byte, handle.Size)
	if _, err := file.ReadAt(blockData, int64(handle.Offset)); err != nil {
		return nil, false
	}

	// 4. 在内存 buffer 里扫描 KV
	// 💡 Trade-off (架构权衡 - 块内查找):
	// 目前块内采用了 O(K) 的线性扫描。因为 Block 通常较小 (4KB)，此开销极低。
	// 优化方向：LevelDB 在块内写入了 Restart Points，允许在 Data Block 内部再次进行二分查找。
	reader := bytes.NewReader(blockData)
	lenBuf := make([]byte, 4)

	for reader.Len() > 0 {
		if _, err := io.ReadFull(reader, lenBuf); err != nil {
			break
		}
		keyLen := binary.LittleEndian.Uint32(lenBuf)

		keyBuf := make([]byte, keyLen)
		if _, err := io.ReadFull(reader, keyBuf); err != nil {
			break
		}

		if _, err := io.ReadFull(reader, lenBuf); err != nil {
			break
		}
		valLen := binary.LittleEndian.Uint32(lenBuf)

		var valBuf []byte
		if valLen == 0xFFFFFFFF {
			valBuf = nil
		} else {
			valBuf = make([]byte, valLen)
			if _, err := io.ReadFull(reader, valBuf); err != nil {
				break
			}
		}

		// 找到了目标 Key，直接返回（墓碑返回 nil，外部会判断）
		if bytes.Equal(keyBuf, targetKey) {
			return valBuf, true
		}
	}

	return nil, false
}
