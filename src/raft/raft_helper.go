package raft

import (
	"sync/atomic"

	"6.5840/labrpc"
)

/*
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
*/
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {

	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.dead = 0
	rf.applyCh = applyCh

	rf.commitIndex = 0
	rf.lastApplied = 0

	// persistent states
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.log = make([]LogEntry, 0)

	rf.role = FOLLOWER
	rf.msgReceived = false
	rf.currentLeader = -1
	rf.currentAppended = 0

	rf.nextIndices = make([]int, len(rf.peers))
	rf.matchIndices = make([]int, len(rf.peers))
	rf.latestIssuedEntryIndices = make([]int, len(rf.peers))
	rf.trailingReplyChan = make(chan AppendEntriesReply)
	rf.quitTrailingReplyChan = make(chan int)
	go func() {
		rf.quitTrailingReplyChan <- 0
	}()

	rf.hbTimeOut = HBTIMEOUT
	rf.eleTimeOut = ELETIMEOUT
	rf.randomRange = RANDOMRANGE

	// initialize from state persisted before a crash
	recover := rf.readPersist(persister.ReadRaftState())
	if !recover {
		PersistenceDPrintf("Not previous state to recover from, persist initialization\n")
		rf.persist("server %v initialization", rf.me)
	} else {
		PersistenceDPrintf("Recover form previous state\n")
	}

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	// Your code here (2A).
	// currentTerm := atomic.LoadUint64(&rf.currentTerm)
	// role := atomic.LoadInt32(&rf.role)
	rf.mu.Lock()
	defer rf.mu.Unlock()
	currentTerm := rf.currentTerm
	role := rf.role
	return currentTerm, role == LEADER
}

/*
the tester doesn't halt goroutines created by Raft after each test,
but it does call the Kill() method. your code can use killed() to
check whether Kill() has been called. the use of atomic avoids the
need for a lock.

the issue is that long-running goroutines use memory and may chew
up CPU time, perhaps causing later tests to fail and generating
confusing debug output. any goroutine with a long-running loop
should call killed() to check whether it should stop.
*/
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) isLeader() bool {
	// ans := atomic.LoadInt32(&rf.role)
	// return ans == LEADER
	rf.mu.Lock()
	defer rf.mu.Unlock()
	ans := rf.role
	return ans == LEADER
}

/*
must hold the lock rf.mu to call this function

	rf.currentTerm = term
	rf.role = FOLLOWER
	rf.votedFor = -1
	rf.currentLeader = -1
	rf.currentAppended = 0

need to call persist() since votedFor is changed
*/
func (rf *Raft) onReceiveHigherTerm(term int) int {
	originalTerm := rf.currentTerm

	rf.currentTerm = term
	rf.role = FOLLOWER
	rf.votedFor = -1
	rf.currentLeader = -1
	rf.currentAppended = 0

	return originalTerm
}
