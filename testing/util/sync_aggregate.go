package util

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	p2pType "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

func generateSyncAggregate(bState state.BeaconState, privs []bls.SecretKey, parentRoot [32]byte) (*ethpb.SyncAggregate, error) {
	st, ok := bState.(state.BeaconStateAltair)
	if !ok || bState.Version() == version.Phase0 {
		return nil, errors.Errorf("state cannot be asserted to altair state")
	}
	nextSlotEpoch := time.SlotToEpoch(st.Slot() + 1)
	currEpoch := time.SlotToEpoch(st.Slot())

	var syncCommittee *ethpb.SyncCommittee
	var err error
	if time.SyncCommitteePeriod(currEpoch) == time.SyncCommitteePeriod(nextSlotEpoch) {
		syncCommittee, err = st.CurrentSyncCommittee()
		if err != nil {
			return nil, err
		}
	} else {
		syncCommittee, err = st.NextSyncCommittee()
		if err != nil {
			return nil, err
		}
	}
	sigs := make([]bls.Signature, 0, len(syncCommittee.Pubkeys))
	var bVector []byte
	currSize := new(ethpb.SyncAggregate).SyncCommitteeBits.Len()
	switch currSize {
	case 512:
		bVector = bitfield.NewBitvector512()
	case 32:
		bVector = bitfield.NewBitvector32()
	default:
		return nil, errors.New("invalid bit vector size")
	}

	for i, p := range syncCommittee.Pubkeys {
		idx, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(p))
		if !ok {
			continue
		}
		d, err := signing.Domain(st.Fork(), time.SlotToEpoch(st.Slot()), params.BeaconConfig().DomainSyncCommittee, st.GenesisValidatorRoot())
		if err != nil {
			return nil, err
		}
		sszBytes := p2pType.SSZBytes(parentRoot[:])
		r, err := signing.ComputeSigningRoot(&sszBytes, d)
		if err != nil {
			return nil, err
		}
		sigs = append(sigs, privs[idx].Sign(r[:]))
		if currSize == 512 {
			bitfield.Bitvector512(bVector).SetBitAt(uint64(i), true)
		}
		if currSize == 32 {
			bitfield.Bitvector32(bVector).SetBitAt(uint64(i), true)
		}
	}
	if len(sigs) == 0 {
		fakeSig := [96]byte{0xC0}
		return &ethpb.SyncAggregate{SyncCommitteeSignature: fakeSig[:], SyncCommitteeBits: bVector}, nil
	}
	aggSig := bls.AggregateSignatures(sigs)
	return &ethpb.SyncAggregate{SyncCommitteeSignature: aggSig.Marshal(), SyncCommitteeBits: bVector}, nil
}
