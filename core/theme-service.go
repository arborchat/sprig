package core

import (
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

// ThemeService provides methods to fetch and manipulate the current
// application theme.
type ThemeService interface {
	Current() *sprigTheme.Theme
}

// themeService implements ThemeService.
type themeService struct {
	*sprigTheme.Theme
}

var _ ThemeService = &themeService{}

func newThemeService() (ThemeService, error) {
	return &themeService{
		Theme: sprigTheme.New(),
	}, nil
}

// Current returns the current theme.
func (t *themeService) Current() *sprigTheme.Theme {
	return t.Theme
}
