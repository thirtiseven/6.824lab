package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"

	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.824/labgob"
	"6.824/labrpc"
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

type LogEntry struct {
	Term    int
	Command interface{}
	Index   int
}

// state of raft: follower, candidate, leader
type State int

const (
	Follower = iota
	Candidate
	Leader
)

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// 2A: Persistent state on all servers
	state       State
	currentTerm int
	votedFor    int
	log         []LogEntry

	commitIndex int
	lastApplied int

	// for leader only
	nextIndex  []int
	matchIndex []int

	nextElectTime time.Time

	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).
	term = rf.currentTerm
	isleader = false
	if rf.votedFor == rf.me {
		isleader = true
	}
	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
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
}

// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	// term, candidateId, lastLogIndex, lastLogTerm
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (2A).
	// term, voteGranted
	Term        int
	VoteGranted bool
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	// Reply false if term < currentTerm
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}
	// If votedFor is null or candidateId, and candidate's log is at
	// least as up-to-date as receiver's log, grant vote
	var upToDate bool = false
	if len(rf.log) == 0 {
		upToDate = true
	} else {
		lastLog := rf.log[len(rf.log)-1]
		if lastLog.Term < args.LastLogTerm {
			upToDate = true
		} else if lastLog.Term == args.LastLogTerm && lastLog.Index <= args.LastLogIndex {
			upToDate = true
		}
	}
	if (rf.votedFor == -1 || rf.votedFor == args.CandidateId) && upToDate {
		reply.Term = rf.currentTerm
		reply.VoteGranted = true
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

type AppendEntriesArgs struct {
	term         int
	LedaderId    int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {

	// heartbeats
	if len(args.Entries) == 0 {
		rf.nextElectTime = time.Now().Add(time.Duration(rand.Intn(150)+150) * time.Millisecond)
		reply.Success = true
		return
	}

	if args.term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}
	// Reply false if log does not contain an entry at prevLogIndex whose term matches prevLogTerm
	if rf.log[args.PrevLogIndex].Term != args.PrevLogTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}
	// If an existing entry conflicts with a new one (same index but different terms),
	// delete the existing entry and all that follow it
	for i := 0; i < len(args.Entries); i++ {
		if rf.log[args.PrevLogIndex+i].Term != args.Entries[i].Term {
			rf.log = rf.log[:args.PrevLogIndex+i]
			break
		}
	}
	// Append any new entries not already in the log
	rf.log = append(rf.log, args.Entries...)
	// If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
	if args.LeaderCommit > rf.commitIndex {
		if args.LeaderCommit < len(rf.log) {
			rf.commitIndex = args.LeaderCommit
		} else {
			rf.commitIndex = len(rf.log) - 1
		}
	}
	reply.Term = rf.currentTerm
	reply.Success = true
}

// send AppendEntries RPC to a server
func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

// broadcast AppendEntries RPCs to all other servers
func (rf *Raft) broadcastAppendEntries() {
	for i := 0; i < len(rf.peers); i++ {
		if i != rf.me {
			var args AppendEntriesArgs
			args.term = rf.currentTerm
			args.LedaderId = rf.me
			args.PrevLogIndex = rf.nextIndex[i] - 1
			args.PrevLogTerm = rf.log[args.PrevLogIndex].Term
			args.Entries = rf.log[rf.nextIndex[i]:]
			args.LeaderCommit = rf.commitIndex
			var reply AppendEntriesReply
			go rf.sendAppendEntries(i, &args, &reply)
			if reply.Term > rf.currentTerm {
				rf.currentTerm = reply.Term
				rf.state = Follower
				rf.votedFor = -1
				return
			}
			if reply.Success {
				rf.nextIndex[i] = len(rf.log)
				rf.matchIndex[i] = len(rf.log) - 1
			} else {
				rf.nextIndex[i]--
			}
		}
	}
}

// send heartbeats to all other servers
func (rf *Raft) broadcastHeartbeats() {
	for !rf.killed() && rf.state == Leader {
		for i := 0; i < len(rf.peers); i++ {
			if i != rf.me {
				// send heartbeats, no entries
				var args AppendEntriesArgs
				args.term = rf.currentTerm
				args.LedaderId = rf.me
				args.PrevLogIndex = rf.nextIndex[i] - 1
				args.PrevLogTerm = rf.log[args.PrevLogIndex].Term
				args.Entries = make([]LogEntry, 0)
				args.LeaderCommit = rf.commitIndex
				var reply AppendEntriesReply
				go rf.sendAppendEntries(i, &args, &reply)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// broadcast AppendEntries RPCs to all other servers
	if rf.state == Leader {
		rf.mu.Lock()
		defer rf.mu.Unlock()
		rf.log = append(rf.log, LogEntry{Term: rf.currentTerm, Command: command, Index: len(rf.log)})
		rf.nextIndex[rf.me] = len(rf.log)
		rf.matchIndex[rf.me] = len(rf.log) - 1
		rf.broadcastAppendEntries()
		index = len(rf.log) - 1
		term = rf.currentTerm
		isLeader = true
	}

	// Your code here (2B).

	return index, term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for rf.killed() == false {

		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep().

		// if election timeout elapses without receiving AppendEntries
		// RPC from current leader or granting vote to candidate:
		// convert to candidate
		if time.Now().After(rf.nextElectTime) {
			rf.hostElection()
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (rf *Raft) hostElection() {
	// print some log
	// println("host election")
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.currentTerm++
	rf.votedFor = rf.me
	rf.state = Candidate
	rf.nextElectTime = time.Now().Add(time.Duration(rand.Intn(150)+150) * time.Millisecond)
	var voteCount int = 1
	for i := 0; i < len(rf.peers); i++ {
		if i != rf.me {
			var args RequestVoteArgs
			args.Term = rf.currentTerm
			args.CandidateId = rf.me
			args.LastLogIndex = rf.log[len(rf.log)-1].Index
			args.LastLogTerm = rf.log[len(rf.log)-1].Term
			var reply RequestVoteReply
			go rf.sendRequestVote(i, &args, &reply)
			if reply.Term > rf.currentTerm {
				rf.currentTerm = reply.Term
				rf.state = Follower
				rf.votedFor = -1
				return
			}
			if reply.VoteGranted {
				voteCount++
			}
		}
	}
	if voteCount > len(rf.peers)/2+1 {
		rf.state = Leader
		rf.nextIndex = make([]int, len(rf.peers))
		rf.matchIndex = make([]int, len(rf.peers))
		for i := 0; i < len(rf.peers); i++ {
			rf.nextIndex[i] = len(rf.log)
			rf.matchIndex[i] = 0
		}
		go rf.broadcastHeartbeats()
	} else {
		rf.state = Follower
		rf.votedFor = -1
	}
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
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	// 2A
	rf.state = Follower
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.log = make([]LogEntry, 0)
	rf.log = append(rf.log, LogEntry{Term: 0, Command: nil})
	rf.commitIndex = 0
	rf.lastApplied = 0

	// 2B

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}
