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
	Value     string
	Exists    bool
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
			var val string
			var exists bool
			switch op.Operation {
			case "PUT":
				err = kv.db.Put(op.Key, op.Value)
			case "DELETE":
				err = kv.db.Delete(op.Key)
			case "GET":
				// GET 操作不修改状态机，只读取当前快照数据
				val, exists = kv.db.Get(op.Key)
			}
			kv.mu.Lock()
			if ch, ok := kv.waitChans[msg.CommandIndex]; ok {
				ch <- OpResult{
					AppliedOp: op,
					Value:     val,
					Exists:    exists,
					Err:       err,
				}
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
	case <-time.After(5 * time.Second): // 5秒超时，真实环境可调
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
	case <-time.After(5 * time.Second): // 5秒超时，真实环境可调
		return fmt.Errorf("request timeout")
	}
}

// Get 修改为强一致性读：通过 Raft 日志同步
func (kv *KVStore) Get(key string) (string, bool, error) {
	op := Op{
		Operation: "GET",
		Key:       key,
		OpId:      time.Now().UnixNano(),
	}

	index, _, isLeader := kv.rf.Start(op)
	if !isLeader {
		return "", false, fmt.Errorf("not leader")
	}

	// 注册通知管道
	ch := make(chan OpResult, 1)
	kv.mu.Lock()
	kv.waitChans[index] = ch
	kv.mu.Unlock()

	// 确保退出时清理管道
	defer func() {
		kv.mu.Lock()
		delete(kv.waitChans, index)
		kv.mu.Unlock()
	}()

	select {
	case result := <-ch:
		// 校验 OpId，防止网络分区导致读取到新 Leader 覆盖后的错误 index 数据
		if result.AppliedOp.OpId != op.OpId {
			return "", false, fmt.Errorf("leadership changed before commit")
		}
		return result.Value, result.Exists, nil

	case <-time.After(5 * time.Second): // 读请求可能稍慢，给予 5 秒超时
		return "", false, fmt.Errorf("request timeout")
	}
}

func (kv *KVStore) ActiveWaitChans() int {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	return len(kv.waitChans)
}

func (kv *KVStore) RaftMetrics() raftapi.RaftMetrics {
	return kv.rf.DebugMetrics()
}
