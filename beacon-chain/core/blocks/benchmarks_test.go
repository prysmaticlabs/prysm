package blocks_test

import (
	"context"
	"io/ioutil"
	"math"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

func setBenchmarkConfig(conditions string, validatorCount uint64) {
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
		c.MaxDeposits = 2
		c.MaxVoluntaryExits = 2
	}
	params.OverrideBeaconConfig(c)
}

func cleanUp() {
	params.OverrideBeaconConfig(params.BeaconConfig())
}

func TestBenchmarkProcessBlock_PerformsSuccessfully(t *testing.T) {
	beaconState, deposits := createGenesisState()
	cfg := &state.TransitionConfig{
		VerifySignatures: false,
		Logging:          false,
	}

	beaconState.Slot = params.BeaconConfig().SlotsPerEpoch*2048 + 3
	beaconState.CurrentJustifiedCheckpoint.Epoch = helpers.PrevEpoch(beaconState)
	beaconState.CurrentCrosslinks[0].EndEpoch = helpers.PrevEpoch(beaconState) - 1
	beaconState.CurrentCrosslinks[0].StartEpoch = helpers.PrevEpoch(beaconState) - 3
	fullBlock, benchRoot := createFullBlock(beaconState, deposits)
	beaconState.Eth1Data = &pb.Eth1Data{
		BlockHash:   benchRoot,
		DepositRoot: benchRoot,
	}
	if _, err := state.ProcessBlock(context.Background(), beaconState, fullBlock, cfg); err != nil {
		t.Fatalf("failed to process block, benchmarks will fail: %v", err)
	}
	cleanUp()
}

func BenchmarkProcessBlockHeader(b *testing.B) {
	genesisState, deposits := createGenesisState()
	block, _ := createFullBlock(genesisState, deposits)
	cleanStates := createCleanStates(genesisState)
	b.N = 30
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
	genesisState, deposits := createGenesisState()
	block, _ := createFullBlock(genesisState, deposits)
	cleanStates := createCleanStates(genesisState)
	b.N = 30
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessRandao(
			cleanStates[i],
			block.Body,
			false, /* verify signatures */
			false, /* disable logging */
		)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func BenchmarkProcessEth1Data(b *testing.B) {
	genesisState, deposits := createGenesisState()
	block, root := createFullBlock(genesisState, deposits)
	cleanStates := createCleanStates(genesisState)

	eth1DataVotes := []*pb.Eth1Data{
		{
			BlockHash:   root,
			DepositRoot: root,
		},
	}

	b.N = 30
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cleanStates[i].Eth1DataVotes = eth1DataVotes
		_, err := blocks.ProcessEth1DataInBlock(cleanStates[i], block)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func BenchmarkProcessValidatorExits(b *testing.B) {
	genesisState, deposits := createGenesisState()
	block, _ := createFullBlock(genesisState, deposits)
	cleanStates := createCleanStates(genesisState)
	b.N = 30
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cleanStates[i].Slot = params.BeaconConfig().SlotsPerEpoch * 2048
		_, err := blocks.ProcessVolundaryExits(cleanStates[i], block.Body, false)
		if err != nil {
			b.Fatalf("run %d, %v", i, err)
		}
	}
	cleanUp()
}

func BenchmarkProcessProposerSlashings(b *testing.B) {
	genesisState, deposits := createGenesisState()
	block, _ := createFullBlock(genesisState, deposits)
	cleanStates := createCleanStates(genesisState)
	b.N = 30
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessProposerSlashings(
			cleanStates[i],
			block.Body,
			false,
		)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func BenchmarkProcessAttesterSlashings(b *testing.B) {
	genesisState, deposits := createGenesisState()
	block, _ := createFullBlock(genesisState, deposits)
	cleanStates := createCleanStates(genesisState)
	b.N = 30
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessAttesterSlashings(cleanStates[i], block.Body, false)
		if err != nil {
			b.Fatal(i)
		}
	}
	cleanUp()
}

func BenchmarkProcessBlockAttestations(b *testing.B) {
	genesisState, deposits := createGenesisState()
	block, _ := createFullBlock(genesisState, deposits)
	cleanStates := createCleanStates(genesisState)
	b.N = 30
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessAttestations(
			cleanStates[i],
			block.Body,
			false,
		)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func BenchmarkProcessValidatorDeposits(b *testing.B) {
	genesisState, deposits := createGenesisState()
	block, root := createFullBlock(genesisState, deposits)
	cleanStates := createCleanStates(genesisState)
	b.N = 30
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cleanStates[i].Eth1Data = &pb.Eth1Data{
			BlockHash:   root,
			DepositRoot: root,
		}
		_, err := blocks.ProcessDeposits(cleanStates[i], block.Body, true)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func BenchmarkProcessBlock(b *testing.B) {
	genesisState, deposits := createGenesisState()
	cfg := &state.TransitionConfig{
		VerifySignatures: false,
		Logging:          false,
	}

	cleanStates := createCleanStates(genesisState)
	cleanStates[0].Slot = params.BeaconConfig().SlotsPerEpoch*2048 + 3
	cleanStates[0].CurrentJustifiedCheckpoint.Epoch = helpers.PrevEpoch(cleanStates[0])
	cleanStates[0].CurrentCrosslinks[0].EndEpoch = helpers.PrevEpoch(cleanStates[0]) - 1
	cleanStates[0].CurrentCrosslinks[0].StartEpoch = helpers.PrevEpoch(cleanStates[0]) - 3
	fullBlock, benchRoot := createFullBlock(cleanStates[0], deposits)

	b.N = 30
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		beaconState := cleanStates[i]
		beaconState.Eth1Data = &pb.Eth1Data{
			BlockHash:   benchRoot,
			DepositRoot: benchRoot,
		}
		beaconState.Slot = params.BeaconConfig().SlotsPerEpoch*2048 + 4
		beaconState.CurrentJustifiedCheckpoint.Epoch = helpers.PrevEpoch(beaconState)
		beaconState.CurrentCrosslinks[0] = cleanStates[0].CurrentCrosslinks[0]
		fullBlock.Slot = params.BeaconConfig().SlotsPerEpoch*2048 + 4
		if _, err := state.ProcessBlock(context.Background(), beaconState, fullBlock, cfg); err != nil {
			b.Fatal(err)
		}
	}
	cleanUp()
}

func createFullBlock(bState *pb.BeaconState, previousDeposits []*pb.Deposit) (*pb.BeaconBlock, []byte) {
	currentSlot := bState.Slot
	currentEpoch := helpers.CurrentEpoch(bState)
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	validatorCount, err := helpers.ActiveValidatorCount(bState, currentEpoch)
	if err != nil {
		panic(err)
	}

	committeesPerEpoch, err := helpers.EpochCommitteeCount(bState, currentEpoch)
	if err != nil {
		panic(err)
	}

	committeeSize := int(math.Ceil(float64(validatorCount) / float64(committeesPerEpoch)))

	proposerSlashings := make([]*pb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
	for i := uint64(0); i < params.BeaconConfig().MaxProposerSlashings; i++ {
		slashing := &pb.ProposerSlashing{
			ProposerIndex: i + uint64(validatorCount/4),
			Header_1: &pb.BeaconBlockHeader{
				Slot:     currentSlot - (i % slotsPerEpoch),
				BodyRoot: []byte{0, 1, 0},
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:     currentSlot - (i % slotsPerEpoch),
				BodyRoot: []byte{0, 2, 0},
			},
		}
		proposerSlashings[i] = slashing
	}

	maxSlashes := params.BeaconConfig().MaxAttesterSlashings
	attesterSlashings := make([]*pb.AttesterSlashing, maxSlashes)
	for i := uint64(0); i < maxSlashes; i++ {
		crosslink := &pb.Crosslink{
			Shard:    i % params.BeaconConfig().ShardCount,
			EndEpoch: i,
		}
		attData1 := &pb.AttestationData{
			Crosslink:   crosslink,
			TargetEpoch: i,
			SourceEpoch: i + 1,
		}
		attData2 := &pb.AttestationData{
			Crosslink:   crosslink,
			TargetEpoch: i,
			SourceEpoch: i,
		}
		aggregationBits, err := bitutil.SetBitfield(int(i), committeeSize)
		if err != nil {
			panic(err)
		}
		att1 := &pb.Attestation{
			Data:            attData1,
			AggregationBits: aggregationBits,
		}
		att2 := &pb.Attestation{
			Data:            attData2,
			AggregationBits: aggregationBits,
		}

		indexedAtt1, err := blocks.ConvertToIndexed(bState, att1)
		if err != nil {
			panic(err)
		}
		indexedAtt2, err := blocks.ConvertToIndexed(bState, att2)
		if err != nil {
			panic(err)
		}
		slashing := &pb.AttesterSlashing{
			Attestation_1: indexedAtt1,
			Attestation_2: indexedAtt2,
		}
		attesterSlashings[i] = slashing
	}

	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().SlotsPerHistoricalRoot; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

	attestations := make([]*pb.Attestation, params.BeaconConfig().MaxAttestations)
	for i := uint64(0); i < params.BeaconConfig().MaxAttestations; i++ {
		crosslink := &pb.Crosslink{
			Shard:      i % params.BeaconConfig().ShardCount,
			StartEpoch: currentEpoch - 2,
			EndEpoch:   currentEpoch,
			DataRoot:   params.BeaconConfig().ZeroHash[:],
		}
		parentCrosslink := bState.CurrentCrosslinks[crosslink.Shard]
		crosslinkParentRoot, err := ssz.HashTreeRoot(parentCrosslink)
		if err != nil {
			panic(err)
		}
		crosslink.ParentRoot = crosslinkParentRoot[:]

		aggregationBitfield, err := bitutil.SetBitfield(int(i), committeeSize)
		if err != nil {
			panic(err)
		}
		custodyBitfield := bitutil.FillBitfield(committeeSize)
		att1 := &pb.Attestation{
			Data: &pb.AttestationData{
				Crosslink:       crosslink,
				SourceEpoch:     helpers.PrevEpoch(bState),
				TargetEpoch:     currentEpoch,
				BeaconBlockRoot: params.BeaconConfig().ZeroHash[:],
				SourceRoot:      params.BeaconConfig().ZeroHash[:],
				TargetRoot:      params.BeaconConfig().ZeroHash[:],
			},
			AggregationBits: aggregationBitfield,
			CustodyBits:     custodyBitfield,
		}
		attestations[i] = att1
	}

	voluntaryExits := make([]*pb.VoluntaryExit, params.BeaconConfig().MaxVoluntaryExits)
	for i := 0; i < len(voluntaryExits); i++ {
		voluntaryExits[i] = &pb.VoluntaryExit{
			Epoch:          currentEpoch - 1,
			ValidatorIndex: validatorCount*uint64(2/3) + uint64(i),
		}
	}

	previousDepsLen := uint64(len(previousDeposits))
	newDeposits, _ := testutil.GenerateDeposits(&testing.B{}, params.BeaconConfig().MaxDeposits, false)
	encodedDeposits := make([][]byte, previousDepsLen)
	for i := 0; i < int(previousDepsLen); i++ {
		hashedDeposit, err := ssz.HashTreeRoot(previousDeposits[i].Data)
		if err != nil {
			panic(err)
		}
		encodedDeposits[i] = hashedDeposit[:]
	}
	newHashes := make([][]byte, len(newDeposits))
	for i := 0; i < len(newDeposits); i++ {
		hashedDeposit, err := ssz.HashTreeRoot(newDeposits[i].Data)
		if err != nil {
			panic(err)
		}
		newHashes[i] = hashedDeposit[:]
	}
	allData := append(encodedDeposits, newHashes...)
	depositTrie, err := trieutil.GenerateTrieFromItems(allData, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		panic(err)
	}
	for i := 0; i < len(newDeposits); i++ {
		proof, err := depositTrie.MerkleProof(int(previousDepsLen) + i)
		if err != nil {
			panic(err)
		}
		newDeposits[i] = &pb.Deposit{
			Data:  newDeposits[i].Data,
			Proof: proof,
		}
	}
	root := depositTrie.Root()

	parentRoot, err := ssz.SigningRoot(bState.LatestBlockHeader)
	if err != nil {
		panic(err)
	}

	block := &pb.BeaconBlock{
		Slot:       currentSlot,
		ParentRoot: parentRoot[:],
		Body: &pb.BeaconBlockBody{
			RandaoReveal: []byte{2, 3, 4},
			Eth1Data: &pb.Eth1Data{
				DepositRoot: root[:],
				BlockHash:   root[:],
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

func createGenesisState(numDeposits int) (*pb.BeaconState, []*pb.Deposit) {
	setBenchmarkConfig("BIG")
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		pubkey := []byte{}
		pubkey = make([]byte, params.BeaconConfig().BLSPubkeyLength)
		copy(pubkey[:], []byte(strconv.FormatUint(uint64(i), 10)))

		depositData := &pb.DepositData{
			Pubkey:                pubkey,
			Amount:                params.BeaconConfig().MaxEffectiveBalance,
			WithdrawalCredentials: []byte{1},
		}
		deposits[i] = &pb.Deposit{
			Data: depositData,
		}
	}

	encodedDeposits := make([][]byte, len(deposits))
	for i := 0; i < len(encodedDeposits); i++ {
		hashedDeposit, err := ssz.HashTreeRoot(deposits[i].Data)
		if err != nil {
			panic(err)
		}
		encodedDeposits[i] = hashedDeposit[:]
	}

	depositTrie, err := trieutil.GenerateTrieFromItems(encodedDeposits, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		panic(err)
	}

	for i := range deposits {
		proof, err := depositTrie.MerkleProof(i)
		if err != nil {
			panic(err)
		}
		deposits[i].Proof = proof
	}

	root := depositTrie.Root()
	eth1Data := &pb.Eth1Data{
		BlockHash:   root[:],
		DepositRoot: root[:],
	}

	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		panic(err)
	}

	genesisState.Slot = 4*params.BeaconConfig().SlotsPerEpoch - 1
	genesisState.CurrentJustifiedCheckpoint.Epoch = helpers.CurrentEpoch(genesisState) - 1
	genesisState.CurrentCrosslinks = []*pb.Crosslink{
		{
			Shard:      0,
			StartEpoch: 0,
			EndEpoch:   1,
			DataRoot:   params.BeaconConfig().ZeroHash[:],
		},
	}
	genesisState.LatestBlockHeader = &pb.BeaconBlockHeader{
		Slot: genesisState.Slot,
	}

	return genesisState, deposits
}

func createCleanStates(beaconState *pb.BeaconState) []*pb.BeaconState {
	cleanStates := make([]*pb.BeaconState, 30)
	for i := 0; i < 30; i++ {
		cleanStates[i] = proto.Clone(beaconState).(*pb.BeaconState)
	}
	return cleanStates
}
