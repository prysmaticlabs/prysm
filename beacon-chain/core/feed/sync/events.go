// Package sync contains types for sync specific operations such as subnet
// subscriptions.
package sync

const (
	// SubscribedToSubnet is sent after an attached validator subscribes to a subnet.
	SubscribedToSubnet = iota + 1
	// UnSubscribeFromSubnet is sent when a validator unsubscribes from a subnet.
	UnSubscribeFromSubnet
)

// SubnetSubscribe describes the subscription event.
type SubnetSubscribe struct {
	// Subnet that has been subscribed to by an attached validator.
	Subnet uint64
}

// SubnetUnsubscribe describes the unsubscribe event.
type SubnetUnsubscribe struct {
	// Subnet that has been unsubscribed to by an attached validator.
	Subnet uint64
}
