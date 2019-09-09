package testing

// Sync defines a mock for the sync service.
type Sync struct {
	IsSyncing bool
}

// Syncing --
func (s *Sync) Syncing() bool {
	return s.IsSyncing
}

// Status --
func (s *Sync) Status() error {
	return nil
}
