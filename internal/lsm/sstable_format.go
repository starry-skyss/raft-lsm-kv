package lsm

// FooterSize = IndexHandle(16 bytes) + MagicNumber(8 bytes).
const FooterSize = 24

// SSTableMagicNumber 是文件的魔数。
// 作用：在打开文件时读取最后 8 字节，如果不是这个数字，说明文件损坏或根本不是我们的 SSTable。
const SSTableMagicNumber uint64 = 0x12345678ABCDEF00

// BlockHandle 指向文件中的一个连续数据区域（可以是 Data Block，也可以是 Index Block）。
// 💡 Trade-off (架构权衡 - 编码方式):
// 真实的 LevelDB/RocksDB 在序列化 BlockHandle 时，会使用变长整数 (Uvarint) 来压缩空间。
// 例如一个只有 4KB 大小的 Block，其 Size 字段用 Varint 可能只占 2 个字节，而用 uint64 永远占 8 字节。
// MVP 阶段为了降低二进制解析的心智负担，并保证定长寻址的精确性，我们选用了固定 8 字节的小端序编码。
type BlockHandle struct {
	Offset uint64 // 数据块在文件中的偏移位置
	Size   uint64 // 数据块的大小
}

// Footer 永远固定在 SSTable 文件的最末尾。
// 💡 Trade-off (架构权衡 - 文件尾部设计):
// 真实的 LevelDB Footer 包含两个 Handle：MetaIndexHandle (指向布隆过滤器) 和 IndexHandle (指向数据索引)。
// 在 MVP 阶段，根据“如无必要，勿增实体”的原则，我们舍弃了 Meta 块，Footer 直接且仅包含 Index Block 的指针。
type Footer struct {
	IndexHandle BlockHandle // 指向 Index Block (数据索引块) 的位置和大小
	MagicNumber uint64      // 文件格式标识，方便未来版本升级
}

// IndexEntry 存在于 Index Block 中，也是加载到内存后的核心索引条目。
type IndexEntry struct {
	MaxKey []byte      // 该数据块中包含的最大 Key，用于二分查找
	Handle BlockHandle // 该数据块的具体位置信息
}

// SSTableMeta 是纯粹的【内存结构】。
// 当系统启动或 Flush 完成后，它常驻内存，代表了一个磁盘上的 SSTable 文件。
// 获取数据时，先查内存里的 SSTableMeta 的范围和索引，真正命中后才去读磁盘。
type SSTableMeta struct {
	FileID uint64       // 比如 1 代表 000001.sst
	MinKey []byte       // 该文件的最小 Key
	MaxKey []byte       // 该文件的最大 Key
	Size   uint64       // 文件大小
	Index  []IndexEntry // 核心：将该文件的完整 Index Block 缓存在内存中，避免读盘
	// Level int // 等 Week 4 做 Compaction 时，我们再给它分层
}
