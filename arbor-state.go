package main

import (
	"log"
	"sort"
	"strings"
	"sync"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/store"
	sprout "git.sr.ht/~whereswaldon/sprout-go"
	"git.sr.ht/~whereswaldon/wisteria/replylist"
)

type ArborState struct {
	sync.Once
	SubscribableStore store.ExtendedStore
	*replylist.ReplyList

	communities []*forest.Community
	replies     []*forest.Reply

	workerLock sync.Mutex
	workerDone chan struct{}
	workerLog  *log.Logger
}

func (a *ArborState) init() {
	a.Once.Do(func() {
		a.SubscribableStore.SubscribeToNewMessages(func(node forest.Node) {
			switch concreteNode := node.(type) {
			case *forest.Community:
				index := sort.Search(len(a.communities), func(i int) bool {
					return a.communities[i].ID().Equals(concreteNode.ID())
				})
				if index >= len(a.communities) {
					a.communities = append(a.communities, concreteNode)
					sort.SliceStable(a.communities, func(i, j int) bool {
						return strings.Compare(string(a.communities[i].Name.Blob), string(a.communities[j].Name.Blob)) < 0
					})
				}
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
