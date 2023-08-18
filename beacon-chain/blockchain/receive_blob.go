package blockchain

import fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"

// SendNewBlobEvent sends a message to the BlobNotifier channel that the blob
// for the blocroot `root` is ready in the database
func (s *Service) SendNewBlobEvent(root [32]byte, index uint64) {
	s.blobNotifier.Lock()
	nc, ok := s.blobNotifier.chanForRoot[root]
	if !ok {
		nc = &blobNotifierChan{indices: make(map[uint64]struct{}), channel: make(chan struct{}, fieldparams.MaxBlobsPerBlock)}
		s.blobNotifier.chanForRoot[root] = nc
	}
	_, ok = nc.indices[index]
	if ok {
		s.blobNotifier.Unlock()
		return
	}
	nc.indices[index] = struct{}{}
	s.blobNotifier.Unlock()
	nc.channel <- struct{}{}
}
