package testutil

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// BlockGenConfig is used to define the requested conditions
// for block generation.
type BlockGenConfig struct {
	NumProposerSlashings uint64
	NumAttesterSlashings uint64
	NumAttestations      uint64
	NumDeposits          uint64
	NumVoluntaryExits    uint64
}

// DefaultBlockGenConfig returns the block config that utilizes the
// current params in the beacon config.
func DefaultBlockGenConfig() *BlockGenConfig {
	return &BlockGenConfig{
		NumProposerSlashings: 0,
		NumAttesterSlashings: 0,
		NumAttestations:      1,
		NumDeposits:          0,
		NumVoluntaryExits:    0,
	}
}

// GenerateFullBlock generates a fully valid block with the requested parameters.
// Use BlockGenConfig to declare the conditions you would like the block generated under.
func GenerateFullBlock(
	bState *pb.BeaconState,
	privs []*bls.SecretKey,
	conf *BlockGenConfig,
	slot uint64,
) (*ethpb.BeaconBlock, error) {
	currentSlot := bState.Slot
	if currentSlot > slot {
		return nil, fmt.Errorf("current slot in state is larger than given slot. %d > %d", currentSlot, slot)
	}

	if conf == nil {
		conf = &BlockGenConfig{}
	}

	var err error
	pSlashings := []*ethpb.ProposerSlashing{}
	numToGen := conf.NumProposerSlashings
	if numToGen > 0 {
		pSlashings, err = generateProposerSlashings(bState, privs, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d proposer slashings:", numToGen)
		}
	}

	numToGen = conf.NumAttesterSlashings
	aSlashings := []*ethpb.AttesterSlashing{}
	if numToGen > 0 {
		aSlashings, err = generateAttesterSlashings(bState, privs, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attester slashings:", numToGen)
		}
	}

	numToGen = conf.NumAttestations
	atts := []*ethpb.Attestation{}
	if numToGen > 0 {
		atts, err = GenerateAttestations(bState, privs, numToGen, slot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attestations:", numToGen)
		}
	}

	numToGen = conf.NumDeposits
	newDeposits, eth1Data := []*ethpb.Deposit{}, bState.Eth1Data
	if numToGen > 0 {
		newDeposits, eth1Data, err = generateDepositsAndEth1Data(bState, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d deposits:", numToGen)
		}
	}

	numToGen = conf.NumVoluntaryExits
	exits := []*ethpb.VoluntaryExit{}
	if numToGen > 0 {
		exits, err = generateVoluntaryExits(bState, privs, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attester slashings:", numToGen)
		}
	}

	newHeader := proto.Clone(bState.LatestBlockHeader).(*ethpb.BeaconBlockHeader)
	prevStateRoot, err := ssz.HashTreeRoot(bState)
	if err != nil {
		return nil, err
	}
	newHeader.StateRoot = prevStateRoot[:]
	parentRoot, err := ssz.SigningRoot(newHeader)
	if err != nil {
		return nil, err
	}

	if slot == currentSlot {
		slot = currentSlot + 1
	}

	// Temporarily incrementing the beacon state slot here since BeaconProposerIndex is a
	// function deterministic on beacon state slot.
	bState.Slot = slot
	reveal, err := RandaoReveal(bState, helpers.CurrentEpoch(bState), privs)
	if err != nil {
		return nil, err
	}

	block := &ethpb.BeaconBlock{
		Slot:       slot,
		ParentRoot: parentRoot[:],
		Body: &ethpb.BeaconBlockBody{
			Eth1Data:          eth1Data,
			RandaoReveal:      reveal,
			ProposerSlashings: pSlashings,
			AttesterSlashings: aSlashings,
			Attestations:      atts,
			VoluntaryExits:    exits,
			Deposits:          newDeposits,
		},
	}
	bState.Slot = currentSlot

	signature, err := BlockSignature(bState, block, privs)
	if err != nil {
		return nil, err
	}
	block.Signature = signature.Marshal()

	return block, nil
}

func generateProposerSlashings(
	bState *pb.BeaconState,
	privs []*bls.SecretKey,
	numSlashings uint64,
) ([]*ethpb.ProposerSlashing, error) {
	currentEpoch := helpers.CurrentEpoch(bState)

	proposerSlashings := make([]*ethpb.ProposerSlashing, numSlashings)
	for i := uint64(0); i < numSlashings; i++ {
		proposerIndex, err := randValIndex(bState)
		if err != nil {
			return nil, err
		}
		header1 := &ethpb.BeaconBlockHeader{
			Slot:     bState.Slot,
			BodyRoot: []byte{0, 1, 0},
		}
		root, err := ssz.SigningRoot(header1)
		if err != nil {
			return nil, err
		}
		domain := helpers.Domain(bState.Fork, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
		header1.Signature = privs[proposerIndex].Sign(root[:], domain).Marshal()

		header2 := &ethpb.BeaconBlockHeader{
			Slot:     bState.Slot,
			BodyRoot: []byte{0, 2, 0},
		}
		root, err = ssz.SigningRoot(header2)
		if err != nil {
			return nil, err
		}
		header2.Signature = privs[proposerIndex].Sign(root[:], domain).Marshal()

		slashing := &ethpb.ProposerSlashing{
			ProposerIndex: proposerIndex,
			Header_1:      header1,
			Header_2:      header2,
		}
		proposerSlashings[i] = slashing
	}
	return proposerSlashings, nil
}

func generateAttesterSlashings(
	bState *pb.BeaconState,
	privs []*bls.SecretKey,
	numSlashings uint64,
) ([]*ethpb.AttesterSlashing, error) {
	currentEpoch := helpers.CurrentEpoch(bState)
	attesterSlashings := make([]*ethpb.AttesterSlashing, numSlashings)
	for i := uint64(0); i < numSlashings; i++ {
		committeeIndex := rand.Uint64() % params.BeaconConfig().MaxCommitteesPerSlot
		committee, err := helpers.BeaconCommittee(bState, bState.Slot, committeeIndex)
		if err != nil {
			return nil, err
		}
		committeeSize := uint64(len(committee))
		randIndex := rand.Uint64() % uint64(len(committee))
		valIndex := committee[randIndex]

		aggregationBits := bitfield.NewBitlist(committeeSize)
		aggregationBits.SetBitAt(randIndex, true)
		att1 := &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Slot:           bState.Slot,
				CommitteeIndex: committeeIndex,
				Target: &ethpb.Checkpoint{
					Epoch: currentEpoch,
					Root:  params.BeaconConfig().ZeroHash[:],
				},
				Source: &ethpb.Checkpoint{
					Epoch: currentEpoch + 1,
					Root:  params.BeaconConfig().ZeroHash[:],
				},
			},
			AggregationBits: aggregationBits,
		}
		dataRoot, err := ssz.HashTreeRoot(&pb.AttestationDataAndCustodyBit{
			Data:       att1.Data,
			CustodyBit: false,
		})
		if err != nil {
			return nil, err
		}
		domain := helpers.Domain(bState.Fork, i, params.BeaconConfig().DomainBeaconAttester)
		sig := privs[valIndex].Sign(dataRoot[:], domain)
		att1.Signature = bls.AggregateSignatures([]*bls.Signature{sig}).Marshal()

		att2 := &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Slot:           bState.Slot,
				CommitteeIndex: committeeIndex,
				Target: &ethpb.Checkpoint{
					Epoch: currentEpoch,
					Root:  params.BeaconConfig().ZeroHash[:],
				},
				Source: &ethpb.Checkpoint{
					Epoch: currentEpoch,
					Root:  params.BeaconConfig().ZeroHash[:],
				},
			},
			AggregationBits: aggregationBits,
		}
		dataRoot, err = ssz.HashTreeRoot(&pb.AttestationDataAndCustodyBit{
			Data:       att2.Data,
			CustodyBit: false,
		})
		if err != nil {
			return nil, err
		}
		sig = privs[valIndex].Sign(dataRoot[:], domain)
		att2.Signature = bls.AggregateSignatures([]*bls.Signature{sig}).Marshal()

		indexedAtt1, err := blocks.ConvertToIndexed(context.Background(), att1, committee)
		if err != nil {
			return nil, err
		}
		indexedAtt2, err := blocks.ConvertToIndexed(context.Background(), att2, committee)
		if err != nil {
			return nil, err
		}
		slashing := &ethpb.AttesterSlashing{
			Attestation_1: indexedAtt1,
			Attestation_2: indexedAtt2,
		}
		attesterSlashings[i] = slashing
	}
	return attesterSlashings, nil
}

// GenerateAttestations creates attestations that are entirely valid, for all
// the committees of the current state slot. This function expects attestations
// requested to be cleanly divisible by committees per slot. If there is 1 committee
// in the slot, and numToGen is set to 4, then it will return 4 attestations
// for the same data with their aggregation bits split uniformly.
//
// If you request 4 attestations, but there are 8 committees, you will get 4 fully aggregated attestations.
func GenerateAttestations(
	bState *pb.BeaconState,
	privs []*bls.SecretKey,
	numToGen uint64,
	slot uint64,
) ([]*ethpb.Attestation, error) {
	currentEpoch := helpers.SlotToEpoch(slot)
	attestations := []*ethpb.Attestation{}
	generateHeadState := false
	if slot > bState.Slot {
		// Going back a slot here so there's no inclusion delay issues.
		slot--
		generateHeadState = true
	}

	var err error
	targetRoot := make([]byte, 32)
	headRoot := make([]byte, 32)
	// Only calculate head state if its an attestation for the current slot or future slot.
	if generateHeadState || slot == bState.Slot {
		headState := proto.Clone(bState).(*pb.BeaconState)
		headState, err := state.ProcessSlots(context.Background(), headState, slot+1)
		if err != nil {
			return nil, err
		}
		headRoot, err = helpers.BlockRootAtSlot(headState, slot)
		if err != nil {
			return nil, err
		}
		targetRoot, err = helpers.BlockRoot(headState, currentEpoch)
		if err != nil {
			return nil, err
		}
	} else {
		headRoot, err = helpers.BlockRootAtSlot(bState, slot)
		if err != nil {
			return nil, err
		}
	}

	committeesPerSlot, err := helpers.CommitteeCountAtSlot(bState, slot)
	if err != nil {
		return nil, err
	}

	if numToGen < committeesPerSlot {
		log.Printf(
			"Warning: %d attestations requested is less than %d committees in current slot, not all validators will be attesting.",
			numToGen,
			committeesPerSlot,
		)
	} else if numToGen > committeesPerSlot {
		log.Printf(
			"Warning: %d attestations requested are more than %d committees in current slot, attestations will not be perfectly efficient.",
			numToGen,
			committeesPerSlot,
		)
	}

	attsPerCommittee := math.Max(float64(numToGen/committeesPerSlot), 1)
	if math.Trunc(attsPerCommittee) != attsPerCommittee {
		return nil, fmt.Errorf(
			"requested attestations %d must be easily divisible by committees in slot %d, calculated %f",
			numToGen,
			committeesPerSlot,
			attsPerCommittee,
		)
	}

	domain := helpers.Domain(bState.Fork, currentEpoch, params.BeaconConfig().DomainBeaconAttester)
	for c := uint64(0); c < committeesPerSlot && c < numToGen; c++ {
		committee, err := helpers.BeaconCommittee(bState, slot, c)
		if err != nil {
			return nil, err
		}

		attData := &ethpb.AttestationData{
			Slot:            slot,
			CommitteeIndex:  c,
			BeaconBlockRoot: headRoot,
			Source:          bState.CurrentJustifiedCheckpoint,
			Target: &ethpb.Checkpoint{
				Epoch: currentEpoch,
				Root:  targetRoot,
			},
		}

		dataRoot, err := ssz.HashTreeRoot(&pb.AttestationDataAndCustodyBit{
			Data:       attData,
			CustodyBit: false,
		})
		if err != nil {
			return nil, err
		}

		committeeSize := uint64(len(committee))
		bitsPerAtt := committeeSize / uint64(attsPerCommittee)
		for i := uint64(0); i < committeeSize; i += bitsPerAtt {
			aggregationBits := bitfield.NewBitlist(committeeSize)
			custodyBits := bitfield.NewBitlist(committeeSize)
			sigs := []*bls.Signature{}
			for b := i; b < i+bitsPerAtt; b++ {
				aggregationBits.SetBitAt(b, true)
				sigs = append(sigs, privs[committee[b]].Sign(dataRoot[:], domain))
			}

			att := &ethpb.Attestation{
				Data:            attData,
				AggregationBits: aggregationBits,
				CustodyBits:     custodyBits,
				Signature:       bls.AggregateSignatures(sigs).Marshal(),
			}
			attestations = append(attestations, att)
		}
	}
	return attestations, nil
}

func generateDepositsAndEth1Data(
	bState *pb.BeaconState,
	numDeposits uint64,
) (
	[]*ethpb.Deposit,
	*ethpb.Eth1Data,
	error,
) {
	previousDepsLen := bState.Eth1DepositIndex
	currentDeposits, _, err := DeterministicDepositsAndKeys(previousDepsLen + numDeposits)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get deposits")
	}
	eth1Data, err := DeterministicEth1Data(len(currentDeposits))
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get eth1data")
	}
	return currentDeposits[previousDepsLen:], eth1Data, nil
}

func generateVoluntaryExits(
	bState *pb.BeaconState,
	privs []*bls.SecretKey,
	numExits uint64,
) ([]*ethpb.VoluntaryExit, error) {
	currentEpoch := helpers.CurrentEpoch(bState)

	voluntaryExits := make([]*ethpb.VoluntaryExit, numExits)
	for i := 0; i < len(voluntaryExits); i++ {
		valIndex, err := randValIndex(bState)
		if err != nil {
			return nil, err
		}
		exit := &ethpb.VoluntaryExit{
			Epoch:          helpers.PrevEpoch(bState),
			ValidatorIndex: valIndex,
		}
		root, err := ssz.SigningRoot(exit)
		if err != nil {
			return nil, err
		}
		domain := helpers.Domain(bState.Fork, currentEpoch, params.BeaconConfig().DomainVoluntaryExit)
		exit.Signature = privs[valIndex].Sign(root[:], domain).Marshal()
		voluntaryExits[i] = exit
	}
	return voluntaryExits, nil
}

func randValIndex(bState *pb.BeaconState) (uint64, error) {
	activeCount, err := helpers.ActiveValidatorCount(bState, helpers.CurrentEpoch(bState))
	if err != nil {
		return 0, err
	}
	return rand.Uint64() % activeCount, nil
}
