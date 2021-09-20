package kvraft

import (
	"time"

	"6.824/raft"
)

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	op := Op{
		Type:     "Get",
		Key:      args.Key,
		ClientID: args.ClientID,
		ID:       args.ID,
	}
	//just to make sure it is the leader here.
	err := kv.exe(op)
	if err {
		reply.Err = ErrWrongLeader
		return
	}
	kv.mu.Lock()
	value, ok := kv.data[args.Key]
	kv.mu.Unlock()
	if !ok {
		reply.Err = ErrNoKey
		reply.Value = ""
		return
	}
	reply.Value = value
	reply.Err = OK

}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	op := Op{
		Key:      args.Key,
		Value:    args.Value,
		ClientID: args.ClientID,
		ID:       args.ID,
		Type:     args.Op,
	}

	err := kv.exe(op)
	if err {
		reply.Err = ErrWrongLeader
		return
	}
	reply.Err = OK
}

//
// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
//
func (kv *KVServer) checklate(op Op) bool {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	return kv.counters[op.ClientID] > op.ID
}
func (kv *KVServer) exe(op Op) bool {
	// kv.mu.Lock()
	// defer kv.mu.Unlock()
	if kv.checklate(op) {
		return false
	}
	idx, _, is_leader := kv.rf.Start(op)
	if !is_leader {
		return true
	}
	kv.mu.Lock()
	if _, ok := kv.to_exe[idx]; !ok {
		kv.to_exe[idx] = make(chan bool)
	}
	wc := kv.to_exe[idx]
	kv.mu.Unlock()
	// kv.mu.Unlock()
	select {
	case <-wc:
	case <-time.After(2000 * time.Millisecond):
	}
	// kv.mu.Lock()
	if kv.checklate(op) {
		return false
	}
	return true
}

func (kv *KVServer) resetchan(idx int) {
	if v, ok := kv.to_exe[idx]; ok {
		// select {
		// case v <- true:
		// default:
		// }
		// to release memory
		close(v)
		delete(kv.to_exe, idx)
	}
}
func (kv *KVServer) Apply(msg *raft.ApplyMsg) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if msg.CommandIndex == kv.lastindex+1 {
		args, ok := msg.Command.(Op)
		if ok {
			if kv.counters[args.ClientID] == args.ID {
				switch args.Type {
				case "Put":
					kv.data[args.Key] = args.Value
				case "Append":
					kv.data[args.Key] = kv.data[args.Key] + args.Value
				default:
				}
				kv.counters[args.ClientID]++
			}
			kv.lastindex = msg.CommandIndex
		}
		kv.TryTakeSanpshot()
		kv.resetchan(msg.CommandIndex)
	}

}
