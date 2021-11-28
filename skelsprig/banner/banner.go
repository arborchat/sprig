package banner

import (
	"sort"

	"git.sr.ht/~gioverse/skel/scheduler"
)

// Banner is a type that provides details for a persistent on-screen
// notification banner
type Banner interface {
	BannerPriority() Priority
	Cancel()
	IsCancelled() bool
}

type Service struct {
	conn       scheduler.Connection
	newBanners chan Banner
	banners    []Banner
}

// New constructs a Service connected on the provided bus connection.
func New(conn scheduler.Connection) *Service {
	s := &Service{
		newBanners: make(chan Banner, 1),
		conn:       conn,
	}
	go s.run()
	return s
}

func (s *Service) run() {
	for event := range s.conn.Output() {
		switch event.(type) {
		case Request:
			s.conn.Message(Event{Service: s})
		}
	}
}

// Request asks for an Event to be sent over the bus.
type Request struct{}

// Event provides a handle to the banner service.
type Event struct {
	*Service
}

// Add inserts a banner into the banner list. It may block, and
// should not be called directly from a UI goroutine.
func (b *Service) Add(banner Banner) {
	b.newBanners <- banner
}

// Top returns the banner that should currently be displayed, if any.
func (b *Service) Top() Banner {
	select {
	case banner := <-b.newBanners:
		b.banners = append(b.banners, banner)
		sort.Slice(b.banners, func(i, j int) bool {
			return b.banners[i].BannerPriority() > b.banners[j].BannerPriority()
		})
	default:
	}
	if len(b.banners) < 1 {
		return nil
	}
	first := b.banners[0]
	for first.IsCancelled() {
		b.banners = b.banners[1:]
		if len(b.banners) < 1 {
			return nil
		}
		first = b.banners[0]
	}
	return first
}

type Priority uint8

const (
	Debug Priority = iota
	Info
	Warn
	Error
)

// LoadingBanner requests a banner with a loading spinner displayed along with
// the provided text. It will not disappear until cancelled.
type LoadingBanner struct {
	Priority
	Text      string
	cancelled bool
}

func (l *LoadingBanner) BannerPriority() Priority {
	return l.Priority
}

func (l *LoadingBanner) Cancel() {
	l.cancelled = true
}

func (l *LoadingBanner) IsCancelled() bool {
	return l.cancelled
}
