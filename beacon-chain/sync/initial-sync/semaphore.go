package initialsync

// is a synchonization primitive for sync and initial-sync to coordinate control over the blockchain package.
// Whichever service currently holds the Semaphore has exclusive writes to interact with blockchain.
// If initial-sync has the Semaphore, sync should pause, and vice versa.
type Status struct {
}
