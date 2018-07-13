package shared

// Service is a struct that can be registered into a ServiceRegistry.  Using a
// ServiceRegistry for easy dependency management. For example, a proposer
// service might depend on a p2p server, a txpool, an smc client, etc, but
// we want this proposer to have the same copy of the p2p server all other
// services use in-memory.
type Service interface {
	// Start spawns any goroutines required by the service.
	Start()
	// Stop terminates all goroutines belonging to the service,
	// blocking until they are all terminated.
	Stop() error
}
