package shardctrler

//
// Shardctrler clerk.
//

import (
	"crypto/rand"
	"math/big"
	"time"

	"6.824/labrpc"
)

type Clerk struct {
	servers []*labrpc.ClientEnd
	// Your data here.
	leaderID int
	clientID int64
	counter  int64
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	ck.clientID = nrand()
	ck.leaderID = 0
	ck.counter = 0
	// Your code here.
	return ck
}

func (ck *Clerk) Query(num int) Config {
	args := &QueryArgs{}
	// Your code here.
	args.ClientID = ck.clientID
	args.ID = ck.counter
	args.Op = "Query"
	args.Num = num
	ck.counter++
	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply QueryReply
			ok := srv.Call("ShardCtrler.Query", args, &reply)
			if ok && reply.WrongLeader == false {
				return reply.Config
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (ck *Clerk) Join(servers map[int][]string) {
	args := &JoinArgs{}
	// Your code here.
	args.Servers = servers
	args.ClientID = ck.clientID
	args.ID = ck.counter
	args.Op = "Join"
	ck.counter++
	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply JoinReply
			ok := srv.Call("ShardCtrler.Join", args, &reply)
			if ok && reply.WrongLeader == false {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (ck *Clerk) Leave(gids []int) {
	args := &LeaveArgs{}
	// Your code here.
	args.GIDs = gids
	args.ClientID = ck.clientID
	args.ID = ck.counter
	args.Op = "Leave"
	ck.counter++
	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply LeaveReply
			ok := srv.Call("ShardCtrler.Leave", args, &reply)
			if ok && reply.WrongLeader == false {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (ck *Clerk) Move(shard int, gid int) {
	args := &MoveArgs{}
	// Your code here.
	args.Shard = shard
	args.ClientID = ck.clientID
	args.GID = gid
	args.Op = "Move"
	args.ID = ck.counter
	ck.counter++

	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply MoveReply
			ok := srv.Call("ShardCtrler.Move", args, &reply)
			if ok && reply.WrongLeader == false {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (ck *Clerk) Get(key string) string {

	// You will have to modify this function.

	//repeat calling, in incase there's a on going election
	// i := 0
	args := GetArgs{
		Key:      key,
		ClientID: ck.clientID,
		ID:       ck.counter,
	}
	ck.counter++
	olde := ck.leaderID
	for {
		reply := GetReply{}
		ok := ck.servers[ck.leaderID].Call("KVServer.Get", &args, &reply)
		if ok && reply.Err == OK {
			return reply.Value
		}
		if ok && reply.Err == ErrNoKey {
			return ""
		}
		ck.leaderID = (ck.leaderID + 1) % len(ck.servers)
		//doing election? wait to avoid using resources
		if olde == ck.leaderID {
			<-time.After(10 * time.Millisecond)
		}
	}
}

func (ck *Clerk) PutAppend(key string, value string, op string) {
	// You will have to modify this function.

	args := PutAppendArgs{
		Key:      key,
		Value:    value,
		Op:       op,
		ClientID: ck.clientID,
		ID:       ck.counter,
	}
	ck.counter++
	//keep calling until success
	olde := ck.leaderID
	for {
		reply := PutAppendReply{}
		ok := ck.servers[ck.leaderID].Call("KVServer.PutAppend", &args, &reply)
		if reply.Err == OK && ok {
			return
		}
		ck.leaderID = (ck.leaderID + 1) % len(ck.servers)
		//doing election? wait to avoid using resources
		if olde == ck.leaderID {
			<-time.After(10 * time.Millisecond)
		}
	}
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
