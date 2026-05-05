package raftapi

// The Raft interface
type Raft interface {
	// Start agreement on a new log entry, and return the log index
	// for that entry, the term, and whether the peer is the leader.
	Start(command interface{}) (int, int, bool)

	// Ask a Raft for its current term, and whether it thinks it is
	// leader
	GetState() (int, bool)

	// For Snaphots (3D)
	Snapshot(index int, snapshot []byte)
	PersistBytes() int

	// DebugMetrics returns a read-only snapshot used by diagnostic tests.
	DebugMetrics() RaftMetrics
}

// RaftMetrics is a lightweight, read-only diagnostics snapshot.
type RaftMetrics struct {
	NodeID                   int     `json:"node_id"`
	IsLeader                 bool    `json:"is_leader"`
	CurrentTerm              int     `json:"current_term"`
	LeaderChangeTotal        uint64  `json:"leader_change_total"`
	ElectionStartedTotal     uint64  `json:"election_started_total"`
	TermChangeTotal          uint64  `json:"term_change_total"`
	AppendEntriesFailedTotal uint64  `json:"append_entries_failed_total"`
	RaftPersistCount         uint64  `json:"raft_persist_count"`
	RaftPersistTotalMS       float64 `json:"raft_persist_total_ms"`
	RaftPersistMaxMS         float64 `json:"raft_persist_max_ms"`
	CommitIndex              int     `json:"commit_index"`
	LastApplied              int     `json:"last_applied"`
	LogLength                int     `json:"log_length"`
}

// As each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the server (or
// tester), via the applyCh passed to Make(). Set CommandValid to true
// to indicate that the ApplyMsg contains a newly committed log entry.
//
// You'll find the Snapshot fields useful later in the lab.
// Exactly one of CommandValid and SnapshotValid should be true.
// 当每个 Raft 副本意识到连续的日志条目已经提交时，它应该通过传给 Make() 的 applyCh 向服务端（或测试程序）发送一个 ApplyMsg。
// 将 CommandValid 设为 true，表示这个 ApplyMsg 包含一条新提交的日志条目。
// 在本实验的后续部分，你会发现 Snapshot 相关字段会很有用。
// CommandValid 和 SnapshotValid 必须且只能有一个为 true。
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}
