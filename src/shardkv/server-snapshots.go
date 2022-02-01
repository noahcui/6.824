package shardkv

import (
	"bytes"
	"log"

	"6.824/labgob"
	"6.824/raft"
)

func (kv *ShardKV) installsnapshot(snapshot []byte) {
	if snapshot == nil || len(snapshot) == 0 {
		return
	}
	lastindex := kv.lastindex
	r := bytes.NewBuffer(snapshot)
	d := labgob.NewDecoder(r)
	if d.Decode(&kv.data) != nil ||
		d.Decode(&kv.pc) != nil ||
		d.Decode(&kv.saved) != nil ||
		d.Decode(&kv.ps) != nil ||
		d.Decode((&kv.s)) != nil ||
		d.Decode(&kv.counters) != nil ||
		// if d.Decode(&kv.p) != nil ||
		d.Decode(&kv.lastindex) != nil ||
		d.Decode(&kv.config) != nil {
		log.Fatalf("decode error\n")
	}
	// making sure all channels are freed
	for idx := lastindex; idx <= kv.lastindex; idx++ {
		kv.resetchan(idx)
	}
}
func (kv *ShardKV) TryInstallSnapshot(msg *raft.ApplyMsg) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.rf.CondInstallSnapshot(msg.SnapshotTerm,
		msg.SnapshotIndex, msg.Snapshot) {
		kv.installsnapshot(msg.Snapshot)
	}
}

func (kv *ShardKV) TryTakeSanpshot() {
	if kv.maxraftstate != -1 && kv.maxraftstate <= kv.persister.RaftStateSize() {
		w := new(bytes.Buffer)
		e := labgob.NewEncoder(w)
		e.Encode(kv.data)
		e.Encode(kv.pc)
		e.Encode(kv.saved)
		e.Encode(kv.ps)
		e.Encode(kv.s)
		e.Encode(kv.counters)
		e.Encode(kv.lastindex)
		e.Encode(kv.config)
		kv.rf.Snapshot(kv.lastindex, w.Bytes())
	}
}
