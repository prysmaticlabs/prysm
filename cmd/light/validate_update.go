package main

import (
	"math"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/crypto/bls/blst"
	"github.com/prysmaticlabs/prysm/crypto/bls/common"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	v2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
	"google.golang.org/protobuf/proto"
)

func validateLightClientUpdate(
	snapshot *LightClientSnapshot,
	update *LightClientUpdate,
	genesisValidatorsRoot [32]byte,
) error {
	if update.Header.Slot <= snapshot.Header.Slot {
		return errors.New("wrong")
	}
	snapshotPeriod := slots.ToEpoch(snapshot.Header.Slot) / params.BeaconConfig().EpochsPerSyncCommitteePeriod
	updatePeriod := slots.ToEpoch(update.Header.Slot) / params.BeaconConfig().EpochsPerSyncCommitteePeriod
	if updatePeriod != snapshotPeriod || updatePeriod != snapshotPeriod+1 {
		return errors.New("unwanted")
	}

	// Verify finality headers.
	var signedHeader *v1.BeaconBlockHeader
	if isEmptyBlockHeader(update.FinalityHeader) {
		signedHeader = update.Header
		// Check if branch is empty.
		for _, elem := range update.FinalityBranch {
			if elem != [32]byte{} {
				return errors.New("branch not empty")
			}
		}
	} else {
		leaf, err := update.Header.HashTreeRoot()
		if err != nil {
			return err
		}
		depth := FinalizedRootIndexFloorLog2
		index := getSubtreeIndex(FinalizedRootIndex)
		root := update.FinalityHeader.StateRoot
		merkleBranch := make([][]byte, len(update.FinalityBranch))
		for i, item := range update.FinalityBranch {
			merkleBranch[i] = item[:]
		}
		if !trie.VerifyMerkleBranch(root, leaf[:], int(index), merkleBranch, uint64(depth)) {
			return errors.New("does not verify")
		}
	}

	// Verify update next sync committee if the update period incremented.
	var syncCommittee *v2.SyncCommittee
	if updatePeriod == snapshotPeriod {
		syncCommittee = snapshot.CurrentSyncCommittee
		for _, elem := range update.NextSyncCommitteeBranch {
			if elem != [32]byte{} {
				return errors.New("branch not empty")
			}
		}
	} else {
		syncCommittee = snapshot.NextSyncCommittee
		v1Sync := &v1alpha1.SyncCommittee{
			Pubkeys:         syncCommittee.Pubkeys,
			AggregatePubkey: syncCommittee.AggregatePubkey,
		}
		leaf, err := v1Sync.HashTreeRoot()
		if err != nil {
			return err
		}
		depth := NextSyncCommitteeIndexFloorLog2
		index := getSubtreeIndex(NextSyncCommitteeIndex)
		root := update.Header.StateRoot
		merkleBranch := make([][]byte, len(update.NextSyncCommitteeBranch))
		for i, item := range update.NextSyncCommitteeBranch {
			merkleBranch[i] = item[:]
		}
		if !trie.VerifyMerkleBranch(root, leaf[:], int(index), merkleBranch, uint64(depth)) {
			return errors.New("does not verify")
		}
	}

	// Verify sync committee has sufficient participants
	if update.SyncCommitteeBits.Count() < params.BeaconConfig().MinSyncCommitteeParticipants {
		return errors.New("insufficient participants")
	}

	// Verify sync committee aggregate signature
	participantPubkeys := make([][]byte, 0)
	for i, pubKey := range syncCommittee.Pubkeys {
		bit := update.SyncCommitteeBits.BitAt(uint64(i))
		if bit {
			participantPubkeys = append(participantPubkeys, pubKey)
		}
	}
	domain, err := signing.ComputeDomain(
		params.BeaconConfig().DomainSyncCommittee,
		[]byte(update.ForkVersion.Version),
		genesisValidatorsRoot[:],
	)
	if err != nil {
		return err
	}
	signingRoot, err := signing.ComputeSigningRoot(signedHeader, domain)
	if err != nil {
		return err
	}
	sig, err := blst.SignatureFromBytes(update.SyncCommitteeSignature[:])
	if err != nil {
		return err
	}
	pubKeys := make([]common.PublicKey, 0)
	for _, pubkey := range participantPubkeys {
		pk, err := blst.PublicKeyFromBytes(pubkey)
		if err != nil {
			return err
		}
		pubKeys = append(pubKeys, pk)
	}
	if !sig.FastAggregateVerify(pubKeys, signingRoot) {
		return errors.New("failed to verify")
	}
	return nil
}

func isEmptyBlockHeader(header *v1.BeaconBlockHeader) bool {
	emptyRoot := params.BeaconConfig().ZeroHash
	return proto.Equal(header, &v1.BeaconBlockHeader{
		Slot:          0,
		ProposerIndex: 0,
		ParentRoot:    emptyRoot[:],
		StateRoot:     emptyRoot[:],
		BodyRoot:      emptyRoot[:],
	})
}

func getSubtreeIndex(index uint64) uint64 {
	return index % uint64(math.Pow(2, floorLog2(index)))
}

func floorLog2(x uint64) float64 {
	return math.Floor(math.Log2(float64(x)))
}
