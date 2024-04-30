package blockchain

import (
	"context"

	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

func (s *Service) ReceiveDataColumn(ctx context.Context, ds *ethpb.DataColumnSidecar) error {
	if err := s.blobStorage.SaveDataColumn(ds); err != nil {
		return err
	}
	hRoot, err := ds.SignedBlockHeader.Header.HashTreeRoot()
	if err != nil {
		return err
	}

	// TODO use a custom event or new method of for data columns. For speed
	// we are simply reusing blob paths here.
	s.sendNewBlobEvent(hRoot, uint64(ds.SignedBlockHeader.Header.Slot))
	return nil
}
