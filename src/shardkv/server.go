package shardkv

import (
	"sync"
	"sync/atomic"

	"6.824/labgob"
	"6.824/labrpc"
	"6.824/raft"
	"6.824/shardctrler"
)

type rt struct {
	val string
	err Err
}

type ShardKV struct {
	mu           sync.Mutex
	mu1          sync.Mutex
	me           int
	rf           *raft.Raft
	persister    *raft.Persister
	applyCh      chan raft.ApplyMsg
	make_end     func(string) *labrpc.ClientEnd
	gid          int
	ctrlers      []*labrpc.ClientEnd
	maxraftstate int // snapshot if log grows this big
	// Your definitions here.

	// pre-sharding and packaging, making life eaiser
	to_exe    map[int]chan bool
	lastindex int
	dead      int32 // set by Kill()
	sm        *shardctrler.Clerk
	config    shardctrler.Config
	s         [shardctrler.NShards]bool
	// nextConfig shardctrler.Config
	data     [shardctrler.NShards]map[string]string
	counters [shardctrler.NShards]map[int64]int64
	todate   bool
	saved    int
	ss       [shardctrler.NShards]int
	ps       [shardctrler.NShards]map[string]string
	pc       [shardctrler.NShards]map[int64]int64
}

//
// the tester calls Kill() when a ShardKV instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (kv *ShardKV) Kill() {
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *ShardKV) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

//
// servers[] contains the ports of the servers in this group.
//
// me is the index of the current server in servers[].
//
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
//
// the k/v server should snapshot when Raft's saved state exceeds
// maxraftstate bytes, in order to allow Raft to garbage-collect its
// log. if maxraftstate is -1, you don't need to snapshot.
//
// gid is this group's GID, for interacting with the shardctrler.
//
// pass ctrlers[] to shardctrler.MakeClerk() so you can send
// RPCs to the shardctrler.
//
// make_end(servername) turns a server name from a
// Config.Groups[gid][i] into a labrpc.ClientEnd on which you can
// send RPCs. You'll need this to send RPCs to other groups.
//
// look at client.go for examples of how to use ctrlers[]
// and make_end() to send RPCs to the group owning a specific shard.
//
// StartServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func (kv *ShardKV) Startfrompersister() {
	kv.installsnapshot(kv.persister.ReadSnapshot())
	kv.lastindex = kv.rf.LastIncludedIndex
}

func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int, gid int, ctrlers []*labrpc.ClientEnd, make_end func(string) *labrpc.ClientEnd) *ShardKV {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(ShardKV)
	kv.me = me
	kv.maxraftstate = maxraftstate
	kv.make_end = make_end
	// gid is uniq and not changing
	kv.gid = gid
	kv.ctrlers = ctrlers
	kv.persister = persister
	// Your initialization code here.
	// Use something like this to talk to the shardctrler:
	// kv.mck = shardctrler.MakeClerk(kv.ctrlers)

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)
	kv.sm = shardctrler.MakeClerk(kv.ctrlers)
	kv.to_exe = make(map[int]chan bool)
	// we only care about order, val doesn't matter
	// kv.counters = make(map[int64]int64)
	// You may need initialization code here.
	for sid := 0; sid < shardctrler.NShards; sid++ {
		kv.data[sid] = make(map[string]string)
		kv.counters[sid] = make(map[int64]int64)
		kv.ps[sid] = make(map[string]string)
		kv.pc[sid] = make(map[int64]int64)
	}
	kv.Startfrompersister()
	go kv.applier(maxraftstate)
	go kv.Poll()
	return kv
}
