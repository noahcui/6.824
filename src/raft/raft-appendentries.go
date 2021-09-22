package raft

func (rf *Raft) apply() {
	if rf.lastApplied < rf.commitIndex &&
		rf.lastApplied < rf.getLastindex() &&
		rf.lastApplied+1 > rf.LastIncludedIndex {
		for rf.lastApplied < rf.commitIndex {
			rf.lastApplied++
			msg := ApplyMsg{
				CommandValid:  true,
				Command:       rf.log[rf.getLogindex(rf.lastApplied)].Command,
				CommandIndex:  rf.lastApplied,
				SnapshotValid: false,
			}
			rf.mu.Unlock()
			rf.applyCh <- msg
			rf.mu.Lock()
		}
	}
}

func (rf *Raft) tryCommit() {
	// if there exists an N such that N > commitIndex, a majority of
	// matchIndex[i] >= N, and log[N].term == currentTerm, set commitIndex = N

	for n := rf.commitIndex + 1; n <= rf.getLastindex(); n++ {
		c := 1
		if rf.getterm(n) == rf.currentTerm {
			for i := 0; i < len(rf.peers); i++ {
				if i != rf.me && rf.matchIndex[i] >= n {
					c++
				}
			}
		}
		if c > len(rf.peers)/2 {
			rf.commitIndex = n
		}
	}
	if rf.commitIndex > rf.lastApplied {
		// rf.sendgo(rf.tryapply, 1)
		// rf.apply()
		rf.applywaiter.Signal()
	}
}

func (rf *Raft) AppendEntriesHandler(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()
	if args.Term < rf.currentTerm {
		reply.Success = false
		reply.Term = rf.currentTerm
		return
	}
	if args.Term > rf.currentTerm {
		rf.tof(args.Term)
	}
	rf.send(rf.beat, 1)
	reply.Success = false
	reply.Term = rf.currentTerm
	reply.NewIndex = -1
	reply.NewTerm = -1
	if args.PrevLogIndex < rf.LastIncludedIndex {
		return
	}
	// check if there's no enty at previndex at all
	if args.PrevLogIndex > rf.getLastindex() {
		reply.NewIndex = rf.getLastindex() + 1
		return
	}
	//Reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm
	if rf.getterm(args.PrevLogIndex) != args.PrevLogTerm {
		if cfTerm := rf.getterm(args.PrevLogIndex); cfTerm != args.PrevLogTerm {
			reply.NewTerm = cfTerm
			for i := args.PrevLogIndex; i >= rf.LastIncludedIndex && rf.getterm(i) == cfTerm; i-- {
				reply.NewIndex = i
			}
			return
		}
	}
	i := rf.getLogindex(args.PrevLogIndex) + 1
	j := 0
	for i < len(rf.log) && j < len(args.Entries) {
		if rf.log[i].Term != args.Entries[j].Term {
			// cut = true
			break
		}
		i++
		j++
	}

	if j < len(args.Entries) {
		rf.log = rf.log[:i]
		to_append := args.Entries[j:]
		rf.log = append(rf.log, to_append...)
		rf.persist()
	}
	// if follower's log is longer than the leader's, there may be some problem
	// but may can be solved by blindly cutting and appending.
	// case: rejoin or snapshot

	reply.Success = true

	//If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
	if args.LeaderCommit > rf.commitIndex {
		rf.commitIndex = args.LeaderCommit
		lastnew := args.PrevLogIndex + len(args.Entries)
		if args.LeaderCommit > lastnew {
			rf.commitIndex = lastnew
		}
		if rf.commitIndex > rf.lastApplied {
			// rf.sendgo(rf.tryapply, 1)
			// rf.apply()
			rf.applywaiter.Signal()
		}
	}
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntriesHandler", args, reply)
	if !ok {
		return ok
	}
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()
	//reseted, out of date. Just checking everything I can think of, not sure if actually useful or not
	if rf.state != LEADER || args.Term != rf.currentTerm || reply.Term < rf.currentTerm {
		return ok
	}
	//out of date, which means this is been blocked out and there's a new leader
	if reply.Term > rf.currentTerm {
		rf.tof(args.Term)
		return ok
	}

	if reply.Success {
		match := args.PrevLogIndex + len(args.Entries)
		next := match + 1
		if match > rf.matchIndex[server] {
			rf.matchIndex[server] = args.PrevLogIndex + len(args.Entries)
		}
		if next > rf.nextIndex[server] {
			rf.nextIndex[server] = rf.matchIndex[server] + 1
		}

		// return ok
	} else if reply.NewIndex < 0 {
		return ok
	} else if reply.NewTerm < 0 {
		rf.nextIndex[server] = reply.NewIndex
		rf.sendto(server)
	} else {
		newNextIndex := rf.getLastindex()
		for ; newNextIndex > rf.LastIncludedIndex; newNextIndex-- {
			//found match.
			if rf.getterm(newNextIndex) == reply.NewTerm {
				break
			}
		}
		// if not found, set nextIndex to conflictIndex
		if newNextIndex <= rf.LastIncludedIndex {
			rf.nextIndex[server] = reply.NewIndex
		} else {
			rf.nextIndex[server] = newNextIndex
		}
		rf.sendto(server)
	}
	rf.tryCommit()
	return ok
}

func (rf *Raft) sendto(i int) {
	if rf.state != LEADER || i == rf.me {
		return
	}
	args := AppendEntriesArgs{}
	args.Term = rf.currentTerm
	args.LeaderID = rf.me
	args.PrevLogIndex = rf.nextIndex[i] - 1
	if args.PrevLogIndex < rf.LastIncludedIndex {
		rf.snapshotsender(i)
		// fmt.Println(rf.snapshot)
		return
	}
	args.PrevLogTerm = rf.getterm(args.PrevLogIndex)
	args.LeaderCommit = rf.commitIndex
	newentries := rf.log[rf.getLogindex(rf.nextIndex[i]):]
	args.Entries = make([]LogEntry, len(newentries))
	// make a deep copy to prevent "repeated call", which may happen if
	// a server deals so slow that the next beat arrived.
	copy(args.Entries, newentries)
	reply := AppendEntriesReply{}
	go rf.sendAppendEntries(i, &args, &reply)
}

//leader send heartbeats to others.
func (rf *Raft) sendbeats() {
	if rf.state != LEADER {
		return
	}
	for i := range rf.peers {
		rf.sendto(i)
	}
}
