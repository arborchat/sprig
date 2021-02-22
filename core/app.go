package core

import (
	"fmt"
	"log"
	"os"
	"time"

	gioapp "gioui.org/app"
	status "git.sr.ht/~athorp96/forest-ex/active-status"
	"git.sr.ht/~whereswaldon/forest-go"
)

// App bundles core application services into a single convenience type.
type App interface {
	Notifications() NotificationService
	Arbor() ArborService
	Settings() SettingsService
	Sprout() SproutService
	Theme() ThemeService
	Status() StatusService
	Haptic() HapticService
	Banner() BannerService
	Shutdown()
}

// app bundles services together.
type app struct {
	NotificationService
	SettingsService
	ArborService
	SproutService
	ThemeService
	StatusService
	HapticService
	BannerService
	window *gioapp.Window
}

var _ App = &app{}

// NewApp constructs an App or fails with an error. This process will fail
// if any of the application services fail to initialize correctly.
func NewApp(w *gioapp.Window, stateDir string) (application App, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed constructing app: %w", err)
		}
	}()
	a := &app{
		window: w,
	}

	// ensure our state directory exists
	if err := os.MkdirAll(stateDir, 0770); err != nil {
		return nil, err
	}

	// Instantiate all of the services.
	// Settings must be initialized first, as other services rely on derived
	// values from it
	if a.SettingsService, err = newSettingsService(stateDir); err != nil {
		return nil, err
	}
	a.BannerService = NewBannerService()
	if a.ArborService, err = newArborService(a.SettingsService); err != nil {
		return nil, err
	}
	if a.NotificationService, err = newNotificationService(a.SettingsService, a.ArborService); err != nil {
		return nil, err
	}
	if a.SproutService, err = newSproutService(a.ArborService, a.BannerService); err != nil {
		return nil, err
	}
	if a.ThemeService, err = newThemeService(); err != nil {
		return nil, err
	}
	if a.StatusService, err = newStatusService(); err != nil {
		return nil, err
	}
	a.HapticService = newHapticService(w)

	// Connect services together
	if addr := a.Settings().Address(); addr != "" {
		a.Sprout().ConnectTo(addr)
	}
	a.Notifications().Register(a.Arbor().Store())
	a.Status().Register(a.Arbor().Store())

	a.Arbor().Store().SubscribeToNewMessages(func(n forest.Node) {
		a.Window().Invalidate()
	})

	return a, nil
}

// Settings returns the app's settings service implementation.
func (a *app) Settings() SettingsService {
	return a.SettingsService
}

// Arbor returns the app's arbor service implementation.
func (a *app) Arbor() ArborService {
	return a.ArborService
}

// Notifications returns the app's notification service implementation.
func (a *app) Notifications() NotificationService {
	return a.NotificationService
}

// Sprout returns the app's sprout service implementation.
func (a *app) Sprout() SproutService {
	return a.SproutService
}

// Theme returns the app's theme service implmentation.
func (a *app) Theme() ThemeService {
	return a.ThemeService
}

// Status returns the app's sprout service implementation.
func (a *app) Status() StatusService {
	return a.StatusService
}

// Haptic returns the app's haptic service implementation.
func (a *app) Haptic() HapticService {
	return a.HapticService
}

// Banner returns the app's banner service implementation.
func (a *app) Banner() BannerService {
	return a.BannerService
}

// Shutdown performs cleanup, and blocks for the duration.
func (a *app) Shutdown() {
	log.Printf("cleaning up")
	defer log.Printf("shutting down")
	for _, conn := range a.Sprout().Connections() {
		if worker := a.Sprout().WorkerFor(conn); worker != nil {
			var (
				nodes []forest.Node
			)
			a.Arbor().Communities().WithCommunities(func(coms []*forest.Community) {
				if a.Settings().ActiveArborIdentityID() != nil {
					builder, err := a.Settings().Builder()
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

// Window returns the window handle.
func (a app) Window() *gioapp.Window {
	return a.window
}
