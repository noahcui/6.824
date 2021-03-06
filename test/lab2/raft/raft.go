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
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"../labrpc"
)

const (
	LEADER    = 0
	FOLLOWER  = 1
	CANDIDATE = 2
)

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
}

type LogEntry struct {
	Term    int
	Command interface{}
}

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
	ticker      *time.Timer
	lasttime    time.Time
	d           time.Duration
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	term = rf.currentTerm
	isleader = rf.state == LEADER
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
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

//
// restore previously persisted state.
//
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

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// if args term is less than currentterm, means request is out of date.
	// send info back and ignore this request
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}
	// if args.term > currentterm, update current term and vote for the args server.
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.votedFor = args.CandidateID
		rf.state = FOLLOWER
		reply.Term = rf.currentTerm
	}
	if rf.state == CANDIDATE && rf.me > args.CandidateID {
		rf.votedFor = args.CandidateID
		rf.state = FOLLOWER
	}
	if rf.votedFor == -1 || rf.votedFor == args.CandidateID {
		reply.Term = rf.currentTerm
		rf.votedFor = args.CandidateID
		reply.VoteGranted = true
		return
	}
	reply.VoteGranted = false
	return

}

//
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
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntriesHandler", args, reply)
	return ok
}

//
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
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

type AppendEntriesArgs struct {
	Term         int
	LeaderID     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
}
type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (rf *Raft) AppendEntriesHandler(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm {
		reply.Success = false
		reply.Term = rf.currentTerm
	} else {
		rf.state = FOLLOWER
		rf.currentTerm = args.Term
		reply.Term = rf.currentTerm
		reply.Success = true
		// rf.ticker.Stop()
		rf.lasttime = time.Now()
		rf.ticker.Reset(rf.d)
	}
}

//leader send heartbeats to others.
func (rf *Raft) heartbeats() {
	ticker := time.NewTicker(150 * time.Millisecond)
	// Dead is dead forever
	for !rf.killed() {
		<-ticker.C
		rf.mu.Lock()
		if rf.state == LEADER {
			//fmt.Printf("abc\n")
			args := AppendEntriesArgs{}

			args.Term = rf.currentTerm
			args.LeaderID = rf.me
			rf.mu.Unlock()

			for i, _ := range rf.peers {
				if i != rf.me {
					//a := time.Now()
					go func(id int) {
						reply := AppendEntriesReply{}
						rf.sendAppendEntries(id, &args, &reply)
						//out of date, which means this is been blocked out and there's a new leader
						rf.mu.Lock()
						if reply.Term > rf.currentTerm {
							rf.currentTerm = reply.Term
							rf.state = FOLLOWER
							rf.votedFor = -1
						}
						rf.mu.Unlock()
					}(i)
					//fmt.Printf("%v\n", time.Since(a))
				}
			}

		} else {
			rf.mu.Unlock()
		}
	}
}

func (rf *Raft) elect() {
	rf.mu.Lock()
	rf.currentTerm++
	rf.state = CANDIDATE
	rf.votedFor = rf.me
	args := RequestVoteArgs{}
	// fmt.Printf("election started\n")
	args.Term = rf.currentTerm
	args.CandidateID = rf.me
	rf.mu.Unlock()
	// var wg sync.WaitGroup
	// var votes []int
	//var w sync.WaitGroup
	votes := 1
	var ctr = make(chan int)
	//var vote [len(rf.peers)]int
	for i, _ := range rf.peers {
		if i != rf.me {
			// wg.Add(1)
			go func(ret chan int, id int) {
				reply := RequestVoteReply{}
				ok := rf.sendRequestVote(id, &args, &reply)
				rf.mu.Lock()
				// if ok && reply.VoteGranted && reply.Term == rf.currentTerm{
				// 	return true
				// }
				// return false
				voted := ok && reply.VoteGranted && reply.Term == rf.currentTerm
				rf.mu.Unlock()
				if voted {

					ret <- 1
				} else {

					ret <- 0
				}

				// wg.Done()
			}(ctr, i)
		}
	}
	// wg.Wait()
	// c := 1

	for i := 0; i < len(rf.peers); i++ {

		vote := <-ctr
		votes += vote
		// in case someone is down, set as leader as soon as it got majority.
		if votes > len(rf.peers)/2 {
			rf.mu.Lock()
			rf.state = LEADER
			rf.mu.Unlock()
			break
		}
	}

}

//schedule a function to begin election under some cases.
func (rf *Raft) election() {
	//d := time.Duration(rand.Float32()*150+200) * time.Millisecond
	//ticker := time.NewTicker(d)
	rf.mu.Lock()
	rf.ticker = time.NewTimer(rf.d)
	rf.mu.Unlock()
	for !rf.killed() {
		<-rf.ticker.C
		go func() {
			// var timeout = make(chan bool)
			rf.mu.Lock()
			if rf.state == CANDIDATE {
				go rf.elect()

			} else if rf.state == FOLLOWER {
				go rf.elect()

			}
			rf.mu.Unlock()
		}()
		rf.ticker.Reset(rf.d)
	}
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.log = make([]LogEntry, 1)
	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.state = FOLLOWER
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	rf.d = time.Duration(rand.Float32()*250+250) * time.Millisecond

	go rf.heartbeats()

	go rf.election()

	return rf
}