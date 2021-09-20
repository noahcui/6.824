package raft

//helper functions
//clean up channels
func (rf *Raft) cleanupCh() {
	rf.win = make(chan int)
	rf.beat = make(chan int)
	rf.giveup = make(chan int)
	rf.vote = make(chan int)
	// rf.beatagain = make(chan int)
	// rf.beatnow = make(chan int)
	rf.reset = make(chan int)
	rf.tryapply = make(chan int)
}

//stitch to leader
func (rf *Raft) tol() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// just checking any errors can happen. not sure if it actually works.
	if rf.state != CANDIDATE {
		return
	}
	rf.state = LEADER
	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))
	for i := range rf.peers {
		rf.nextIndex[i] = rf.getLastindex() + 1
		rf.matchIndex[i] = 0
	}
	rf.send(rf.giveup, 1)
	rf.sendbeats()
}

//switch to follower
func (rf *Raft) tof(term int) {
	state := rf.state
	rf.currentTerm = term
	rf.state = FOLLOWER
	rf.votedFor = -1
	if state != FOLLOWER {
		rf.send(rf.giveup, 1)
	}
}

//switch to canadiate
func (rf *Raft) toc(state int) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// Do nothing if the state is already changed. avoid conflicts.
	if rf.state != state {
		return
	}
	rf.state = CANDIDATE
	rf.votedFor = rf.me
	rf.vc = 1
	rf.currentTerm++
	rf.persist()
	rf.send(rf.giveup, 1)
	rf.election()
}

func (rf *Raft) getLastindex() int {
	return len(rf.log) - 1 + rf.lastIncludedIndex
}
func (rf *Raft) getLogindex(idx int) int {
	return idx - rf.lastIncludedIndex
}

func (rf *Raft) getterm(idx int) int {
	return rf.log[rf.getLogindex(idx)].Term
}

//avoid blocking when disconnected.
func (rf *Raft) send(ch chan int, value int) {
	select {
	case ch <- value:
	default:
	}
}

//avoid blocking when disconnected.
// will create too much goroutines if race mode is on.
func (rf *Raft) sendgo(ch chan int, value int) {
	go func() {
		ch <- value
	}()
}
