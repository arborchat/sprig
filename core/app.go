package core

import (
	"fmt"
	"log"
	"strings"
	"time"

	"git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/forest-go/store"
	"git.sr.ht/~whereswaldon/niotify"
)

type App interface {
	Notifications() NotificationService
	Arbor() ArborService
	Settings() SettingsService
}

type ArborService interface {
	Store() store.ExtendedStore
}

type SettingsService interface {
	NotificationsGloballyAllowed() bool
	ActiveArborIdentityID() *fields.QualifiedHash
}

type NotificationService interface {
	Register(store.ExtendedStore)
	Notify(title, content string) error
}

type notificationManager struct {
	App
	niotify.Manager
	TimeLaunched uint64
}

var _ NotificationService = &notificationManager{}

func newNotificationService(app App) (NotificationService, error) {
	m, err := niotify.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed initializing notification support: %w", err)
	}
	return &notificationManager{
		App:          app,
		Manager:      m,
		TimeLaunched: uint64(time.Now().UnixNano() / 1000000),
	}, nil
}

func (n *notificationManager) Register(s store.ExtendedStore) {
	s.SubscribeToNewMessages(n.handleNode)
}

func (n *notificationManager) shouldNotify(reply *forest.Reply) bool {
	if !n.Settings().NotificationsGloballyAllowed() {
		return false
	}
	localUserID := n.Settings().ActiveArborIdentityID()
	if localUserID == nil {
		return false
	}
	localUserNode, has, err := n.Arbor().Store().GetIdentity(localUserID)
	if err != nil || !has {
		return false
	}
	localUser := localUserNode.(*forest.Identity)
	messageContent := strings.ToLower(string(reply.Content.Blob))
	username := strings.ToLower(string(localUser.Name.Blob))
	if strings.Contains(messageContent, username) {
		// local user directly mentioned
		return true
	}
	if uint64(reply.Created) < n.TimeLaunched {
		// do not send old notifications
		return false
	}
	if reply.Author.Equals(localUserID) {
		// Do not send notifications for replies created by the local
		// user's identity.
		return false
	}
	if reply.TreeDepth() == 1 {
		// Notify of new conversation
		return true
	}
	parent, known, err := n.Arbor().Store().Get(reply.ParentID())
	if err != nil || !known {
		// Don't notify if we don't know about this conversation.
		return false
	}
	if parent.(*forest.Reply).Author.Equals(localUserID) {
		// Direct response to local user.
		return true
	}
	return false
}

func (n *notificationManager) Notify(title, content string) error {
	_, err := n.CreateNotification(title, content)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	return nil
}

func (n *notificationManager) handleNode(node forest.Node) {
	if asReply, ok := node.(*forest.Reply); ok {
		go func(reply *forest.Reply) {
			if !n.shouldNotify(reply) {
				return
			}
			var title, authorName string
			author, _, err := n.Arbor().Store().GetIdentity(&reply.Author)
			if err != nil {
				authorName = "???"
			} else {
				authorName = string(author.(*forest.Identity).Name.Blob)
			}
			switch {
			case reply.Depth == 1:
				title = fmt.Sprintf("New conversation by %s", authorName)
			default:
				title = fmt.Sprintf("New reply from %s", authorName)
			}
			_, err = n.Manager.CreateNotification(title, string(reply.Content.Blob))
			if err != nil {
				log.Printf("failed sending notification: %v", err)
			}
		}(asReply)
	}
}

type app struct {
	NotificationService
	SettingsService
	ArborService
}

var _ App = &app{}

func NewApp() (application App, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed constructing app: %w", err)
		}
	}()
	a := &app{}
	if a.NotificationService, err = newNotificationService(a); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *app) Settings() SettingsService {
	return a.SettingsService
}

func (a *app) Arbor() ArborService {
	return a.ArborService
}

func (a *app) Notifications() NotificationService {
	return a.NotificationService
}
