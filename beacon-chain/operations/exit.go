package operations

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"go.opencensus.io/trace"
)

// HandleValidatorExits processes a validator exit operation.
func (s *Service) HandleValidatorExits(ctx context.Context, message proto.Message) error {
	ctx, span := trace.StartSpan(ctx, "operations.HandleValidatorExits")
	defer span.End()

	exit := message.(*ethpb.VoluntaryExit)
	hash, err := hashutil.HashProto(exit)
	if err != nil {
		return err
	}
	if err := s.beaconDB.SaveVoluntaryExit(ctx, exit); err != nil {
		return err
	}
	log.WithField("hash", fmt.Sprintf("%#x", hash)).Info("Exit request saved in DB")
	return nil
}
