package main

import (
	"fmt"
	"log"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/niotify"
)

type NotificationManager struct {
	*AppState
	niotify.Manager
}

func NewNotificationManager(state *AppState) (*NotificationManager, error) {
	mgr, err := niotify.NewManager()
	if err != nil {
		return nil, err
	}

	return &NotificationManager{
		AppState: state,
		Manager:  mgr,
	}, nil
}

func (n *NotificationManager) ShouldNotify(reply *forest.Reply) bool {
	if reply.Author.Equals(n.AppState.Settings.ActiveIdentity) {
		// Do not send notifications for replies created by the local
		// user's identity.
		return false
	}
	if reply.TreeDepth() == 1 {
		// Notify of new conversation
		return true
	}
	parent, known, err := n.AppState.ArborState.SubscribableStore.Get(reply.ParentID())
	if err != nil || !known {
		// Don't notify if we don't know about this conversation.
		return false
	}
	if parent.(*forest.Reply).Author.Equals(n.AppState.Settings.ActiveIdentity) {
		// Direct response to local user.
		return true
	}
	return false
}

func (n *NotificationManager) HandleNode(node forest.Node) {
	if asReply, ok := node.(*forest.Reply); ok {
		go func(reply *forest.Reply) {
			if !n.ShouldNotify(reply) {
				return
			}
			var title, authorName string
			author, _, err := n.AppState.SubscribableStore.GetIdentity(&reply.Author)
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
