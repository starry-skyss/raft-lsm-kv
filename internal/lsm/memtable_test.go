// 文件路径: internal/lsm/memtable_test.go
package lsm

import (
	"path/filepath"
	"raft-lsm-kv/internal/wal"
	"strconv"
	"sync"
	"testing"
)

// 测试基础的增删改查逻辑
func TestKVStore_Basic(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "test_wal.log")

	w, err := wal.OpenWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer w.Close()

	// 适配修改 1：传入 tempDir 作为引擎数据目录
	kv := NewDB(w, tempDir)

	err = kv.Put("key1", "value1")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	val, exists := kv.Get("key1")
	if !exists || val != "value1" {
		t.Fatalf("Expected value1, got %v (exists: %v)", val, exists)
	}

	_, exists = kv.Get("not_exist")
	if exists {
		t.Fatalf("Expected key to not exist")
	}

	err = kv.Delete("key1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, exists = kv.Get("key1")
	if exists {
		t.Fatalf("Expected key1 to be deleted")
	}
}

// 测试高并发下的读写安全性 (必须配合 -race 运行)
func TestKVStore_Concurrent(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "test_concurrent_wal.log")

	w, err := wal.OpenWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer w.Close()

	kv := NewDB(w, tempDir)
	var wg sync.WaitGroup

	workers := 100 // 启动 100 个并发写和 100 个并发读

	// 并发写
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key_" + strconv.Itoa(n)
			val := "val_" + strconv.Itoa(n)
			_ = kv.Put(key, val)
		}(i)
	}

	// 并发读
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key_" + strconv.Itoa(n)
			_, _ = kv.Get(key)
		}(i)
	}

	wg.Wait() // 等待所有 goroutine 执行完毕

	// 适配修改 2：不要用底层的结构查长度，因为部分数据可能已经被 Flush 到磁盘了！
	// 最靠谱的测试方法是：遍历所有写入的 key，看能不能成功 Get 出来。
	for i := 0; i < workers; i++ {
		key := "key_" + strconv.Itoa(i)
		expected := "val_" + strconv.Itoa(i)

		val, exists := kv.Get(key)
		if !exists || val != expected {
			t.Fatalf("Concurrent Write/Read failed for %s: got %s, want %s", key, val, expected)
		}
	}
}

// 测试崩溃恢复链路：写入 -> 关闭 -> 重启 -> 读取
func TestKVStore_Recovery(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "test_recovery_wal.log")

	// ==========================================
	// 阶段 1：模拟系统初次启动，写入数据后崩溃
	// ==========================================
	w1, err := wal.OpenWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}

	kv1 := NewDB(w1, tempDir)
	_ = kv1.Put("recover_key1", "val1")
	_ = kv1.Put("recover_key2", "val2")
	_ = kv1.Delete("recover_key1") // 删除 key1，留下 key2

	w1.Close()

	// ==========================================
	// 阶段 2：模拟系统重启，重放 WAL
	// ==========================================
	w2, err := wal.OpenWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer w2.Close()

	// 实例化全新内存表，共用同一个数据目录
	kv2 := NewDB(w2, tempDir)

	err = kv2.RecoverFromWAL()
	if err != nil {
		t.Fatalf("Recovery failed: %v", err)
	}

	// ==========================================
	// 阶段 3：验证数据正确性
	// ==========================================
	if _, exists := kv2.Get("recover_key1"); exists {
		t.Fatalf("Expected recover_key1 to be deleted")
	}

	val, exists := kv2.Get("recover_key2")
	if !exists || val != "val2" {
		t.Fatalf("Expected recover_key2=val2, got %v", val)
	}
}
