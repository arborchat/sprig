package core

import (
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

// ThemeService provides methods to fetch and manipulate the current
// application theme.
type ThemeService interface {
	Current() *sprigTheme.Theme
	SetDarkMode(bool)
}

// themeService implements ThemeService.
type themeService struct {
	*sprigTheme.Theme
	darkTheme *sprigTheme.Theme
	useDark   bool
}

var _ ThemeService = &themeService{}

func newThemeService() (ThemeService, error) {
	dark := sprigTheme.New()
	dark.ToDark()
	return &themeService{
		Theme:     sprigTheme.New(),
		darkTheme: dark,
	}, nil
}

// Current returns the current theme.
func (t *themeService) Current() *sprigTheme.Theme {
	if !t.useDark {
		return t.Theme
	}
	return t.darkTheme
}

func (t *themeService) SetDarkMode(enabled bool) {
	t.useDark = enabled
}
