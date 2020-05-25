package main

import (
	"log"
	"sync"

	"git.sr.ht/~whereswaldon/forest-go/store"
	"git.sr.ht/~whereswaldon/sprig/ds"
	sprout "git.sr.ht/~whereswaldon/sprout-go"
)

type ArborState struct {
	SubscribableStore store.ExtendedStore
	*ds.ReplyList
	*ds.CommunityList

	workerLock sync.Mutex
	workerDone chan struct{}
	workerLog  *log.Logger
}

func (a *ArborState) RestartWorker(address string) {
	a.workerLock.Lock()
	defer a.workerLock.Unlock()
	if a.workerDone != nil {
		close(a.workerDone)
	}
	a.workerDone = make(chan struct{})
	a.workerLog = log.New(log.Writer(), "worker "+address, log.LstdFlags|log.Lshortfile)
	go sprout.LaunchSupervisedWorker(a.workerDone, address, a.SubscribableStore, nil, a.workerLog)
}
