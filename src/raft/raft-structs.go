package raft

import (
	"sync"

	"6.824/labrpc"
)

const (
	LEADER    = 0
	FOLLOWER  = 1
	CANDIDATE = 2
)

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	currentTerm int
	votedFor    int
	log         []LogEntry
	commitIndex int
	lastApplied int
	nextIndex   []int
	matchIndex  []int
	state       int
	applyCh     chan ApplyMsg
	//vote count
	vc int

	//helper channels, use bool to save memory.
	// just to make life eaiser, withoout channals there's a bunch of stuffs need to be done
	beat   chan int
	win    chan int
	vote   chan int
	giveup chan int
	// beatagain         chan int
	reset    chan int
	tryapply chan int
	// beatnow           chan int
	applywaiter       *sync.Cond
	LastIncludedIndex int
	LastIncludedTerm  int
	snapshot          []byte
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateID  int
	LastLogIndex int
	LastLogTerm  int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

type SnapshotArg struct {
	Term              int
	LastIncludedIndex int
	LastIncludedTerm  int
	Data              []byte
}

type SnapshotReply struct {
	Term int
	Ok   bool
}

// import "bytes"
// import "../labgob"

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//

type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
	// for snapshots
	SnapshotIndex int
	Snapshot      []byte
	SnapshotValid bool
	SnapshotTerm  int
}

type LogEntry struct {
	Term    int
	Command interface{}
}
