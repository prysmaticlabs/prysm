package testutil

import (
	"context"
	"fmt"
	"log"
	"math"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
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

// NewBeaconBlock creates a beacon block with minimum marshalable fields.
func NewBeaconBlock() *ethpb.SignedBeaconBlock {
	return &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: make([]byte, 32),
			StateRoot:  make([]byte, 32),
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: make([]byte, 96),
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot: make([]byte, 32),
					BlockHash:   make([]byte, 32),
				},
				Graffiti: make([]byte, 32),
			},
		},
		Signature: make([]byte, 96),
	}
}

// GenerateFullBlock generates a fully valid block with the requested parameters.
// Use BlockGenConfig to declare the conditions you would like the block generated under.
func GenerateFullBlock(
	bState *stateTrie.BeaconState,
	privs []bls.SecretKey,
	conf *BlockGenConfig,
	slot uint64,
) (*ethpb.SignedBeaconBlock, error) {
	ctx := context.Background()
	currentSlot := bState.Slot()
	if currentSlot > slot {
		return nil, fmt.Errorf("current slot in state is larger than given slot. %d > %d", currentSlot, slot)
	}
	bState = bState.Copy()

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
		atts, err = GenerateAttestations(bState, privs, numToGen, slot, false)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attestations:", numToGen)
		}
	}

	numToGen = conf.NumDeposits
	newDeposits, eth1Data := []*ethpb.Deposit{}, bState.Eth1Data()
	if numToGen > 0 {
		newDeposits, eth1Data, err = generateDepositsAndEth1Data(bState, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d deposits:", numToGen)
		}
	}

	numToGen = conf.NumVoluntaryExits
	exits := []*ethpb.SignedVoluntaryExit{}
	if numToGen > 0 {
		exits, err = generateVoluntaryExits(bState, privs, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attester slashings:", numToGen)
		}
	}

	newHeader := bState.LatestBlockHeader()
	prevStateRoot, err := bState.HashTreeRoot(ctx)
	if err != nil {
		return nil, err
	}
	newHeader.StateRoot = prevStateRoot[:]
	parentRoot, err := stateutil.BlockHeaderRoot(newHeader)
	if err != nil {
		return nil, err
	}

	if slot == currentSlot {
		slot = currentSlot + 1
	}

	// Temporarily incrementing the beacon state slot here since BeaconProposerIndex is a
	// function deterministic on beacon state slot.
	if err := bState.SetSlot(slot); err != nil {
		return nil, err
	}
	reveal, err := RandaoReveal(bState, helpers.CurrentEpoch(bState), privs)
	if err != nil {
		return nil, err
	}

	idx, err := helpers.BeaconProposerIndex(bState)
	if err != nil {
		return nil, err
	}

	block := &ethpb.BeaconBlock{
		Slot:          slot,
		ParentRoot:    parentRoot[:],
		ProposerIndex: idx,
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
	if err := bState.SetSlot(currentSlot); err != nil {
		return nil, err
	}

	signature, err := BlockSignature(bState, block, privs)
	if err != nil {
		return nil, err
	}

	return &ethpb.SignedBeaconBlock{Block: block, Signature: signature.Marshal()}, nil
}

// GenerateProposerSlashingForValidator for a specific validator index.
func GenerateProposerSlashingForValidator(
	bState *stateTrie.BeaconState,
	priv bls.SecretKey,
	idx uint64,
) (*ethpb.ProposerSlashing, error) {
	header1 := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: idx,
			Slot:          bState.Slot(),
			BodyRoot:      bytesutil.PadTo([]byte{0, 1, 0}, 32),
			StateRoot:     make([]byte, 32),
			ParentRoot:    make([]byte, 32),
		},
	}
	currentEpoch := helpers.CurrentEpoch(bState)
	domain, err := helpers.Domain(bState.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer, bState.GenesisValidatorRoot())
	if err != nil {
		return nil, err
	}
	root, err := helpers.ComputeSigningRoot(header1.Header, domain)
	if err != nil {
		return nil, err
	}
	header1.Signature = priv.Sign(root[:]).Marshal()

	header2 := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: idx,
			Slot:          bState.Slot(),
			BodyRoot:      bytesutil.PadTo([]byte{0, 2, 0}, 32),
			StateRoot:     make([]byte, 32),
			ParentRoot:    make([]byte, 32),
		},
	}
	root, err = helpers.ComputeSigningRoot(header2.Header, domain)
	if err != nil {
		return nil, err
	}
	header2.Signature = priv.Sign(root[:]).Marshal()

	return &ethpb.ProposerSlashing{
		Header_1: header1,
		Header_2: header2,
	}, nil
}

func generateProposerSlashings(
	bState *stateTrie.BeaconState,
	privs []bls.SecretKey,
	numSlashings uint64,
) ([]*ethpb.ProposerSlashing, error) {
	proposerSlashings := make([]*ethpb.ProposerSlashing, numSlashings)
	for i := uint64(0); i < numSlashings; i++ {
		proposerIndex, err := randValIndex(bState)
		if err != nil {
			return nil, err
		}
		slashing, err := GenerateProposerSlashingForValidator(bState, privs[proposerIndex], proposerIndex)
		if err != nil {
			return nil, err
		}
		proposerSlashings[i] = slashing
	}
	return proposerSlashings, nil
}

// GenerateAttesterSlashingForValidator for a specific validator index.
func GenerateAttesterSlashingForValidator(
	bState *stateTrie.BeaconState,
	priv bls.SecretKey,
	idx uint64,
) (*ethpb.AttesterSlashing, error) {
	currentEpoch := helpers.CurrentEpoch(bState)

	att1 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Slot:           bState.Slot(),
			CommitteeIndex: 0,
			Target: &ethpb.Checkpoint{
				Epoch: currentEpoch,
				Root:  params.BeaconConfig().ZeroHash[:],
			},
			Source: &ethpb.Checkpoint{
				Epoch: currentEpoch + 1,
				Root:  params.BeaconConfig().ZeroHash[:],
			},
		},
		AttestingIndices: []uint64{idx},
	}
	domain, err := helpers.Domain(bState.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconAttester, bState.GenesisValidatorRoot())
	if err != nil {
		return nil, err
	}
	dataRoot, err := helpers.ComputeSigningRoot(att1.Data, domain)
	if err != nil {
		return nil, err
	}
	sig := priv.Sign(dataRoot[:])
	att1.Signature = bls.AggregateSignatures([]bls.Signature{sig}).Marshal()

	att2 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Slot:           bState.Slot(),
			CommitteeIndex: 0,
			Target: &ethpb.Checkpoint{
				Epoch: currentEpoch,
				Root:  params.BeaconConfig().ZeroHash[:],
			},
			Source: &ethpb.Checkpoint{
				Epoch: currentEpoch,
				Root:  params.BeaconConfig().ZeroHash[:],
			},
		},
		AttestingIndices: []uint64{idx},
	}
	dataRoot, err = helpers.ComputeSigningRoot(att2.Data, domain)
	if err != nil {
		return nil, err
	}
	sig = priv.Sign(dataRoot[:])
	att2.Signature = bls.AggregateSignatures([]bls.Signature{sig}).Marshal()

	return &ethpb.AttesterSlashing{
		Attestation_1: att1,
		Attestation_2: att2,
	}, nil
}

func generateAttesterSlashings(
	bState *stateTrie.BeaconState,
	privs []bls.SecretKey,
	numSlashings uint64,
) ([]*ethpb.AttesterSlashing, error) {
	attesterSlashings := make([]*ethpb.AttesterSlashing, numSlashings)
	randGen := rand.NewDeterministicGenerator()
	for i := uint64(0); i < numSlashings; i++ {
		committeeIndex := randGen.Uint64() % params.BeaconConfig().MaxCommitteesPerSlot
		committee, err := helpers.BeaconCommitteeFromState(bState, bState.Slot(), committeeIndex)
		if err != nil {
			return nil, err
		}
		randIndex := randGen.Uint64() % uint64(len(committee))
		valIndex := committee[randIndex]
		slashing, err := GenerateAttesterSlashingForValidator(bState, privs[valIndex], valIndex)
		if err != nil {
			return nil, err
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
func GenerateAttestations(bState *stateTrie.BeaconState, privs []bls.SecretKey, numToGen uint64, slot uint64, randomRoot bool) ([]*ethpb.Attestation, error) {
	currentEpoch := helpers.SlotToEpoch(slot)
	attestations := []*ethpb.Attestation{}
	generateHeadState := false
	bState = bState.Copy()
	if slot > bState.Slot() {
		// Going back a slot here so there's no inclusion delay issues.
		slot--
		generateHeadState = true
	}

	targetRoot := make([]byte, 32)
	headRoot := make([]byte, 32)
	var err error
	// Only calculate head state if its an attestation for the current slot or future slot.
	if generateHeadState || slot == bState.Slot() {
		headState, err := stateTrie.InitializeFromProtoUnsafe(bState.CloneInnerState())
		if err != nil {
			return nil, err
		}
		headState, err = state.ProcessSlots(context.Background(), headState, slot+1)
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
	if randomRoot {
		randGen := rand.NewDeterministicGenerator()
		b := make([]byte, 32)
		_, err := randGen.Read(b)
		if err != nil {
			return nil, err
		}
		headRoot = b
	}

	activeValidatorCount, err := helpers.ActiveValidatorCount(bState, currentEpoch)
	if err != nil {
		return nil, err
	}
	committeesPerSlot := helpers.SlotCommitteeCount(activeValidatorCount)

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

	domain, err := helpers.Domain(bState.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconAttester, bState.GenesisValidatorRoot())
	if err != nil {
		return nil, err
	}
	for c := uint64(0); c < committeesPerSlot && c < numToGen; c++ {
		committee, err := helpers.BeaconCommitteeFromState(bState, slot, c)
		if err != nil {
			return nil, err
		}

		attData := &ethpb.AttestationData{
			Slot:            slot,
			CommitteeIndex:  c,
			BeaconBlockRoot: headRoot,
			Source:          bState.CurrentJustifiedCheckpoint(),
			Target: &ethpb.Checkpoint{
				Epoch: currentEpoch,
				Root:  targetRoot,
			},
		}

		dataRoot, err := helpers.ComputeSigningRoot(attData, domain)
		if err != nil {
			return nil, err
		}

		committeeSize := uint64(len(committee))
		bitsPerAtt := committeeSize / uint64(attsPerCommittee)
		for i := uint64(0); i < committeeSize; i += bitsPerAtt {
			aggregationBits := bitfield.NewBitlist(committeeSize)
			sigs := []bls.Signature{}
			for b := i; b < i+bitsPerAtt; b++ {
				aggregationBits.SetBitAt(b, true)
				sigs = append(sigs, privs[committee[b]].Sign(dataRoot[:]))
			}

			// bls.AggregateSignatures will return nil if sigs is 0.
			if len(sigs) == 0 {
				continue
			}

			att := &ethpb.Attestation{
				Data:            attData,
				AggregationBits: aggregationBits,
				Signature:       bls.AggregateSignatures(sigs).Marshal(),
			}
			attestations = append(attestations, att)
		}
	}
	return attestations, nil
}

func generateDepositsAndEth1Data(
	bState *stateTrie.BeaconState,
	numDeposits uint64,
) (
	[]*ethpb.Deposit,
	*ethpb.Eth1Data,
	error,
) {
	previousDepsLen := bState.Eth1DepositIndex()
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
	bState *stateTrie.BeaconState,
	privs []bls.SecretKey,
	numExits uint64,
) ([]*ethpb.SignedVoluntaryExit, error) {
	currentEpoch := helpers.CurrentEpoch(bState)

	voluntaryExits := make([]*ethpb.SignedVoluntaryExit, numExits)
	for i := 0; i < len(voluntaryExits); i++ {
		valIndex, err := randValIndex(bState)
		if err != nil {
			return nil, err
		}
		exit := &ethpb.SignedVoluntaryExit{
			Exit: &ethpb.VoluntaryExit{
				Epoch:          helpers.PrevEpoch(bState),
				ValidatorIndex: valIndex,
			},
		}
		domain, err := helpers.Domain(bState.Fork(), currentEpoch, params.BeaconConfig().DomainVoluntaryExit, bState.GenesisValidatorRoot())
		if err != nil {
			return nil, err
		}
		root, err := helpers.ComputeSigningRoot(exit.Exit, domain)
		if err != nil {
			return nil, err
		}
		exit.Signature = privs[valIndex].Sign(root[:]).Marshal()
		voluntaryExits[i] = exit
	}
	return voluntaryExits, nil
}

func randValIndex(bState *stateTrie.BeaconState) (uint64, error) {
	activeCount, err := helpers.ActiveValidatorCount(bState, helpers.CurrentEpoch(bState))
	if err != nil {
		return 0, err
	}
	return rand.NewGenerator().Uint64() % activeCount, nil
}
