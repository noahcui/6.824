package shardkv

import "6.824/shardctrler"

//
// Sharded key/value server.
// Lots of replica groups, each running op-at-a-time paxos.
// Shardctrler decides which group serves each shard.
// Shardctrler may change shard assignment from time to time.
//
// You will have to modify these definitions.
//

const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongGroup  = "ErrWrongGroup"
	ErrWrongLeader = "ErrWrongLeader"
	PUT            = "Put"
	GET            = "Get"
	APPEND         = "Append"
	NEWCONFIG      = "NewConfig"
	SAVE           = "Save"

	UPDATE = "Update"
	DELETE = "Delete"
)

type Err string

// Put or Append
type PutAppendArgs struct {
	// You'll have to add definitions here.
	Key   string
	Value string
	Op    string // "Put" or "Append"
	// You'll have to add definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	ClientID int64
	ID       int64
	Num      int
}

type PutAppendReply struct {
	Err Err
}

type GetArgs struct {
	Key      string
	ClientID int64
	ID       int64
	Num      int
	// You'll have to add definitions here.
}

type GetReply struct {
	Err   Err
	Value string
}

type MArgs struct {
	Num    int
	Shards []int
}

type MReply struct {
	OK       bool
	Data     [shardctrler.NShards]map[string]string
	Counters [shardctrler.NShards]map[int64]int64
}

type CpyArgs struct {
	Servers []string
	Num     int
}

type CpyReply struct {
	Stop bool
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
	Servers  []string
	Config   shardctrler.Config
	ShardID  int
	Data     [shardctrler.NShards]map[string]string
	Counters [shardctrler.NShards]map[int64]int64
	CUD      bool
	WC       chan bool
	Num      int
	Start    bool
	Shards   []int
}
