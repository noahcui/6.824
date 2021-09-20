package raft

func (rf *Raft) uptodate(index int, term int) bool {
	if term == rf.getterm(rf.getLastindex()) {
		return index >= rf.getLastindex()
	}

	return term > rf.getterm(rf.getLastindex())
}

func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {

	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()
	// if args term is less than currentterm, means request is out of date.
	// send info back and ignore this request
	reply.VoteGranted = false
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		return
	}
	// if args.term > currentterm, update current term and vote for the args server.
	if args.Term > rf.currentTerm {
		rf.tof(args.Term)
	}
	reply.Term = rf.currentTerm
	if rf.state != FOLLOWER {
		return
	}
	if (rf.votedFor == -1 || rf.votedFor == args.CandidateID) && rf.uptodate(args.LastLogIndex, args.LastLogTerm) {
		reply.Term = rf.currentTerm
		rf.votedFor = args.CandidateID
		reply.VoteGranted = true
		// avoid start election after voting. Can help with rising success rate.
		// and makes the logic more clear.
		rf.send(rf.vote, 1)
	}
}

func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {

	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()
	if rf.state != CANDIDATE || args.Term != rf.currentTerm || reply.Term < rf.currentTerm {
		return ok
	}
	if reply.Term > rf.currentTerm {
		rf.tof(reply.Term)
		return ok
	}
	if reply.VoteGranted {
		rf.vc++
		//only when got majority, reduce repeating.
		if rf.vc == len(rf.peers)/2+1 {
			rf.send(rf.win, 1)
		}
	}
	return ok
}

func (rf *Raft) election() {

	if rf.state != CANDIDATE {
		return
	}
	args := RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateID:  rf.me,
		LastLogIndex: rf.getLastindex(),
		LastLogTerm:  rf.getterm(rf.getLastindex()),
	}
	for i := range rf.peers {
		if i != rf.me {
			reply := RequestVoteReply{}
			go rf.sendRequestVote(i, &args, &reply)
		}
	}
}
