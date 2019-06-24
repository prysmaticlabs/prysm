package blocks_test

import (
	"context"
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
)

var runAmount = 50
var genesisState16K, deposits16K = createGenesisState(16000)
var cleanStates16K = createCleanStates16K(runAmount)

// var genesisState4M = createGenesisState(4000000)

func setBenchmarkConfig(conditions string) {
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

func BenchmarkProcessBlockHeader(b *testing.B) {
	var err error

	block, _ := createFullBlock(b, genesisState16K, deposits16K)

	b.Run("16K", func(b *testing.B) {
		b.N = runAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = blocks.ProcessBlockHeader(cleanStates16K[i], block)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkProcessBlockRandao(b *testing.B) {
	blockBody := &pb.BeaconBlockBody{
		RandaoReveal: []byte{2, 3, 4},
	}

	b.Run("16K", func(b *testing.B) {
		var err error
		b.N = runAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = blocks.ProcessRandao(
				cleanStates16K[i],
				blockBody,
				false, /* verify signatures */
				false, /* disable logging */
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// b.Run("300K", func(b *testing.B) {
	// 	b.N = runAmount
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		_, _ = blocks.ProcessRandao(
	// 			genesisState300K,
	// 			blockBody,
	// 			false, /* verify signatures */
	// 			false, /* disable logging */
	// 		)
	// 	}
	// })

	// b.Run("4M Validators", func(b *testing.B) {
	// 	b.N = runAmount
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		_, _ = blocks.ProcessBlockRandao(
	// 			genesisState4M,
	// 			block,
	// 			false, /* verify signatures */
	// 			false, /* disable logging */
	// 		)
	// 	}
	// })
}

// func BenchmarkProcessValidatorExits(b *testing.B) {
// 	block, _ := createFullBlock(b, genesisState16K, deposits16K)

// 	b.Run("16K", func(b *testing.B) {
// 		b.N = runAmount
// 		b.ResetTimer()
// 		for i := 0; i < b.N; i++ {
// 			cleanStates16K[i].Slot = params.BeaconConfig().SlotsPerEpoch * 2048
// 			_, err := blocks.ProcessValidatorExits(cleanStates16K[i], block, false)
// 			if err != nil {
// 				b.Fatalf("run %d, %v", i, err)
// 			}
// 			cleanStates16K[i].Slot = 2*params.BeaconConfig().SlotsPerEpoch - 1
// 		}
// 	})
// }

// 	block, _ = createFullBlock(b, genesisState300K, deposits300K)

// 	genesisState300K.Slot = params.BeaconConfig().SlotsPerEpoch*2048 + 1
// 	b.Run("300K", func(b *testing.B) {
// 		b.N = runAmount
// 		b.ResetTimer()
// 		for i := 0; i < b.N; i++ {
// 			_, err := blocks.ProcessValidatorExits(cleanStates300K[i], block, false)
// 			if err != nil {
// 				b.Fatal(err)
// 			}
// 			// genesisState300K = cleanStates300K[i]
// 		}
// 	})
// 	genesisState300K.Slot = 2*params.BeaconConfig().SlotsPerEpoch - 1
// }

// func BenchmarkProcessProposerSlashings(b *testing.B) {
// 	var err error

// 	block, _ := createFullBlock(b, genesisState16K, deposits16K)

// 	b.Run("16K", func(b *testing.B) {
// 		b.N = runAmount
// 		b.ResetTimer()
// 		for i := 0; i < b.N; i++ {
// 			_, err = blocks.ProcessProposerSlashings(
// 				cleanStates16K[i],
// 				block,
// 				false,
// 			)
// 			if err != nil {
// 				b.Fatal(err)
// 			}
// 		}
// 	})

// 	block, _ = createFullBlock(b, genesisState300K, deposits300K)

// 	b.Run("300K", func(b *testing.B) {
// 		b.N = runAmount
// 		b.ResetTimer()
// 		for i := 0; i < b.N; i++ {
// 			_, err = blocks.ProcessProposerSlashings(
// 				genesisState300K,
// 				block,
// 				false,
// 			)
// 			if err != nil {
// 				b.Fatal(err)
// 			}
// 			genesisState300K = cleanStates300K[i]
// 		}
// 	})
// }

func BenchmarkProcessAttesterSlashings(b *testing.B) {
	block, _ := createFullBlock(b, genesisState16K, deposits16K)

	var cleanStates16K = createCleanStates16K(runAmount)
	b.Run("16K", func(b *testing.B) {
		b.N = runAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := blocks.ProcessAttesterSlashings(
				cleanStates16K[i],
				block,
				false,
			)
			if err != nil {
				b.Fatal(i)
			}
		}
	})

	// 	block, _ = createFullBlock(b, genesisState300K, deposits300K)

	// 	b.Run("300K", func(b *testing.B) {
	// 		b.N = 1
	// 		b.ResetTimer()
	// 		for i := 0; i < b.N; i++ {
	// 			_, err := blocks.ProcessAttesterSlashings(
	// 				cleanStates300K[i],
	// 				block,
	// 				false,
	// 			)
	// 			if err != nil {
	// 				b.Fatal(err)
	// 			}
	// 		}
	// 	})
}

func BenchmarkProcessBlockAttestations(b *testing.B) {
	block, _ := createFullBlock(b, genesisState16K, deposits16K)

	b.Run("16K", func(b *testing.B) {
		b.N = runAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := blocks.ProcessBlockAttestations(
				cleanStates16K[i],
				block,
				false,
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// block, _ = createFullBlock(b, genesisState300K, deposits300K)

	// b.Run("300K", func(b *testing.B) {
	// 	b.N = runAmount(300000)
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		_, err := blocks.ProcessBlockAttestations(
	// 			genesisState300K,
	// 			block,
	// 			false,
	// 		)
	// 		if err != nil {
	// 			b.Fatal(err)
	// 		}
	// 	}
	// })
}

func BenchmarkProcessValidatorDeposits(b *testing.B) {
	block, root := createFullBlock(b, genesisState16K, deposits16K)

	b.Run("16K", func(b *testing.B) {
		b.N = runAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			beaconState := cleanStates16K[i]
			beaconState.LatestEth1Data = &pb.Eth1Data{
				BlockHash:   root,
				DepositRoot: root,
			}
			_, err := blocks.ProcessValidatorDeposits(
				cleanStates16K[i],
				block,
				true,
			)
			beaconState.DepositIndex = 16000
			if err != nil {
				b.Fatalf("Expected block deposits to process correctly, received: %v", err)
			}
		}
	})

	// block, root = createFullBlock(b, genesisState300K, deposits300K)
	// genesisState300K.LatestEth1Data = &pb.Eth1Data{
	// 	BlockHash:   root,
	// 	DepositRoot: root,
	// }

	// b.Run("300K", func(b *testing.B) {
	// 	b.N = runAmount(300000)
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		_, err := blocks.ProcessValidatorDeposits(
	// 			genesisState300K,
	// 			block,
	// 			true,
	// 		)
	// 		genesisState300K.DepositIndex = 300000
	// 		if err != nil {
	// 			b.Fatalf("Expected block deposits to process correctly, received: %v", err)
	// 		}
	// 	}
	// })
}

func BenchmarkProcessEth1Data(b *testing.B) {
	block, root := createFullBlock(b, genesisState16K, deposits16K)
	b.Run("16K", func(b *testing.B) {
		eth1DataVotes := []*pb.Eth1Data{
			{
				BlockHash:   root,
				DepositRoot: root,
			},
		}

		b.N = runAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			beaconState := cleanStates16K[i]
			beaconState.Eth1DataVotes = eth1DataVotes
			_, err := blocks.ProcessEth1DataInBlock(beaconState, block)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// 	block, _ = createFullBlock(b, genesisState300K, deposits300K)
	// 	eth1DataVotes = []*pb.Eth1Data{
	// 		{
	// 			BlockHash:   root,
	// 			DepositRoot: root,
	// 		},
	// 	}
	// 	genesisState300K.Eth1DataVotes = eth1DataVotes
	// 	b.Run("300K", func(b *testing.B) {
	// 		b.N = runAmount
	// 		b.ResetTimer()
	// 		for i := 0; i < b.N; i++ {
	// 			_, err := blocks.ProcessEth1DataInBlock(genesisState300K, block)
	// 			if err != nil {
	// 				b.Fatal(err)
	// 			}
	// 		}
	// 	})

	// genesisState4M.Eth1DataVotes = eth1DataVotes
	// b.Run("4M Validators", func(b *testing.B) {
	// 	b.N = runAmount
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		_ = blocks.ProcessEth1DataInBlock(genesisState4M, block)
	// 	}
	// })
}

func BenchmarkProcessBlock(b *testing.B) {
	block, root := createFullBlock(b, genesisState16K, deposits16K)

	var cleanStates16KFull = createCleanStates16K(runAmount)

	cfg := &state.TransitionConfig{
		VerifySignatures: false,
		Logging:          false,
	}
	b.Run("16K", func(b *testing.B) {
		b.N = 1
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			beaconState := cleanStates16KFull[i]
			beaconState.LatestEth1Data = &pb.Eth1Data{
				BlockHash:   root,
				DepositRoot: root,
			}
			if _, err := state.ProcessBlock(context.Background(), beaconState, block, cfg); err != nil {
				b.Fatal(err)
			}
			beaconState.DepositIndex = 16000
		}
	})

	// 	block, root = createFullBlock(b, genesisState300K, deposits300K)

	// 	genesisState300K.LatestEth1Data = &pb.Eth1Data{
	// 		BlockHash:   root,
	// 		DepositRoot: root,
	// 	}

	// 	b.Run("300K", func(b *testing.B) {
	// 		b.N = 1
	// 		b.ResetTimer()
	// 		for i := 0; i < b.N; i++ {
	// 			if _, err := state.ProcessBlock(context.Background(), genesisState300K, block, cfg); err != nil {
	// 				b.Fatal(err)
	// 			}
	// 			genesisState300K.DepositIndex = 300000
	// 		}
	// 	})
}

func createFullBlock(b *testing.B, bState *pb.BeaconState, previousDeposits []*pb.Deposit) (*pb.BeaconBlock, []byte) {
	currentSlot := bState.Slot
	currentEpoch := helpers.CurrentEpoch(bState)
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	validatorIndices, err := helpers.ActiveValidatorIndices(bState, currentEpoch)
	if err != nil {
		b.Fatal(err)
	}
	validatorCount := len(validatorIndices)

	committeesPerEpoch, err := helpers.EpochCommitteeCount(bState, currentEpoch)
	if err != nil {
		b.Fatal(err)
	}

	if float64(validatorCount)/float64(committeesPerEpoch) > float64(params.BeaconConfig().MaxIndicesPerAttestation) {
		committeesPerEpoch = uint64(math.Ceil(float64(validatorCount) / float64(params.BeaconConfig().MaxIndicesPerAttestation)))
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
		aggregationBitfield, err := bitutil.SetBitfield(int(i), committeeSize)
		if err != nil {
			b.Fatal(err)
		}

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

		att1 := &pb.Attestation{
			Data:                attData1,
			AggregationBitfield: aggregationBitfield,
		}
		att2 := &pb.Attestation{
			Data:                attData2,
			AggregationBitfield: aggregationBitfield,
		}

		indexedAtt1, err := blocks.ConvertToIndexed(bState, att1)
		if err != nil {
			b.Fatal(err)
		}
		indexedAtt2, err := blocks.ConvertToIndexed(bState, att2)
		if err != nil {
			b.Fatal(err)
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
			b.Fatal(err)
		}

		att1 := &pb.Attestation{
			Data: &pb.AttestationData{
				Crosslink:       crosslink,
				SourceEpoch:     helpers.PrevEpoch(bState),
				TargetEpoch:     currentEpoch,
				BeaconBlockRoot: params.BeaconConfig().ZeroHash[:],
				SourceRoot:      params.BeaconConfig().ZeroHash[:],
				TargetRoot:      params.BeaconConfig().ZeroHash[:],
			},
			AggregationBitfield: aggregationBitfield,
			CustodyBitfield:     []byte{1},
		}
		attestations[i] = att1
	}

	voluntaryExits := make([]*pb.VoluntaryExit, params.BeaconConfig().MaxVoluntaryExits)
	for i := 0; i < len(voluntaryExits); i++ {
		voluntaryExits[i] = &pb.VoluntaryExit{
			Epoch:          currentEpoch - 1,
			ValidatorIndex: uint64(validatorCount*2/3 + i),
		}
	}

	previousDepsLen := uint64(len(previousDeposits))
	newDeposits, _ := testutil.GenerateDeposits(b, params.BeaconConfig().MaxDeposits, false)
	for i := 0; i < len(newDeposits); i++ {
		newDeposits[i].Index = previousDepsLen + uint64(i)
	}

	encodedDeposits := make([][]byte, previousDepsLen)
	for i := 0; i < int(previousDepsLen); i++ {
		hashedDeposit, err := ssz.HashTreeRoot(previousDeposits[i].Data)
		if err != nil {
			b.Fatalf("could not tree hash deposit data: %v", err)
		}
		encodedDeposits[i] = hashedDeposit[:]
	}

	newHashes := make([][]byte, len(newDeposits))
	for i := 0; i < len(newDeposits); i++ {
		hashedDeposit, err := ssz.HashTreeRoot(newDeposits[i].Data)
		if err != nil {
			b.Fatalf("could not tree hash deposit data: %v", err)
		}
		newHashes[i] = hashedDeposit[:]
	}

	allData := append(encodedDeposits, newHashes...)

	depositTrie, err := trieutil.GenerateTrieFromItems(allData, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		b.Errorf("Could not generate trie: %v", err)
	}

	for i := 0; i < len(newDeposits); i++ {
		proof, err := depositTrie.MerkleProof(int(previousDepsLen) + i)
		if err != nil {
			b.Errorf("Could not generate proof: %v", err)
		}

		newDeposits[i] = &pb.Deposit{
			Data:  newDeposits[i].Data,
			Proof: proof,
			Index: previousDepsLen + uint64(i),
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
			RandaoReveal: []byte{1, 2, 3},
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
			Amount:                params.BeaconConfig().MaxDepositAmount,
			WithdrawalCredentials: []byte{1},
		}
		deposits[i] = &pb.Deposit{
			Data:  depositData,
			Index: uint64(i),
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
		proof, err := depositTrie.MerkleProof(int(deposits[i].Index))
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
	genesisState.CurrentJustifiedEpoch = helpers.CurrentEpoch(genesisState) - 1
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

func createCleanStates16K(num int) []*pb.BeaconState {
	cleanStates := make([]*pb.BeaconState, num)
	for i := 0; i < num; i++ {
		cleanStates[i] = proto.Clone(genesisState16K).(*pb.BeaconState)
	}
	return cleanStates
}

func createCleanStates300K(num int) []*pb.BeaconState {
	cleanStates := make([]*pb.BeaconState, num)
	// for i := 0; i < num; i++ {
	// 	cleanStates[i] = proto.Clone(genesisState300K).(*pb.BeaconState)
	// }
	return cleanStates
}
