package blockchain

import (
	"context"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
)

func (s *Service) ReceiveDataColumn(ctx context.Context, ds blocks.VerifiedRODataColumn) error {
	if err := s.blobStorage.SaveDataColumn(ds); err != nil {
		return err
	}

	s.blobDataColumnNotifier.receiveBlobDataColumn(ds.BlockRoot(), ds.ColumnIndex)
	return nil
}
