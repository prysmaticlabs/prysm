package sync

import "github.com/prysmaticlabs/prysm/shared/event"

// Notifier interface defines the methods of the service that provides sync updates to consumers.
type Notifier interface {
	SyncFeed() *event.Feed
}
