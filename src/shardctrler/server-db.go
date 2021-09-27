package shardctrler

import (
	"time"

	"6.824/raft"
)

func (sc *ShardCtrler) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	op := Op{
		Type:     "Get",
		Key:      args.Key,
		ClientID: args.ClientID,
		ID:       args.ID,
	}
	//just to make sure it is the leader here.
	err := sc.exe(op)
	if err {
		reply.Err = ErrWrongLeader
		return
	}
	sc.mu.Lock()
	value, ok := sc.data[args.Key]
	sc.mu.Unlock()
	if !ok {
		reply.Err = ErrNoKey
		reply.Value = ""
		return
	}
	reply.Value = value
	reply.Err = OK

}

func (sc *ShardCtrler) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	op := Op{
		Key:      args.Key,
		Value:    args.Value,
		ClientID: args.ClientID,
		ID:       args.ID,
		Type:     args.Op,
	}

	err := sc.exe(op)
	if err {
		reply.Err = ErrWrongLeader
		return
	}
	reply.Err = OK
}

//
// the tester calls Kill() when a ShardCtrler instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
//
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

func (sc *ShardCtrler) Apply(msg *raft.ApplyMsg) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if msg.CommandIndex > sc.lastindex {
		args, ok := msg.Command.(Op)
		if ok {
			if sc.counters[args.ClientID] == args.ID {
				switch args.Type {
				case "Put":
					sc.data[args.Key] = args.Value
				case "Append":
					sc.data[args.Key] = sc.data[args.Key] + args.Value
				case "Join":

				case "Leave":
				default:
				}
				sc.counters[args.ClientID]++
			}
			sc.lastindex = msg.CommandIndex
		}
		sc.TryTakeSanpshot()
		sc.resetchan(msg.CommandIndex)
	}

}
