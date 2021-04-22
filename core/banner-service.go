package core

import "sort"

// Banner is a type that provides details for a persistent on-screen
// notification banner
type Banner interface {
	BannerPriority() Priority
	Cancel()
	IsCancelled() bool
}

// BannerService provides methods for creating and managing on-screen
// persistent banners. The methods must be safe for concurrent use.
type BannerService interface {
	// Add establishes a new banner managed by the service.
	Add(Banner)
	// Top returns the banner that should be displayed right now
	Top() Banner
}

type bannerService struct {
	App
	newBanners chan Banner
	banners    []Banner
}

var _ BannerService = &bannerService{}

func NewBannerService(app App) BannerService {
	return &bannerService{
		newBanners: make(chan Banner, 1),
		App:        app,
	}
}

func (b *bannerService) Add(banner Banner) {
	b.newBanners <- banner
	b.App.Window().Invalidate()
}

func (b *bannerService) Top() Banner {
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
