package shared

import (
	"fmt"
	"log"
	"reflect"
)

// ServiceRegistry provides a useful pattern for managing services attached to a struct.
// This can be used for managing routines that are part of items such as a
// beacon chain node, or a sharding client. That is, it allows for ease of dependency
// management and ensures services dependent on others use the same references in memory.
//
// A beacon chain node, for example, has various services including p2p, proposers, attesters,
// and more. Proposers depend on p2p, so upon registering a proposer service, a p2p service
// can be fetched from the service registry and passed into the p2p constructor.
// See beacon-chain/node/node.go for examples.
type ServiceRegistry struct {
	services     map[reflect.Type]Service // map of types to services.
	serviceTypes []reflect.Type           // keep an ordered slice of registered service types.
}

// NewServiceRegistry starts a registry instance for convenience
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[reflect.Type]Service),
	}
}

// StartAll initialized each service in order of registration.
func (s *ServiceRegistry) StartAll() {
	for _, kind := range s.serviceTypes {
		s.services[kind].Start()
	}
}

// StopAll ends every service, logging a panic if any of them fail to stop.
func (s *ServiceRegistry) StopAll() {
	for kind, service := range s.services {
		if err := service.Stop(); err != nil {
			log.Panicf("Could not stop the following service: %v, %v", kind, err)
		}
	}
}

// RegisterService appends a service constructor function to the service
// registry.
func (s *ServiceRegistry) RegisterService(service Service) error {
	kind := reflect.TypeOf(service)
	if _, exists := s.services[kind]; exists {
		return fmt.Errorf("service already exists: %v", kind)
	}
	s.services[kind] = service
	s.serviceTypes = append(s.serviceTypes, kind)
	return nil
}

// FetchService takes in a struct pointer and sets the value of that pointer
// to a service currently stored in the service registry. This ensures the input argument is
// set to the right pointer that refers to the originally registered service.
func (s *ServiceRegistry) FetchService(service interface{}) error {
	if reflect.TypeOf(service).Kind() != reflect.Ptr {
		return fmt.Errorf("input must be of pointer type, received value type instead: %T", service)
	}
	element := reflect.ValueOf(service).Elem()
	if running, ok := s.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return fmt.Errorf("unknown service: %T", service)
}
