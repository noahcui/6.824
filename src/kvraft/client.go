package kvraft

import (
	"crypto/rand"
	"math/big"
	"time"

	"6.824/labrpc"
)

type Clerk struct {
	servers []*labrpc.ClientEnd
	// You will have to modify this struct.
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
	// You'll have to add code here.
	ck.clientID = nrand()
	ck.leaderID = 0
	ck.counter = 0
	return ck
}

//
// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
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
		// if ok && reply.Err == ErrWrongLeader {
		// 	ck.leaderID = (ck.leaderID + 1) % len(ck.servers)
		// 	continue
		// }
		// <-time.After(10 * time.Millisecond)
		ck.leaderID = (ck.leaderID + 1) % len(ck.servers)
		//doing election? wait to avoid using resources
		if olde == ck.leaderID {
			<-time.After(10 * time.Millisecond)
		}
	}
}

//
// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
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
		// if ok && reply.Err == ErrWrongLeader {
		// 	ck.leaderID = (ck.leaderID + 1) % len(ck.servers)
		// 	continue
		// }
		// <-time.After(10 * time.Millisecond)
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
