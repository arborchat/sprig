package main

import (
	"log"
	"sync"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/store"
	"git.sr.ht/~whereswaldon/sprig/ds"
	sprout "git.sr.ht/~whereswaldon/sprout-go"
	"git.sr.ht/~whereswaldon/wisteria/replylist"
)

type ArborState struct {
	sync.Once
	SubscribableStore store.ExtendedStore
	*replylist.ReplyList
	*ds.CommunityList

	workerLock sync.Mutex
	workerDone chan struct{}
	workerLog  *log.Logger
}

func (a *ArborState) init() {
	a.Once.Do(func() {
		a.SubscribableStore.SubscribeToNewMessages(func(node forest.Node) {
			switch node.(type) {
			case *forest.Community:
				go a.CommunityList.Sort()
			case *forest.Reply:
				go a.ReplyList.Sort()
			}
		})
	})
}

func (a *ArborState) RestartWorker(address string) {
	a.init()
	a.workerLock.Lock()
	defer a.workerLock.Unlock()
	if a.workerDone != nil {
		close(a.workerDone)
	}
	a.workerDone = make(chan struct{})
	a.workerLog = log.New(log.Writer(), "worker "+address, log.LstdFlags|log.Lshortfile)
	go sprout.LaunchSupervisedWorker(a.workerDone, address, a.SubscribableStore, nil, a.workerLog)
}
