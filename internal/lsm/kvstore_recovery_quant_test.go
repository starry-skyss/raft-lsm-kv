package lsm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const recoveryQuantKeyCount = 1000

type recoveryQuantResult struct {
	Scenario           string
	RunID              int
	WriteKeyCount      int
	OverwriteKeyCount  int
	DeleteKeyCount     int
	VerifyKeyCount     int
	VerifySuccessCount int
	RecoveryMS         float64
	WALFileCountBefore int
	SSTableCountBefore int
	SSTableCountAfter  int
	ManifestLoaded     bool
	Pass               bool
}

func TestQuantitativeRecoveryScenarios(t *testing.T) {
	scenarios := []struct {
		name string
		run  func(t *testing.T, runID int) recoveryQuantResult
	}{
		{name: "wal", run: runQuantWALRecovery},
		{name: "sstable_manifest", run: runQuantSSTableManifestRecovery},
		{name: "tombstone", run: runQuantTombstoneRecovery},
		{name: "mixed", run: runQuantMixedRecovery},
	}

	for _, scenario := range scenarios {
		scenario := scenario
		for runID := 1; runID <= 3; runID++ {
			runID := runID
			t.Run(fmt.Sprintf("%s_run_%d", scenario.name, runID), func(t *testing.T) {
				result := scenario.run(t, runID)
				logRecoveryQuantResult(t, result)
				if !result.Pass {
					t.Fatalf("%s run %d failed: verified %d/%d keys",
						result.Scenario, result.RunID, result.VerifySuccessCount, result.VerifyKeyCount)
				}
			})
		}
	}
}

func runQuantWALRecovery(t *testing.T, runID int) recoveryQuantResult {
	t.Helper()
	rootDir := t.TempDir()
	db := newQuantDBForTest(rootDir)

	expected := make(map[string]string, recoveryQuantKeyCount)
	for i := 0; i < recoveryQuantKeyCount; i++ {
		key := fmt.Sprintf("%03d", i)
		value := "v"
		if err := db.Put(key, value); err != nil {
			t.Fatalf("wal put %s: %v", key, err)
		}
		expected[key] = value
	}

	sstBefore := countSSTablesForTest(t, rootDir)
	if sstBefore != 0 {
		t.Fatalf("wal recovery setup unexpectedly flushed before restart: sstable_count=%d", sstBefore)
	}
	walBefore := countWALFilesForQuantTest(t, rootDir)
	closeDBForTest(t, db)

	reopened, recoveryMS := reopenDBForQuantTest(t, rootDir)
	defer closeDBForTest(t, reopened)

	verifyCount, successCount := verifyExpectedValuesForQuantTest(reopened, expected)
	sstAfter := countSSTablesForTest(t, rootDir)
	manifestLoaded := manifestLoadedForQuantTest(t, rootDir)

	return recoveryQuantResult{
		Scenario:           "wal",
		RunID:              runID,
		WriteKeyCount:      len(expected),
		VerifyKeyCount:     verifyCount,
		VerifySuccessCount: successCount,
		RecoveryMS:         recoveryMS,
		WALFileCountBefore: walBefore,
		SSTableCountBefore: sstBefore,
		SSTableCountAfter:  sstAfter,
		ManifestLoaded:     manifestLoaded,
		Pass:               verifyCount == recoveryQuantKeyCount && successCount == verifyCount,
	}
}

func runQuantSSTableManifestRecovery(t *testing.T, runID int) recoveryQuantResult {
	t.Helper()
	rootDir := t.TempDir()
	db := newQuantDBForTest(rootDir)
	waitForNextWALForTest(t, db)

	expected := make(map[string]string, recoveryQuantKeyCount)
	value := strings.Repeat("s", 64)
	for i := 0; i < recoveryQuantKeyCount; i++ {
		key := fmt.Sprintf("sst_%04d", i)
		if err := db.Put(key, value); err != nil {
			t.Fatalf("sstable put %s: %v", key, err)
		}
		expected[key] = value
	}

	waitForSSTablesForTest(t, rootDir, 1)
	sstBefore := countSSTablesForTest(t, rootDir)
	walBefore := countWALFilesForQuantTest(t, rootDir)
	closeDBForTest(t, db)

	reopened, recoveryMS := reopenDBForQuantTest(t, rootDir)
	defer closeDBForTest(t, reopened)

	verifyCount, successCount := verifyExpectedValuesForQuantTest(reopened, expected)
	sstAfter := countSSTablesForTest(t, rootDir)
	manifestLoaded := manifestLoadedForQuantTest(t, rootDir)

	return recoveryQuantResult{
		Scenario:           "sstable_manifest",
		RunID:              runID,
		WriteKeyCount:      len(expected),
		VerifyKeyCount:     verifyCount,
		VerifySuccessCount: successCount,
		RecoveryMS:         recoveryMS,
		WALFileCountBefore: walBefore,
		SSTableCountBefore: sstBefore,
		SSTableCountAfter:  sstAfter,
		ManifestLoaded:     manifestLoaded,
		Pass:               sstBefore > 0 && manifestLoaded && verifyCount == recoveryQuantKeyCount && successCount == verifyCount,
	}
}

func runQuantTombstoneRecovery(t *testing.T, runID int) recoveryQuantResult {
	t.Helper()
	rootDir := t.TempDir()
	db := newQuantDBForTest(rootDir)

	deleted := make([]string, 0, recoveryQuantKeyCount)
	for i := 0; i < recoveryQuantKeyCount; i++ {
		key := fmt.Sprintf("del_%04d", i)
		if err := db.Put(key, "v1"); err != nil {
			t.Fatalf("tombstone put %s: %v", key, err)
		}
		if err := db.Delete(key); err != nil {
			t.Fatalf("tombstone delete %s: %v", key, err)
		}
		deleted = append(deleted, key)
	}

	sstBefore := countSSTablesForTest(t, rootDir)
	walBefore := countWALFilesForQuantTest(t, rootDir)
	closeDBForTest(t, db)

	reopened, recoveryMS := reopenDBForQuantTest(t, rootDir)
	defer closeDBForTest(t, reopened)

	verifyCount, successCount := verifyDeletedKeysForQuantTest(reopened, deleted)
	sstAfter := countSSTablesForTest(t, rootDir)
	manifestLoaded := manifestLoadedForQuantTest(t, rootDir)

	return recoveryQuantResult{
		Scenario:           "tombstone",
		RunID:              runID,
		WriteKeyCount:      recoveryQuantKeyCount,
		DeleteKeyCount:     recoveryQuantKeyCount,
		VerifyKeyCount:     verifyCount,
		VerifySuccessCount: successCount,
		RecoveryMS:         recoveryMS,
		WALFileCountBefore: walBefore,
		SSTableCountBefore: sstBefore,
		SSTableCountAfter:  sstAfter,
		ManifestLoaded:     manifestLoaded,
		Pass:               verifyCount == recoveryQuantKeyCount && successCount == verifyCount,
	}
}

func runQuantMixedRecovery(t *testing.T, runID int) recoveryQuantResult {
	t.Helper()
	rootDir := t.TempDir()
	db := newQuantDBForTest(rootDir)

	expected := make(map[string]string, recoveryQuantKeyCount)
	deleted := make([]string, 0, 250)
	for i := 0; i < recoveryQuantKeyCount; i++ {
		key := fmt.Sprintf("mix_%04d", i)
		if err := db.Put(key, "v1"); err != nil {
			t.Fatalf("mixed initial put %s: %v", key, err)
		}
		expected[key] = "v1"
	}
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("mix_%04d", i)
		if err := db.Put(key, "v2"); err != nil {
			t.Fatalf("mixed overwrite %s: %v", key, err)
		}
		expected[key] = "v2"
	}
	for i := 500; i < 750; i++ {
		key := fmt.Sprintf("mix_%04d", i)
		if err := db.Delete(key); err != nil {
			t.Fatalf("mixed delete %s: %v", key, err)
		}
		delete(expected, key)
		deleted = append(deleted, key)
	}

	sstBefore := countSSTablesForTest(t, rootDir)
	walBefore := countWALFilesForQuantTest(t, rootDir)
	closeDBForTest(t, db)

	reopened, recoveryMS := reopenDBForQuantTest(t, rootDir)
	defer closeDBForTest(t, reopened)

	valueVerifyCount, valueSuccessCount := verifyExpectedValuesForQuantTest(reopened, expected)
	deleteVerifyCount, deleteSuccessCount := verifyDeletedKeysForQuantTest(reopened, deleted)
	verifyCount := valueVerifyCount + deleteVerifyCount
	successCount := valueSuccessCount + deleteSuccessCount
	sstAfter := countSSTablesForTest(t, rootDir)
	manifestLoaded := manifestLoadedForQuantTest(t, rootDir)

	return recoveryQuantResult{
		Scenario:           "mixed",
		RunID:              runID,
		WriteKeyCount:      recoveryQuantKeyCount,
		OverwriteKeyCount:  500,
		DeleteKeyCount:     len(deleted),
		VerifyKeyCount:     verifyCount,
		VerifySuccessCount: successCount,
		RecoveryMS:         recoveryMS,
		WALFileCountBefore: walBefore,
		SSTableCountBefore: sstBefore,
		SSTableCountAfter:  sstAfter,
		ManifestLoaded:     manifestLoaded,
		Pass:               verifyCount == recoveryQuantKeyCount && successCount == verifyCount,
	}
}

func reopenDBForQuantTest(t *testing.T, rootDir string) (*DB, float64) {
	t.Helper()
	start := time.Now()
	db := newQuantDBForTest(rootDir)
	elapsed := time.Since(start)
	return db, float64(elapsed.Microseconds()) / 1000.0
}

func newQuantDBForTest(rootDir string) *DB {
	db := NewDB(rootDir)
	db.mu.Lock()
	db.compactionThreshold = 1 << 30
	db.mu.Unlock()
	return db
}

func verifyExpectedValuesForQuantTest(db *DB, expected map[string]string) (int, int) {
	verifyCount := 0
	successCount := 0
	for key, want := range expected {
		verifyCount++
		got, ok := db.Get(key)
		if ok && got == want {
			successCount++
		}
	}
	return verifyCount, successCount
}

func verifyDeletedKeysForQuantTest(db *DB, keys []string) (int, int) {
	verifyCount := 0
	successCount := 0
	for _, key := range keys {
		verifyCount++
		if _, ok := db.Get(key); !ok {
			successCount++
		}
	}
	return verifyCount, successCount
}

func countWALFilesForQuantTest(t *testing.T, rootDir string) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(rootDir, "data", "wal"))
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatalf("read wal dir: %v", err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".log" {
			count++
		}
	}
	return count
}

func manifestLoadedForQuantTest(t *testing.T, rootDir string) bool {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(rootDir, "data", "manifest.log"))
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		t.Fatalf("read manifest: %v", err)
	}
	return strings.TrimSpace(string(data)) != ""
}

func logRecoveryQuantResult(t *testing.T, result recoveryQuantResult) {
	t.Helper()
	t.Logf("RECOVERY_QUANT|scenario=%s|run_id=%d|write_key_count=%d|overwrite_key_count=%d|delete_key_count=%d|verify_key_count=%d|verify_success_count=%d|recovery_ms=%.3f|wal_file_count_before=%d|sstable_file_count_before=%d|sstable_file_count_after=%d|manifest_loaded=%t|pass=%t",
		result.Scenario,
		result.RunID,
		result.WriteKeyCount,
		result.OverwriteKeyCount,
		result.DeleteKeyCount,
		result.VerifyKeyCount,
		result.VerifySuccessCount,
		result.RecoveryMS,
		result.WALFileCountBefore,
		result.SSTableCountBefore,
		result.SSTableCountAfter,
		result.ManifestLoaded,
		result.Pass)
}
