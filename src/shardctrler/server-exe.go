package shardctrler

import (
	"6.824/raft"
)

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

func (sc *ShardCtrler) rebalance(ct map[int]int, os [NShards]int, new map[int][]string) {
	var ns [NShards]int
	l := len(ct)
	for i := 0; i < NShards; i++ {
		ns[i] = 0
	}
	if l > 0 {
		even := NShards / l
		extra := NShards % l
		opt := make([]int, l)
		for i := 0; i < l; i++ {
			opt[i] = even
		}

		for i := 0; i < l && extra > 0; i++ {
			opt[i]++
			extra--
		}
		idx := make([]int, 0)
		for i := range ct {
			idx = append(idx, i)
		}
		// sort the index list, to make sure every server do the same thing
		for k1, i := range idx {
			for k2, j := range idx {
				if i < j {
					temp := idx[k1]
					idx[k1] = idx[k2]
					idx[k2] = temp
				}
			}
		}

		// optermizing
		// free some shards
		var isfree [NShards]bool
		for i := 0; i < NShards; i++ {
			isfree[i] = true
		}
		for id := range opt {
			for i := 0; i < NShards; i++ {
				if os[i] == opt[id] && opt[id] > 0 {
					isfree[i] = false
					opt[id]--
				}
			}
		}

		// assign new shards
		i := 0
		for j, c := range opt {
			for a := 0; a < c; a++ {
				for !isfree[i] {
					i++
				}
				ns[i] = idx[j]
				i++
			}
		}
	}
	to_append := Config{
		Num:    len(sc.configs),
		Shards: ns,
		Groups: new,
	}

	sc.configs = append(sc.configs, to_append)
	// fmt.Println("in: ", sc.query(-1))
}

func (sc *ShardCtrler) join(op Op) {

	// last one
	old := sc.configs[len(sc.configs)-1]
	new := make(map[int][]string)
	ct := make(map[int]int)
	// deep copy
	for gid, server := range old.Groups {
		new[gid] = server
	}

	// insert any new
	for gid, server := range op.Servers {
		new[gid] = server
	}
	// rebalancing
	for i := range new {
		ct[i] = 1
	}
	sc.rebalance(ct, old.Shards, new)
}

func (sc *ShardCtrler) leave(op Op) {

	old := sc.configs[len(sc.configs)-1]
	new := make(map[int][]string)
	ct := make(map[int]int)
	to_leave := make(map[int]bool)
	for gid := range old.Groups {
		to_leave[gid] = false
	}
	for _, gid := range op.GIDs {
		// fmt.Println(gid)
		to_leave[gid] = true
	}
	for gid, server := range old.Groups {
		if !to_leave[gid] {
			new[gid] = server
		}
	}

	// rebalancing
	for i := range new {
		ct[i] = 1
	}

	// fmt.Println(new)
	sc.rebalance(ct, old.Shards, new)

}

func (sc *ShardCtrler) move(op Op) {
	//deep copy
	old := sc.configs[len(sc.configs)-1]
	var ns [NShards]int
	for i := 0; i < NShards; i++ {
		ns[i] = old.Shards[i]
	}
	new := make(map[int][]string)
	for gid, server := range old.Groups {
		new[gid] = server
	}
	ns[op.Shard] = op.GID
	to_append := Config{
		Num:    len(sc.configs),
		Shards: ns,
		Groups: new,
	}
	sc.configs = append(sc.configs, to_append)
}

func (sc *ShardCtrler) query(num int) Config {
	l := len(sc.configs)
	if num > -1 && num < l {
		return sc.configs[num]
	}
	return sc.configs[l-1]
}

func (sc *ShardCtrler) Apply(msg *raft.ApplyMsg) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if msg.CommandIndex > sc.lastindex {
		args, ok := msg.Command.(Op)
		if ok {
			if sc.counters[args.ClientID] == args.ID {
				switch args.Type {
				case "Join":
					sc.join(msg.Command.(Op))
				case "Leave":
					sc.leave(msg.Command.(Op))
				case "Move":
					sc.move(msg.Command.(Op))
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
