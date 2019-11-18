package operations

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// IncomingProcessedBlockFeed returns a feed that any service can send incoming p2p beacon blocks into.
// The beacon block operation pool service will subscribe to this feed in order to receive incoming beacon blocks.
func (s *Service) IncomingProcessedBlockFeed() *event.Feed {
	return s.incomingProcessedBlockFeed
}

func (s *Service) handleProcessedBlock(ctx context.Context, message proto.Message) error {
	ctx, span := trace.StartSpan(ctx, "operations.handleProcessedBlock")
	defer span.End()

	block := message.(*ethpb.BeaconBlock)
	// Removes the attestations from the pool that have been included
	// in the received block.
	if err := s.removeAttestationsFromPool(ctx, block.Body.Attestations); err != nil {
		return errors.Wrap(err, "could not remove processed attestations from DB")
	}
	s.recentAttestationBitlist.Prune(block.Slot)

	for i, att := range block.Body.Attestations {
		root, err := ssz.HashTreeRoot(att.Data)
		if err != nil {
			return err
		}
		log.WithFields(logrus.Fields{
			"index":            i,
			"root":             fmt.Sprintf("%#x", root),
			"aggregation_bits": fmt.Sprintf("%08b", att.AggregationBits.Bytes()),
			"committeeIndex":   att.Data.Index,
		}).Debug("block attestation")
	}
	return nil
}

// removeAttestationsFromPool removes a list of attestations from the DB
// after they have been included in a beacon block.
func (s *Service) removeAttestationsFromPool(ctx context.Context, attestations []*ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "operations.removeAttestationsFromPool")
	defer span.End()

	s.attestationPoolLock.Lock()
	defer s.attestationPoolLock.Unlock()

	for _, attestation := range attestations {
		root, err := ssz.HashTreeRoot(attestation.Data)
		if err != nil {
			return err
		}

		// TODO(1428): Update this to attestation.Slot.
		// References upstream issue https://github.com/ethereum/eth2.0-specs/pull/1428
		slot := helpers.StartSlot(attestation.Data.Target.Epoch)
		s.recentAttestationBitlist.Insert(slot, root, attestation.AggregationBits)

		log = log.WithField("root", fmt.Sprintf("%#x", root))

		ac, ok := s.attestationPool[root]
		if ok {
			atts := ac.ToAttestations()
			for i, att := range atts {
				if s.recentAttestationBitlist.Contains(root, att.AggregationBits) {
					log.Debug("deleting attestation")
					if i < len(atts)-1 {
						copy(atts[i:], atts[i+1:])
					}
					atts[len(atts)-1] = nil
					atts = atts[:len(atts)-1]
				}
			}

			if len(atts) == 0 {
				delete(s.attestationPool, root)
				continue
			}

			s.attestationPool[root] = dbpb.NewContainerFromAttestations(atts)
		} else {
			log.Debug("No attestations found with this root.")
		}
	}
	return nil
}
