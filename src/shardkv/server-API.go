package shardkv

import (
	"time"
)

func (kv *ShardKV) checklate(op Op) bool {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	return kv.counters[op.ShardID][op.ClientID] > op.ID
}

func (kv *ShardKV) resetchan(idx int) {
	if v, ok := kv.to_exe[idx]; ok {
		// to release memory
		close(v)
		delete(kv.to_exe, idx)
	}
}

func (kv *ShardKV) CanDelete(args *MArgs, reply *MReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	op := Op{
		Type:   DELETE,
		Shards: args.Shards,
		Num:    args.Num,
	}
	kv.rf.Start(op)
}

func (kv *ShardKV) MData(args *MArgs, reply *MReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if args.Num > kv.saved {
		reply.OK = false
		// fmt.Println("FUCK", args.Num, kv.saved)
		return
	}
	reply.OK = true

	for _, sid := range args.Shards {
		to_send := kv.ps[sid]
		to_counter := kv.pc[sid]
		reply.Data[sid] = make(map[string]string)
		reply.Counters[sid] = make(map[int64]int64)
		for k, v := range to_send {
			reply.Data[sid][k] = v
		}
		for c, n := range to_counter {
			reply.Counters[sid][c] = n
		}
	}
}

func (kv *ShardKV) Get(args *GetArgs, reply *GetReply) {
	op := Op{
		Type:     GET,
		Key:      args.Key,
		ClientID: args.ClientID,
		ID:       args.ID,
		Num:      args.Num,
		ShardID:  key2shard(args.Key),
	}
	//just to make sure it is the leader here.
	reply.Value, reply.Err = kv.exe(op)
	if reply.Err == OK {
		kv.mu.Lock()
		value, ok := kv.data[op.ShardID][args.Key]
		kv.mu.Unlock()
		if !ok {
			reply.Err = ErrNoKey
			reply.Value = ""
			return
		}
		reply.Value = value
	}
}

func (kv *ShardKV) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	op := Op{
		Key:      args.Key,
		Value:    args.Value,
		ClientID: args.ClientID,
		ID:       args.ID,
		Type:     args.Op,
		Num:      args.Num,
		ShardID:  key2shard(args.Key),
	}
	_, reply.Err = kv.exe(op)
}

func (kv *ShardKV) crtgroup(op Op) bool {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	shard := key2shard(op.Key)
	// return kv.config.Shards[shard] == kv.gid && kv.config.Num == op.Num
	return op.Num == kv.config.Num && kv.s[shard]
}
func (kv *ShardKV) exe(op Op) (string, Err) {

	if !kv.crtgroup(op) {
		// kv.mu.Lock()
		// log.Debugf("group %v, serverid %v, serverconfig %v, s %v opconfig %v", kv.gid, kv.me, kv.config.Num, kv.s, op.Num)
		// kv.mu.Unlock()
		return "", ErrWrongGroup
	}
	if kv.checklate(op) {
		return "", OK
	}

	idx, _, is_leader := kv.rf.Start(op)
	if !is_leader {
		return "", ErrWrongLeader
	}
	kv.mu.Lock()
	if _, ok := kv.to_exe[idx]; !ok {
		kv.to_exe[idx] = make(chan bool)
	}
	wc := kv.to_exe[idx]
	kv.mu.Unlock()
	select {
	case <-wc:
	case <-time.After(500 * time.Millisecond):
	}
	if kv.checklate(op) {
		return "", OK
	}
	return "", ErrWrongLeader
}
