package shared

import (
	"errors"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

type mockService struct {
	status error
}
type secondMockService struct {
	status error
}

func (m *mockService) Start() {
}

func (m *mockService) Stop() error {
	return nil
}

func (m *mockService) Status() error {
	return m.status
}

func (s *secondMockService) Start() {
}

func (s *secondMockService) Stop() error {
	return nil
}

func (s *secondMockService) Status() error {
	return s.status
}

func TestRegisterService_Twice(t *testing.T) {
	registry := &ServiceRegistry{
		services: make(map[reflect.Type]Service),
	}

	m := &mockService{}
	if err := registry.RegisterService(m); err != nil {
		t.Fatalf("failed to register first service")
	}

	// Checks if first service was indeed registered.
	require.Equal(t, 1, len(registry.serviceTypes))

	if err := registry.RegisterService(m); err == nil {
		t.Errorf("should not be able to register a service twice, got nil error")
	}
}

func TestRegisterService_Different(t *testing.T) {
	registry := &ServiceRegistry{
		services: make(map[reflect.Type]Service),
	}

	m := &mockService{}
	s := &secondMockService{}
	if err := registry.RegisterService(m); err != nil {
		t.Fatalf("failed to register first service")
	}

	if err := registry.RegisterService(s); err != nil {
		t.Fatalf("failed to register second service")
	}

	require.Equal(t, 2, len(registry.serviceTypes))

	if _, exists := registry.services[reflect.TypeOf(m)]; !exists {
		t.Errorf("service of type %v not registered", reflect.TypeOf(m))
	}

	if _, exists := registry.services[reflect.TypeOf(s)]; !exists {
		t.Errorf("service of type %v not registered", reflect.TypeOf(s))
	}
}

func TestFetchService_OK(t *testing.T) {
	registry := &ServiceRegistry{
		services: make(map[reflect.Type]Service),
	}

	m := &mockService{}
	if err := registry.RegisterService(m); err != nil {
		t.Fatalf("failed to register first service")
	}

	if err := registry.FetchService(*m); err == nil {
		t.Errorf("passing in a value should throw an error, received nil error")
	}

	var s *secondMockService
	if err := registry.FetchService(&s); err == nil {
		t.Errorf("fetching an unregistered service should return an error, got nil")
	}

	var m2 *mockService
	if err := registry.FetchService(&m2); err != nil {
		t.Fatalf("failed to fetch service")
	}

	require.Equal(t, m, m2)
}

func TestServiceStatus_OK(t *testing.T) {
	registry := &ServiceRegistry{
		services: make(map[reflect.Type]Service),
	}

	m := &mockService{}
	if err := registry.RegisterService(m); err != nil {
		t.Fatalf("failed to register first service")
	}

	s := &secondMockService{}
	if err := registry.RegisterService(s); err != nil {
		t.Fatalf("failed to register first service")
	}

	m.status = errors.New("something bad has happened")
	s.status = errors.New("woah, horsee")

	statuses := registry.Statuses()

	mStatus := statuses[reflect.TypeOf(m)]
	if mStatus == nil || mStatus.Error() != "something bad has happened" {
		t.Errorf("Received unexpected status for %T = %v", m, mStatus)
	}

	sStatus := statuses[reflect.TypeOf(s)]
	if sStatus == nil || sStatus.Error() != "woah, horsee" {
		t.Errorf("Received unexpected status for %T = %v", s, sStatus)
	}
}
