package sync

import (
	"context"
	"fmt"
	"strings"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	prysmTime "github.com/prysmaticlabs/prysm/v5/time"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

func (s *Service) validateDataColumn(ctx context.Context, pid peer.ID, msg *pubsub.Message) (pubsub.ValidationResult, error) {
	receivedTime := prysmTime.Now()

	if pid == s.cfg.p2p.PeerID() {
		return pubsub.ValidationAccept, nil
	}
	if s.cfg.initialSync.Syncing() {
		return pubsub.ValidationIgnore, nil
	}
	if msg.Topic == nil {
		return pubsub.ValidationReject, errInvalidTopic
	}
	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Error("Failed to decode message")
		return pubsub.ValidationReject, err
	}

	ds, ok := m.(*eth.DataColumnSidecar)
	if !ok {
		log.WithField("message", m).Error("Message is not of type *eth.DataColumnSidecar")
		return pubsub.ValidationReject, errWrongMessage
	}
	if ds.ColumnIndex >= params.BeaconConfig().NumberOfColumns {
		return pubsub.ValidationReject, errors.Errorf("invalid column index provided, got %d", ds.ColumnIndex)
	}
	want := fmt.Sprintf("data_column_sidecar_%d", computeSubnetForColumnSidecar(ds.ColumnIndex))
	if !strings.Contains(*msg.Topic, want) {
		log.Debug("Column Sidecar index does not match topic")
		return pubsub.ValidationReject, fmt.Errorf("wrong topic name: %s", *msg.Topic)
	}
	if err := slots.VerifyTime(uint64(s.cfg.clock.GenesisTime().Unix()), ds.SignedBlockHeader.Header.Slot, params.BeaconConfig().MaximumGossipClockDisparityDuration()); err != nil {
		log.WithError(err).Debug("Ignored sidecar: could not verify slot time")
		return pubsub.ValidationIgnore, nil
	}
	cp := s.cfg.chain.FinalizedCheckpt()
	startSlot, err := slots.EpochStart(cp.Epoch)
	if err != nil {
		log.WithError(err).Debug("Ignored column sidecar: could not calculate epoch start slot")
		return pubsub.ValidationIgnore, nil
	}
	if startSlot >= ds.SignedBlockHeader.Header.Slot {
		err := fmt.Errorf("finalized slot %d greater or equal to block slot %d", startSlot, ds.SignedBlockHeader.Header.Slot)
		log.Debug(err)
		return pubsub.ValidationIgnore, err
	}
	// Handle sidecar when the parent is unknown.
	if !s.cfg.chain.HasBlock(ctx, [32]byte(ds.SignedBlockHeader.Header.ParentRoot)) {
		err := errors.Errorf("unknown parent for data column sidecar with slot %d and parent root %#x", ds.SignedBlockHeader.Header.Slot, ds.SignedBlockHeader.Header.ParentRoot)
		log.WithError(err).Debug("Could not identify parent for data column sidecar")
		return pubsub.ValidationIgnore, err
	}
	if s.hasBadBlock([32]byte(ds.SignedBlockHeader.Header.ParentRoot)) {
		bRoot, err := ds.SignedBlockHeader.Header.HashTreeRoot()
		if err != nil {
			return pubsub.ValidationIgnore, err
		}
		s.setBadBlock(ctx, bRoot)
		return pubsub.ValidationReject, errors.Errorf("column sidecar with bad parent provided")
	}
	parentSlot, err := s.cfg.chain.RecentBlockSlot([32]byte(ds.SignedBlockHeader.Header.ParentRoot))
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	if ds.SignedBlockHeader.Header.Slot <= parentSlot {
		return pubsub.ValidationReject, errors.Errorf("invalid column sidecar slot: %d", ds.SignedBlockHeader.Header.Slot)
	}
	if !s.cfg.chain.InForkchoice([32]byte(ds.SignedBlockHeader.Header.ParentRoot)) {
		return pubsub.ValidationReject, blockchain.ErrNotDescendantOfFinalized
	}
	// TODO Verify KZG inclusion proof of data column sidecar

	// TODO Verify KZG proofs of column sidecar

	startTime, err := slots.ToTime(uint64(s.cfg.chain.GenesisTime().Unix()), ds.SignedBlockHeader.Header.Slot)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}

	sinceSlotStartTime := receivedTime.Sub(startTime)
	validationTime := s.cfg.clock.Now().Sub(receivedTime)
	fields["sinceSlotStartTime"] = sinceSlotStartTime
	fields["validationTime"] = validationTime
	log.WithFields(fields).Debug("Received blob sidecar gossip")

	msg.ValidatorData = vBlobData

	return pubsub.ValidationAccept, nil
}

func computeSubnetForColumnSidecar(colIdx uint64) uint64 {
	return colIdx % params.BeaconConfig().DataColumnSidecarSubnetCount
}
