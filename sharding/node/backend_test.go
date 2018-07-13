package node

import (
	"flag"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/geth-sharding/sharding/params"
	"github.com/prysmaticlabs/geth-sharding/sharding/types"

	"github.com/urfave/cli"
)

// Verifies that ShardEthereum implements the Node interface.
var _ = types.Node(&ShardEthereum{})

type mockService struct{}
type secondMockService struct{}

func (m *mockService) Start() {
	return
}

func (m *mockService) Stop() error {
	return nil
}

func (s *secondMockService) Start() {
	return
}

func (s *secondMockService) Stop() error {
	return nil
}

// Test that the sharding node can build with default flag values.
func TestNode_Builds(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)

	context := cli.NewContext(app, set, nil)

	_, err := New(context)
	if err != nil {
		t.Fatalf("Failed to create ShardEthereum: %v", err)
	}
}

func TestRegisterServiceTwice(t *testing.T) {
	shardEthereum := &ShardEthereum{
		services: make(map[reflect.Type]types.Service),
		stop:     make(chan struct{}),
	}

	// Configure shardConfig by loading the default.
	shardEthereum.shardConfig = params.DefaultConfig

	m := &mockService{}
	if err := shardEthereum.registerService(m); err != nil {
		t.Fatalf("failed to register first service")
	}

	// checks if first service was indeed registered
	if len(shardEthereum.serviceTypes) != 1 {
		t.Fatalf("service types slice should contain 1 service, contained %v", len(shardEthereum.serviceTypes))
	}

	if err := shardEthereum.registerService(m); err == nil {
		t.Errorf("should not be able to register a service twice, got nil error")
	}
}

func TestRegisterDifferentServices(t *testing.T) {
	shardEthereum := &ShardEthereum{
		services: make(map[reflect.Type]types.Service),
		stop:     make(chan struct{}),
	}

	// Configure shardConfig by loading the default.
	shardEthereum.shardConfig = params.DefaultConfig

	m := &mockService{}
	s := &secondMockService{}
	if err := shardEthereum.registerService(m); err != nil {
		t.Fatalf("failed to register first service")
	}

	if err := shardEthereum.registerService(s); err != nil {
		t.Fatalf("failed to register second service")
	}

	if len(shardEthereum.serviceTypes) != 2 {
		t.Errorf("service types slice should contain 2 services, contained %v", len(shardEthereum.serviceTypes))
	}

	if _, exists := shardEthereum.services[reflect.TypeOf(m)]; !exists {
		t.Errorf("service of type %v not registered", reflect.TypeOf(m))
	}

	if _, exists := shardEthereum.services[reflect.TypeOf(s)]; !exists {
		t.Errorf("service of type %v not registered", reflect.TypeOf(s))
	}
}

func TestFetchService(t *testing.T) {
	shardEthereum := &ShardEthereum{
		services: make(map[reflect.Type]types.Service),
		stop:     make(chan struct{}),
	}

	// Configure shardConfig by loading the default.
	shardEthereum.shardConfig = params.DefaultConfig

	m := &mockService{}
	if err := shardEthereum.registerService(m); err != nil {
		t.Fatalf("failed to register first service")
	}

	if err := shardEthereum.fetchService(*m); err == nil {
		t.Errorf("passing in a value should throw an error, received nil error")
	}

	var s *secondMockService
	if err := shardEthereum.fetchService(&s); err == nil {
		t.Errorf("fetching an unregistered service should return an error, got nil")
	}

	var m2 *mockService
	if err := shardEthereum.fetchService(&m2); err != nil {
		t.Fatalf("failed to fetch service")
	}

	if m2 != m {
		t.Errorf("pointers were not equal, instead got %p, %p", m2, m)
	}
}
