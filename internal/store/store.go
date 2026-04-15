package store

import (
	"fmt"
	"raft-lsm-kv/internal/lsm"

	//"raft-lsm-kv/internal/raft"
	"raft-lsm-kv/internal/raft/raftapi"
	"sync"
	"time"
)

// 定义具体的操作指令
type Op struct {
	Operation string // "PUT", "DELETE","GET"
	Key       string
	Value     string
	OpId      int64 //防伪码，用于区分重复请求
}

type OpResult struct {
	AppliedOp Op
	Err       error
}

type KVStore struct {
	mu      sync.RWMutex
	db      *lsm.DB
	rf      raftapi.Raft
	applyCh chan raftapi.ApplyMsg

	// 用于阻塞等待 Raft 达成共识的通知管道
	// Key 是 Raft 的 Log Index，Value 是一个 channel，Raft 达成共识后会通过这个 channel 通知等待的 goroutine
	waitChans map[int]chan OpResult // log index -> result channel
}

func NewKVStore(rootDir string, rf raftapi.Raft, applyCh chan raftapi.ApplyMsg) *KVStore {
	kv := &KVStore{
		db:        lsm.NewDB(rootDir),
		rf:        rf,
		applyCh:   applyCh,
		waitChans: make(map[int]chan OpResult),
	}
	go kv.applyLoop()
	return kv
}

func (kv *KVStore) applyLoop() {
	for msg := range kv.applyCh {
		if msg.CommandValid {
			op := msg.Command.(Op)
			var err error
			switch op.Operation {
			case "PUT":
				err = kv.db.Put(op.Key, op.Value)
			case "DELETE":
				err = kv.db.Delete(op.Key)
			}
			kv.mu.Lock()
			if ch, ok := kv.waitChans[msg.CommandIndex]; ok {
				ch <- OpResult{AppliedOp: op, Err: err}
			}
			kv.mu.Unlock()
		}
	}
}

func (kv *KVStore) Put(key, value string) error {
	op := Op{Operation: "PUT", Key: key, Value: value, OpId: time.Now().UnixNano()}
	index, _, isLeader := kv.rf.Start(op)
	if !isLeader {
		return fmt.Errorf("not leader")
	}

	ch := make(chan OpResult, 1)
	kv.mu.Lock()
	kv.waitChans[index] = ch
	kv.mu.Unlock()

	defer func() {
		kv.mu.Lock()
		delete(kv.waitChans, index)
		kv.mu.Unlock()
	}()

	select {
	case result := <-ch:
		// 如果被唤醒时发现 id 变了，说明之前网络分区了，旧 Leader 的日志被新 Leader 覆盖了
		if result.AppliedOp.OpId != op.OpId {
			return fmt.Errorf("leadership changed before commit")
		}
		return result.Err
	case <-time.After(1 * time.Second): // 1秒超时，真实环境可调
		return fmt.Errorf("request timeout")
	}
}

func (kv *KVStore) Delete(key string) error {
	op := Op{Operation: "DELETE", Key: key, OpId: time.Now().UnixNano()}
	index, _, isLeader := kv.rf.Start(op)
	if !isLeader {
		return fmt.Errorf("not leader")
	}

	ch := make(chan OpResult, 1)
	kv.mu.Lock()
	kv.waitChans[index] = ch
	kv.mu.Unlock()

	defer func() {
		kv.mu.Lock()
		delete(kv.waitChans, index)
		kv.mu.Unlock()
	}()

	select {
	case result := <-ch:
		// 如果被唤醒时发现 id 变了，说明之前网络分区了，旧 Leader 的日志被新 Leader 覆盖了
		if result.AppliedOp.OpId != op.OpId {
			return fmt.Errorf("leadership changed before commit")
		}
		return result.Err
	case <-time.After(1 * time.Second): // 1秒超时，真实环境可调
		return fmt.Errorf("request timeout")
	}
}

func (kv *KVStore) Get(key string) (string, bool) {
	return kv.db.Get(key)
}
