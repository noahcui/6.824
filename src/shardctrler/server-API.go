package shardctrler

import (
	"time"

	"6.824/raft"
)

func (sc *ShardCtrler) checklate(op Op) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.counters[op.ClientID] > op.ID
}
func (sc *ShardCtrler) exe(op Op) bool {
	// sc.mu.Lock()
	// defer sc.mu.Unlock()
	if sc.checklate(op) {
		return false
	}
	idx, _, is_leader := sc.rf.Start(op)
	if !is_leader {
		return true
	}
	sc.mu.Lock()
	if _, ok := sc.to_exe[idx]; !ok {
		sc.to_exe[idx] = make(chan bool)
	}
	wc := sc.to_exe[idx]
	sc.mu.Unlock()
	// sc.mu.Unlock()
	select {
	case <-wc:
	case <-time.After(2000 * time.Millisecond):
	}
	// sc.mu.Lock()
	if sc.checklate(op) {
		return false
	}
	return true
}

// needed by shardkv tester
func (sc *ShardCtrler) Raft() *raft.Raft {
	return sc.rf
}
func (sc *ShardCtrler) Join(args *JoinArgs, reply *JoinReply) {
	// Your code here.
	op := Op{
		ClientID: args.ClientID,
		ID:       args.ID,
		Type:     args.Op,
		Servers:  args.Servers,
	}

	err := sc.exe(op)
	if err {
		reply.Err = ErrWrongLeader
		reply.WrongLeader = true
		return
	}
	reply.WrongLeader = false
	reply.Err = OK
}

func (sc *ShardCtrler) Leave(args *LeaveArgs, reply *LeaveReply) {
	// Your code here.
	op := Op{
		ClientID: args.ClientID,
		ID:       args.ID,
		Type:     args.Op,
		GIDs:     args.GIDs,
	}

	err := sc.exe(op)
	if err {
		reply.Err = ErrWrongLeader
		reply.WrongLeader = true
		return
	}
	reply.WrongLeader = false
	reply.Err = OK
}

func (sc *ShardCtrler) Move(args *MoveArgs, reply *MoveReply) {
	// Your code here.
	op := Op{
		ClientID: args.ClientID,
		ID:       args.ID,
		Type:     args.Op,
		Shard:    args.Shard,
		GID:      args.GID,
		// Servers:  args.Servers,
	}
	err := sc.exe(op)
	if err {
		reply.Err = ErrWrongLeader
		reply.WrongLeader = true
		return
	}
	reply.WrongLeader = false
	reply.Err = OK
}

func (sc *ShardCtrler) Query(args *QueryArgs, reply *QueryReply) {
	// Your code here.
	op := Op{
		ClientID: args.ClientID,
		ID:       args.ID,
		Type:     args.Op,
		Num:      args.Num,
		// Servers:  args.Servers,
	}

	err := sc.exe(op)
	if err {
		reply.Err = ErrWrongLeader
		reply.WrongLeader = true
		return
	}
	reply.WrongLeader = false
	reply.Err = OK
	reply.Config = sc.query(op.Num)
}
