package blocks_test

import (
	"context"
	"io/ioutil"
	"math"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

var validatorCount = 128
var runAmount = 50
var conditions = "SML"

var deposits, privs = testutil.GenerateDeposits(&testing.B{}, uint64(validatorCount))

func setBenchmarkConfig() {
	logrus.Printf("Running block benchmarks for %d validators", validatorCount)
	logrus.SetLevel(logrus.PanicLevel)
	logrus.SetOutput(ioutil.Discard)
	c := params.DemoBeaconConfig()
	if conditions == "BIG" {
		c.MaxProposerSlashings = 16
		c.MaxAttesterSlashings = 1
		c.MaxAttestations = 128
		c.MaxDeposits = 16
		c.MaxVoluntaryExits = 16
	} else if conditions == "SML" {
		c.MaxAttesterSlashings = 1
		c.MaxProposerSlashings = 1
		c.MaxAttestations = 16
		c.MaxDeposits = 1
		c.MaxVoluntaryExits = 1
	}
	params.OverrideBeaconConfig(c)
}

func cleanUp() {
	params.OverrideBeaconConfig(params.MainnetConfig())
}

func TestBenchmarkProcessBlock_PerformsSuccessfully(t *testing.T) {
	beaconState, block := createBeaconStateAndBlock(t)
	if _, err := state.ProcessBlock(context.Background(), beaconState, block); err != nil {
		t.Fatalf("failed to process block, benchmarks will fail: %v", err)
	}
	cleanUp()
}

func BenchmarkProcessBlockHeader(b *testing.B) {
	beaconState, block := createBeaconStateAndBlock(b)
	cleanStates := createCleanStates(beaconState)
	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessBlockHeader(cleanStates[i], block)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func BenchmarkProcessBlockRandao(b *testing.B) {
	beaconState, block := createBeaconStateAndBlock(b)
	cleanStates := createCleanStates(beaconState)

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessRandao(cleanStates[i], block.Body)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func BenchmarkProcessEth1Data(b *testing.B) {
	beaconState, block := createBeaconStateAndBlock(b)
	cleanStates := createCleanStates(beaconState)

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessEth1DataInBlock(cleanStates[i], block)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func BenchmarkProcessValidatorExits(b *testing.B) {
	beaconState, block := createBeaconStateAndBlock(b)
	cleanStates := createCleanStates(beaconState)

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessVoluntaryExits(cleanStates[i], block.Body)
		if err != nil {
			b.Fatalf("run %d, %v", i, err)
		}
	}
	cleanUp()
}

func BenchmarkProcessProposerSlashings(b *testing.B) {
	beaconState, block := createBeaconStateAndBlock(b)
	cleanStates := createCleanStates(beaconState)

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessProposerSlashings(cleanStates[i], block.Body)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func BenchmarkProcessAttesterSlashings(b *testing.B) {
	beaconState, block := createBeaconStateAndBlock(b)
	cleanStates := createCleanStates(beaconState)
	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessAttesterSlashings(cleanStates[i], block.Body)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func BenchmarkProcessAttestations(b *testing.B) {
	beaconState, block := createBeaconStateAndBlock(b)
	cleanStates := createCleanStates(beaconState)

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessAttestations(cleanStates[i], block.Body)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func BenchmarkProcessValidatorDeposits(b *testing.B) {
	beaconState, block := createBeaconStateAndBlock(b)
	cleanStates := createCleanStates(beaconState)

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessDeposits(cleanStates[i], block.Body)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func BenchmarkProcessBlock(b *testing.B) {
	beaconState, block := createBeaconStateAndBlock(b)
	cleanStates := createCleanStates(beaconState)

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := state.ProcessBlock(context.Background(), cleanStates[i], block); err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func BenchmarkExecuteStateTransition(b *testing.B) {
	beaconState, block := createBeaconStateAndBlock(b)
	cleanStates := createCleanStates(beaconState)

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := state.ExecuteStateTransitionNoVerify(context.Background(), cleanStates[i], block); err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func BenchmarkBeaconProposerIndex(b *testing.B) {
	beaconState, _ := createBeaconStateAndBlock(b)

	b.N = 100
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := helpers.BeaconProposerIndex(beaconState)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func BenchmarkCrosslinkCommitee(b *testing.B) {
	beaconState, _ := createBeaconStateAndBlock(b)

	b.N = 100
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := helpers.CrosslinkCommittee(beaconState, helpers.CurrentEpoch(beaconState), 0)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

// MAKE FULL BLOCK SIMULATOR
func createFullBlock(b testing.TB, bState *pb.BeaconState) (*ethpb.BeaconBlock, []byte) {
	currentSlot := bState.Slot
	currentEpoch := helpers.CurrentEpoch(bState)
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	validatorCount, err := helpers.ActiveValidatorCount(bState, currentEpoch)
	if err != nil {
		b.Fatal(err)
	}

	committeesPerEpoch, err := helpers.CommitteeCount(bState, currentEpoch)
	if err != nil {
		b.Fatal(err)
	}

	committeeSize := uint64(math.Ceil(float64(validatorCount) / float64(committeesPerEpoch)))

	proposerSlashings := make([]*ethpb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
	for i := uint64(0); i < params.BeaconConfig().MaxProposerSlashings; i++ {
		proposerIndex := i + uint64(validatorCount/4)
		header1 := &ethpb.BeaconBlockHeader{
			Slot:     currentSlot - (i % slotsPerEpoch),
			BodyRoot: []byte{0, 1, 0},
		}
		root, err := ssz.SigningRoot(header1)
		if err != nil {
			b.Fatal(err)
		}
		domain := helpers.Domain(bState, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
		header1.Signature = privs[proposerIndex].Sign(root[:], domain).Marshal()

		header2 := &ethpb.BeaconBlockHeader{
			Slot:     currentSlot - (i % slotsPerEpoch),
			BodyRoot: []byte{0, 2, 0},
		}
		root, err = ssz.SigningRoot(header2)
		if err != nil {
			b.Fatal(err)
		}
		header2.Signature = privs[proposerIndex].Sign(root[:], domain).Marshal()

		slashing := &ethpb.ProposerSlashing{
			ProposerIndex: proposerIndex,
			Header_1:      header1,
			Header_2:      header2,
		}
		proposerSlashings[i] = slashing
	}

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
			b.Fatal(err)
		}
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
			b.Fatal(err)
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
			b.Fatal(err)
		}
		sig = privs[committee[i]].Sign(dataRoot[:], domain)
		att2.Signature = bls.AggregateSignatures([]*bls.Signature{sig}).Marshal()

		indexedAtt1, err := blocks.ConvertToIndexed(bState, att1)
		if err != nil {
			b.Fatal(err)
		}
		indexedAtt2, err := blocks.ConvertToIndexed(bState, att2)
		if err != nil {
			b.Fatal(err)
		}
		slashing := &ethpb.AttesterSlashing{
			Attestation_1: indexedAtt1,
			Attestation_2: indexedAtt2,
		}
		attesterSlashings[i] = slashing
	}

	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().SlotsPerHistoricalRoot; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

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
			b.Fatal(err)
		}
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
			b.Fatal(err)
		}
		domain := helpers.Domain(bState, parentCrosslink.EndEpoch+1, params.BeaconConfig().DomainAttestation)
		sig := privs[committee[i]].Sign(dataRoot[:], domain)
		att.Signature = bls.AggregateSignatures([]*bls.Signature{sig}).Marshal()
		attestations[i] = att
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
			b.Fatal(err)
		}
		domain := helpers.Domain(bState, currentEpoch, params.BeaconConfig().DomainVoluntaryExit)
		exit.Signature = privs[valIndex].Sign(root[:], domain).Marshal()
		voluntaryExits[i] = exit
	}

	previousDepsLen := uint64(len(deposits))
	newDeposits, _ := testutil.GenerateDeposits(&testing.B{}, params.BeaconConfig().MaxDeposits)
	encodedDeposits := make([][]byte, previousDepsLen)
	for i := 0; i < int(previousDepsLen); i++ {
		hashedDeposit, err := ssz.HashTreeRoot(deposits[i].Data)
		if err != nil {
			b.Fatal(err)
		}
		encodedDeposits[i] = hashedDeposit[:]
	}
	newHashes := make([][]byte, len(newDeposits))
	for i := 0; i < len(newDeposits); i++ {
		hashedDeposit, err := ssz.HashTreeRoot(newDeposits[i].Data)
		if err != nil {
			b.Fatal(err)
		}
		newHashes[i] = hashedDeposit[:]
	}
	allData := append(encodedDeposits, newHashes...)
	depositTrie, err := trieutil.GenerateTrieFromItems(allData, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < len(newDeposits); i++ {
		proof, err := depositTrie.MerkleProof(int(previousDepsLen) + i)
		if err != nil {
			b.Fatal(err)
		}
		newDeposits[i] = &ethpb.Deposit{
			Data:  newDeposits[i].Data,
			Proof: proof,
		}
	}
	root := depositTrie.Root()

	parentRoot, err := ssz.SigningRoot(bState.LatestBlockHeader)
	if err != nil {
		b.Fatal(err)
	}

	block := &ethpb.BeaconBlock{
		Slot:       currentSlot,
		ParentRoot: parentRoot[:],
		Body: &ethpb.BeaconBlockBody{
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot:  root[:],
				BlockHash:    root[:],
				DepositCount: uint64(len(deposits)),
			},
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			VoluntaryExits:    voluntaryExits,
			Deposits:          newDeposits,
		},
	}

	return block, root[:]
}

func getStateRoot(bState *pb.BeaconState, block *ethpb.BeaconBlock) ([]byte, error) {
	s, err := state.ExecuteStateTransitionNoVerify(context.Background(), bState, block)
	if err != nil {
		return nil, err
	}

	root, err := ssz.HashTreeRoot(s)
	if err != nil {
		return nil, err
	}
	return root[:], nil
}

func signBlockAndRandao(bState *pb.BeaconState, block *ethpb.BeaconBlock) error {
	reveal, err := testutil.CreateRandaoReveal(bState, helpers.CurrentEpoch(bState), privs)
	if err != nil {
		return err
	}
	block.Body.RandaoReveal = reveal
	s, err := state.ExecuteStateTransitionNoVerify(context.Background(), bState, block)
	if err != nil {
		return err
	}
	root, err := ssz.HashTreeRoot(s)
	if err != nil {
		return err
	}
	block.StateRoot = root[:]
	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		return err
	}
	proposerIdx, err := helpers.BeaconProposerIndex(bState)
	if err != nil {
		return err
	}
	domain := helpers.Domain(bState, helpers.CurrentEpoch(bState), params.BeaconConfig().DomainBeaconProposer)
	block.Signature = privs[proposerIdx].Sign(blockRoot[:], domain).Marshal()
	return nil
}

func createBeaconStateAndBlock(b testing.TB) (*pb.BeaconState, *ethpb.BeaconBlock) {
	setBenchmarkConfig()

	eth1Data := testutil.GenerateEth1Data(b, deposits)
	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		b.Fatal(err)
	}

	genesisState.Slot = params.BeaconConfig().PersistentCommitteePeriod*8 + (params.BeaconConfig().SlotsPerEpoch / 2)
	genesisState.CurrentJustifiedCheckpoint.Epoch = helpers.PrevEpoch(genesisState)
	crosslinks := make([]*ethpb.Crosslink, params.BeaconConfig().ShardCount)
	for i := 0; i < len(crosslinks); i++ {
		crosslinks[i] = &ethpb.Crosslink{
			Shard:      uint64(i),
			StartEpoch: helpers.PrevEpoch(genesisState) - 1,
			EndEpoch:   helpers.PrevEpoch(genesisState),
			DataRoot:   params.BeaconConfig().ZeroHash[:],
		}
	}
	genesisState.CurrentCrosslinks = crosslinks

	genesisState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot: genesisState.Slot,
	}
	genesisState.Eth1DepositIndex = uint64(len(deposits))
	fullBlock, root := createFullBlock(b, genesisState)
	genesisState.Eth1Data = &ethpb.Eth1Data{
		BlockHash:    root,
		DepositRoot:  root,
		DepositCount: uint64(len(deposits)) + params.BeaconConfig().MaxDeposits,
	}
	genesisState.Eth1DataVotes = []*ethpb.Eth1Data{
		{
			BlockHash:   root,
			DepositRoot: root,
		},
	}
	if err := signBlockAndRandao(genesisState, fullBlock); err != nil {
		b.Fatal(err)
	}
	return genesisState, fullBlock
}

func createCleanStates(beaconState *pb.BeaconState) []*pb.BeaconState {
	cleanStates := make([]*pb.BeaconState, runAmount)
	for i := 0; i < runAmount; i++ {
		cleanStates[i] = proto.Clone(beaconState).(*pb.BeaconState)
	}
	return cleanStates
}
