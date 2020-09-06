package core

import (
	status "git.sr.ht/~athorp96/forest-ex/active-status"
	"git.sr.ht/~whereswaldon/forest-go/store"
)

// StatusService provides information on the online status of users.
type StatusService interface {
	Register(store.ExtendedStore)
}

type statusService struct {
	*status.StatusManager
}

var _ StatusService = &statusService{}

func newStatusService() (StatusService, error) {
	return &statusService{
		StatusManager: status.NewStatusManager(),
	}, nil
}

// Register subscribes the StatusService to new nodes within
// the provided store.
func (s *statusService) Register(stor store.ExtendedStore) {
	stor.SubscribeToNewMessages(s.StatusManager.HandleNode)
}
