package lsm

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"raft-lsm-kv/internal/wal" // 💡 导师注：确保这里的路径和你的项目一致
	"sync"
	"testing"
	"time"
)

// 辅助函数：初始化测试用的 KVStore
func setupTestKVStore(t *testing.T) (*KVStore, string) {
	// 创建一个隔离的临时目录，测试结束后操作系统会自动清理
	testDir := t.TempDir()

	// 💡 导师注：这里假设你的 WAL 有一个类似 NewWAL 的初始化函数。
	// 如果你的函数签名不同（比如叫 InitWAL），请在这里修改。
	// 为了不让 WAL 报错干扰 LSM 测试，建议你可以在 wal 模块加个 Mock，或者直接用真实的。
	testWal, err := wal.OpenWAL(filepath.Join(testDir, "wal"))
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	kv := NewKVStore(testWal, testDir)
	return kv, testDir
}

// ================= 测试 1：基础增删改查 =================
func TestKVStore_BasicOperations(t *testing.T) {
	kv, _ := setupTestKVStore(t)

	// 1. 测试 Put 和 Get
	kv.Put("hello", "world")
	kv.Put("foo", "bar")

	if val, ok := kv.Get("hello"); !ok || val != "world" {
		t.Errorf("Expected 'world', got '%s'", val)
	}

	// 2. 测试更新 (Update)
	kv.Put("hello", "lsm-tree")
	if val, ok := kv.Get("hello"); !ok || val != "lsm-tree" {
		t.Errorf("Expected 'lsm-tree', got '%s'", val)
	}

	// 3. 测试删除 (Delete/Tombstone)
	kv.Delete("foo")
	if _, ok := kv.Get("foo"); ok {
		t.Errorf("Expected 'foo' to be deleted, but it was found")
	}

	// 4. 测试不存在的 Key
	if _, ok := kv.Get("not-exist"); ok {
		t.Errorf("Expected key not to exist")
	}
}

// ================= 测试 2：触发大批量 Flush 落盘 =================
func TestKVStore_MassiveFlush(t *testing.T) {
	kv, _ := setupTestKVStore(t)

	numEntries := 5000 // 5000 条数据，绝对会突破 4KB 的 MemTable 阈值，触发多次 Flush

	t.Logf("Writing %d entries to trigger flushes...", numEntries)
	for i := 0; i < numEntries; i++ {
		key := fmt.Sprintf("key_%05d", i)
		val := fmt.Sprintf("val_%05d_with_some_padding_data_to_increase_size", i)
		kv.Put(key, val)
	}

	// 导师注：因为 Flush 是在后台 Goroutine 异步跑的，我们需要稍微等一下它落盘完毕
	time.Sleep(2 * time.Second)

	// 验证：数据是否还在？（此时大量数据应该已经躺在磁盘 SSTable 里了）
	t.Logf("Verifying data from SSTables...")
	for i := 0; i < numEntries; i += 500 { // 抽样检查
		key := fmt.Sprintf("key_%05d", i)
		expectedVal := fmt.Sprintf("val_%05d_with_some_padding_data_to_increase_size", i)

		val, ok := kv.Get(key)
		if !ok || val != expectedVal {
			t.Fatalf("Failed to get expected value for %s from SSTable. Got: %s", key, val)
		}
	}

	// 检查磁盘上是否真的生成了 .sst 文件
	files, _ := os.ReadDir(kv.sstDir)
	sstCount := 0
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".sst" {
			sstCount++
		}
	}
	t.Logf("Generated %d SSTable files.", sstCount)
	if sstCount == 0 {
		t.Errorf("Expected SSTables to be generated, but found 0.")
	}
}

// ================= 测试 3：高并发竞态轰炸 (核心测试) =================
func TestKVStore_HighConcurrencyRace(t *testing.T) {
	kv, _ := setupTestKVStore(t)

	var wg sync.WaitGroup
	numWriters := 50        // 50 个并发写协程
	numReaders := 50        // 50 个并发读协程
	opsPerGoroutine := 1000 // 每个协程 1000 次操作
	// 总计：50,000 次随机写，50,000 次随机读

	t.Logf("Starting Concurrent Race Test: %d Writers, %d Readers...", numWriters, numReaders)

	// 启动写协程 (疯狂 Put 和 Delete)
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := fmt.Sprintf("concurrent_key_%d_%d", writerID, j)
				val := fmt.Sprintf("concurrent_val_%d_%d", writerID, j)

				// 90% 概率写入，10% 概率删除
				if rand.Float32() < 0.9 {
					kv.Put(key, val)
				} else {
					kv.Delete(key)
				}
			}
		}(i)
	}

	// 启动读协程 (疯狂 Get 随机数据)
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				// 随机去查一些可能存在、也可能不存在的 key
				targetWriter := rand.Intn(numWriters)
				targetOp := rand.Intn(opsPerGoroutine)
				key := fmt.Sprintf("concurrent_key_%d_%d", targetWriter, targetOp)

				// 这里我们不在乎读到什么，我们在乎的是系统会不会死锁、会不会报 Index Out of Range 或 OOM！
				// 这将极限压榨你的 mu.RLock() 和 ReadAt() 机制！
				kv.Get(key)
			}
		}(i)
	}

	// 等待所有协程执行完毕
	wg.Wait()
	t.Logf("Concurrent Test Finished smoothly without panics!")
}

// ================= 测试 4：崩溃与重启恢复 (Crash & Recovery) =================
func TestKVStore_CrashAndRecovery(t *testing.T) {
	testDir := t.TempDir()
	testWal, _ := wal.OpenWAL(filepath.Join(testDir, "wal"))

	// 1. 启动第一个实例，写入数据
	kv1 := NewKVStore(testWal, testDir)
	kv1.Put("survivor_1", "alive")
	kv1.Put("survivor_2", "alive")

	// 强行写入大量数据触发落盘
	for i := 0; i < 3000; i++ {
		kv1.Put(fmt.Sprintf("dummy_%d", i), "pad")
	}
	time.Sleep(1 * time.Second) // 等待落盘完成

	// 写入内存表还没来得及落盘的数据
	kv1.Put("survivor_3", "in_memtable")

	// 2. 模拟系统崩溃 (这里由于我们不能真杀进程，就让它被 Go GC 掉，不再调用它)
	// 此时：survivor_1/2 在 SSTable 里，survivor_3 在 WAL 和 MemTable 里

	// 3. 重新启动一个新的实例，指向同一个目录！
	t.Logf("Simulating crash and rebooting...")
	testWal2, _ := wal.OpenWAL(filepath.Join(testDir, "wal")) // 重新打开 WAL
	kv2 := NewKVStore(testWal2, testDir)                      // 这一步会触发 loadSSTables 和 RecoverFromWAL

	// 4. 验证数据是否全部生还
	if val, ok := kv2.Get("survivor_1"); !ok || val != "alive" {
		t.Errorf("Failed to recover survivor_1 from SSTable")
	}
	if val, ok := kv2.Get("survivor_3"); !ok || val != "in_memtable" {
		t.Errorf("Failed to recover survivor_3 from WAL")
	}
}
