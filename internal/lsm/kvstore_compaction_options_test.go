package lsm

import (
	"fmt"
	"strings"
	"testing"
)

func TestCompactionCanBeDisabledWhileFlushContinues(t *testing.T) {
	rootDir := t.TempDir()
	db := NewDBWithOptions(rootDir, Options{EnableCompaction: false})
	defer closeDBForTest(t, db)

	writeUntilSSTableCountForCompactionTest(t, db, rootDir, 4)

	metrics := db.DebugMetrics()
	if metrics.CompactionEnabled {
		t.Fatalf("compaction should be disabled")
	}
	if metrics.SSTableCount < 4 {
		t.Fatalf("expected flushed SSTables to remain, got %d", metrics.SSTableCount)
	}
	if metrics.CompactionTotal != 0 {
		t.Fatalf("background compaction should not run, got compaction_total=%d", metrics.CompactionTotal)
	}
}

func TestManualCompactionRecordsMetrics(t *testing.T) {
	rootDir := t.TempDir()
	db := NewDBWithOptions(rootDir, Options{EnableCompaction: false})
	defer closeDBForTest(t, db)

	writeUntilSSTableCountForCompactionTest(t, db, rootDir, 4)
	before := db.DebugMetrics()
	if before.SSTableCount < 4 {
		t.Fatalf("expected at least 4 SSTables before compaction, got %d", before.SSTableCount)
	}

	if err := db.doCompaction(); err != nil {
		t.Fatalf("manual compaction failed: %v", err)
	}

	after := db.DebugMetrics()
	if after.CompactionTotal != before.CompactionTotal+1 {
		t.Fatalf("expected one recorded compaction, before=%d after=%d", before.CompactionTotal, after.CompactionTotal)
	}
	if after.CompactionInputFilesTotal < before.CompactionInputFilesTotal+4 {
		t.Fatalf("expected at least 4 input files, before=%d after=%d", before.CompactionInputFilesTotal, after.CompactionInputFilesTotal)
	}
	if after.CompactionOutputFilesTotal < before.CompactionOutputFilesTotal+1 {
		t.Fatalf("expected one output file, before=%d after=%d", before.CompactionOutputFilesTotal, after.CompactionOutputFilesTotal)
	}
	if after.CompactionLastMS <= 0 || after.CompactionMaxMS <= 0 {
		t.Fatalf("expected positive compaction timing, last=%.3f max=%.3f", after.CompactionLastMS, after.CompactionMaxMS)
	}
}

func writeUntilSSTableCountForCompactionTest(t *testing.T, db *DB, rootDir string, want int) {
	t.Helper()
	value := strings.Repeat("c", 512)
	for i := 0; countSSTablesForTest(t, rootDir) < want && i < 1000; i++ {
		if err := db.Put(fmt.Sprintf("compaction_key_%04d", i), value); err != nil {
			t.Fatalf("put compaction key %d: %v", i, err)
		}
		if i%8 == 0 {
			waitForNextWALForTest(t, db)
		}
	}
	waitForSSTablesForTest(t, rootDir, want)
}
