package shardkv

import (
	"sync"
	"time"

	"6.824/raft"
	"6.824/shardctrler"
)

func (kv *ShardKV) Poll() {
	for !kv.killed() {
		time.Sleep(50 * time.Millisecond)
		kv.mu.Lock()
		num := kv.config.Num
		kv.mu.Unlock()
		lnum := kv.sm.Query(-1).Num

		_, leader := kv.rf.GetState()
		if !leader {
			continue
		}

		for cnum := num + 1; cnum <= lnum; cnum++ {
			for sid := 0; sid < shardctrler.NShards; sid++ {
				kv.s[sid] = false
			}
			kv.todate = false
			newConfig := kv.sm.Query(cnum)
			if !kv.mirtoraft(newConfig) {
				break
			}
		}

	}
}

func (kv *ShardKV) shardpuller(gid int, shards []int, num int, ND [shardctrler.NShards]map[string]string, NC [shardctrler.NShards]map[int64]int64) bool {
	args := &MArgs{
		Num:    num,
		Shards: shards,
	}
	for !kv.killed() {
		for _, server := range kv.config.Groups[gid] {
			reply := &MReply{
				OK: false,
			}
			srv := kv.make_end(server)
			ok := srv.Call("ShardKV.MData", args, reply)
			if !ok || !reply.OK {
				continue
			}

			//do data copy
			kv.mu1.Lock()
			for sid, data := range reply.Data {
				for k, v := range data {
					ND[sid][k] = v
				}
			}
			for sid, cts := range reply.Counters {
				for c, n := range cts {
					NC[sid][c] = n
				}
			}
			kv.mu1.Unlock()
			return true
		}
	}

	return false
}

func (kv *ShardKV) godelete(gid int, shards []int, num int, oldconfig shardctrler.Config) {
	args := &MArgs{
		Num:    num,
		Shards: shards,
	}
	for i := 0; i < 3; i++ {
		for _, server := range oldconfig.Groups[gid] {
			reply := &MReply{
				OK: false,
			}
			srv := kv.make_end(server)
			ok := srv.Call("ShardKV.CanDelete", args, reply)
			if !ok || !reply.OK {
				continue
			}
		}
	}
}

func (kv *ShardKV) mirtoraft(newConfig shardctrler.Config) bool {
	if newConfig.Num != kv.config.Num+1 {

		return false
	}
	if _, leader := kv.rf.GetState(); !leader {

		return false
	}

	kv.mu.Lock()
	gid := kv.gid
	oldConfig := kv.config
	kv.mu.Unlock()
	// saving datas
	to_save := make([]int, 0)

	for sid := 0; sid < shardctrler.NShards; sid++ {
		if newConfig.Shards[sid] != kv.gid && oldConfig.Shards[sid] == kv.gid && newConfig.Shards[sid] != 0 {
			to_save = append(to_save, sid)
		}
	}
	// bk := false
this:
	for i := 0; i < 3; i++ {
		op := Op{
			Type:   SAVE,
			Num:    newConfig.Num,
			Shards: to_save,
			Start:  len(to_save) > 0,
		}
		kv.rf.Start(op)
		idx, _, leader := kv.rf.Start(op)
		if !leader {
			return false
		}
		kv.mu.Lock()
		if _, ok := kv.to_exe[idx]; !ok {
			kv.to_exe[idx] = make(chan bool)
		}
		wc := kv.to_exe[idx]
		kv.mu.Unlock()
		select {
		case <-wc:
			kv.mu.Lock()
			stored := kv.saved
			kv.mu.Unlock()
			if stored == newConfig.Num {
				// bk = true
				break this
			}
		case <-time.After(300 * time.Millisecond):
		}
	}
	// if !bk {
	// 	return false
	// }
	to_merge := make(map[int][]int)
	var m_shards []int
	for sid := 0; sid < shardctrler.NShards; sid++ {
		if newConfig.Shards[sid] == gid && oldConfig.Shards[sid] != gid {
			//figuring out what the hell should I merge from which mother fucker
			// if oldgid ==0, means that init, nothing need to be done now
			if oldg := oldConfig.Shards[sid]; oldg != 0 {
				m_shards = append(m_shards, sid)
				if _, ok := to_merge[oldg]; !ok {
					to_merge[oldg] = []int{}
				}
				to_merge[oldg] = append(to_merge[oldg], sid)
			}
		}
	}
	//pull datas from other groups
	var newCounters [shardctrler.NShards]map[int64]int64
	var newData [shardctrler.NShards]map[string]string
	for sid := 0; sid < shardctrler.NShards; sid++ {
		newData[sid] = make(map[string]string)
		newCounters[sid] = make(map[int64]int64)
	}
	OK := true
	var wg sync.WaitGroup
	for g, s := range to_merge {
		wg.Add(1)
		go func(gid int, shards []int) {
			defer wg.Done()
			if kv.shardpuller(gid, shards, newConfig.Num, newData, newCounters) {
				return
			}
			kv.mu1.Lock()
			OK = false
			kv.mu1.Unlock()
		}(g, s)
	}
	wg.Wait()
	if !OK {
		return false
	}
	//Ok, now let's update our data and configuration
	start := newConfig.Num == kv.sm.Query(-1).Num
	for {
		op := Op{
			Type:     UPDATE,
			Data:     newData,
			Counters: newCounters,
			Config:   newConfig,
			Start:    start,
			Shards:   m_shards,
		}
		idx, _, leader := kv.rf.Start(op)
		if !leader {
			return false
		}
		kv.mu.Lock()
		if _, ok := kv.to_exe[idx]; !ok {
			kv.to_exe[idx] = make(chan bool)
		}
		wc := kv.to_exe[idx]
		kv.mu.Unlock()
		select {
		case <-wc:
			kv.mu.Lock()
			cfgNum := kv.config.Num
			kv.mu.Unlock()
			if cfgNum == newConfig.Num {
				for gid, shards := range to_merge {
					go kv.godelete(gid, shards, newConfig.Num, oldConfig)
				}
			}
			return cfgNum == newConfig.Num
		case <-time.After(300 * time.Millisecond):
		}
	}
}

func (kv *ShardKV) goupdate(op Op) {
	newConfig := op.Config
	if newConfig.Num != kv.config.Num+1 {
		return
	}
	for _, sid := range op.Shards {
		kv.data[sid] = make(map[string]string)
		kv.counters[sid] = make(map[int64]int64)
		for k, v := range op.Data[sid] {
			kv.data[sid][k] = v
		}
		for c, n := range op.Counters[sid] {
			kv.counters[sid][c] = n
		}
	}

	kv.config = newConfig
	// don't start servering unless reach the newest config
	if kv.config.Num != kv.sm.Query(-1).Num {
		return
	}
	for sid := 0; sid < shardctrler.NShards; sid++ {
		if kv.config.Shards[sid] == kv.gid {
			kv.s[sid] = true
		} else {
			kv.data[sid] = nil
			kv.counters[sid] = nil
		}
	}
}

func (kv *ShardKV) deleteshards(op Op) {
	for _, sid := range op.Shards {
		// for num := range kv.ps[sid] {
		if kv.saved <= op.Num {
			kv.ps[sid] = nil
			kv.pc[sid] = nil
		}
		// }
	}
	// for _, sid := range op.Shards {
	// 	kv.pc[sid][op.Num] = nil
	// }
}

func (kv *ShardKV) store(op Op) {
	shards := op.Shards
	num := op.Num
	if !op.Start {
		kv.saved = num
		return
	}
	if kv.saved+1 != num {
		return
	}
	for _, sid := range shards {

		kv.ps[sid] = make(map[string]string)
		kv.pc[sid] = make(map[int64]int64)
		for k, v := range kv.data[sid] {
			kv.ps[sid][k] = v
		}
		for c, n := range kv.counters[sid] {
			kv.pc[sid][c] = n
		}
		kv.data[sid] = nil
		kv.counters[sid] = nil
	}

	// history clean up, only need the newest one
	// for sid := range kv.ps {
	// 	max := -1
	// 	for hnum := range kv.ps[sid] {
	// 		if max < hnum {
	// 			max = hnum
	// 		}
	// 	}
	// 	for hnum := range kv.ps[sid] {
	// 		if max > hnum {
	// 			kv.ps[sid][hnum] = nil
	// 		}
	// 	}
	// }
	// for sid := range kv.pc {
	// 	max := -1
	// 	for hnum := range kv.pc[sid] {
	// 		if max < hnum {
	// 			max = hnum
	// 		}
	// 	}
	// 	for hnum := range kv.pc[sid] {
	// 		if max > hnum {
	// 			kv.ps[sid][hnum] = nil
	// 		}
	// 	}
	// }
	kv.saved = num

}
func (kv *ShardKV) Apply(msg *raft.ApplyMsg) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if msg.CommandIndex == kv.lastindex+1 {
		op, ok := msg.Command.(Op)
		if ok {
			switch op.Type {
			case UPDATE:
				kv.goupdate(op)
			case DELETE:
				kv.deleteshards(op)
			case SAVE:
				kv.store(op)
			default:
				if kv.counters[op.ShardID][op.ClientID] == op.ID {
					switch op.Type {
					case PUT:
						kv.data[op.ShardID][op.Key] = op.Value
					case APPEND:
						kv.data[op.ShardID][op.Key] += op.Value
					default:
					}
					kv.counters[op.ShardID][op.ClientID]++
					// fmt.Println(kv.counters[op.ShardID][op.ClientID], op.ID)
				}
			}
			kv.resetchan(msg.CommandIndex)
			kv.lastindex = msg.CommandIndex
		}
		kv.TryTakeSanpshot()
	}
}

func (kv *ShardKV) applier(maxraftstate int) {
	for !kv.killed() {
		msg := <-kv.applyCh
		if &msg != nil {
			if msg.CommandValid {
				kv.Apply(&msg)
			} else if msg.SnapshotValid && maxraftstate != -1 {
				kv.TryInstallSnapshot(&msg)
			}
		}
	}
}
