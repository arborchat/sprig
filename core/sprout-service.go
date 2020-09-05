package core

import (
	"log"
	"sync"

	"git.sr.ht/~whereswaldon/sprout-go"
)

type SproutService interface {
	ConnectTo(address string) error
}

type sproutService struct {
	ArborService
	workerLock sync.Mutex
	workerDone chan struct{}
	workerLog  *log.Logger
}

var _ SproutService = &sproutService{}

func newSproutService(arbor ArborService) (SproutService, error) {
	s := &sproutService{
		ArborService: arbor,
	}
	return s, nil
}

// ConnectTo (re)connects to the specified address.
func (s *sproutService) ConnectTo(address string) error {
	s.workerLock.Lock()
	defer s.workerLock.Unlock()
	if s.workerDone != nil {
		close(s.workerDone)
	}
	s.workerDone = make(chan struct{})
	s.workerLog = log.New(log.Writer(), "worker "+address, log.LstdFlags|log.Lshortfile)
	go sprout.LaunchSupervisedWorker(s.workerDone, address, s.ArborService.Store(), nil, s.workerLog)
	return nil
}
