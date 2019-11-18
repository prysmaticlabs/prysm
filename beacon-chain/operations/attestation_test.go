package operations

import (
	"bytes"
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestHandleAttestation_Saves_NewAttestation(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, beaconDB)
	service := NewService(context.Background(), &Config{
		BeaconDB: beaconDB,
	})

	deposits, _, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}

	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: []byte("block-root"),
			Source:          &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target:          &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
		},
		AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
		CustodyBits:     bitfield.Bitlist{0x00, 0x00, 0x00, 0x00, 0x01},
	}

	attestingIndices, err := helpers.AttestingIndices(beaconState, att.Data, att.AggregationBits)
	if err != nil {
		t.Error(err)
	}
	dataAndCustodyBit := &pb.AttestationDataAndCustodyBit{
		Data:       att.Data,
		CustodyBit: false,
	}
	domain := helpers.Domain(beaconState.Fork, 0, params.BeaconConfig().DomainBeaconAttester)
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

	beaconState.CurrentJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	newBlock := &ethpb.BeaconBlock{
		Slot: 0,
	}
	newBlockRoot, err := ssz.HashTreeRoot(newBlock)
	if err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveBlock(context.Background(), newBlock); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveHeadBlockRoot(context.Background(), newBlockRoot); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveState(context.Background(), beaconState, newBlockRoot); err != nil {
		t.Fatal(err)
	}
	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay

	if err := service.HandleAttestation(context.Background(), att); err != nil {
		t.Error(err)
	}
}

func TestHandleAttestation_Aggregates_LargeNumValidators(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, beaconDB)
	ctx := context.Background()
	opsSrv := NewService(ctx, &Config{
		BeaconDB: beaconDB,
	})
	opsSrv.attestationPool = make(map[[32]byte]*dbpb.AttestationContainer)

	// First, we create a common attestation data.
	data := &ethpb.AttestationData{
		Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
		Target: &ethpb.Checkpoint{Epoch: 0},
	}
	dataAndCustodyBit := &pb.AttestationDataAndCustodyBit{
		Data:       data,
		CustodyBit: false,
	}
	root, err := ssz.HashTreeRoot(dataAndCustodyBit)
	if err != nil {
		t.Error(err)
	}
	att := &ethpb.Attestation{
		Data:        data,
		CustodyBits: bitfield.Bitlist{0x00, 0x00, 0x00, 0x00, 0x01},
	}

	// We setup the genesis state with 256 validators.
	deposits, _, privKeys := testutil.SetupInitialDeposits(t, 256)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}
	stateRoot, err := ssz.HashTreeRoot(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	block := blocks.NewGenesisBlock(stateRoot[:])
	blockRoot, err := ssz.HashTreeRoot(block)
	if err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveBlock(ctx, block); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveHeadBlockRoot(ctx, blockRoot); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveState(ctx, beaconState, blockRoot); err != nil {
		t.Fatal(err)
	}

	// Next up, we compute the committee for the attestation we're testing.
	committee, err := helpers.BeaconCommittee(beaconState, att.Data.Slot, att.Data.Index)
	if err != nil {
		t.Error(err)
	}
	attDataRoot, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		t.Error(err)
	}
	totalAggBits := bitfield.NewBitlist(uint64(len(committee)))
	domain := helpers.Domain(beaconState.Fork, 0, params.BeaconConfig().DomainBeaconAttester)

	// For every single member of the committee, we sign the attestation data and handle
	// the attestation through the operations service, which will perform basic aggregation
	// and verification.
	//
	// We perform these operations concurrently with a wait group to more closely
	// emulate a production environment.
	var wg sync.WaitGroup
	for i := 0; i < len(committee); i++ {
		wg.Add(1)
		go func(tt *testing.T, j int, w *sync.WaitGroup) {
			defer w.Done()
			newAtt := &ethpb.Attestation{
				AggregationBits: bitfield.NewBitlist(uint64(len(committee))),
				Data:            data,
				CustodyBits:     bitfield.Bitlist{0x00, 0x00, 0x00, 0x00, 0x01},
				Signature:       privKeys[committee[j]].Sign(root[:], domain).Marshal(),
			}
			newAtt.AggregationBits.SetBitAt(uint64(j), true)
			if err := opsSrv.HandleAttestation(ctx, newAtt); err != nil {
				tt.Fatalf("Could not handle attestation %d: %v", j, err)
			}
			totalAggBits = totalAggBits.Or(newAtt.AggregationBits)
		}(t, i, &wg)
	}
	wg.Wait()

	// We fetch the final attestation from the attestation pool, which should be an aggregation of
	// all committee members effectively.
	aggAtt := opsSrv.attestationPool[attDataRoot].ToAttestations()[0]
	b1 := aggAtt.AggregationBits.Bytes()
	b2 := totalAggBits.Bytes()

	// We check if the aggregation bits are what we want.
	if !bytes.Equal(b1, b2) {
		t.Errorf("Wanted aggregation bytes %v, received %v", b2, b1)
	}

	// If the committee is larger than 1, the signature from the attestation fetched from the DB
	// should be an aggregate of signatures and not equal to an individual signature from a validator.
	if len(committee) > 1 && bytes.Equal(aggAtt.Signature, att.Signature) {
		t.Errorf("Expected aggregate signature %#x to be different from individual sig %#x", aggAtt.Signature, att.Signature)
	}
}

func TestHandleAttestation_Skips_PreviouslyAggregatedAttestations(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, beaconDB)
	helpers.ClearAllCaches()
	service := NewService(context.Background(), &Config{
		BeaconDB: beaconDB,
	})
	service.attestationPool = make(map[[32]byte]*dbpb.AttestationContainer)

	deposits, _, privKeys := testutil.SetupInitialDeposits(t, 200)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}

	beaconState.CurrentJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	att1 := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
		},
		CustodyBits: bitfield.Bitlist{0x00, 0x00, 0x00, 0x00, 0x01},
	}

	committee, err := helpers.BeaconCommittee(beaconState, att1.Data.Slot, att1.Data.Index)
	if err != nil {
		t.Error(err)
	}
	aggregationBits := bitfield.NewBitlist(uint64(len(committee)))
	aggregationBits.SetBitAt(0, true)
	att1.AggregationBits = aggregationBits

	dataAndCustodyBit := &pb.AttestationDataAndCustodyBit{
		Data:       att1.Data,
		CustodyBit: false,
	}
	hashTreeRoot, err := ssz.HashTreeRoot(dataAndCustodyBit)
	if err != nil {
		t.Error(err)
	}
	domain := helpers.Domain(beaconState.Fork, 0, params.BeaconConfig().DomainBeaconAttester)
	att1.Signature = privKeys[committee[0]].Sign(hashTreeRoot[:], domain).Marshal()

	att2 := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target: &ethpb.Checkpoint{Epoch: 0},
		},
		CustodyBits: bitfield.Bitlist{0x00, 0x00, 0x00, 0x00, 0x01},
	}
	aggregationBits = bitfield.NewBitlist(uint64(len(committee)))
	aggregationBits.SetBitAt(1, true)
	att2.AggregationBits = aggregationBits

	att2.Signature = privKeys[committee[1]].Sign(hashTreeRoot[:], domain).Marshal()

	att3 := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target: &ethpb.Checkpoint{Epoch: 0},
		},
		CustodyBits: bitfield.Bitlist{0x00, 0x00, 0x00, 0x00, 0x01},
	}
	aggregationBits = bitfield.NewBitlist(uint64(len(committee)))
	aggregationBits.SetBitAt(0, true)
	aggregationBits.SetBitAt(1, true)
	att3.AggregationBits = aggregationBits

	att3Sig1 := privKeys[committee[0]].Sign(hashTreeRoot[:], domain)
	att3Sig2 := privKeys[committee[1]].Sign(hashTreeRoot[:], domain)
	aggregatedSig := bls.AggregateSignatures([]*bls.Signature{att3Sig1, att3Sig2}).Marshal()
	att3.Signature = aggregatedSig[:]

	newBlock := &ethpb.BeaconBlock{
		Slot: 0,
	}
	newBlockRoot, err := ssz.HashTreeRoot(newBlock)
	if err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveBlock(context.Background(), newBlock); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveHeadBlockRoot(context.Background(), newBlockRoot); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveState(context.Background(), beaconState, newBlockRoot); err != nil {
		t.Fatal(err)
	}
	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay

	if err := service.HandleAttestation(context.Background(), att1); err != nil {
		t.Error(err)
	}
	if err := service.HandleAttestation(context.Background(), att2); err != nil {
		t.Error(err)
	}
	if err := service.HandleAttestation(context.Background(), att1); err != nil {
		t.Error(err)
	}

	attDataHash, err := ssz.HashTreeRoot(att2.Data)
	if err != nil {
		t.Error(err)
	}
	dbAtt := service.attestationPool[attDataHash].ToAttestations()[0]

	dbAttBits := dbAtt.AggregationBits.Bytes()
	aggregatedBits := att1.AggregationBits.Or(att2.AggregationBits).Bytes()
	if !bytes.Equal(dbAttBits, aggregatedBits) {
		t.Error("Expected aggregation bits to be equal.")
	}

	if !bytes.Equal(dbAtt.Signature, aggregatedSig) {
		t.Error("Expected aggregated signatures to be equal")
	}

	if err := service.HandleAttestation(context.Background(), att2); err != nil {
		t.Error(err)
	}
	dbAtt = service.attestationPool[attDataHash].ToAttestations()[0]

	dbAttBits = dbAtt.AggregationBits.Bytes()
	if !bytes.Equal(dbAttBits, aggregatedBits) {
		t.Error("Expected aggregation bits to be equal.")
	}

	if !bytes.Equal(dbAtt.Signature, aggregatedSig) {
		t.Error("Expected aggregated signatures to be equal")
	}

	if err := service.HandleAttestation(context.Background(), att3); err != nil {
		t.Error(err)
	}
	dbAtt = service.attestationPool[attDataHash].ToAttestations()[0]

	dbAttBits = dbAtt.AggregationBits.Bytes()
	if !bytes.Equal(dbAttBits, aggregatedBits) {
		t.Error("Expected aggregation bits to be equal.")
	}

	if !bytes.Equal(dbAtt.Signature, aggregatedSig) {
		t.Error("Expected aggregated signatures to be equal")
	}
}

func TestRetrieveAttestations_OK(t *testing.T) {
	helpers.ClearAllCaches()
	beaconDB := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, beaconDB)
	service := NewService(context.Background(), &Config{BeaconDB: beaconDB})
	service.attestationPool = make(map[[32]byte]*dbpb.AttestationContainer)

	deposits, _, privKeys := testutil.SetupInitialDeposits(t, 32)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}

	aggBits := bitfield.NewBitlist(1)
	aggBits.SetBitAt(1, true)
	custodyBits := bitfield.NewBitlist(1)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: params.BeaconConfig().ZeroHash[:]},
			Target: &ethpb.Checkpoint{Epoch: 0},
		},
		AggregationBits: aggBits,
		CustodyBits:     custodyBits,
	}
	attestingIndices, err := helpers.AttestingIndices(beaconState, att.Data, att.AggregationBits)
	if err != nil {
		t.Error(err)
	}
	dataAndCustodyBit := &pb.AttestationDataAndCustodyBit{
		Data:       att.Data,
		CustodyBit: false,
	}
	domain := helpers.Domain(beaconState.Fork, 0, params.BeaconConfig().DomainBeaconAttester)
	sigs := make([]*bls.Signature, len(attestingIndices))

	zeroSig := [96]byte{}
	att.Signature = zeroSig[:]
	for i, indice := range attestingIndices {
		hashTreeRoot, err := ssz.HashTreeRoot(dataAndCustodyBit)
		if err != nil {
			t.Error(err)
		}
		sig := privKeys[indice].Sign(hashTreeRoot[:], domain)
		sigs[i] = sig
	}

	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay

	att.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	r, _ := ssz.HashTreeRoot(att.Data)
	service.attestationPool[r] = dbpb.NewContainerFromAttestations([]*ethpb.Attestation{att})

	headBlockRoot := [32]byte{1, 2, 3}
	if err := beaconDB.SaveHeadBlockRoot(context.Background(), headBlockRoot); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveState(context.Background(), beaconState, headBlockRoot); err != nil {
		t.Fatal(err)
	}
	// Test we can retrieve attestations from slot1 - slot61.
	attestations, err := service.AttestationPool(context.Background(), 1)
	if err != nil {
		t.Fatalf("Could not retrieve attestations: %v", err)
	}

	if !reflect.DeepEqual(attestations[0], att) {
		t.Error("Retrieved attestations did not match")
	}
}

func TestRetrieveAttestations_PruneInvalidAtts(t *testing.T) {
	helpers.ClearAllCaches()
	beaconDB := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, beaconDB)
	service := NewService(context.Background(), &Config{BeaconDB: beaconDB})

	origAttestations := make([]*ethpb.Attestation, 140)
	for i := 0; i < len(origAttestations); i++ {
		origAttestations[i] = &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Slot:   uint64(i),
				Source: &ethpb.Checkpoint{},
				Target: &ethpb.Checkpoint{},
			},
			AggregationBits: bitfield.Bitlist{0b11},
		}
		if err := service.beaconDB.SaveAttestation(context.Background(), origAttestations[i]); err != nil {
			t.Fatalf("Failed to save attestation: %v", err)
		}
	}

	headBlockRoot := [32]byte{1, 2, 3}
	if err := beaconDB.SaveHeadBlockRoot(context.Background(), headBlockRoot); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveState(context.Background(), &pb.BeaconState{
		Slot: 200}, headBlockRoot); err != nil {
		t.Fatal(err)
	}
	attestations, err := service.AttestationPool(context.Background(), 200)
	if err != nil {
		t.Fatalf("Could not retrieve attestations: %v", err)
	}

	if len(attestations) != 0 {
		t.Error("Incorrect pruned attestations")
	}

	// Verify the invalid attestations are deleted.
	hash, err := hashutil.HashProto(origAttestations[1])
	if err != nil {
		t.Fatal(err)
	}
	if service.beaconDB.HasAttestation(context.Background(), hash) {
		t.Error("Invalid attestation is not deleted")
	}
}

func TestRemoveProcessedAttestations_Ok(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, beaconDB)
	s := NewService(context.Background(), &Config{BeaconDB: beaconDB})

	attestations := make([]*ethpb.Attestation, 10)
	for i := 0; i < len(attestations); i++ {
		attestations[i] = &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Slot:   uint64(i),
				Source: &ethpb.Checkpoint{},
				Target: &ethpb.Checkpoint{},
			},
			AggregationBits: bitfield.Bitlist{0b11},
		}
		if err := s.beaconDB.SaveAttestation(context.Background(), attestations[i]); err != nil {
			t.Fatalf("Failed to save attestation: %v", err)
		}
	}
	headBlockRoot := [32]byte{1, 2, 3}
	if err := beaconDB.SaveHeadBlockRoot(context.Background(), headBlockRoot); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveState(context.Background(), &pb.BeaconState{Slot: 15}, headBlockRoot); err != nil {
		t.Fatal(err)
	}

	if err := s.removeAttestationsFromPool(context.Background(), attestations); err != nil {
		t.Fatalf("Could not remove attestations: %v", err)
	}

	atts, err := s.AttestationPool(context.Background(), 15)
	if err != nil {
		t.Fatal(err)
	}
	if len(atts) != 0 {
		t.Errorf("Attestation pool should be empty but got a length of %d", len(atts))
	}
}

func TestForkchoiceRetrieveAttestations_NotVoted(t *testing.T) {
	helpers.ClearAllCaches()
	beaconDB := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, beaconDB)
	service := NewService(context.Background(), &Config{BeaconDB: beaconDB})
	service.attestationPool = make(map[[32]byte]*dbpb.AttestationContainer)

	aggBits := bitfield.NewBitlist(8)
	aggBits.SetBitAt(1, true)
	custodyBits := bitfield.NewBitlist(8)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{},
			Target: &ethpb.Checkpoint{},
		},
		AggregationBits: aggBits,
		CustodyBits:     custodyBits,
	}

	r, _ := ssz.HashTreeRoot(att.Data)
	service.attestationPool[r] = dbpb.NewContainerFromAttestations([]*ethpb.Attestation{att})

	atts, err := service.AttestationPoolForForkchoice(context.Background())
	if err != nil {
		t.Fatalf("Could not retrieve attestations: %v", err)
	}

	if !reflect.DeepEqual(atts[0], att) {
		t.Error("Did not receive wanted attestation")
	}
}

func TestForkchoiceRetrieveAttestations_AlreadyVoted(t *testing.T) {
	helpers.ClearAllCaches()
	beaconDB := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, beaconDB)
	service := NewService(context.Background(), &Config{BeaconDB: beaconDB})
	service.attestationPool = make(map[[32]byte]*dbpb.AttestationContainer)

	aggBits := bitfield.NewBitlist(8)
	aggBits.SetBitAt(1, true)
	custodyBits := bitfield.NewBitlist(8)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{},
			Target: &ethpb.Checkpoint{},
		},
		AggregationBits: aggBits,
		CustodyBits:     custodyBits,
	}

	r, _ := ssz.HashTreeRoot(att.Data)
	service.attestationPool[r] = dbpb.NewContainerFromAttestations([]*ethpb.Attestation{att})

	_, err := service.AttestationPoolForForkchoice(context.Background())
	if err != nil {
		t.Fatalf("Could not retrieve attestations: %v", err)
	}

	atts, err := service.AttestationPoolForForkchoice(context.Background())
	if err != nil {
		t.Fatalf("Could not retrieve attestations: %v", err)
	}

	if len(atts) != 0 {
		t.Errorf("Wanted att count 0, got %d", len(atts))
	}
}
