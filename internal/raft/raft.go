package raft

// The file ../raftapi/raftapi.go defines the interface that raft must
// expose to servers (or the tester), but see comments below for each
// of these functions for more details.
//
// In addition,  Make() creates a new raft peer that implements the
// raft interface.

import (
	"bytes"

	"math/rand"
	"sync"
	"time"

	"raft-lsm-kv/internal/raft/labgob"
	"raft-lsm-kv/internal/raft/labrpc"
	"raft-lsm-kv/internal/raft/raftapi"
)

type NodeState int

const (
	StateFollower  NodeState = iota // 0
	StateCandidate                  // 1
	StateLeader                     // 2
)

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]

	currentTerm     int
	votedFor        int
	state           NodeState
	lastHeartbeat   time.Time
	electionTimeout time.Duration

	log         []LogEntry
	commitIndex int //已经提交的日志的索引,即已经被大多数服务器复制的日志的索引,但还没有被应用到状态机的日志的索引,>=lastApplied,已记录
	lastApplied int //已经应用到状态机的日志的索引,即已经被应用到状态机的日志的索引,但还没有被大多数服务器复制的日志的索引,<=commitIndex,已执行

	nextIndex  []int //准备发送给这个 Follower 的下一条日志的 Index
	matchIndex []int
	applyCh    chan raftapi.ApplyMsg

	leaderChangeTotal        uint64
	electionStartedTotal     uint64
	termChangeTotal          uint64
	appendEntriesFailedTotal uint64
	raftPersistCount         uint64
	raftPersistTotalNs       int64
	raftPersistMaxNs         int64

	// Your data here (3A, 3B, 3C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

}

// 变量名首字母必须大写才能通过 RPC 传输
type AppendEntriesArgs struct {
	Term         int
	LeaderID     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool

	XTerm  int // for optimization: 失败时，返回冲突日志的任期
	XIndex int // for optimization: 失败时，返回冲突日志的第一个索引
	XLen   int // for optimization: 失败时，返回日志的总长度
}

type LogEntry struct {
	Term    int
	Command interface{}
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	// var term int
	// var isleader bool
	// // Your code here (3A).
	// return term, isleader
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.currentTerm, rf.state == StateLeader
}

func (rf *Raft) setTermLocked(term int) {
	if term == rf.currentTerm {
		return
	}
	rf.currentTerm = term
	rf.termChangeTotal++
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	start := time.Now()
	// Your code here (3C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
	w := new(bytes.Buffer)
	encoder := labgob.NewEncoder(w)
	encoder.Encode(rf.currentTerm)
	encoder.Encode(rf.votedFor)
	encoder.Encode(rf.log)
	rf.persister.Save(w.Bytes(), nil)

	elapsed := time.Since(start).Nanoseconds()
	rf.raftPersistCount++
	rf.raftPersistTotalNs += elapsed
	if elapsed > rf.raftPersistMaxNs {
		rf.raftPersistMaxNs = elapsed
	}
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
	r := bytes.NewBuffer(data)
	decoder := labgob.NewDecoder(r)
	var currentTerm int
	var votedFor int
	var log []LogEntry
	if decoder.Decode(&currentTerm) != nil ||
		decoder.Decode(&votedFor) != nil ||
		decoder.Decode(&log) != nil {
		// 读取失败，可能是数据损坏或者版本不兼容，直接返回不修改状态
		return
	} else {
		rf.currentTerm = currentTerm
		rf.votedFor = votedFor
		rf.log = log
	}
}

// how many bytes in Raft's persisted log?
func (rf *Raft) PersistBytes() int {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.persister.RaftStateSize()
}

func (rf *Raft) DebugMetrics() raftapi.RaftMetrics {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	return raftapi.RaftMetrics{
		NodeID:                   rf.me,
		IsLeader:                 rf.state == StateLeader,
		CurrentTerm:              rf.currentTerm,
		LeaderChangeTotal:        rf.leaderChangeTotal,
		ElectionStartedTotal:     rf.electionStartedTotal,
		TermChangeTotal:          rf.termChangeTotal,
		AppendEntriesFailedTotal: rf.appendEntriesFailedTotal,
		RaftPersistCount:         rf.raftPersistCount,
		RaftPersistTotalMS:       float64(rf.raftPersistTotalNs) / float64(time.Millisecond),
		RaftPersistMaxMS:         float64(rf.raftPersistMaxNs) / float64(time.Millisecond),
		CommitIndex:              rf.commitIndex,
		LastApplied:              rf.lastApplied,
		LogLength:                len(rf.log),
	}
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (3A, 3B).
	Term        int // 候选人的当前任期
	CandidateId int // 候选人的 ID

	LastLogIndex int // 候选人最后一条日志的索引
	LastLogTerm  int // 候选人最后一条日志的任期
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (3A).
	Term        int  // 投票人的当前任期
	VoteGranted bool //投票结果 (true 代表赞成，false 代表拒绝)
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (3A, 3B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}
	if args.Term > rf.currentTerm {
		rf.setTermLocked(args.Term)
		rf.state = StateFollower
		rf.votedFor = -1
		rf.persist()
	}

	// 比较日志新旧：
	lastLogIndex := len(rf.log) - 1
	lastLogTerm := rf.log[lastLogIndex].Term

	// 如果 Candidate 的日志比我旧，拒绝
	upToDate := false
	if args.LastLogTerm > lastLogTerm {
		upToDate = true
	} else if args.LastLogTerm == lastLogTerm && args.LastLogIndex >= lastLogIndex {
		upToDate = true
	}

	if !upToDate {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}

	if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
		rf.votedFor = args.CandidateId
		reply.Term = rf.currentTerm
		reply.VoteGranted = true
		rf.resetElectionTimeout()
		rf.persist()
		return
	}
	reply.Term = rf.currentTerm
	reply.VoteGranted = false
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
// 使用 Raft 的服务（例如 k/v 服务器）希望就下一条要追加到 Raft 日志中的命令发起一致性协议。
// 如果该服务器不是 Leader，则返回 false；否则发起一致性过程并立即返回。
// 不能保证这条命令最终一定会被提交到 Raft 日志中，因为 Leader 可能会故障或在选举中失去领导权。
//
// 第一个返回值是这条命令将出现的日志索引（如果它最终被提交）。
// 第二个返回值是当前任期。
// 第三个返回值表示该服务器是否认为自己是 Leader。

func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	isleader := (rf.state == StateLeader)
	if !isleader {
		return -1, -1, false
	}

	rf.log = append(rf.log, LogEntry{Term: rf.currentTerm, Command: command})
	rf.persist() // 追加日志后立即持久化，确保不会丢失
	index := len(rf.log) - 1
	term := rf.currentTerm
	go rf.broadcastHeartbeats() // 立即向所有 Follower 发送心跳，尝试提交日志
	return index, term, true

	// Your code here (3B).

}

func (rf *Raft) ticker() {
	for true {

		// Your code here (3A)
		// Check if a leader election should be started.

		// pause for a random amount of time between 50 and 350
		// milliseconds.
		// 每次只睡一小会儿，确保不会错过关键的时间点
		time.Sleep(10 * time.Millisecond)

		rf.mu.Lock()
		if rf.state != StateLeader {
			elapsed := time.Since(rf.lastHeartbeat)
			// 当超时时间真正达到阈值时才触发选举
			if elapsed >= rf.electionTimeout {
				rf.startElection()
				// 重置超时时间，防止脑裂和短时间重复选举
				rf.resetElectionTimeout()
			}
		}
		rf.mu.Unlock()
	}
}

func (rf *Raft) resetElectionTimeout() {
	// 随机生成一个 150-300ms 的选举超时时间
	ms := 300 + (rand.Int63() % 200)
	rf.electionTimeout = time.Duration(ms) * time.Millisecond
	rf.lastHeartbeat = time.Now()
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan raftapi.ApplyMsg) raftapi.Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.state = StateFollower
	rf.resetElectionTimeout()
	rf.log = make([]LogEntry, 0)
	rf.log = append(rf.log, LogEntry{Term: 0, Command: nil})
	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.nextIndex = make([]int, len(peers))
	rf.matchIndex = make([]int, len(peers))
	rf.applyCh = applyCh

	// Your initialization code here (3A, 3B, 3C).

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()
	go rf.heartbeatTicker()
	go rf.applyLoop(applyCh)
	return rf
}

// tricker已加锁
func (rf *Raft) startElection() {
	rf.currentTerm++
	rf.termChangeTotal++
	rf.electionStartedTotal++
	rf.state = StateCandidate
	rf.votedFor = rf.me
	rf.resetElectionTimeout()
	rf.persist()
	args := RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateId:  rf.me,
		LastLogIndex: len(rf.log) - 1,
		LastLogTerm:  rf.log[len(rf.log)-1].Term,
	}
	votecount := 1
	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		go func(peer int) {
			var reply RequestVoteReply
			ok := rf.sendRequestVote(peer, &args, &reply)
			if ok {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				if rf.state != StateCandidate || rf.currentTerm != args.Term {
					return
				}
				if reply.Term > rf.currentTerm {
					rf.setTermLocked(reply.Term)
					rf.state = StateFollower
					rf.votedFor = -1
					rf.persist()
					return
				}
				if reply.VoteGranted {
					votecount++
					if votecount > len(rf.peers)/2 {
						if rf.state == StateCandidate {
							rf.state = StateLeader
							rf.leaderChangeTotal++
							for i := range rf.peers {
								rf.nextIndex[i] = len(rf.log)
							}
							go rf.broadcastHeartbeats()
						}
					}
				}
			}
		}(i)
	}
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.currentTerm
	reply.Success = false
	needPersist := false

	defer func() {
		if !reply.Success {
			rf.appendEntriesFailedTotal++
		}
		if needPersist {
			rf.persist()
		}
	}()

	if args.Term < rf.currentTerm {
		return
	}
	if args.Term > rf.currentTerm {
		rf.setTermLocked(args.Term)
		reply.Term = rf.currentTerm
		rf.votedFor = -1
		needPersist = true
	}
	rf.state = StateFollower
	rf.resetElectionTimeout()

	lastLogIndex := len(rf.log) - 1
	// 1. Follower 根本就没有 PrevLogIndex 这么长的日志
	if lastLogIndex < args.PrevLogIndex {
		reply.XTerm = -1
		reply.XIndex = -1
		reply.XLen = len(rf.log) // 告诉 Leader 我的真实长度
		return
	}

	// 2. 长度够，但是在 PrevLogIndex 处的 Term 冲突了
	if rf.log[args.PrevLogIndex].Term != args.PrevLogTerm {
		reply.XTerm = rf.log[args.PrevLogIndex].Term
		reply.XIndex = args.PrevLogIndex

		// 往前找，找到属于 XTerm 这个任期的第一条日志
		for reply.XIndex > 0 && rf.log[reply.XIndex-1].Term == reply.XTerm {
			reply.XIndex--
		}
		return
	}

	reply.Success = true
	for i, entry := range args.Entries {
		logIndex := args.PrevLogIndex + 1 + i
		// 如果超出了当前日志长度，或者在同位置 Term 不匹配
		if logIndex >= len(rf.log) || rf.log[logIndex].Term != entry.Term {
			// 从冲突点截断，并将从 i 开始的所有新 entry 一次性追加
			rf.log = rf.log[:logIndex]
			rf.log = append(rf.log, args.Entries[i:]...)
			needPersist = true
			break // 截断+追加完毕，不用再循环了
		}
	}
	if args.LeaderCommit > rf.commitIndex {
		lastNewEntryIndex := args.PrevLogIndex + len(args.Entries)
		rf.commitIndex = min(args.LeaderCommit, lastNewEntryIndex)
	}
}

func (rf *Raft) broadcastHeartbeats() {
	rf.mu.Lock()
	// 如果自己已经不是 Leader 了，直接退出
	if rf.state != StateLeader {
		rf.mu.Unlock()
		return
	}
	rf.mu.Unlock()
	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		go func(peer int) {
			for {
				rf.mu.Lock()
				if rf.state != StateLeader {
					rf.mu.Unlock()
					return
				}
				// 组装心跳包
				sendLen := len(rf.log) - rf.nextIndex[peer]
				if sendLen < 0 {
					sendLen = 0
				}
				entriesToCopy := make([]LogEntry, sendLen)
				copy(entriesToCopy, rf.log[rf.nextIndex[peer]:])
				args := AppendEntriesArgs{
					Term:         rf.currentTerm,
					LeaderID:     rf.me,
					Entries:      entriesToCopy,
					LeaderCommit: rf.commitIndex,
					PrevLogIndex: rf.nextIndex[peer] - 1,
					PrevLogTerm:  rf.log[rf.nextIndex[peer]-1].Term,
				}
				rf.mu.Unlock()
				var reply AppendEntriesReply
				ok := rf.sendAppendEntries(peer, &args, &reply)
				if ok {
					rf.mu.Lock()

					// 检查过期 RPC
					if rf.state != StateLeader || rf.currentTerm != args.Term {
						rf.mu.Unlock()
						return
					}

					// 如果发现有人的任期比自己大（比如遇到了刚苏醒的新 Leader）
					// 自己必须立刻退位变成 Follower
					if reply.Term > rf.currentTerm {
						rf.setTermLocked(reply.Term)
						rf.state = StateFollower
						rf.resetElectionTimeout()
						rf.votedFor = -1
						rf.persist()
						rf.mu.Unlock()
						return
					}
					if reply.Success {
						newNext := args.PrevLogIndex + len(args.Entries) + 1
						newMatch := args.PrevLogIndex + len(args.Entries)

						// 防御乱序 RPC 导致的状态倒退
						if newNext > rf.nextIndex[peer] {
							rf.nextIndex[peer] = newNext
						}
						if newMatch > rf.matchIndex[peer] {
							rf.matchIndex[peer] = newMatch
						}
						// 更新 commitIndex
						for N := len(rf.log) - 1; N > rf.commitIndex; N-- {
							count := 1 // 包括自己
							for j := range rf.peers {
								if j != rf.me && rf.matchIndex[j] >= N && rf.log[N].Term == rf.currentTerm {
									count++
								}
							}
							if count > len(rf.peers)/2 {
								rf.commitIndex = N
								break
							}
						}
						rf.mu.Unlock()
						break
					} else {
						if reply.XTerm != -1 {
							// Case 2 & 3: 发生了 Term 冲突
							// Leader 尝试在自己的日志里找找有没有 XTerm
							lastMatchedIndex := -1
							for i := len(rf.log) - 1; i > 0; i-- {
								if rf.log[i].Term == reply.XTerm {
									lastMatchedIndex = i
									break
								}
							}

							if lastMatchedIndex != -1 {
								// Case 2: Leader 有这个 Term，跳到该 Term 的最后一条日志的后一个位置
								rf.nextIndex[peer] = lastMatchedIndex + 1
							} else {
								// Case 3: Leader 根本没有这个 Term，直接跳过 Follower 的整个冲突 Term
								rf.nextIndex[peer] = reply.XIndex
							}
						} else {
							// 没有任期信息，只能一个一个回退了
							rf.nextIndex[peer] = reply.XLen
						}
						if rf.nextIndex[peer] < 1 {
							rf.nextIndex[peer] = 1
						}
						rf.mu.Unlock()
					}

				} else {
					rf.mu.Lock()
					rf.appendEntriesFailedTotal++
					rf.mu.Unlock()
					return
				}
			}
		}(i)
	}
}

func (rf *Raft) heartbeatTicker() {
	for {
		// 休眠 100 毫秒（满足每秒不超过 10 次心跳的要求）
		time.Sleep(100 * time.Millisecond)

		rf.mu.Lock()
		isLeader := (rf.state == StateLeader)
		rf.mu.Unlock()

		if isLeader {
			rf.broadcastHeartbeats()
		}
	}
}

func (rf *Raft) applyLoop(applyCh chan raftapi.ApplyMsg) {
	for {
		time.Sleep(10 * time.Millisecond) // 避免频繁检查
		var msgs []raftapi.ApplyMsg
		rf.mu.Lock()
		for rf.commitIndex > rf.lastApplied {
			rf.lastApplied++
			applyMsg := raftapi.ApplyMsg{
				CommandValid: true,
				Command:      rf.log[rf.lastApplied].Command,
				CommandIndex: rf.lastApplied,
			}
			msgs = append(msgs, applyMsg)
		}
		rf.mu.Unlock()

		for _, msg := range msgs {
			applyCh <- msg
		}
	}
}
