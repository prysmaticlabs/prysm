package eth

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethrpc "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	"math/bits"
	"strconv"
)

const (
	CurrentSyncCommitteeIndex = uint64(54)
	NextSyncCommitteeIndex    = uint64(55)
	FinalizedRootIndex        = uint64(105)
)

type LightClientGenericUpdate interface {
	GetAttestedHeader() *ethpbv1.BeaconBlockHeader
	GetNextSyncCommittee() *SyncCommittee
	GetNextSyncCommitteeBranch() [][]byte
	GetFinalizedHeader() *ethpbv1.BeaconBlockHeader
	SetFinalizedHeader(*ethpbv1.BeaconBlockHeader)
	GetFinalityBranch() [][]byte
	GetSyncAggregate() *ethpbv1.SyncAggregate
	GetSignatureSlot() types.Slot
}

// TODO: move this somewhere common
func FloorLog2(x uint64) int {
	return bits.Len64(uint64(x - 1))
}

func (x *SyncCommittee) Equals(other *SyncCommittee) bool {
	if len(x.Pubkeys) != len(other.Pubkeys) {
		return false
	}
	for i := range x.Pubkeys {
		if !bytes.Equal(x.Pubkeys[i], other.Pubkeys[i]) {
			return false
		}
	}
	return bytes.Equal(x.AggregatePubkey, other.AggregatePubkey)
}

func headerFromJSON(header *ethrpc.BeaconBlockHeaderJson) (*ethpbv1.BeaconBlockHeader, error) {
	slot, err := strconv.ParseUint(header.Slot, 10, 64)
	if err != nil {
		return nil, err
	}
	proposerIndex, err := strconv.ParseUint(header.ProposerIndex, 10, 64)
	if err != nil {
		return nil, err
	}
	return &ethpbv1.BeaconBlockHeader{
		Slot:          types.Slot(slot),
		ProposerIndex: types.ValidatorIndex(proposerIndex),
		ParentRoot:    hexutil.MustDecode(header.ParentRoot),
		StateRoot:     hexutil.MustDecode(header.StateRoot),
		BodyRoot:      hexutil.MustDecode(header.BodyRoot),
	}, nil
}

func syncCommitteeFromJSON(syncCommittee *ethrpc.SyncCommitteeJson) *SyncCommittee {
	pubKeys := make([][]byte, len(syncCommittee.Pubkeys))
	for i, pubKey := range syncCommittee.Pubkeys {
		pubKeys[i] = hexutil.MustDecode(pubKey)
	}
	return &SyncCommittee{
		Pubkeys:         pubKeys,
		AggregatePubkey: hexutil.MustDecode(syncCommittee.AggregatePubkey),
	}
}

func branchFromJSON(branch []string) [][]byte {
	branchBytes := [][]byte{}
	for _, root := range branch {
		branchBytes = append(branchBytes, hexutil.MustDecode(root))
	}
	return branchBytes
}

func trustedBlockRoot(bootstrap *LightClientBootstrap) ([32]byte, error) {
	return bootstrap.Header.HashTreeRoot()
}

func NewLightClientBootstrapFromJSON(bootstrap *ethrpc.LightClientBootstrapJson) (*LightClientBootstrap, error) {
	header, err := headerFromJSON(bootstrap.Header)
	if err != nil {
		return nil, err
	}
	return &LightClientBootstrap{
		Header:                     header,
		CurrentSyncCommittee:       syncCommitteeFromJSON(bootstrap.CurrentSyncCommittee),
		CurrentSyncCommitteeBranch: branchFromJSON(bootstrap.CurrentSyncCommitteeBranch),
	}, nil
}

func NewLightClientUpdateFromJSON(update *ethrpc.LightClientUpdateDataJson) (*LightClientUpdate, error) {
	attestedHeader, err := headerFromJSON(update.AttestedHeader)
	if err != nil {
		return nil, err
	}
	finalizedHeader, err := headerFromJSON(update.FinalizedHeader)
	if err != nil {
		return nil, err
	}
	signatureSlot, err := strconv.ParseUint(update.SignatureSlot, 10, 64)
	return &LightClientUpdate{
		AttestedHeader:          attestedHeader,
		NextSyncCommittee:       syncCommitteeFromJSON(update.NextSyncCommittee),
		NextSyncCommitteeBranch: branchFromJSON(update.NextSyncCommitteeBranch),
		FinalizedHeader:         finalizedHeader,
		FinalityBranch:          branchFromJSON(update.FinalityBranch),
		SyncAggregate:           ethpbv1.NewSyncAggregateFromJSON(update.SyncAggregate),
		SignatureSlot:           types.Slot(signatureSlot),
	}, nil
}
