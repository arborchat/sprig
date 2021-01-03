package core

import (
	"crypto/tls"
	"fmt"
	"log"
	"sync"
	"time"

	"git.sr.ht/~whereswaldon/forest-go/store"
	"git.sr.ht/~whereswaldon/sprout-go"
)

type SproutService interface {
	ConnectTo(address string) error
	Connections() []string
	WorkerFor(address string) *sprout.Worker
}

type sproutService struct {
	ArborService
	BannerService
	workerLock sync.Mutex
	workerDone chan struct{}
	workers    map[string]*sprout.Worker
}

var _ SproutService = &sproutService{}

func newSproutService(arbor ArborService, banner BannerService) (SproutService, error) {
	s := &sproutService{
		ArborService:  arbor,
		BannerService: banner,
		workers:       make(map[string]*sprout.Worker),
		workerDone:    make(chan struct{}),
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
	go s.launchWorker(address)
	return nil
}

func (s *sproutService) Connections() []string {
	s.workerLock.Lock()
	defer s.workerLock.Unlock()
	out := make([]string, 0, len(s.workers))
	for addr := range s.workers {
		out = append(out, addr)
	}
	return out
}

func (s *sproutService) WorkerFor(address string) *sprout.Worker {
	s.workerLock.Lock()
	defer s.workerLock.Unlock()
	out, defined := s.workers[address]
	if !defined {
		return nil
	}
	return out
}

func (s *sproutService) launchWorker(addr string) {
	firstAttempt := true
	logger := log.New(log.Writer(), "worker "+addr, log.LstdFlags|log.Lshortfile)
	for {
		connectionBanner := &LoadingBanner{
			Priority: Info,
			Text:     "Connecting to " + addr + "...",
		}
		s.BannerService.Add(connectionBanner)
		if !firstAttempt {
			logger.Printf("Restarting worker for address %s", addr)
			time.Sleep(time.Second)
		}
		firstAttempt = false

		s.workerLock.Lock()
		done := s.workerDone
		s.workerLock.Unlock()

		worker, err := NewWorker(addr, done, s.ArborService.Store())
		if err != nil {
			log.Printf("Failed starting worker: %v", err)
			continue
		}
		worker.Logger = log.New(logger.Writer(), fmt.Sprintf("worker-%v ", addr), log.Flags())

		s.workerLock.Lock()
		s.workers[addr] = worker
		s.workerLock.Unlock()
		connectionBanner.Cancel()

		synchronizingBanner := &LoadingBanner{
			Priority: Info,
			Text:     "Syncing with " + addr + "...",
		}
		s.BannerService.Add(synchronizingBanner)
		go func() {
			worker.BootstrapLocalStore(1024)
			synchronizingBanner.Cancel()
		}()

		worker.Run()
		select {
		case <-done:
			return
		default:
		}
	}
}

// NewWorker creates a sprout worker connected to the provided address using
// TLS over TCP as a transport.
func NewWorker(addr string, done <-chan struct{}, s store.ExtendedStore) (*sprout.Worker, error) {
	conn, err := tls.Dial("tcp", addr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %v", addr, err)
	}

	worker, err := sprout.NewWorker(done, conn, s)
	if err != nil {
		return nil, fmt.Errorf("failed launching worker to connect to address %s: %v", addr, err)
	}

	return worker, nil
}
