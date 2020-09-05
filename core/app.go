package core

import (
	"fmt"
)

// App bundles core application services into a single convenience type.
type App interface {
	Notifications() NotificationService
	Arbor() ArborService
	Settings() SettingsService
}

// app bundles services together.
type app struct {
	NotificationService
	SettingsService
	ArborService
}

var _ App = &app{}

// NewApp constructs an App or fails with an error. This process will fail
// if any of the application services fail to initialize correctly.
func NewApp(stateDir string) (application App, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed constructing app: %w", err)
		}
	}()
	a := &app{}
	// Settings must be initialized first, as other services rely on derived
	// values from it
	if a.SettingsService, err = newSettingsService(a, stateDir); err != nil {
		return nil, err
	}
	if a.NotificationService, err = newNotificationService(a); err != nil {
		return nil, err
	}
	if a.ArborService, err = newArborService(a); err != nil {
		return nil, err
	}

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
