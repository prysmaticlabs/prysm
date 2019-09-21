package testutil

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

// GenerateFullBlock generates a fully valid block with the requested parameters.
func GenerateFullBlock(
	t testing.TB,
	bState *pb.BeaconState,
	privs []*bls.SecretKey,
) *ethpb.BeaconBlock {
	currentSlot := bState.Slot

	proposerSlashings := generateProposerSlashings(t, bState, privs)
	attesterSlashings := generateAttestorSlashings(t, bState, privs)
	attestations := generateAttestations(t, bState, privs)
	newDeposits, eth1Data := generateDepositsAndEth1Data(t, bState)
	voluntaryExits := generateVoluntaryExits(t, bState, privs)

	parentRoot, err := ssz.SigningRoot(bState.LatestBlockHeader)
	if err != nil {
		t.Fatal(err)
	}

	reveal, err := CreateRandaoReveal(bState, helpers.CurrentEpoch(bState), privs)
	if err != nil {
		t.Fatal(err)
	}

	block := &ethpb.BeaconBlock{
		Slot:       currentSlot,
		ParentRoot: parentRoot[:],
		Body: &ethpb.BeaconBlockBody{
			Eth1Data:          eth1Data,
			RandaoReveal:      reveal,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			VoluntaryExits:    voluntaryExits,
			Deposits:          newDeposits,
		},
	}

	s, err := state.ExecuteStateTransitionNoVerify(context.Background(), bState, block)
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
	proposerIdx, err := helpers.BeaconProposerIndex(bState)
	if err != nil {
		t.Fatal(err)
	}
	domain := helpers.Domain(bState, helpers.CurrentEpoch(bState), params.BeaconConfig().DomainBeaconProposer)
	block.Signature = privs[proposerIdx].Sign(blockRoot[:], domain).Marshal()

	return block
}

func generateProposerSlashings(
	t testing.TB,
	bState *pb.BeaconState,
	privs []*bls.SecretKey,
) []*ethpb.ProposerSlashing {
	currentSlot := bState.Slot
	currentEpoch := helpers.CurrentEpoch(bState)
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch

	validatorCount, err := helpers.ActiveValidatorCount(bState, currentEpoch)
	if err != nil {
		t.Fatal(err)
	}
	proposerSlashings := make([]*ethpb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
	for i := uint64(0); i < params.BeaconConfig().MaxProposerSlashings; i++ {
		proposerIndex := i + uint64(validatorCount/4)
		header1 := &ethpb.BeaconBlockHeader{
			Slot:     currentSlot - (i % slotsPerEpoch),
			BodyRoot: []byte{0, 1, 0},
		}
		root, err := ssz.SigningRoot(header1)
		if err != nil {
			t.Fatal(err)
		}
		domain := helpers.Domain(bState, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
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

func generateAttestorSlashings(
	t testing.TB,
	bState *pb.BeaconState,
	privs []*bls.SecretKey,
) []*ethpb.AttesterSlashing {
	maxSlashes := params.BeaconConfig().MaxAttesterSlashings
	attesterSlashings := make([]*ethpb.AttesterSlashing, maxSlashes)
	for i := uint64(0); i < maxSlashes; i++ {
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
		domain := helpers.Domain(bState, i, params.BeaconConfig().DomainAttestation)
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

func generateAttestations(
	t testing.TB,
	bState *pb.BeaconState,
	privs []*bls.SecretKey,
) []*ethpb.Attestation {
	attestations := make([]*ethpb.Attestation, params.BeaconConfig().MaxAttestations)
	for i := uint64(0); i < params.BeaconConfig().MaxAttestations; i++ {
		shard := (i % (bState.Slot % params.BeaconConfig().SlotsPerEpoch)) % params.BeaconConfig().ShardCount
		parentCrosslink := bState.CurrentCrosslinks[shard]
		crosslink := &ethpb.Crosslink{
			Shard:      shard,
			StartEpoch: parentCrosslink.EndEpoch,
			EndEpoch:   parentCrosslink.EndEpoch + 1,
			DataRoot:   params.BeaconConfig().ZeroHash[:],
		}
		committee, err := helpers.CrosslinkCommittee(bState, helpers.CurrentEpoch(bState), shard)
		if err != nil {
			t.Fatal(err)
		}
		committeeSize := uint64(len(committee))
		crosslinkParentRoot, err := ssz.HashTreeRoot(parentCrosslink)
		if err != nil {
			panic(err)
		}
		crosslink.ParentRoot = crosslinkParentRoot[:]

		aggregationBits := bitfield.NewBitlist(committeeSize)
		aggregationBits.SetBitAt(i, true)
		custodyBits := bitfield.NewBitlist(committeeSize)
		att := &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Crosslink: crosslink,
				Source: &ethpb.Checkpoint{
					Epoch: helpers.PrevEpoch(bState),
					Root:  params.BeaconConfig().ZeroHash[:],
				},
				Target: &ethpb.Checkpoint{
					Epoch: parentCrosslink.EndEpoch + 1,
					Root:  params.BeaconConfig().ZeroHash[:],
				},
			},
			AggregationBits: aggregationBits,
			CustodyBits:     custodyBits,
		}
		dataRoot, err := ssz.HashTreeRoot(&pb.AttestationDataAndCustodyBit{
			Data:       att.Data,
			CustodyBit: false,
		})
		if err != nil {
			t.Fatal(err)
		}
		domain := helpers.Domain(bState, parentCrosslink.EndEpoch+1, params.BeaconConfig().DomainAttestation)
		sig := privs[committee[i]].Sign(dataRoot[:], domain)
		att.Signature = bls.AggregateSignatures([]*bls.Signature{sig}).Marshal()
		attestations[i] = att
	}
	return attestations
}

func generateDepositsAndEth1Data(
	t testing.TB,
	bState *pb.BeaconState,
) (
	[]*ethpb.Deposit,
	*ethpb.Eth1Data,
) {
	previousDepsLen := uint64(len(deposits))
	newDeposits, _ := GenerateDeposits(t, params.BeaconConfig().MaxDeposits)
	encodedDeposits := make([][]byte, previousDepsLen)
	for i := 0; i < int(previousDepsLen); i++ {
		hashedDeposit, err := ssz.HashTreeRoot(deposits[i].Data)
		if err != nil {
			t.Fatal(err)
		}
		encodedDeposits[i] = hashedDeposit[:]
	}
	newHashes := make([][]byte, len(newDeposits))
	for i := 0; i < len(newDeposits); i++ {
		hashedDeposit, err := ssz.HashTreeRoot(newDeposits[i].Data)
		if err != nil {
			t.Fatal(err)
		}
		newHashes[i] = hashedDeposit[:]
	}
	allData := append(encodedDeposits, newHashes...)
	depositTrie, err := trieutil.GenerateTrieFromItems(allData, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(newDeposits); i++ {
		proof, err := depositTrie.MerkleProof(int(previousDepsLen) + i)
		if err != nil {
			t.Fatal(err)
		}
		newDeposits[i] = &ethpb.Deposit{
			Data:  newDeposits[i].Data,
			Proof: proof,
		}
	}
	root := depositTrie.Root()

	eth1Data := &ethpb.Eth1Data{
		DepositRoot:  root[:],
		BlockHash:    root[:],
		DepositCount: uint64(len(deposits)),
	}
	return newDeposits, eth1Data
}
func generateVoluntaryExits(
	t testing.TB,
	bState *pb.BeaconState,
	privs []*bls.SecretKey,
) []*ethpb.VoluntaryExit {
	currentEpoch := helpers.CurrentEpoch(bState)
	validatorCount, err := helpers.ActiveValidatorCount(bState, currentEpoch)
	if err != nil {
		t.Fatal(err)
	}

	voluntaryExits := make([]*ethpb.VoluntaryExit, params.BeaconConfig().MaxVoluntaryExits)
	for i := 0; i < len(voluntaryExits); i++ {
		valIndex := validatorCount*uint64(2/3) + uint64(i)
		exit := &ethpb.VoluntaryExit{
			Epoch:          helpers.PrevEpoch(bState),
			ValidatorIndex: valIndex,
		}
		root, err := ssz.SigningRoot(exit)
		if err != nil {
			t.Fatal(err)
		}
		domain := helpers.Domain(bState, currentEpoch, params.BeaconConfig().DomainVoluntaryExit)
		exit.Signature = privs[valIndex].Sign(root[:], domain).Marshal()
		voluntaryExits[i] = exit
	}
	return voluntaryExits
}
