package shardctrler

import (
	"sync"
	"sync/atomic"

	"6.824/labgob"
	"6.824/labrpc"
	"6.824/raft"
)

type ShardCtrler struct {
	mu        sync.Mutex
	me        int
	rf        *raft.Raft
	applyCh   chan raft.ApplyMsg
	persister *raft.Persister
	dead      int32 // set by Kill()
	// Your data here.

	maxraftstate int // snapshot if log grows this big
	lastindex    int
	// Your definitions here.
	data     map[string]string
	to_exe   map[int]chan bool
	counters map[int64]int64
	configs  []Config // indexed by config num
}
type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	Type     string
	Key      string
	Value    string
	ClientID int64
	ID       int64
	Num      int
	Servers  map[int][]string
	GIDs     []int
	Shard    int
	GID      int
}

type waiter struct {
	ch chan bool
	op *Op
}

//
// the tester calls Kill() when a ShardCtrler instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (sc *ShardCtrler) Kill() {
	sc.rf.Kill()
	// Your code here, if desired.
}
func (sc *ShardCtrler) killed() bool {
	z := atomic.LoadInt32(&sc.dead)
	return z == 1
}
func (sc *ShardCtrler) resetchan(idx int) {
	if v, ok := sc.to_exe[idx]; ok {
		// select {
		// case v <- true:
		// default:
		// }
		// to release memory
		close(v)
		delete(sc.to_exe, idx)
	}
}

func (sc *ShardCtrler) Startfrompersister() {
	sc.installsnapshot(sc.persister.ReadSnapshot())
	sc.lastindex = sc.rf.LastIncludedIndex
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Paxos to
// form the fault-tolerant shardctrler service.
// me is the index of the current server in servers[].
//
func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister) *ShardCtrler {
	sc := new(ShardCtrler)
	sc.me = me
	sc.maxraftstate = -1

	sc.configs = make([]Config, 1)
	sc.configs[0].Groups = map[int][]string{}
	sc.configs[0].Num = 0

	labgob.Register(Op{})
	sc.applyCh = make(chan raft.ApplyMsg)
	sc.rf = raft.Make(servers, me, persister, sc.applyCh)

	// Your code here.
	sc.persister = persister
	sc.lastindex = 0
	sc.applyCh = make(chan raft.ApplyMsg)
	sc.rf = raft.Make(servers, me, persister, sc.applyCh)
	sc.to_exe = make(map[int]chan bool)
	sc.counters = make(map[int64]int64)

	// You may need initialization code here.
	sc.data = make(map[string]string)

	sc.Startfrompersister()
	go func() {
		for !sc.killed() {
			msg := <-sc.applyCh
			// for msg := range kv.applyCh {
			// if &msg != nil {
			if msg.CommandValid {
				sc.Apply(&msg)
			} else if msg.SnapshotValid {
				sc.TryInstallSnapshot(&msg)
			}
		}
		// }

	}()

	return sc
}
