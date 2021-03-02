package core

import (
	"crypto/tls"
	"fmt"
	"log"
	"sync"
	"time"

	status "git.sr.ht/~athorp96/forest-ex/active-status"
	"git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/forest-go/store"
	"git.sr.ht/~whereswaldon/sprout-go"
)

type SproutService interface {
	ConnectTo(address string) error
	Connections() []string
	WorkerFor(address string) *sprout.Worker
	MarkSelfOffline()
}

type sproutService struct {
	ArborService
	BannerService
	SettingsService
	workerLock sync.Mutex
	workerDone chan struct{}
	workers    map[string]*sprout.Worker
}

var _ SproutService = &sproutService{}

func newSproutService(arbor ArborService, banner BannerService, settings SettingsService) (SproutService, error) {
	s := &sproutService{
		ArborService:    arbor,
		BannerService:   banner,
		SettingsService: settings,
		workers:         make(map[string]*sprout.Worker),
		workerDone:      make(chan struct{}),
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
		worker, done := func() (*sprout.Worker, chan struct{}) {
			connectionBanner := &LoadingBanner{
				Priority: Info,
				Text:     "Connecting to " + addr + "...",
			}
			defer connectionBanner.Cancel()
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
				return nil, nil
			}
			worker.Logger = log.New(logger.Writer(), fmt.Sprintf("worker-%v ", addr), log.Flags())

			s.workerLock.Lock()
			s.workers[addr] = worker
			s.workerLock.Unlock()
			return worker, done
		}()
		if worker == nil {
			continue
		}

		go func() {
			synchronizingBanner := &LoadingBanner{
				Priority: Info,
				Text:     "Syncing with " + addr + "...",
			}
			s.BannerService.Add(synchronizingBanner)
			defer synchronizingBanner.Cancel()
			BootstrapSubscribed(worker, s.SettingsService.Subscriptions())
		}()

		worker.Run()
		select {
		case <-done:
			return
		default:
		}
	}
}

// MarkSelfOffline announces that the local user is offline in all known
// communities.
func (s *sproutService) MarkSelfOffline() {
	for _, conn := range s.Connections() {
		if worker := s.WorkerFor(conn); worker != nil {
			var (
				nodes []forest.Node
			)
			s.ArborService.Communities().WithCommunities(func(coms []*forest.Community) {
				if s.SettingsService.ActiveArborIdentityID() != nil {
					builder, err := s.SettingsService.Builder()
					if err == nil {
						log.Printf("killing active-status heartbeat")
						for _, c := range coms {
							n, err := status.NewActivityNode(c, builder, status.Inactive, time.Minute*5)
							if err != nil {
								log.Printf("creating inactive node: %v", err)
								continue
							}
							log.Printf("sending offline node to community %s", c.ID())
							nodes = append(nodes, n)
						}
					} else {
						log.Printf("aquiring builder: %v", err)
					}
				}
			})
			if err := worker.SendAnnounce(nodes, time.NewTicker(time.Second*5).C); err != nil {
				log.Printf("sending shutdown messages: %v", err)
			}
		}
	}
}

func makeTicker(duration time.Duration) <-chan time.Time {
	return time.NewTicker(duration).C
}

func BootstrapSubscribed(worker *sprout.Worker, subscribed []string) error {
	leaves := 1024
	communities, err := worker.SendList(fields.NodeTypeCommunity, leaves, makeTicker(worker.DefaultTimeout))
	if err != nil {
		worker.Printf("Failed listing peer communities: %v", err)
		return err
	}
	subbed := map[string]bool{}
	for _, s := range subscribed {
		subbed[s] = true
	}
	for _, node := range communities.Nodes {
		community, isCommunity := node.(*forest.Community)
		if !isCommunity {
			worker.Printf("Got response in community list that isn't a community: %s", node.ID().String())
			continue
		}
		if !subbed[community.ID().String()] {
			continue
		}
		if err := worker.IngestNode(community); err != nil {
			worker.Printf("Couldn't ingest community %s: %v", community.ID().String(), err)
			continue
		}
		if err := worker.SendSubscribe(community, makeTicker(worker.DefaultTimeout)); err != nil {
			worker.Printf("Couldn't subscribe to community %s", community.ID().String())
			continue
		}
		worker.Subscribe(community.ID())
		worker.Printf("Subscribed to %s", community.ID().String())
		if err := worker.SynchronizeFullTree(community, leaves, worker.DefaultTimeout); err != nil {
			worker.Printf("Couldn't fetch message tree rooted at community %s: %v", community.ID().String(), err)
			continue
		}
	}
	return nil
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
