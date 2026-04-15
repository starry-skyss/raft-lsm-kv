package raft

//
// support for Raft and kvraft to save persistent
// Raft state (log &c) and k/v server snapshots.
//
// we will use the original persister.go to test your code for grading.
// so, while you can modify this code to help you debug, please
// test with the original before submitting.
//

import (
	"os"
	//"path/filepath"
	"sync"
)

type Persister struct {
	mu        sync.Mutex
	raftstate []byte
	filepath  string //持久化文件路径
	snapshot  []byte
}

func MakePersister(filepath string) *Persister {
	return &Persister{
		filepath: filepath,
	}
}

func clone(orig []byte) []byte {
	x := make([]byte, len(orig))
	copy(x, orig)
	return x
}

func (ps *Persister) Checkpoint() *Persister {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	np := MakePersister(ps.filepath)
	np.raftstate = clone(ps.raftstate)
	np.snapshot = clone(ps.snapshot)
	return np
}

func (ps *Persister) ReadRaftState() []byte {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	// 【新增核心逻辑】：从操作系统的文件里读取
	state, err := os.ReadFile(ps.filepath)
	if err != nil {
		// 正常情况（首次启动）
		if os.IsNotExist(err) {
			return nil
		}
		// 如果是权限不足、磁盘坏道等其他严重错误， panic
		panic("failed to read persisted raft state: " + err.Error())
	}
	ps.raftstate = state
	return clone(ps.raftstate)
}

func (ps *Persister) RaftStateSize() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ps.raftstate)
}

// Save both Raft state and K/V snapshot as a single atomic action,
// to help avoid them getting out of sync.
func (ps *Persister) Save(raftstate []byte, snapshot []byte) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.raftstate = clone(raftstate)
	// 【新增核心逻辑】：把字节数组真正写到操作系统的文件里
	// 注意：为了安全性，工业界通常用“写临时文件+Rename”保证原子性，
	// 但在 MVP 阶段，直接 WriteFile 已经足够。
	err := os.WriteFile(ps.filepath, raftstate, 0644)
	if err != nil {
		panic("failed to persist raft state: " + err.Error())
	}
	ps.snapshot = clone(snapshot)
}

func (ps *Persister) ReadSnapshot() []byte {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return clone(ps.snapshot)
}

func (ps *Persister) SnapshotSize() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ps.snapshot)
}
