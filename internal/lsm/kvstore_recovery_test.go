package lsm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWALRecoveryPreservesUnflushedWrites(t *testing.T) {
	rootDir := t.TempDir()

	db := NewDB(rootDir)
	if err := db.Put("wal_key_1", "wal_value_1"); err != nil {
		t.Fatalf("put wal_key_1: %v", err)
	}
	if err := db.Put("wal_key_2", "wal_value_2"); err != nil {
		t.Fatalf("put wal_key_2: %v", err)
	}
	sstBefore := countSSTablesForTest(t, rootDir)
	closeDBForTest(t, db)

	reopened := NewDB(rootDir)
	defer closeDBForTest(t, reopened)

	got, ok := reopened.Get("wal_key_1")
	if !ok || got != "wal_value_1" {
		t.Fatalf("wal_key_1 after recovery = (%q, %v), want (wal_value_1, true)", got, ok)
	}
	got, ok = reopened.Get("wal_key_2")
	if !ok || got != "wal_value_2" {
		t.Fatalf("wal_key_2 after recovery = (%q, %v), want (wal_value_2, true)", got, ok)
	}

	sstAfter := countSSTablesForTest(t, rootDir)
	manifestSize := fileSizeForTest(t, filepath.Join(rootDir, "data", "manifest.log"))
	t.Logf("RECOVERY_RESULT|WAL Recovery|data_scale=2 keys|flush_triggered=false before restart, true during recovery bootstrap|sstable_count_before=%d|sstable_count_after=%d|manifest_size=%d|actual=both WAL keys readable after reopen|pass=true",
		sstBefore, sstAfter, manifestSize)
}

func TestSSTableManifestRecoveryLoadsFlushedData(t *testing.T) {
	rootDir := t.TempDir()

	db := NewDB(rootDir)
	waitForNextWALForTest(t, db)

	const totalKeys = 20
	value := strings.Repeat("v", 256)
	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("sst_key_%03d", i)
		if err := db.Put(key, value); err != nil {
			t.Fatalf("put %s: %v", key, err)
		}
	}

	waitForSSTablesForTest(t, rootDir, 1)
	sstBefore := countSSTablesForTest(t, rootDir)
	manifestBefore := fileSizeForTest(t, filepath.Join(rootDir, "data", "manifest.log"))
	closeDBForTest(t, db)

	reopened := NewDB(rootDir)
	defer closeDBForTest(t, reopened)

	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("sst_key_%03d", i)
		got, ok := reopened.Get(key)
		if !ok || got != value {
			t.Fatalf("%s after recovery = (%q, %v), want (%q, true)", key, got, ok, value)
		}
	}

	sstAfter := countSSTablesForTest(t, rootDir)
	manifestAfter := fileSizeForTest(t, filepath.Join(rootDir, "data", "manifest.log"))
	t.Logf("RECOVERY_RESULT|SSTable Manifest Recovery|data_scale=%d keys, value_size=256 bytes|flush_triggered=true|sstable_count_before=%d|sstable_count_after=%d|manifest_size_before=%d|manifest_size_after=%d|actual=flushed and recovered keys readable after reopen|pass=true",
		totalKeys, sstBefore, sstAfter, manifestBefore, manifestAfter)
}

func TestTombstoneRecoveryHidesDeletedValue(t *testing.T) {
	rootDir := t.TempDir()

	db := NewDB(rootDir)
	if err := db.Put("deleted_key", "old_value"); err != nil {
		t.Fatalf("put deleted_key: %v", err)
	}
	if err := db.Delete("deleted_key"); err != nil {
		t.Fatalf("delete deleted_key: %v", err)
	}
	closeDBForTest(t, db)

	reopened := NewDB(rootDir)
	defer closeDBForTest(t, reopened)

	got, ok := reopened.Get("deleted_key")
	if ok {
		t.Fatalf("deleted_key after tombstone recovery = (%q, true), want not found", got)
	}

	sstAfter := countSSTablesForTest(t, rootDir)
	manifestSize := fileSizeForTest(t, filepath.Join(rootDir, "data", "manifest.log"))
	t.Logf("RECOVERY_RESULT|Tombstone Recovery|data_scale=1 put + 1 delete|flush_triggered=true during recovery bootstrap|sstable_count_after=%d|manifest_size=%d|actual=deleted key returns not found after reopen|pass=true",
		sstAfter, manifestSize)
}

func closeDBForTest(t *testing.T, db *DB) {
	t.Helper()
	if db == nil {
		return
	}

	db.mu.Lock()
	activeWAL := db.wal
	nextWAL := db.nextwal
	db.mu.Unlock()

	if activeWAL != nil {
		if err := activeWAL.Close(); err != nil {
			t.Fatalf("close active WAL: %v", err)
		}
	}
	if nextWAL != nil {
		if err := nextWAL.Close(); err != nil {
			t.Fatalf("close preallocated WAL: %v", err)
		}
	}
}

func waitForNextWALForTest(t *testing.T, db *DB) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		db.mu.RLock()
		ready := db.nextwal != nil
		db.mu.RUnlock()
		if ready {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("next WAL was not preallocated before deadline")
}

func waitForSSTablesForTest(t *testing.T, rootDir string, wantAtLeast int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if countSSTablesForTest(t, rootDir) >= wantAtLeast {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("sstable count did not reach %d before deadline", wantAtLeast)
}

func countSSTablesForTest(t *testing.T, rootDir string) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(rootDir, "data", "sst"))
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatalf("read sst dir: %v", err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sst" {
			count++
		}
	}
	return count
}

func fileSizeForTest(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Size()
}
