package core

import (
	status "git.sr.ht/~athorp96/forest-ex/active-status"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/forest-go/store"
)

// StatusService provides information on the online status of users.
type StatusService interface {
	Register(store.ExtendedStore)
	IsActive(*fields.QualifiedHash) bool
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

// IsActive returns whether or not a given user is listed as currently
// active. If the user has never been registered by the StatusManager,
// they are considered inactive.
func (s *statusService) IsActive(id *fields.QualifiedHash) bool {
	return s.StatusManager.IsActive(*id)
}
