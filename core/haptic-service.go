package core

import (
	"log"

	gioapp "gioui.org/app"
	"gioui.org/x/haptic"
)

// HapticService provides access to haptic feedback devices features.
type HapticService interface {
	UpdateAndroidViewRef(uintptr)
	Buzz()
}

type hapticService struct {
	*haptic.Buzzer
}

func newHapticService(w *gioapp.Window) HapticService {
	return &hapticService{
		Buzzer: haptic.NewBuzzer(w),
	}
}

func (h *hapticService) UpdateAndroidViewRef(view uintptr) {
	h.Buzzer.SetView(view)
}

func (h *hapticService) Buzz() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Recovered from buzz panic: %v", err)
		}
	}()
	h.Buzzer.Buzz()
}
