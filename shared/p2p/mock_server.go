package p2p

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"

	bhost "github.com/libp2p/go-libp2p-blankhost"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
)

// MockServer is a p2p server with a blankhost that can be used
// for testing in other services.
func MockServer(t *testing.T) (*Server, error) {
	ctx := context.Background()
	h := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))

	gsub, err := pubsub.NewFloodSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub: %v", err)
	}

	s := &Server{
		ctx:          ctx,
		gsub:         gsub,
		host:         h,
		feeds:        make(map[reflect.Type]Feed),
		mutex:        &sync.Mutex{},
		topicMapping: make(map[reflect.Type]string),
	}

	return s, nil
}
