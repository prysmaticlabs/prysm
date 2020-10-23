package shared

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

type mockService struct {
	status error
}
type secondMockService struct {
	status error
}

func (m *mockService) Start(ctx context.Context) {
}

func (m *mockService) Stop(ctx context.Context) error {
	return nil
}

func (m *mockService) Status() error {
	return m.status
}

func (s *secondMockService) Start(ctx context.Context) {
}

func (s *secondMockService) Stop(ctx context.Context) error {
	return nil
}

func (s *secondMockService) Status() error {
	return s.status
}

func TestRegisterService_Twice(t *testing.T) {
	registry := &ServiceRegistry{
		contexts: make(map[reflect.Type]*ServiceContext),
		services: make(map[reflect.Type]Service),
	}

	m := &mockService{}
	ctx, cancel := context.WithCancel(context.Background())
	serviceCtx := &ServiceContext{Ctx: ctx, Cancel: cancel}
	require.NoError(t, registry.RegisterService(m, serviceCtx), "Failed to register first service")

	// Checks if first service was indeed registered.
	require.Equal(t, 1, len(registry.serviceTypes))
	assert.ErrorContains(t, "service already exists", registry.RegisterService(m, serviceCtx))
}

func TestRegisterService_Different(t *testing.T) {
	registry := &ServiceRegistry{
		contexts: make(map[reflect.Type]*ServiceContext),
		services: make(map[reflect.Type]Service),
	}

	m := &mockService{}
	s := &secondMockService{}
	ctx, cancel := context.WithCancel(context.Background())
	serviceCtx := &ServiceContext{Ctx: ctx, Cancel: cancel}
	require.NoError(t, registry.RegisterService(m, serviceCtx), "Failed to register first service")
	require.NoError(t, registry.RegisterService(s, serviceCtx), "Failed to register second service")

	require.Equal(t, 2, len(registry.serviceTypes))

	_, exists := registry.services[reflect.TypeOf(m)]
	assert.Equal(t, true, exists, "service of type %v not registered", reflect.TypeOf(m))

	_, exists = registry.services[reflect.TypeOf(s)]
	assert.Equal(t, true, exists, "service of type %v not registered", reflect.TypeOf(s))
}

func TestFetchService_OK(t *testing.T) {
	registry := &ServiceRegistry{
		contexts: make(map[reflect.Type]*ServiceContext),
		services: make(map[reflect.Type]Service),
	}

	m := &mockService{}
	ctx, cancel := context.WithCancel(context.Background())
	serviceCtx := &ServiceContext{Ctx: ctx, Cancel: cancel}
	require.NoError(t, registry.RegisterService(m, serviceCtx), "Failed to register first service")

	assert.ErrorContains(t, "input must be of pointer type, received value type instead", registry.FetchService(*m))

	var s *secondMockService
	assert.ErrorContains(t, "unknown service", registry.FetchService(&s))

	var m2 *mockService
	require.NoError(t, registry.FetchService(&m2), "Failed to fetch service")
	require.Equal(t, m, m2)
}

func TestServiceStatus_OK(t *testing.T) {
	registry := &ServiceRegistry{
		contexts: make(map[reflect.Type]*ServiceContext),
		services: make(map[reflect.Type]Service),
	}

	ctx, cancel := context.WithCancel(context.Background())
	serviceCtx := &ServiceContext{Ctx: ctx, Cancel: cancel}

	m := &mockService{}
	require.NoError(t, registry.RegisterService(m, serviceCtx), "Failed to register first service")

	s := &secondMockService{}
	require.NoError(t, registry.RegisterService(s, serviceCtx), "Failed to register first service")

	m.status = errors.New("something bad has happened")
	s.status = errors.New("woah, horsee")

	statuses := registry.Statuses()

	assert.ErrorContains(t, "something bad has happened", statuses[reflect.TypeOf(m)])
	assert.ErrorContains(t, "woah, horsee", statuses[reflect.TypeOf(s)])
}
