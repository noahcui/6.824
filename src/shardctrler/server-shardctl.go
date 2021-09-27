package shardctrler

import "6.824/raft"

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
	reply.Err = OK
}

func (sc *ShardCtrler) Move(args *MoveArgs, reply *MoveReply) {
	// Your code here.
}

func (sc *ShardCtrler) Query(args *QueryArgs, reply *QueryReply) {
	// Your code here.
}
