package raft

//
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//

func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, LastIncludedIndex int, snapshot []byte) bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.commitIndex >= LastIncludedIndex {
		return false
	}
	if rf.getterm(rf.commitIndex) > lastIncludedTerm {
		return false
	}
	defer func() {
		rf.snapshot = make([]byte, len(snapshot))
		copy(rf.snapshot, snapshot)
		rf.lastApplied = LastIncludedIndex
		rf.LastIncludedIndex = LastIncludedIndex
		rf.commitIndex = LastIncludedIndex
		rf.LastIncludedTerm = lastIncludedTerm
		rf.persistsnapshot()
	}()
	var to_append []LogEntry = nil
	if LastIncludedIndex <= rf.getLastindex() &&
		rf.getterm(LastIncludedIndex) == lastIncludedTerm {
		to_append = rf.log[rf.getLogindex(LastIncludedIndex+1):]
	}
	rf.log = make([]LogEntry, 1)
	rf.log[0].Term = lastIncludedTerm
	rf.log[0].Command = nil
	rf.log = append(rf.log, to_append...)
	return true
}
func (rf *Raft) InstallSnapshot(args *SnapshotArg, reply *SnapshotReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	reply.Term = rf.currentTerm
	if args.Term < rf.currentTerm {
		return
	}
	if args.Term > rf.currentTerm {
		rf.tof(args.Term)
		reply.Term = rf.currentTerm
	}

	rf.send(rf.beat, 1)
	reply.Ok = true
	if rf.LastIncludedIndex >= args.LastIncludedIndex {
		return
	}

	rf.mu.Unlock()
	msg := ApplyMsg{
		CommandValid:  false,
		SnapshotValid: true,
		SnapshotTerm:  args.LastIncludedTerm,
		SnapshotIndex: args.LastIncludedIndex,
		Snapshot:      args.Data,
	}
	msg.Snapshot = make([]byte, len(args.Data))
	copy(msg.Snapshot, args.Data)
	rf.applyCh <- msg
	rf.mu.Lock()
	return
}

func (rf *Raft) sendSnapshot(server int, args *SnapshotArg, reply *SnapshotReply) bool {
	ok := rf.peers[server].Call("Raft.InstallSnapshot", args, reply)
	if !ok {
		return false
	}
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.state != LEADER || args.Term != rf.currentTerm || reply.Term < rf.currentTerm {
		return ok
	}
	if reply.Term > rf.currentTerm {
		rf.tof(reply.Term)
		return false
	}
	rf.nextIndex[server] = args.LastIncludedIndex + 1
	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if index <= rf.LastIncludedIndex {
		return
	}
	if index > rf.lastApplied {
		return
	}
	to_append := rf.log[rf.getLogindex(index+1):]
	rf.LastIncludedTerm = rf.getterm(index)
	rf.LastIncludedIndex = index
	rf.log = make([]LogEntry, 1)
	rf.log[0].Term = rf.LastIncludedTerm
	rf.log[0].Command = nil
	rf.log = append(rf.log, to_append...)
	rf.snapshot = make([]byte, len(snapshot))
	copy(rf.snapshot, snapshot)
	rf.persistsnapshot()
}

func (rf *Raft) snapshotsender(id int) {
	if rf.state != LEADER || rf.me == id {
		return
	}
	args := SnapshotArg{
		Term:              rf.currentTerm,
		LastIncludedIndex: rf.LastIncludedIndex,
		LastIncludedTerm:  rf.LastIncludedTerm,
	}
	args.Data = make([]byte, len(rf.snapshot))
	copy(args.Data, rf.snapshot)
	reply := SnapshotReply{
		Term: -1,
		Ok:   false,
	}
	rf.mu.Unlock()
	go rf.sendSnapshot(id, &args, &reply)
	rf.mu.Lock()
}
