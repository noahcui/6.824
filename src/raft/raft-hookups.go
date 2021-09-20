package raft

import (
	"math/rand"
	"time"
)

// will be too slow if race mode is on
func (rf *Raft) applierhookupchannel() {
	for !rf.killed() {
		<-rf.tryapply
		rf.mu.Lock()
		if rf.lastApplied < rf.commitIndex &&
			rf.lastApplied < rf.getLastindex() &&
			rf.lastApplied+1 > rf.lastIncludedIndex {
			for rf.lastApplied < rf.commitIndex {
				rf.lastApplied++
				msg := ApplyMsg{
					CommandValid:  true,
					Command:       rf.log[rf.getLogindex(rf.lastApplied)].Command,
					CommandIndex:  rf.lastApplied,
					SnapshotValid: false,
				}
				rf.mu.Unlock()
				rf.applyCh <- msg
				rf.mu.Lock()
			}
		}
		rf.mu.Unlock()
	}
}

func (rf *Raft) applierhookup() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	for !rf.killed() {
		if rf.lastApplied < rf.commitIndex &&
			rf.lastApplied < rf.getLastindex() &&
			rf.lastApplied+1 > rf.lastIncludedIndex {
			for rf.lastApplied < rf.commitIndex {
				rf.lastApplied++
				msg := ApplyMsg{
					CommandValid:  true,
					Command:       rf.log[rf.getLogindex(rf.lastApplied)].Command,
					CommandIndex:  rf.lastApplied,
					SnapshotValid: false,
				}
				rf.mu.Unlock()
				rf.applyCh <- msg
				rf.mu.Lock()
			}
		} else {
			rf.applywaiter.Wait()
		}
	}
}
func (rf *Raft) hookup() {
	for !rf.killed() {
		rf.mu.Lock()
		currentstate := rf.state
		rf.mu.Unlock()
		switch currentstate {
		case LEADER:
			select {
			case <-rf.giveup:
			case <-rf.reset:
			case <-time.After(50 * time.Millisecond):
				rf.mu.Lock()
				rf.sendbeats()
				rf.mu.Unlock()
			}

		case FOLLOWER:
			select {
			case <-rf.giveup:
			case <-rf.vote:
			case <-rf.beat:
			case <-time.After(time.Duration(rand.Float32()*150+150) * time.Millisecond):
				rf.toc(FOLLOWER)
			}
		case CANDIDATE:
			select {
			case <-rf.giveup:
			case <-rf.beat:
			case <-rf.win:
				rf.tol()
			case <-time.After(time.Duration(rand.Float32()*150+150) * time.Millisecond):
				rf.toc(CANDIDATE)
			}
		}
	}

}
