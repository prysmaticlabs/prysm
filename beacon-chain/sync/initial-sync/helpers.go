package initialsync

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

const noMsgData = "message contains no data"

func (s *InitialSync) checkBlockValidity(ctx context.Context, block *pb.BeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.sync.initial-sync.checkBlockValidity")
	defer span.End()
	beaconState, err := s.db.HeadState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get beacon state: %v", err)
	}

	if block.Slot < beaconState.FinalizedEpoch*params.BeaconConfig().SlotsPerEpoch {
		return errors.New(debugError + "discarding received block with a slot number smaller than the last finalized slot")
	}
	// Attestation from proposer not verified as, other nodes only store blocks not proposer
	// attestations.
	return nil
}

func (s *InitialSync) doesParentExist(block *pb.BeaconBlock) bool {
	parentHash := bytesutil.ToBytes32(block.ParentRootHash32)
	return s.db.HasBlock(parentHash)
}

// safelyHandleMessage will recover and log any panic that occurs from the
// function argument.
func safelyHandleMessage(fn func(p2p.Message), msg p2p.Message) {
	defer func() {
		if r := recover(); r != nil {
			printedMsg := noMsgData
			if msg.Data != nil {
				printedMsg = proto.MarshalTextString(msg.Data)
			}
			log.WithFields(logrus.Fields{
				"r":   r,
				"msg": printedMsg,
			}).Error("Panicked when handling p2p message! Recovering...")

			debug.PrintStack()

			if msg.Ctx == nil {
				return
			}
			if span := trace.FromContext(msg.Ctx); span != nil {
				span.SetStatus(trace.Status{
					Code:    trace.StatusCodeInternal,
					Message: fmt.Sprintf("Panic: %v", r),
				})
			}
		}
	}()

	// Fingers crossed that it doesn't panic...
	fn(msg)
}
