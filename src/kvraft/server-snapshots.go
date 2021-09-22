package kvraft

import (
	"bytes"
	"log"

	"6.824/labgob"
	"6.824/raft"
)

func (kv *KVServer) installsnapshot(snapshot []byte) {
	if snapshot == nil || len(snapshot) == 0 {
		return
	}
	kv.data = make(map[string]string)
	kv.counters = make(map[int64]int64)
	lastindex := kv.lastindex
	r := bytes.NewBuffer(snapshot)
	d := labgob.NewDecoder(r)
	if d.Decode(&kv.data) != nil ||
		d.Decode(&kv.counters) != nil ||
		d.Decode(&kv.lastindex) != nil {
		log.Fatalf("decode error\n")
	}
	// making sure all channels are freed
	for idx := lastindex; idx <= kv.lastindex; idx++ {
		kv.resetchan(idx)
	}
}
func (kv *KVServer) TryInstallSnapshot(msg *raft.ApplyMsg) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.rf.CondInstallSnapshot(msg.SnapshotTerm,
		msg.SnapshotIndex, msg.Snapshot) {
		kv.installsnapshot(msg.Snapshot)
	}
}

func (kv *KVServer) TryTakeSanpshot() {
	if kv.maxraftstate != -1 && kv.maxraftstate <= kv.persister.RaftStateSize() {
		w := new(bytes.Buffer)
		e := labgob.NewEncoder(w)
		e.Encode(kv.data)
		e.Encode(kv.counters)
		e.Encode(kv.lastindex)
		kv.rf.Snapshot(kv.lastindex, w.Bytes())
	}
}
