package core

import (
	"fmt"
	"os"

	gioapp "gioui.org/app"
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
	a := &app{}

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
	if a.ArborService, err = newArborService(a.SettingsService); err != nil {
		return nil, err
	}
	if a.NotificationService, err = newNotificationService(a.SettingsService, a.ArborService); err != nil {
		return nil, err
	}
	if a.SproutService, err = newSproutService(a.ArborService); err != nil {
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

// Haptic returns the app's sprout service implementation.
func (a *app) Haptic() HapticService {
	return a.HapticService
}
