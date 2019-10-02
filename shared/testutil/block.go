package testutil

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// BlockGenConfig is used to define the requested conditions
// for block generation.
type BlockGenConfig struct {
	MaxProposerSlashings uint64
	MaxAttesterSlashings uint64
	MaxAttestations      uint64
	MaxDeposits          uint64
	MaxVoluntaryExits    uint64
}

// GenerateFullBlock generates a fully valid block with the requested parameters.
// Use BlockGenConfig to declare the conditions you would like the block generated under.
func GenerateFullBlock(
	t testing.TB,
	bState *pb.BeaconState,
	privs []*bls.SecretKey,
	conf *BlockGenConfig,
) *ethpb.BeaconBlock {

	currentSlot := bState.Slot

	pSlashings := []*ethpb.ProposerSlashing{}
	if conf.MaxProposerSlashings > 0 {
		pSlashings = generateProposerSlashings(t, bState, privs, conf.MaxProposerSlashings)
	}

	aSlashings := []*ethpb.AttesterSlashing{}
	if conf.MaxAttesterSlashings > 0 {
		aSlashings = generateAttesterSlashings(t, bState, privs, conf.MaxAttesterSlashings)
	}

	atts := []*ethpb.Attestation{}
	if conf.MaxAttestations > 0 {
		atts = generateAttestations(t, bState, privs, conf.MaxAttestations)
	}

	newDeposits, eth1Data := []*ethpb.Deposit{}, bState.Eth1Data
	if conf.MaxDeposits > 0 {
		newDeposits, eth1Data = generateDepositsAndEth1Data(t, bState, conf.MaxDeposits)
	}

	exits := []*ethpb.VoluntaryExit{}
	if conf.MaxVoluntaryExits > 0 {
		exits = generateVoluntaryExits(t, bState, privs, conf.MaxVoluntaryExits)
	}

	newHeader := proto.Clone(bState.LatestBlockHeader).(*ethpb.BeaconBlockHeader)
	prevStateRoot, err := ssz.HashTreeRoot(bState)
	if err != nil {
		t.Fatal(err)
	}
	newHeader.StateRoot = prevStateRoot[:]
	parentRoot, err := ssz.SigningRoot(newHeader)
	if err != nil {
		t.Fatal(err)
	}

	// Temporarily incrementing the beacon state slot here since BeaconProposerIndex is a
	// function deterministic on beacon state slot.
	bState.Slot++
	reveal, err := CreateRandaoReveal(bState, helpers.CurrentEpoch(bState), privs)
	if err != nil {
		t.Fatal(err)
	}
	bState.Slot--

	block := &ethpb.BeaconBlock{
		Slot:       currentSlot + 1,
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

	s, err := state.CalculateStateRoot(context.Background(), bState, block)
	if err != nil {
		t.Fatal(err)
	}
	root, err := ssz.HashTreeRoot(s)
	if err != nil {
		t.Fatal(err)
	}
	block.StateRoot = root[:]
	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatal(err)
	}
	// Temporarily incrementing the beacon state slot here since BeaconProposerIndex is a
	// function deterministic on beacon state slot.
	bState.Slot++
	proposerIdx, err := helpers.BeaconProposerIndex(bState)
	if err != nil {
		t.Fatal(err)
	}
	bState.Slot--
	domain := helpers.Domain(bState.Fork, helpers.CurrentEpoch(bState), params.BeaconConfig().DomainBeaconProposer)
	block.Signature = privs[proposerIdx].Sign(blockRoot[:], domain).Marshal()

	return block
}

func generateProposerSlashings(
	t testing.TB,
	bState *pb.BeaconState,
	privs []*bls.SecretKey,
	maxSlashings uint64,
) []*ethpb.ProposerSlashing {
	currentSlot := bState.Slot
	currentEpoch := helpers.CurrentEpoch(bState)
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch

	validatorCount, err := helpers.ActiveValidatorCount(bState, currentEpoch)
	if err != nil {
		t.Fatal(err)
	}
	proposerSlashings := make([]*ethpb.ProposerSlashing, maxSlashings)
	for i := uint64(0); i < maxSlashings; i++ {
		proposerIndex := i + uint64(validatorCount/4)
		header1 := &ethpb.BeaconBlockHeader{
			Slot:     currentSlot - (i % slotsPerEpoch),
			BodyRoot: []byte{0, 1, 0},
		}
		root, err := ssz.SigningRoot(header1)
		if err != nil {
			t.Fatal(err)
		}
		domain := helpers.Domain(bState.Fork, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
		header1.Signature = privs[proposerIndex].Sign(root[:], domain).Marshal()

		header2 := &ethpb.BeaconBlockHeader{
			Slot:     currentSlot - (i % slotsPerEpoch),
			BodyRoot: []byte{0, 2, 0},
		}
		root, err = ssz.SigningRoot(header2)
		if err != nil {
			t.Fatal(err)
		}
		header2.Signature = privs[proposerIndex].Sign(root[:], domain).Marshal()

		slashing := &ethpb.ProposerSlashing{
			ProposerIndex: proposerIndex,
			Header_1:      header1,
			Header_2:      header2,
		}
		proposerSlashings[i] = slashing
	}
	return proposerSlashings
}

func generateAttesterSlashings(
	t testing.TB,
	bState *pb.BeaconState,
	privs []*bls.SecretKey,
	maxSlashings uint64,
) []*ethpb.AttesterSlashing {
	attesterSlashings := make([]*ethpb.AttesterSlashing, maxSlashings)
	for i := uint64(0); i < maxSlashings; i++ {
		crosslink := &ethpb.Crosslink{
			Shard:      i % params.BeaconConfig().ShardCount,
			StartEpoch: i,
			EndEpoch:   i + 1,
		}
		committee, err := helpers.CrosslinkCommittee(bState, i, crosslink.Shard)
		if err != nil {
			t.Fatal(err)
		}
		committeeSize := uint64(len(committee))
		attData1 := &ethpb.AttestationData{
			Crosslink: crosslink,
			Target: &ethpb.Checkpoint{
				Epoch: i,
				Root:  params.BeaconConfig().ZeroHash[:],
			},
			Source: &ethpb.Checkpoint{
				Epoch: i + 1,
				Root:  params.BeaconConfig().ZeroHash[:],
			},
		}
		aggregationBits := bitfield.NewBitlist(committeeSize)
		aggregationBits.SetBitAt(i, true)
		custodyBits := bitfield.NewBitlist(committeeSize)
		att1 := &ethpb.Attestation{
			Data:            attData1,
			CustodyBits:     custodyBits,
			AggregationBits: aggregationBits,
		}
		dataRoot, err := ssz.HashTreeRoot(&pb.AttestationDataAndCustodyBit{
			Data:       att1.Data,
			CustodyBit: false,
		})
		if err != nil {
			t.Fatal(err)
		}
		domain := helpers.Domain(bState.Fork, i, params.BeaconConfig().DomainAttestation)
		sig := privs[committee[i]].Sign(dataRoot[:], domain)
		att1.Signature = bls.AggregateSignatures([]*bls.Signature{sig}).Marshal()

		attData2 := &ethpb.AttestationData{
			Crosslink: crosslink,
			Target: &ethpb.Checkpoint{
				Epoch: i,
				Root:  params.BeaconConfig().ZeroHash[:],
			},
			Source: &ethpb.Checkpoint{
				Epoch: i,
				Root:  params.BeaconConfig().ZeroHash[:],
			},
		}
		att2 := &ethpb.Attestation{
			Data:            attData2,
			CustodyBits:     custodyBits,
			AggregationBits: aggregationBits,
		}
		dataRoot, err = ssz.HashTreeRoot(&pb.AttestationDataAndCustodyBit{
			Data:       att2.Data,
			CustodyBit: false,
		})
		if err != nil {
			t.Fatal(err)
		}
		sig = privs[committee[i]].Sign(dataRoot[:], domain)
		att2.Signature = bls.AggregateSignatures([]*bls.Signature{sig}).Marshal()

		indexedAtt1, err := blocks.ConvertToIndexed(bState, att1)
		if err != nil {
			t.Fatal(err)
		}
		indexedAtt2, err := blocks.ConvertToIndexed(bState, att2)
		if err != nil {
			t.Fatal(err)
		}
		slashing := &ethpb.AttesterSlashing{
			Attestation_1: indexedAtt1,
			Attestation_2: indexedAtt2,
		}
		attesterSlashings[i] = slashing
	}
	return attesterSlashings
}

// generateAttestations creates attestations that are entirely valid, for the current state slot.
// This function always returns all validators participating, if maxAttestations is 1, then it will
// return 1 attestation with all validators aggregated into it. If maxAttestations is set to 4, then
// it will return 4 attestations for the same data with their aggregation bits split uniformly.
func generateAttestations(
	t testing.TB,
	bState *pb.BeaconState,
	privs []*bls.SecretKey,
	maxAttestations uint64,
) []*ethpb.Attestation {
	headState := proto.Clone(bState).(*pb.BeaconState)
	headState, err := state.ProcessSlots(context.Background(), headState, bState.Slot+1)
	if err != nil {
		t.Fatal(err)
	}

	currentEpoch := helpers.CurrentEpoch(bState)
	attestations := make([]*ethpb.Attestation, maxAttestations)

	committeeCount, err := helpers.CommitteeCount(bState, currentEpoch)
	if err != nil {
		t.Fatal(err)
	}
	committeesPerSlot := committeeCount / params.BeaconConfig().SlotsPerEpoch
	offSet := committeesPerSlot * (bState.Slot % params.BeaconConfig().SlotsPerEpoch)
	startShard, err := helpers.StartShard(bState, currentEpoch)
	if err != nil {
		t.Fatal(err)
	}
	shard := (startShard + offSet) % params.BeaconConfig().ShardCount

	parentCrosslink := bState.CurrentCrosslinks[shard]
	endEpoch := parentCrosslink.EndEpoch + params.BeaconConfig().MaxEpochsPerCrosslink
	if currentEpoch < endEpoch {
		endEpoch = currentEpoch
	}
	parentRoot, err := ssz.HashTreeRoot(parentCrosslink)
	if err != nil {
		t.Fatal(err)
	}
	crosslink := &ethpb.Crosslink{
		Shard:      shard,
		StartEpoch: parentCrosslink.EndEpoch,
		EndEpoch:   endEpoch,
		ParentRoot: parentRoot[:],
		DataRoot:   params.BeaconConfig().ZeroHash[:],
	}
	committee, err := helpers.CrosslinkCommittee(bState, currentEpoch, shard)
	if err != nil {
		t.Fatal(err)
	}
	committeeSize := uint64(len(committee))
	crosslinkParentRoot, err := ssz.HashTreeRoot(parentCrosslink)
	if err != nil {
		panic(err)
	}
	crosslink.ParentRoot = crosslinkParentRoot[:]

	headRoot, err := helpers.BlockRootAtSlot(headState, bState.Slot)
	if err != nil {
		t.Fatal(err)
	}

	targetRoot := make([]byte, 32)
	epochStartSlot := helpers.StartSlot(currentEpoch)
	if epochStartSlot == headState.Slot {
		targetRoot = headRoot[:]
	} else {
		targetRoot, err = helpers.BlockRootAtSlot(headState, epochStartSlot)
		if err != nil {
			t.Fatal(err)
		}
	}

	custodyBits := bitfield.NewBitlist(committeeSize)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: headRoot,
			Crosslink:       crosslink,
			Source:          bState.CurrentJustifiedCheckpoint,
			Target: &ethpb.Checkpoint{
				Epoch: currentEpoch,
				Root:  targetRoot,
			},
		},
		CustodyBits: custodyBits,
	}

	dataRoot, err := ssz.HashTreeRoot(&pb.AttestationDataAndCustodyBit{
		Data:       att.Data,
		CustodyBit: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	if maxAttestations > committeeSize {
		t.Fatalf(
			"requested %d attestations per block but there are only %d committee members",
			maxAttestations,
			len(committee),
		)
	}

	bitsPerAtt := committeeSize / maxAttestations
	domain := helpers.Domain(bState.Fork, parentCrosslink.EndEpoch+1, params.BeaconConfig().DomainAttestation)
	for i := uint64(0); i < committeeSize; i += bitsPerAtt {
		aggregationBits := bitfield.NewBitlist(committeeSize)
		sigs := []*bls.Signature{}
		for b := i; b < i+bitsPerAtt; b++ {
			aggregationBits.SetBitAt(b, true)
			sigs = append(sigs, privs[committee[b]].Sign(dataRoot[:], domain))
		}
		att.AggregationBits = aggregationBits

		att.Signature = bls.AggregateSignatures(sigs).Marshal()
		attestations[i/bitsPerAtt] = att
	}
	return attestations
}

func generateDepositsAndEth1Data(
	t testing.TB,
	bState *pb.BeaconState,
	maxDeposits uint64,
) (
	[]*ethpb.Deposit,
	*ethpb.Eth1Data,
) {
	previousDepsLen := bState.Eth1DepositIndex
	currentDeposits, _, _ := SetupInitialDeposits(t, previousDepsLen+maxDeposits)
	eth1Data := GenerateEth1Data(t, currentDeposits)
	return currentDeposits[previousDepsLen:], eth1Data
}

func generateVoluntaryExits(
	t testing.TB,
	bState *pb.BeaconState,
	privs []*bls.SecretKey,
	maxExits uint64,
) []*ethpb.VoluntaryExit {
	currentEpoch := helpers.CurrentEpoch(bState)
	validatorCount, err := helpers.ActiveValidatorCount(bState, currentEpoch)
	if err != nil {
		t.Fatal(err)
	}

	voluntaryExits := make([]*ethpb.VoluntaryExit, maxExits)
	for i := 0; i < len(voluntaryExits); i++ {
		valIndex := float64(validatorCount)*(2.0/3.0) + float64(i)
		exit := &ethpb.VoluntaryExit{
			Epoch:          helpers.PrevEpoch(bState),
			ValidatorIndex: uint64(valIndex),
		}
		root, err := ssz.SigningRoot(exit)
		if err != nil {
			t.Fatal(err)
		}
		domain := helpers.Domain(bState.Fork, currentEpoch, params.BeaconConfig().DomainVoluntaryExit)
		exit.Signature = privs[uint64(valIndex)].Sign(root[:], domain).Marshal()
		voluntaryExits[i] = exit
	}
	return voluntaryExits
}
