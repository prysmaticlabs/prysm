package blockchain

// SendNewBlobEvent sends a message to the BlobNotifier channel that the blob
// for the blocroot `root` is ready in the database
func (s *Service) SendNewBlobEvent(root [32]byte, index uint64) {
	s.blobNotifiers.forRoot(root) <- index
}
