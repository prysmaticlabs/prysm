// Package p2p handles peer-to-peer networking for Ethereum Serenity clients.
//
// There are three types of p2p communications.
//
// 	- Direct: two peer communication
// 	- Floodsub: peer broadcasting to all peers
// 	- Gossipsub: peer broadcasting to localized peers
//
// This communication is abstracted through the Feed, Broadcast, and Send.
//
// Pub/sub topic has a specific message type that is used for that topic.
//
// Read more about gossipsub at https://github.com/vyzo/gerbil-simsub
package p2p

import peer "github.com/libp2p/go-libp2p-peer"

// AnyPeer represents a Peer ID alias for sending to any available peer(s).
const AnyPeer = peer.ID("AnyPeer")

// Use this file for interfaces only!

// Adapter is used to create middleware.
//
// See http://godoc.org/github.com/prysmaticlabs/prysm/shared/p2p#Server.RegisterTopic
type Adapter func(Handler) Handler

// Handler is a callback used in the adapter/middleware stack chain.
//
// See http://godoc.org/github.com/prysmaticlabs/prysm/shared/p2p#Server.RegisterTopic
type Handler func(Message)
