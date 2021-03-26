package app

import "github.com/prysmaticlabs/prysm/shared/event"

// Notifier interface defines the methods of the service that provides application payload instructions to eth1 clients.
type Notifier interface {
	AppFeed() *event.Feed
}
