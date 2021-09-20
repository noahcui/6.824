package raft

import (
	"bytes"
	"sync"
	"sync/atomic"

	"6.824/labgob"
	"6.824/labrpc"
)

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

func (rf *Raft) persistsnapshot() {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log)
	e.Encode(rf.lastIncludedIndex)
	e.Encode(rf.LastIncludedTerm)
	data := w.Bytes()
	// blindly saving snapshots, this may wast resouses, but makes coding eaiser
	// will optimize this later
	rf.persister.SaveStateAndSnapshot(data, rf.snapshot)
}

func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log)
	e.Encode(rf.lastIncludedIndex)
	e.Encode(rf.LastIncludedTerm)
	data := w.Bytes()
	// blindly saving snapshots, this may wast resouses, but makes coding eaiser
	// will optimize this later
	rf.persister.SaveRaftState(data)
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
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var ct int
	var vf int
	var lg []LogEntry
	var li int
	var lt int
	if d.Decode(&ct) != nil ||
		d.Decode(&vf) != nil ||
		d.Decode(&lg) != nil ||
		d.Decode(&li) != nil ||
		d.Decode(&lt) != nil {
		// fmt.Println("fuckyou")
		return
	} else {
		rf.currentTerm = ct
		rf.votedFor = vf
		rf.log = lg
		rf.lastIncludedIndex = li
		rf.LastIncludedTerm = lt
		rf.commitIndex = li
		rf.lastApplied = li
		snp := rf.persister.ReadSnapshot()
		rf.snapshot = make([]byte, len(snp))
		copy(rf.snapshot, snp)
	}
}

func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	isLeader = rf.state == LEADER
	term = rf.currentTerm
	if isLeader {
		newlog := LogEntry{
			Term:    term,
			Command: command,
		}
		rf.log = append(rf.log, newlog)

		index = rf.getLastindex()
		rf.persist()
	}
	rf.send(rf.reset, 1)
	rf.sendbeats()
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
	LeaderCommit int
}
type AppendEntriesReply struct {
	Term     int
	Success  bool
	NewTerm  int
	NewIndex int
}

//schedule a function to begin election under some cases.

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
	rf.log = append(rf.log, LogEntry{Term: 0})
	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.state = FOLLOWER
	rf.applyCh = applyCh
	rf.vc = 0
	rf.lastIncludedIndex = 0
	rf.LastIncludedTerm = 0
	rf.snapshot = nil
	rf.applywaiter = sync.NewCond(&rf.mu)
	rf.cleanupCh()

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	// channel version, will create too many goroutines and slow the program if race is on
	// go rf.applierhookupchannel()

	go rf.applierhookup()
	go rf.hookup()
	return rf
}
