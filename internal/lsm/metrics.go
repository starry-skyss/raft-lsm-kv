package lsm

import (
	"sync/atomic"
	"time"
)

type DebugMetrics struct {
	CompactionEnabled           bool    `json:"compaction_enabled"`
	GetTotal                    uint64  `json:"get_total"`
	GetSSTableFilesTouchedTotal uint64  `json:"get_sstable_files_touched_total"`
	GetSSTableFilesTouchedAvg   float64 `json:"get_sstable_files_touched_avg"`
	GetSSTableFilesTouchedMax   uint64  `json:"get_sstable_files_touched_max"`
	CompactionTotal             uint64  `json:"compaction_total"`
	CompactionInputFilesTotal   uint64  `json:"compaction_input_files_total"`
	CompactionOutputFilesTotal  uint64  `json:"compaction_output_files_total"`
	CompactionTotalMS           float64 `json:"compaction_total_ms"`
	CompactionMaxMS             float64 `json:"compaction_max_ms"`
	CompactionLastMS            float64 `json:"compaction_last_ms"`
	CompactionFailedTotal       uint64  `json:"compaction_failed_total"`
	SSTableCount                int     `json:"sstable_count"`
}

type dbMetrics struct {
	getTotal                    atomic.Uint64
	getSSTableFilesTouchedTotal atomic.Uint64
	getSSTableFilesTouchedMax   atomic.Uint64
	compactionTotal             atomic.Uint64
	compactionInputFilesTotal   atomic.Uint64
	compactionOutputFilesTotal  atomic.Uint64
	compactionTotalNS           atomic.Uint64
	compactionMaxNS             atomic.Uint64
	compactionLastNS            atomic.Uint64
	compactionFailedTotal       atomic.Uint64
}

func (db *DB) DebugMetrics() DebugMetrics {
	getTotal := db.metrics.getTotal.Load()
	getTouchedTotal := db.metrics.getSSTableFilesTouchedTotal.Load()
	getTouchedAvg := 0.0
	if getTotal > 0 {
		getTouchedAvg = float64(getTouchedTotal) / float64(getTotal)
	}

	db.mu.RLock()
	enabled := db.enableCompaction
	sstableCount := len(db.sstables)
	db.mu.RUnlock()

	return DebugMetrics{
		CompactionEnabled:           enabled,
		GetTotal:                    getTotal,
		GetSSTableFilesTouchedTotal: getTouchedTotal,
		GetSSTableFilesTouchedAvg:   getTouchedAvg,
		GetSSTableFilesTouchedMax:   db.metrics.getSSTableFilesTouchedMax.Load(),
		CompactionTotal:             db.metrics.compactionTotal.Load(),
		CompactionInputFilesTotal:   db.metrics.compactionInputFilesTotal.Load(),
		CompactionOutputFilesTotal:  db.metrics.compactionOutputFilesTotal.Load(),
		CompactionTotalMS:           nsToMS(db.metrics.compactionTotalNS.Load()),
		CompactionMaxMS:             nsToMS(db.metrics.compactionMaxNS.Load()),
		CompactionLastMS:            nsToMS(db.metrics.compactionLastNS.Load()),
		CompactionFailedTotal:       db.metrics.compactionFailedTotal.Load(),
		SSTableCount:                sstableCount,
	}
}

func (db *DB) recordGetSSTableTouches(touched int) {
	db.metrics.getTotal.Add(1)
	db.metrics.getSSTableFilesTouchedTotal.Add(uint64(touched))
	updateAtomicMax(&db.metrics.getSSTableFilesTouchedMax, uint64(touched))
}

func (db *DB) recordCompaction(inputFiles, outputFiles int, elapsed time.Duration, failed bool) {
	if failed {
		db.metrics.compactionFailedTotal.Add(1)
		return
	}
	elapsedNS := uint64(elapsed.Nanoseconds())
	db.metrics.compactionTotal.Add(1)
	db.metrics.compactionInputFilesTotal.Add(uint64(inputFiles))
	db.metrics.compactionOutputFilesTotal.Add(uint64(outputFiles))
	db.metrics.compactionTotalNS.Add(elapsedNS)
	db.metrics.compactionLastNS.Store(elapsedNS)
	updateAtomicMax(&db.metrics.compactionMaxNS, elapsedNS)
}

func updateAtomicMax(counter *atomic.Uint64, value uint64) {
	for {
		current := counter.Load()
		if value <= current {
			return
		}
		if counter.CompareAndSwap(current, value) {
			return
		}
	}
}

func nsToMS(ns uint64) float64 {
	return float64(ns) / float64(time.Millisecond)
}
