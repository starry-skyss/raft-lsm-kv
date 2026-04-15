package lsm

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func setupTestKVStore(t *testing.T) (*DB, string) {
	t.Helper()

	rootDir := t.TempDir()
	kv := NewDB(rootDir)
	if kv == nil {
		t.Fatal("NewKVStore returned nil")
	}

	t.Cleanup(func() {
		if kv.wal != nil {
			_ = kv.wal.Close()
		}
		if kv.nextwal != nil {
			_ = kv.nextwal.Close()
		}
	})

	return kv, rootDir
}

func waitForCondition(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}

func countSSTFiles(t *testing.T, dir string) int {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read sst dir: %v", err)
	}

	count := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".sst" {
			count++
		}
	}
	return count
}

func TestKVStore_BasicOperations(t *testing.T) {
	kv, _ := setupTestKVStore(t)

	if err := kv.Put("hello", "world"); err != nil {
		t.Fatalf("put failed: %v", err)
	}
	if err := kv.Put("foo", "bar"); err != nil {
		t.Fatalf("put failed: %v", err)
	}

	if val, ok := kv.Get("hello"); !ok || val != "world" {
		t.Fatalf("expected hello=world, got ok=%v val=%q", ok, val)
	}

	if err := kv.Put("hello", "lsm-tree"); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if val, ok := kv.Get("hello"); !ok || val != "lsm-tree" {
		t.Fatalf("expected hello=lsm-tree, got ok=%v val=%q", ok, val)
	}

	if err := kv.Delete("foo"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, ok := kv.Get("foo"); ok {
		t.Fatal("expected foo to be deleted")
	}

	if _, ok := kv.Get("not-exist"); ok {
		t.Fatal("expected missing key to return not found")
	}
}

func TestKVStore_FlushAndSSTableRead(t *testing.T) {
	kv, _ := setupTestKVStore(t)

	for i := 0; i < 256; i++ {
		key := fmt.Sprintf("key_%03d", i)
		val := fmt.Sprintf("value_%03d_with_padding_to_force_flush", i)
		if err := kv.Put(key, val); err != nil {
			t.Fatalf("put %d failed: %v", i, err)
		}
	}

	waitForCondition(t, 3*time.Second, func() bool {
		return countSSTFiles(t, kv.sstDir) > 0
	})

	for i := 0; i < 256; i += 17 {
		key := fmt.Sprintf("key_%03d", i)
		val := fmt.Sprintf("value_%03d_with_padding_to_force_flush", i)
		got, ok := kv.Get(key)
		if !ok || got != val {
			t.Fatalf("expected %s=%s from flushed data, got ok=%v val=%q", key, val, ok, got)
		}
	}

	if count := countSSTFiles(t, kv.sstDir); count == 0 {
		t.Fatal("expected at least one SSTable file to be created")
	}
}

func TestKVStore_CrashAndRecovery(t *testing.T) {
	rootDir := t.TempDir()

	kv1 := NewDB(rootDir)
	if kv1 == nil {
		t.Fatal("NewKVStore returned nil")
	}
	defer func() {
		if kv1.wal != nil {
			_ = kv1.wal.Close()
		}
		if kv1.nextwal != nil {
			_ = kv1.nextwal.Close()
		}
	}()

	for i := 0; i < 220; i++ {
		key := fmt.Sprintf("stable_%03d", i)
		val := fmt.Sprintf("stable_value_%03d_with_padding", i)
		if err := kv1.Put(key, val); err != nil {
			t.Fatalf("initial put failed: %v", err)
		}
	}

	waitForCondition(t, 3*time.Second, func() bool {
		return countSSTFiles(t, kv1.sstDir) > 0
	})

	if err := kv1.Put("recovery_put", "from_wal"); err != nil {
		t.Fatalf("wal put failed: %v", err)
	}
	if err := kv1.Delete("stable_007"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if err := kv1.Put("late_key", "late_value"); err != nil {
		t.Fatalf("late put failed: %v", err)
	}

	kv2 := NewDB(rootDir)
	if kv2 == nil {
		t.Fatal("NewKVStore returned nil on recovery")
	}
	defer func() {
		if kv2.wal != nil {
			_ = kv2.wal.Close()
		}
		if kv2.nextwal != nil {
			_ = kv2.nextwal.Close()
		}
	}()

	if val, ok := kv2.Get("recovery_put"); !ok || val != "from_wal" {
		t.Fatalf("expected recovery_put=from_wal after recovery, got ok=%v val=%q", ok, val)
	}
	if val, ok := kv2.Get("late_key"); !ok || val != "late_value" {
		t.Fatalf("expected late_key=late_value after recovery, got ok=%v val=%q", ok, val)
	}
	if _, ok := kv2.Get("stable_007"); ok {
		t.Fatal("expected stable_007 tombstone to survive recovery")
	}
	if _, ok := kv2.Get("stable_008"); !ok {
		t.Fatal("expected flushed SSTable key to survive recovery")
	}
}

func TestKVStore_HighConcurrencyRace(t *testing.T) {
	kv, _ := setupTestKVStore(t)

	const (
		writerCount     = 24
		readerCount     = 24
		opsPerGoroutine = 300
		keySpace        = 128
	)

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < writerCount; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(1000 + writerID)))
			<-start
			for j := 0; j < opsPerGoroutine; j++ {
				keyIndex := r.Intn(keySpace)
				key := fmt.Sprintf("race_key_%03d", keyIndex)
				if r.Float32() < 0.75 {
					if err := kv.Put(key, fmt.Sprintf("writer_%d_%d", writerID, j)); err != nil {
						t.Errorf("put failed: %v", err)
						return
					}
				} else {
					if err := kv.Delete(key); err != nil {
						t.Errorf("delete failed: %v", err)
						return
					}
				}
			}
		}(i)
	}

	for i := 0; i < readerCount; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(2000 + readerID)))
			<-start
			for j := 0; j < opsPerGoroutine; j++ {
				key := fmt.Sprintf("race_key_%03d", r.Intn(keySpace))
				_, _ = kv.Get(key)
			}
		}(i)
	}

	close(start)
	wg.Wait()

	if len(kv.memTable.pairs) == 0 && len(kv.sstables) == 0 {
		t.Fatal("expected concurrent workload to leave some data or flushed SSTables behind")
	}
}
