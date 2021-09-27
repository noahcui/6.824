package shardctrler

import (
	"bytes"
	"log"

	"6.824/labgob"
	"6.824/raft"
)

func (sc *ShardCtrler) installsnapshot(snapshot []byte) {
	if snapshot == nil || len(snapshot) == 0 {
		return
	}
	sc.data = make(map[string]string)
	sc.counters = make(map[int64]int64)
	lastindex := sc.lastindex
	r := bytes.NewBuffer(snapshot)
	d := labgob.NewDecoder(r)
	if d.Decode(&sc.data) != nil ||
		d.Decode(&sc.counters) != nil ||
		d.Decode(&sc.lastindex) != nil {
		log.Fatalf("decode error\n")
	}
	// making sure all channels are freed
	for idx := lastindex; idx <= sc.lastindex; idx++ {
		sc.resetchan(idx)
	}
}
func (sc *ShardCtrler) TryInstallSnapshot(msg *raft.ApplyMsg) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if sc.rf.CondInstallSnapshot(msg.SnapshotTerm,
		msg.SnapshotIndex, msg.Snapshot) {
		sc.installsnapshot(msg.Snapshot)
	}
}

func (sc *ShardCtrler) TryTakeSanpshot() {
	if sc.maxraftstate != -1 && sc.maxraftstate <= sc.persister.RaftStateSize() {
		w := new(bytes.Buffer)
		e := labgob.NewEncoder(w)
		e.Encode(sc.data)
		e.Encode(sc.counters)
		e.Encode(sc.lastindex)
		sc.rf.Snapshot(sc.lastindex, w.Bytes())
	}
}
