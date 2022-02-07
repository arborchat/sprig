package core

import (
	"fmt"
	"log"
	"strings"
	"time"

	niotify "gioui.org/x/notify"
	"git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/store"
)

// NotificationService provides methods to send notifications and to
// configure notifications for collections of arbor nodes.
type NotificationService interface {
	Register(store.ExtendedStore)
	Notify(title, content string) error
}

// notificationManager implements NotificationService and provides
// methods to send notifications and choose (based on settings)
// whether to notify for a given arbor message.
type notificationManager struct {
	SettingsService
	ArborService
	niotify.Notifier
	TimeLaunched uint64
}

var _ NotificationService = &notificationManager{}

// newNotificationService constructs a new NotificationService for the
// provided App.
func newNotificationService(settings SettingsService, arbor ArborService) (NotificationService, error) {
	m, err := niotify.NewNotifier()
	if err != nil {
		return nil, fmt.Errorf("failed initializing notification support: %w", err)
	}
	return &notificationManager{
		SettingsService: settings,
		ArborService:    arbor,
		Notifier:        m,
		TimeLaunched:    uint64(time.Now().UnixNano() / 1000000),
	}, nil
}

// Register configures the store so that new nodes will generate notifications
// if notifications are appropriate (based on current user settings).
func (n *notificationManager) Register(s store.ExtendedStore) {
	s.SubscribeToNewMessages(n.handleNode)
}

// shouldNotify returns whether or not a node should generate a notification
// according to the user's current settings.
func (n *notificationManager) shouldNotify(reply *forest.Reply) bool {
	if !n.SettingsService.NotificationsGloballyAllowed() {
		return false
	}
	if md, err := reply.TwigMetadata(); err != nil || md.Contains("invisible", 1) {
		// Invisible message
		return false
	}
	localUserID := n.SettingsService.ActiveArborIdentityID()
	if localUserID == nil {
		return false
	}
	localUserNode, has, err := n.ArborService.Store().GetIdentity(localUserID)
	if err != nil || !has {
		return false
	}

	twigData, err := reply.TwigMetadata()
	if err != nil {
		log.Printf("Error checking whether to notify while parsing twig metadata for node %s", reply.ID())
	} else {
		if twigData.Contains("invisible", 1) {
			return false
		}
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
	parent, known, err := n.ArborService.Store().Get(reply.ParentID())
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

// Notify sends a notification with the given title and content if
// notifications are currently allowed.
func (n *notificationManager) Notify(title, content string) error {
	if !n.SettingsService.NotificationsGloballyAllowed() {
		return nil
	}
	_, err := n.CreateNotification(title, content)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	return nil
}

// handleNode spawns a worker goroutine to decide whether or not
// to notify for a given node. This makes it appropriate as a subscriber
// function on a store.ExtendedStore, as it will not block.
func (n *notificationManager) handleNode(node forest.Node) {
	if asReply, ok := node.(*forest.Reply); ok {
		go func(reply *forest.Reply) {
			if !n.shouldNotify(reply) {
				return
			}
			var title, authorName string
			author, _, err := n.ArborService.Store().GetIdentity(&reply.Author)
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
			err = n.Notify(title, string(reply.Content.Blob))
			if err != nil {
				log.Printf("failed sending notification: %v", err)
			}
		}(asReply)
	}
}
