package state_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

func TestExecuteStateTransition_IncorrectSlot(t *testing.T) {
	beaconState := &pb.BeaconState{
		Slot: 5,
	}
	block := &ethpb.BeaconBlock{
		Slot: 4,
	}
	want := "expected state.slot"
	if _, err := state.ExecuteStateTransition(context.Background(), beaconState, block); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestExecuteStateTransition_FullProcess(t *testing.T) {

	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	eth1Data := &ethpb.Eth1Data{
		DepositCount: 100,
		DepositRoot:  []byte{2},
	}
	beaconState.Slot = params.BeaconConfig().SlotsPerEpoch - 1
	beaconState.Eth1Data.DepositCount = 100
	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{Slot: beaconState.Slot}
	beaconState.Eth1DataVotes = []*ethpb.Eth1Data{eth1Data}

	oldMix := beaconState.RandaoMixes[1]
	parentRoot, err := ssz.SigningRoot(beaconState.LatestBlockHeader)
	if err != nil {
		t.Error(err)
	}

	beaconState.Slot++
	epoch := helpers.CurrentEpoch(beaconState)
	randaoReveal, err := testutil.RandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot--
	block := &ethpb.BeaconBlock{
		Slot:       beaconState.Slot + 1,
		ParentRoot: parentRoot[:],
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: randaoReveal,
			Eth1Data:     eth1Data,
		},
	}

	stateRoot, err := state.CalculateStateRoot(context.Background(), beaconState, block)
	if err != nil {
		t.Fatal(err)
	}

	block.StateRoot = stateRoot[:]

	sig, err := testutil.BlockSignature(beaconState, block, privKeys)
	if err != nil {
		t.Error(err)
	}
	block.Signature = sig.Marshal()

	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		t.Error(err)
	}

	if beaconState.Slot != params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Unexpected Slot number, expected: 64, received: %d", beaconState.Slot)
	}

	if bytes.Equal(beaconState.RandaoMixes[1], oldMix) {
		t.Errorf("Did not expect new and old randao mix to equal, %#x == %#x", beaconState.RandaoMixes[0], oldMix)
	}
}

func TestProcessBlock_IncorrectProposerSlashing(t *testing.T) {

	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	block, err := testutil.GenerateFullBlock(beaconState, privKeys, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	slashing := &ethpb.ProposerSlashing{
		Header_1: &ethpb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch},
		Header_2: &ethpb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch * 2},
	}
	block.Body.ProposerSlashings = []*ethpb.ProposerSlashing{slashing}

	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot++
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot--
	domain := helpers.Domain(beaconState.Fork, helpers.CurrentEpoch(beaconState), params.BeaconConfig().DomainBeaconProposer)
	sig := privKeys[proposerIdx].Sign(blockRoot[:], domain)
	block.Signature = sig.Marshal()

	beaconState, err = state.ProcessSlots(context.Background(), beaconState, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := "could not process block proposer slashing"
	if _, err := state.ProcessBlock(context.Background(), beaconState, block); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProcessBlockAttestations(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Target: &ethpb.Checkpoint{Epoch: 0},
			Source: &ethpb.Checkpoint{Epoch: 0},
		},
		AggregationBits: bitfield.NewBitlist(3),
		CustodyBits:     bitfield.NewBitlist(3),
	}

	block, err := testutil.GenerateFullBlock(beaconState, privKeys, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	block.Body.Attestations = []*ethpb.Attestation{att}
	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot++
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot--
	domain := helpers.Domain(beaconState.Fork, helpers.CurrentEpoch(beaconState), params.BeaconConfig().DomainBeaconProposer)
	sig := privKeys[proposerIdx].Sign(blockRoot[:], domain)
	block.Signature = sig.Marshal()

	beaconState, err = state.ProcessSlots(context.Background(), beaconState, 1)
	if err != nil {
		t.Fatal(err)
	}

	want := "could not process block attestations"
	if _, err := state.ProcessBlock(context.Background(), beaconState, block); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProcessExits(t *testing.T) {

	beaconState, _ := testutil.DeterministicGenesisState(t, 100)

	proposerSlashings := []*ethpb.ProposerSlashing{
		{
			ProposerIndex: 3,
			Header_1: &ethpb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("A"),
			},
			Header_2: &ethpb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("B"),
			},
		},
	}
	attesterSlashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 0},
					Target: &ethpb.Checkpoint{Epoch: 0},
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
			Attestation_2: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 1},
					Target: &ethpb.Checkpoint{Epoch: 0},
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
		},
	}
	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().SlotsPerHistoricalRoot; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	beaconState.BlockRoots = blockRoots
	blockAtt := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
		},
		AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
		CustodyBits:     bitfield.Bitlist{0x00, 0x00, 0x00, 0x00, 0x01},
	}
	attestations := []*ethpb.Attestation{blockAtt}
	var exits []*ethpb.VoluntaryExit
	for i := uint64(0); i < params.BeaconConfig().MaxVoluntaryExits+1; i++ {
		exits = append(exits, &ethpb.VoluntaryExit{})
	}
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := ssz.HashTreeRoot(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot:       genesisBlock.Slot,
		ParentRoot: genesisBlock.ParentRoot,
		BodyRoot:   bodyRoot[:],
	}
	parentRoot, err := ssz.SigningRoot(beaconState.LatestBlockHeader)
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		ParentRoot: parentRoot[:],
		Slot:       1,
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal:      []byte{},
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			VoluntaryExits:    exits,
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: []byte{2},
				BlockHash:   []byte{3},
			},
		},
	}
	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay
	beaconState.CurrentJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}
	if _, err := state.ProcessBlock(context.Background(), beaconState, block); err == nil {
		t.Error("Expected err, received nil")
	}
}

func TestProcessBlock_PassesProcessingConditions(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 32)
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := ssz.HashTreeRoot(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot:       genesisBlock.Slot,
		ParentRoot: genesisBlock.ParentRoot,
		BodyRoot:   bodyRoot[:],
	}
	beaconState.Slashings = make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)
	beaconState.CurrentJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	proposerSlashIdx := uint64(3)
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	beaconState.Slot = (params.BeaconConfig().PersistentCommitteePeriod * slotsPerEpoch) + params.BeaconConfig().MinAttestationInclusionDelay

	currentEpoch := helpers.CurrentEpoch(beaconState)
	domain := helpers.Domain(
		beaconState.Fork,
		currentEpoch,
		params.BeaconConfig().DomainBeaconProposer,
	)

	header1 := &ethpb.BeaconBlockHeader{
		Slot:      1,
		StateRoot: []byte("A"),
	}
	signingRoot, err := ssz.SigningRoot(header1)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	header1.Signature = privKeys[proposerSlashIdx].Sign(signingRoot[:], domain).Marshal()[:]

	header2 := &ethpb.BeaconBlockHeader{
		Slot:      1,
		StateRoot: []byte("B"),
	}
	signingRoot, err = ssz.SigningRoot(header2)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	header2.Signature = privKeys[proposerSlashIdx].Sign(signingRoot[:], domain).Marshal()[:]

	proposerSlashings := []*ethpb.ProposerSlashing{
		{
			ProposerIndex: proposerSlashIdx,
			Header_1:      header1,
			Header_2:      header2,
		},
	}
	beaconState.Validators[proposerSlashIdx].PublicKey = privKeys[proposerSlashIdx].PublicKey().Marshal()[:]

	att1 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte{'A'}},
			Target: &ethpb.Checkpoint{Epoch: 0}},
		CustodyBit_0Indices: []uint64{0, 1},
	}
	dataAndCustodyBit := &pb.AttestationDataAndCustodyBit{
		Data:       att1.Data,
		CustodyBit: false,
	}
	hashTreeRoot, err := ssz.HashTreeRoot(dataAndCustodyBit)
	if err != nil {
		t.Error(err)
	}
	domain = helpers.Domain(beaconState.Fork, currentEpoch, params.BeaconConfig().DomainBeaconAttester)
	sig0 := privKeys[0].Sign(hashTreeRoot[:], domain)
	sig1 := privKeys[1].Sign(hashTreeRoot[:], domain)
	aggregateSig := bls.AggregateSignatures([]*bls.Signature{sig0, sig1})
	att1.Signature = aggregateSig.Marshal()[:]

	att2 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte{'B'}},
			Target: &ethpb.Checkpoint{Epoch: 0}},
		CustodyBit_0Indices: []uint64{0, 1},
	}
	dataAndCustodyBit = &pb.AttestationDataAndCustodyBit{
		Data:       att2.Data,
		CustodyBit: false,
	}
	hashTreeRoot, err = ssz.HashTreeRoot(dataAndCustodyBit)
	if err != nil {
		t.Error(err)
	}
	sig0 = privKeys[0].Sign(hashTreeRoot[:], domain)
	sig1 = privKeys[1].Sign(hashTreeRoot[:], domain)
	aggregateSig = bls.AggregateSignatures([]*bls.Signature{sig0, sig1})
	att2.Signature = aggregateSig.Marshal()[:]

	attesterSlashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: att1,
			Attestation_2: att2,
		},
	}

	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().SlotsPerHistoricalRoot; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	beaconState.BlockRoots = blockRoots

	aggBits := bitfield.NewBitlist(1)
	aggBits.SetBitAt(0, true)
	custodyBits := bitfield.NewBitlist(1)
	blockAtt := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:   beaconState.Slot - 1,
			Target: &ethpb.Checkpoint{Epoch: helpers.SlotToEpoch(beaconState.Slot)},
			Source: &ethpb.Checkpoint{
				Epoch: 0,
				Root:  []byte("hello-world"),
			}},
		AggregationBits: aggBits,
		CustodyBits:     custodyBits,
	}
	attestingIndices, err := helpers.AttestingIndices(beaconState, blockAtt.Data, blockAtt.AggregationBits)
	if err != nil {
		t.Error(err)
	}
	dataAndCustodyBit = &pb.AttestationDataAndCustodyBit{
		Data:       blockAtt.Data,
		CustodyBit: false,
	}
	hashTreeRoot, err = ssz.HashTreeRoot(dataAndCustodyBit)
	if err != nil {
		t.Error(err)
	}
	sigs := make([]*bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sig := privKeys[indice].Sign(hashTreeRoot[:], domain)
		sigs[i] = sig
	}
	blockAtt.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	exit := &ethpb.VoluntaryExit{
		ValidatorIndex: 10,
		Epoch:          0,
	}
	signingRoot, err = ssz.SigningRoot(exit)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	domain = helpers.Domain(beaconState.Fork, currentEpoch, params.BeaconConfig().DomainVoluntaryExit)
	exit.Signature = privKeys[exit.ValidatorIndex].Sign(signingRoot[:], domain).Marshal()[:]

	parentRoot, err := ssz.SigningRoot(beaconState.LatestBlockHeader)
	if err != nil {
		t.Fatal(err)
	}

	randaoReveal, err := testutil.RandaoReveal(beaconState, currentEpoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		ParentRoot: parentRoot[:],
		Slot:       beaconState.Slot,
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal:      randaoReveal,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      []*ethpb.Attestation{blockAtt},
			VoluntaryExits:    []*ethpb.VoluntaryExit{exit},
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: []byte{2},
				BlockHash:   []byte{3},
			},
		},
	}

	sig, err := testutil.BlockSignature(beaconState, block, privKeys)
	if err != nil {
		t.Error(err)
	}
	block.Signature = sig.Marshal()

	beaconState, err = state.ProcessBlock(context.Background(), beaconState, block)
	if err != nil {
		t.Errorf("Expected block to pass processing conditions: %v", err)
	}

	if !beaconState.Validators[proposerSlashings[0].ProposerIndex].Slashed {
		t.Errorf("Expected validator at index %d to be slashed, received false", proposerSlashings[0].ProposerIndex)
	}

	if !beaconState.Validators[1].Slashed {
		t.Error("Expected validator at index 1 to be slashed, received false")
	}

	received := beaconState.Validators[exit.ValidatorIndex].ExitEpoch
	wanted := params.BeaconConfig().FarFutureEpoch
	if received == wanted {
		t.Errorf("Expected validator at index %d to be exiting, did not expect: %d", exit.ValidatorIndex, wanted)
	}
}

func TestProcessEpochPrecompute_CanProcess(t *testing.T) {
	epoch := uint64(1)

	atts := []*pb.PendingAttestation{{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{}}}}

	newState, err := state.ProcessEpochPrecompute(context.Background(), &pb.BeaconState{
		Slot:                       epoch*params.BeaconConfig().SlotsPerEpoch + 1,
		BlockRoots:                 make([][]byte, 128),
		Slashings:                  []uint64{0, 1e9, 1e9},
		RandaoMixes:                make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CurrentEpochAttestations:   atts,
		FinalizedCheckpoint:        &ethpb.Checkpoint{},
		JustificationBits:          bitfield.Bitvector4{0x00},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{},
	})
	if err != nil {
		t.Fatal(err)
	}

	wanted := uint64(0)
	if newState.Slashings[2] != wanted {
		t.Errorf("Wanted slashed balance: %d, got: %d", wanted, newState.Slashings[2])
	}
}
func BenchmarkProcessBlk_65536Validators_FullBlock(b *testing.B) {
	logrus.SetLevel(logrus.PanicLevel)

	validatorCount := params.BeaconConfig().MinGenesisActiveValidatorCount * 4
	committeeCount := validatorCount / params.BeaconConfig().TargetCommitteeSize
	validators := make([]*ethpb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch:          params.BeaconConfig().FarFutureEpoch,
			ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	randaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(randaoMixes); i++ {
		randaoMixes[i] = params.BeaconConfig().ZeroHash[:]
	}

	s := &pb.BeaconState{
		Slot:              20,
		LatestBlockHeader: &ethpb.BeaconBlockHeader{},
		BlockRoots:        make([][]byte, 254),
		RandaoMixes:       randaoMixes,
		Validators:        validators,
		Balances:          validatorBalances,
		Slashings:         make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{
			Root: []byte("hello-world"),
		},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
	}

	// Set up proposer slashing object for block
	proposerSlashings := []*ethpb.ProposerSlashing{
		{
			ProposerIndex: 1,
			Header_1: &ethpb.BeaconBlockHeader{
				Slot:      0,
				Signature: []byte("A"),
			},
			Header_2: &ethpb.BeaconBlockHeader{
				Slot:      0,
				Signature: []byte("B"),
			},
		},
	}

	// Set up attester slashing object for block
	attesterSlashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: &ethpb.IndexedAttestation{
				Data:                &ethpb.AttestationData{},
				CustodyBit_0Indices: []uint64{2, 3},
			},
			Attestation_2: &ethpb.IndexedAttestation{
				Data:                &ethpb.AttestationData{},
				CustodyBit_0Indices: []uint64{2, 3},
			},
		},
	}

	// Set up deposit object for block
	deposit := &ethpb.Deposit{
		Data: &ethpb.Deposit_Data{
			PublicKey: []byte{1, 2, 3},
			Amount:    params.BeaconConfig().MaxEffectiveBalance,
		},
	}
	leaf, err := ssz.HashTreeRoot(deposit.Data)
	if err != nil {
		b.Fatal(err)
	}
	depositTrie, err := trieutil.GenerateTrieFromItems([][]byte{leaf[:]}, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		b.Fatalf("Could not generate trie: %v", err)
	}
	proof, err := depositTrie.MerkleProof(0)
	if err != nil {
		b.Fatalf("Could not generate proof: %v", err)
	}
	deposit.Proof = proof
	root := depositTrie.Root()

	// Set up randao reveal object for block
	proposerIdx, err := helpers.BeaconProposerIndex(s)
	if err != nil {
		b.Fatal(err)
	}
	priv := bls.RandKey()
	s.Validators[proposerIdx].PublicKey = priv.PublicKey().Marshal()
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, 0)
	domain := helpers.Domain(s.Fork, 0, params.BeaconConfig().DomainRandao)
	epochSignature := priv.Sign(buf, domain)

	buf = []byte{params.BeaconConfig().BLSWithdrawalPrefixByte}
	pubKey := []byte("A")
	hashed := hashutil.Hash(pubKey)
	buf = append(buf, hashed[:]...)
	s.Validators[3].WithdrawalCredentials = buf

	attestations := make([]*ethpb.Attestation, 128)
	for i := 0; i < len(attestations); i++ {
		attestations[i] = &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Root: []byte("hello-world")}},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0,
				0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			CustodyBits: bitfield.NewBitlist(0),
		}
	}

	blk := &ethpb.BeaconBlock{
		Slot: s.Slot,
		Body: &ethpb.BeaconBlockBody{
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: root[:],
				BlockHash:   root[:],
			},
			RandaoReveal:      epochSignature.Marshal(),
			Attestations:      attestations,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
		},
	}

	// Precache the shuffled indices
	for i := uint64(0); i < committeeCount; i++ {
		if _, err := helpers.BeaconCommittee(s, 0, i); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := state.ProcessBlock(context.Background(), s, blk)
		if err != nil {
			b.Fatal(err)
		}
		// Reset state fields to process block again
		s.Validators[1].Slashed = false
		s.Validators[2].Slashed = false
		s.Balances[3] += 2 * params.BeaconConfig().MinDepositAmount
	}
}

func TestProcessBlk_AttsBasedOnValidatorCount(t *testing.T) {
	logrus.SetLevel(logrus.PanicLevel)

	// Default at 256 validators, can raise this number with faster BLS.
	validatorCount := uint64(256)
	s, privKeys := testutil.DeterministicGenesisState(t, validatorCount)
	s.Slot = params.BeaconConfig().SlotsPerEpoch

	bitCount := validatorCount / params.BeaconConfig().SlotsPerEpoch
	aggBits := bitfield.NewBitlist(bitCount)
	custodyBits := bitfield.NewBitlist(bitCount)
	for i := uint64(1); i < bitCount; i++ {
		aggBits.SetBitAt(i, true)
	}
	atts := make([]*ethpb.Attestation, 1)

	for i := 0; i < len(atts); i++ {
		att := &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 0, Root: params.BeaconConfig().ZeroHash[:]},
				Target: &ethpb.Checkpoint{Epoch: 0}},
			AggregationBits: aggBits,
			CustodyBits:     custodyBits,
		}
		attestingIndices, err := helpers.AttestingIndices(s, att.Data, att.AggregationBits)
		if err != nil {
			t.Error(err)
		}
		dataAndCustodyBit := &pb.AttestationDataAndCustodyBit{
			Data:       att.Data,
			CustodyBit: false,
		}

		domain := helpers.Domain(s.Fork, 0, params.BeaconConfig().DomainBeaconAttester)
		sigs := make([]*bls.Signature, len(attestingIndices))
		for i, indice := range attestingIndices {
			hashTreeRoot, err := ssz.HashTreeRoot(dataAndCustodyBit)
			if err != nil {
				t.Error(err)
			}
			sig := privKeys[indice].Sign(hashTreeRoot[:], domain)
			sigs[i] = sig
		}
		att.Signature = bls.AggregateSignatures(sigs).Marshal()[:]
		atts[i] = att
	}

	epochSignature, _ := testutil.RandaoReveal(s, helpers.CurrentEpoch(s), privKeys)
	parentRoot, _ := ssz.SigningRoot(s.LatestBlockHeader)
	blk := &ethpb.BeaconBlock{
		Slot:       s.Slot,
		ParentRoot: parentRoot[:],
		Body: &ethpb.BeaconBlockBody{
			Eth1Data:     &ethpb.Eth1Data{},
			RandaoReveal: epochSignature,
			Attestations: atts,
		},
	}
	sig, _ := testutil.BlockSignature(s, blk, privKeys)
	blk.Signature = sig.Marshal()

	config := params.BeaconConfig()
	config.MinAttestationInclusionDelay = 0
	params.OverrideBeaconConfig(config)

	if _, err := state.ProcessBlock(context.Background(), s, blk); err != nil {
		t.Fatal(err)
	}
}

func TestCanProcessEpoch_TrueOnEpochs(t *testing.T) {
	tests := []struct {
		slot            uint64
		canProcessEpoch bool
	}{
		{
			slot:            1,
			canProcessEpoch: false,
		}, {
			slot:            63,
			canProcessEpoch: true,
		},
		{
			slot:            64,
			canProcessEpoch: false,
		}, {
			slot:            127,
			canProcessEpoch: true,
		}, {
			slot:            1000000000,
			canProcessEpoch: false,
		},
	}

	for _, tt := range tests {
		s := &pb.BeaconState{Slot: tt.slot}
		if state.CanProcessEpoch(s) != tt.canProcessEpoch {
			t.Errorf(
				"CanProcessEpoch(%d) = %v. Wanted %v",
				tt.slot,
				state.CanProcessEpoch(s),
				tt.canProcessEpoch,
			)
		}
	}
}

func TestProcessOperations_OverMaxProposerSlashings(t *testing.T) {
	maxSlashings := params.BeaconConfig().MaxProposerSlashings
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: make([]*ethpb.ProposerSlashing, maxSlashings+1),
		},
	}

	want := fmt.Sprintf("number of proposer slashings (%d) in block body exceeds allowed threshold of %d",
		len(block.Body.ProposerSlashings), params.BeaconConfig().MaxProposerSlashings)
	if _, err := state.ProcessOperations(
		context.Background(),
		&pb.BeaconState{},
		block.Body,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessOperations_OverMaxAttesterSlashings(t *testing.T) {
	maxSlashings := params.BeaconConfig().MaxAttesterSlashings
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			AttesterSlashings: make([]*ethpb.AttesterSlashing, maxSlashings+1),
		},
	}

	want := fmt.Sprintf("number of attester slashings (%d) in block body exceeds allowed threshold of %d",
		len(block.Body.AttesterSlashings), params.BeaconConfig().MaxAttesterSlashings)
	if _, err := state.ProcessOperations(
		context.Background(),
		&pb.BeaconState{},
		block.Body,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessOperations_OverMaxAttestations(t *testing.T) {
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: make([]*ethpb.Attestation, params.BeaconConfig().MaxAttestations+1),
		},
	}

	want := fmt.Sprintf("number of attestations (%d) in block body exceeds allowed threshold of %d",
		len(block.Body.Attestations), params.BeaconConfig().MaxAttestations)
	if _, err := state.ProcessOperations(
		context.Background(),
		&pb.BeaconState{},
		block.Body,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessOperation_OverMaxVoluntaryExits(t *testing.T) {
	maxExits := params.BeaconConfig().MaxVoluntaryExits
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			VoluntaryExits: make([]*ethpb.VoluntaryExit, maxExits+1),
		},
	}

	want := fmt.Sprintf("number of voluntary exits (%d) in block body exceeds allowed threshold of %d",
		len(block.Body.VoluntaryExits), maxExits)
	if _, err := state.ProcessOperations(
		context.Background(),
		&pb.BeaconState{},
		block.Body,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessOperations_IncorrectDeposits(t *testing.T) {
	s := &pb.BeaconState{
		Eth1Data:         &ethpb.Eth1Data{DepositCount: 100},
		Eth1DepositIndex: 98,
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Deposits: []*ethpb.Deposit{{}},
		},
	}

	want := fmt.Sprintf("incorrect outstanding deposits in block body, wanted: %d, got: %d",
		s.Eth1Data.DepositCount-s.Eth1DepositIndex, len(block.Body.Deposits))
	if _, err := state.ProcessOperations(
		context.Background(),
		s,
		block.Body,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}
