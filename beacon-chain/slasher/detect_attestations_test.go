package slasher

import (
	"context"
	"fmt"
	"testing"
	"time"

	mock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	dbtest "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	slashingsmock "github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/slashings/mock"
	slashertypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/common"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func Test_processAttestations(t *testing.T) {
	type (
		attestationInfo struct {
			source          primitives.Epoch
			target          primitives.Epoch
			indices         []uint64
			beaconBlockRoot []byte
		}

		slashingInfo struct {
			attestationInfo_1 *attestationInfo
			attestationInfo_2 *attestationInfo
		}
	)

	tests := []struct {
		name                  string
		currentEpoch          primitives.Epoch
		attestationsInfo      []*attestationInfo
		expectedSlashingsInfo []*slashingInfo
	}{
		{
			name:         "Same target with different signing roots",
			currentEpoch: 4,
			attestationsInfo: []*attestationInfo{
				{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: []byte{1}},
				{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: []byte{2}},
			},
			expectedSlashingsInfo: []*slashingInfo{
				{
					attestationInfo_1: &attestationInfo{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: []byte{1}},
					attestationInfo_2: &attestationInfo{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: []byte{2}},
				},
				{
					attestationInfo_1: &attestationInfo{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: []byte{1}},
					attestationInfo_2: &attestationInfo{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: []byte{2}},
				},
			},
		},
		{
			name:         "Same target with same signing roots",
			currentEpoch: 4,
			attestationsInfo: []*attestationInfo{
				{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: []byte{1}},
				{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: []byte{1}},
			},
			expectedSlashingsInfo: nil,
		},
		{
			name:         "Detects surrounding vote (source 1, target 2), (source 0, target 3)",
			currentEpoch: 4,
			attestationsInfo: []*attestationInfo{
				{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				{source: 0, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
			},
			expectedSlashingsInfo: []*slashingInfo{
				{
					attestationInfo_1: &attestationInfo{source: 0, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
					attestationInfo_2: &attestationInfo{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				},
				{
					attestationInfo_1: &attestationInfo{source: 0, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
					attestationInfo_2: &attestationInfo{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				},
				{
					attestationInfo_1: &attestationInfo{source: 0, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
					attestationInfo_2: &attestationInfo{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				},
				{
					attestationInfo_1: &attestationInfo{source: 0, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
					attestationInfo_2: &attestationInfo{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				},
			},
		},
		{
			name:         "Detects surrounding vote (source 50, target 51), (source 0, target 1000)",
			currentEpoch: 1000,
			attestationsInfo: []*attestationInfo{
				{source: 50, target: 51, indices: []uint64{0}, beaconBlockRoot: nil},
				{source: 0, target: 1000, indices: []uint64{0}, beaconBlockRoot: nil},
			},
			expectedSlashingsInfo: []*slashingInfo{
				{
					attestationInfo_1: &attestationInfo{source: 0, target: 1000, indices: []uint64{0}, beaconBlockRoot: nil},
					attestationInfo_2: &attestationInfo{source: 50, target: 51, indices: []uint64{0}, beaconBlockRoot: nil},
				},
				{
					attestationInfo_1: &attestationInfo{source: 0, target: 1000, indices: []uint64{0}, beaconBlockRoot: nil},
					attestationInfo_2: &attestationInfo{source: 50, target: 51, indices: []uint64{0}, beaconBlockRoot: nil},
				},
			},
		},
		{
			name:         "Detects surrounded vote (source 0, target 3), (source 1, target 2)",
			currentEpoch: 4,
			attestationsInfo: []*attestationInfo{
				{source: 0, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
			},
			expectedSlashingsInfo: []*slashingInfo{
				{
					attestationInfo_1: &attestationInfo{source: 0, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
					attestationInfo_2: &attestationInfo{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				},
				{
					attestationInfo_1: &attestationInfo{source: 0, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
					attestationInfo_2: &attestationInfo{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				},
			},
		},
		{
			name:         "Detects surrounded vote (source 0, target 3), (source 1, target 2)",
			currentEpoch: 4,
			attestationsInfo: []*attestationInfo{
				{source: 0, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
			},
			expectedSlashingsInfo: []*slashingInfo{
				{
					attestationInfo_1: &attestationInfo{source: 0, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
					attestationInfo_2: &attestationInfo{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				},
				{
					attestationInfo_1: &attestationInfo{source: 0, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
					attestationInfo_2: &attestationInfo{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				},
			},
		},
		{
			name:         "Detects double vote, (source 1, target 2), (source 0, target 2)",
			currentEpoch: 4,
			attestationsInfo: []*attestationInfo{
				{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				{source: 0, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
			},
			expectedSlashingsInfo: []*slashingInfo{
				{
					attestationInfo_1: &attestationInfo{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
					attestationInfo_2: &attestationInfo{source: 0, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				},
				{
					attestationInfo_1: &attestationInfo{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
					attestationInfo_2: &attestationInfo{source: 0, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				},
			},
		},
		{
			name:         "Not slashable, surrounding but non-overlapping attesting indices within same validator chunk index",
			currentEpoch: 4,
			attestationsInfo: []*attestationInfo{
				{source: 1, target: 2, indices: []uint64{0}, beaconBlockRoot: nil},
				{source: 0, target: 3, indices: []uint64{1}, beaconBlockRoot: nil},
			},
			expectedSlashingsInfo: nil,
		},
		{
			name:         "Not slashable, surrounded but non-overlapping attesting indices within same validator chunk index",
			currentEpoch: 4,
			attestationsInfo: []*attestationInfo{
				{source: 0, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				{source: 1, target: 2, indices: []uint64{2, 3}, beaconBlockRoot: nil},
			},
			expectedSlashingsInfo: nil,
		},
		{
			name:         "Not slashable, surrounding but non-overlapping attesting indices in different validator chunk index",
			currentEpoch: 4,
			attestationsInfo: []*attestationInfo{
				{source: 0, target: 3, indices: []uint64{0}, beaconBlockRoot: nil},
				{source: 1, target: 2, indices: []uint64{params.BeaconConfig().MinGenesisActiveValidatorCount - 1}, beaconBlockRoot: nil},
			},
			expectedSlashingsInfo: nil,
		},
		{
			name:         "Not slashable, surrounded but non-overlapping attesting indices in different validator chunk index",
			currentEpoch: 4,
			attestationsInfo: []*attestationInfo{
				{source: 0, target: 3, indices: []uint64{0}, beaconBlockRoot: nil},
				{source: 1, target: 2, indices: []uint64{params.BeaconConfig().MinGenesisActiveValidatorCount - 1}, beaconBlockRoot: nil},
			},
			expectedSlashingsInfo: nil,
		},
		{
			name:         "Not slashable, (source 1, target 2), (source 2, target 3)",
			currentEpoch: 4,
			attestationsInfo: []*attestationInfo{
				{source: 1, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				{source: 2, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
			},
			expectedSlashingsInfo: nil,
		},
		{
			name:         "Not slashable, (source 0, target 3), (source 2, target 4)",
			currentEpoch: 4,
			attestationsInfo: []*attestationInfo{
				{source: 0, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				{source: 2, target: 4, indices: []uint64{0, 1}, beaconBlockRoot: nil},
			},
			expectedSlashingsInfo: nil,
		},
		{
			name:         "Not slashable, (source 0, target 2), (source 0, target 3)",
			currentEpoch: 4,
			attestationsInfo: []*attestationInfo{
				{source: 0, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				{source: 0, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
			},
			expectedSlashingsInfo: nil,
		},
		{
			name:         "Not slashable, (source 0, target 3), (source 0, target 2)",
			currentEpoch: 4,
			attestationsInfo: []*attestationInfo{
				{source: 0, target: 3, indices: []uint64{0, 1}, beaconBlockRoot: nil},
				{source: 0, target: 2, indices: []uint64{0, 1}, beaconBlockRoot: nil},
			},
			expectedSlashingsInfo: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context.
			ctx := context.Background()

			// Configure logging.
			hook := logTest.NewGlobal()
			defer hook.Reset()

			// Configure the slasher database.
			slasherDB := dbtest.SetupSlasherDB(t)

			// Configure the beacon state.
			beaconState, err := util.NewBeaconState()
			require.NoError(t, err)

			// Get the currentSlot for the current epoch.
			currentSlot, err := slots.EpochStart(tt.currentEpoch)
			require.NoError(t, err)

			// Create the mock chain service.
			mockChain := &mock.ChainService{State: beaconState}

			// Create the mock slashing pool inserter.
			mockSlashingPoolInserter := &slashingsmock.PoolMock{}

			// Create the service configuration.
			serviceConfig := &ServiceConfig{
				Database:                slasherDB,
				HeadStateFetcher:        mockChain,
				AttestationStateFetcher: mockChain,
				SlashingPoolInserter:    mockSlashingPoolInserter,
			}

			// Create the slasher service.
			slasherService, err := New(context.Background(), serviceConfig)
			require.NoError(t, err)

			// Initialize validators in the state.
			numVals := params.BeaconConfig().MinGenesisActiveValidatorCount
			validators := make([]*ethpb.Validator, numVals)
			privateKeys := make([]bls.SecretKey, numVals)

			for i := uint64(0); i < numVals; i++ {
				// Create a random private key.
				privateKey, err := bls.RandKey()
				require.NoError(t, err)

				// Add the private key to the list.
				privateKeys[i] = privateKey

				// Derive the public key from the private key.
				publicKey := privateKey.PublicKey().Marshal()

				// Initialize the validator.
				validator := &ethpb.Validator{PublicKey: publicKey}

				// Add the validator to the list.
				validators[i] = validator
			}

			// Set the validators into the state.
			err = beaconState.SetValidators(validators)
			require.NoError(t, err)

			// Compute the signing domain.
			domain, err := signing.Domain(
				beaconState.Fork(),
				0,
				params.BeaconConfig().DomainBeaconAttester,
				beaconState.GenesisValidatorsRoot(),
			)
			require.NoError(t, err)

			// Build attestation wrappers.
			attestationsCount := len(tt.attestationsInfo)
			attestationWrappers := make([]*slashertypes.IndexedAttestationWrapper, 0, attestationsCount)
			for _, attestationInfo := range tt.attestationsInfo {
				// Create a wrapped attestation.
				attestationWrapper := createAttestationWrapper(
					t,
					domain,
					privateKeys,
					attestationInfo.source,
					attestationInfo.target,
					attestationInfo.indices,
					attestationInfo.beaconBlockRoot,
				)

				// Add the wrapped attestation to the list.
				attestationWrappers = append(attestationWrappers, attestationWrapper)
			}

			// Build expected attester slashings.
			expectedSlashings := make([]*ethpb.AttesterSlashing, 0, len(tt.expectedSlashingsInfo))
			for _, slashingInfo := range tt.expectedSlashingsInfo {
				// Create attestations.
				attestation_1 := createAttestationWrapper(
					t,
					domain,
					privateKeys,
					slashingInfo.attestationInfo_1.source,
					slashingInfo.attestationInfo_1.target,
					slashingInfo.attestationInfo_1.indices,
					slashingInfo.attestationInfo_1.beaconBlockRoot,
				)

				attestation_2 := createAttestationWrapper(
					t,
					domain,
					privateKeys,
					slashingInfo.attestationInfo_2.source,
					slashingInfo.attestationInfo_2.target,
					slashingInfo.attestationInfo_2.indices,
					slashingInfo.attestationInfo_2.beaconBlockRoot,
				)

				// Create the attester slashing.
				attesterSlashing := &ethpb.AttesterSlashing{
					Attestation_1: attestation_1.IndexedAttestation,
					Attestation_2: attestation_2.IndexedAttestation,
				}

				// Add the attester slashing to the list.
				expectedSlashings = append(expectedSlashings, attesterSlashing)
			}

			// Process the attestations.
			processedSlashings := slasherService.processAttestations(ctx, attestationWrappers, currentSlot)

			// Check the processed slashings correspond to the expected slashings.
			assert.DeepSSZEqual(t, expectedSlashings, processedSlashings)
		})
	}
}

func Test_processQueuedAttestations_MultipleChunkIndices(t *testing.T) {
	hook := logTest.NewGlobal()
	defer hook.Reset()

	slasherDB := dbtest.SetupSlasherDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	slasherParams := DefaultParams()

	// We process submit attestations from chunk index 0 to chunk index 1.
	// What we want to test here is if we can proceed
	// with processing queued attestations once the chunk index changes.
	// For example, epochs 0 - 15 are chunk 0, epochs 16 - 31 are chunk 1, etc.
	startEpoch := primitives.Epoch(slasherParams.chunkSize)
	endEpoch := primitives.Epoch(slasherParams.chunkSize + 1)

	currentTime := time.Now()
	totalSlots := uint64(startEpoch) * uint64(params.BeaconConfig().SlotsPerEpoch)
	secondsSinceGenesis := time.Duration(totalSlots * params.BeaconConfig().SecondsPerSlot)
	genesisTime := currentTime.Add(-secondsSinceGenesis * time.Second)

	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
	mockChain := &mock.ChainService{
		State: beaconState,
	}

	s, err := New(context.Background(),
		&ServiceConfig{
			Database:                slasherDB,
			StateNotifier:           &mock.MockStateNotifier{},
			HeadStateFetcher:        mockChain,
			AttestationStateFetcher: mockChain,
			SlashingPoolInserter:    &slashingsmock.PoolMock{},
			ClockWaiter:             startup.NewClockSynchronizer(),
		})
	require.NoError(t, err)
	s.genesisTime = genesisTime

	currentSlotChan := make(chan primitives.Slot)
	s.wg.Add(1)
	go func() {
		s.processQueuedAttestations(ctx, currentSlotChan)
	}()

	for i := startEpoch; i <= endEpoch; i++ {
		source := primitives.Epoch(0)
		target := primitives.Epoch(0)
		if i != 0 {
			source = i - 1
			target = i
		}
		var sr [32]byte
		copy(sr[:], fmt.Sprintf("%d", i))
		att := createAttestationWrapperEmptySig(t, source, target, []uint64{0}, sr[:])
		s.attsQueue = newAttestationsQueue()
		s.attsQueue.push(att)
		slot, err := slots.EpochStart(i)
		require.NoError(t, err)
		require.NoError(t, mockChain.State.SetSlot(slot))
		s.serviceCfg.HeadStateFetcher = mockChain
		currentSlotChan <- slot
	}

	time.Sleep(time.Millisecond * 200)
	cancel()
	s.wg.Wait()
	require.LogsDoNotContain(t, hook, "Slashable offenses found")
	require.LogsDoNotContain(t, hook, "Could not detect")
}

func Test_processQueuedAttestations_OverlappingChunkIndices(t *testing.T) {
	hook := logTest.NewGlobal()
	defer hook.Reset()

	slasherDB := dbtest.SetupSlasherDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	slasherParams := DefaultParams()

	startEpoch := primitives.Epoch(slasherParams.chunkSize)

	currentTime := time.Now()
	totalSlots := uint64(startEpoch) * uint64(params.BeaconConfig().SlotsPerEpoch)
	secondsSinceGenesis := time.Duration(totalSlots * params.BeaconConfig().SecondsPerSlot)
	genesisTime := currentTime.Add(-secondsSinceGenesis * time.Second)

	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
	mockChain := &mock.ChainService{
		State: beaconState,
	}

	s, err := New(context.Background(),
		&ServiceConfig{
			Database:                slasherDB,
			StateNotifier:           &mock.MockStateNotifier{},
			HeadStateFetcher:        mockChain,
			AttestationStateFetcher: mockChain,
			SlashingPoolInserter:    &slashingsmock.PoolMock{},
			ClockWaiter:             startup.NewClockSynchronizer(),
		})
	require.NoError(t, err)
	s.genesisTime = genesisTime

	currentSlotChan := make(chan primitives.Slot)
	s.wg.Add(1)
	go func() {
		s.processQueuedAttestations(ctx, currentSlotChan)
	}()

	// We create two attestations fully spanning chunk indices 0 and chunk 1
	att1 := createAttestationWrapperEmptySig(t, primitives.Epoch(slasherParams.chunkSize-2), primitives.Epoch(slasherParams.chunkSize), []uint64{0, 1}, nil)
	att2 := createAttestationWrapperEmptySig(t, primitives.Epoch(slasherParams.chunkSize-1), primitives.Epoch(slasherParams.chunkSize+1), []uint64{0, 1}, nil)

	// We attempt to process the batch.
	s.attsQueue = newAttestationsQueue()
	s.attsQueue.push(att1)
	s.attsQueue.push(att2)
	slot, err := slots.EpochStart(att2.IndexedAttestation.Data.Target.Epoch)
	require.NoError(t, err)
	mockChain.Slot = &slot
	s.serviceCfg.HeadStateFetcher = mockChain
	currentSlotChan <- slot

	time.Sleep(time.Millisecond * 200)
	cancel()
	s.wg.Wait()
	require.LogsDoNotContain(t, hook, "Slashable offenses found")
	require.LogsDoNotContain(t, hook, "Could not detect")
}

func Test_epochUpdateForValidators(t *testing.T) {
	ctx := context.Background()
	slasherDB := dbtest.SetupSlasherDB(t)

	// Check if the chunk at chunk index already exists in-memory.
	s := &Service{
		params: &Parameters{
			chunkSize:          2, // 2 epochs in a chunk.
			validatorChunkSize: 2, // 2 validators in a chunk.
			historyLength:      4,
		},
		serviceCfg:                     &ServiceConfig{Database: slasherDB},
		latestEpochWrittenForValidator: map[primitives.ValidatorIndex]primitives.Epoch{},
	}

	t.Run("no update if no latest written epoch", func(t *testing.T) {
		validators := []primitives.ValidatorIndex{
			1, 2,
		}
		currentEpoch := primitives.Epoch(3)
		// No last written epoch for both validators.
		s.latestEpochWrittenForValidator = map[primitives.ValidatorIndex]primitives.Epoch{}

		// Because the validators have no recorded latest epoch written, we expect
		// no chunks to be loaded nor updated to.
		updatedChunks := make(map[uint64]Chunker)
		for _, valIdx := range validators {
			err := s.epochUpdateForValidator(
				ctx,
				updatedChunks,
				0, // validatorChunkIndex
				slashertypes.MinSpan,
				currentEpoch,
				valIdx,
			)
			require.NoError(t, err)
		}
		require.Equal(t, 0, len(updatedChunks))
	})

	t.Run("update from latest written epoch", func(t *testing.T) {
		validators := []primitives.ValidatorIndex{
			1, 2,
		}
		currentEpoch := primitives.Epoch(3)

		// Set the latest written epoch for validators to current epoch - 1.
		latestWrittenEpoch := currentEpoch - 1
		s.latestEpochWrittenForValidator = map[primitives.ValidatorIndex]primitives.Epoch{
			1: latestWrittenEpoch,
			2: latestWrittenEpoch,
		}

		// Because the latest written epoch for the input validators is == 2, we expect
		// that we will update all epochs from 2 up to 3 (the current epoch). This is all
		// safe contained in chunk index 1.
		updatedChunks := make(map[uint64]Chunker)
		for _, valIdx := range validators {
			err := s.epochUpdateForValidator(
				ctx,
				updatedChunks,
				0, // validatorChunkIndex,
				slashertypes.MinSpan,
				currentEpoch,
				valIdx,
			)
			require.NoError(t, err)
		}
		require.Equal(t, 1, len(updatedChunks))
		_, ok := updatedChunks[1]
		require.Equal(t, true, ok)
	})
}

func Test_applyAttestationForValidator_MinSpanChunk(t *testing.T) {
	ctx := context.Background()
	slasherDB := dbtest.SetupSlasherDB(t)
	srv, err := New(context.Background(),
		&ServiceConfig{
			Database:      slasherDB,
			StateNotifier: &mock.MockStateNotifier{},
			ClockWaiter:   startup.NewClockSynchronizer(),
		})
	require.NoError(t, err)

	// We initialize an empty chunks slice.
	currentEpoch := primitives.Epoch(3)
	validatorChunkIndex := uint64(0)
	validatorIdx := primitives.ValidatorIndex(0)
	chunksByChunkIdx := map[uint64]Chunker{}

	// We apply attestation with (source 1, target 2) for our validator.
	source := primitives.Epoch(1)
	target := primitives.Epoch(2)
	att := createAttestationWrapperEmptySig(t, source, target, nil, nil)
	slashing, err := srv.applyAttestationForValidator(
		ctx,
		chunksByChunkIdx,
		att,
		slashertypes.MinSpan,
		validatorChunkIndex,
		validatorIdx,
		currentEpoch,
	)
	require.NoError(t, err)
	require.IsNil(t, slashing)
	att.IndexedAttestation.AttestingIndices = []uint64{uint64(validatorIdx)}
	err = slasherDB.SaveAttestationRecordsForValidators(
		ctx,
		[]*slashertypes.IndexedAttestationWrapper{att},
	)
	require.NoError(t, err)

	// Next, we apply an attestation with (source 0, target 3) and
	// expect a slashable offense to be returned.
	source = primitives.Epoch(0)
	target = primitives.Epoch(3)
	slashableAtt := createAttestationWrapperEmptySig(t, source, target, nil, nil)
	slashing, err = srv.applyAttestationForValidator(
		ctx,
		chunksByChunkIdx,
		slashableAtt,
		slashertypes.MinSpan,
		validatorChunkIndex,
		validatorIdx,
		currentEpoch,
	)
	require.NoError(t, err)
	require.NotNil(t, slashing)
}

func Test_applyAttestationForValidator_MaxSpanChunk(t *testing.T) {
	ctx := context.Background()
	slasherDB := dbtest.SetupSlasherDB(t)
	srv, err := New(context.Background(),
		&ServiceConfig{
			Database:      slasherDB,
			StateNotifier: &mock.MockStateNotifier{},
			ClockWaiter:   startup.NewClockSynchronizer(),
		})
	require.NoError(t, err)

	// We initialize an empty chunks slice.
	currentEpoch := primitives.Epoch(3)
	validatorChunkIndex := uint64(0)
	validatorIdx := primitives.ValidatorIndex(0)
	chunksByChunkIdx := map[uint64]Chunker{}

	// We apply attestation with (source 0, target 3) for our validator.
	source := primitives.Epoch(0)
	target := primitives.Epoch(3)
	att := createAttestationWrapperEmptySig(t, source, target, nil, nil)
	slashing, err := srv.applyAttestationForValidator(
		ctx,
		chunksByChunkIdx,
		att,
		slashertypes.MaxSpan,
		validatorChunkIndex,
		validatorIdx,
		currentEpoch,
	)
	require.NoError(t, err)
	require.Equal(t, true, slashing == nil)
	att.IndexedAttestation.AttestingIndices = []uint64{uint64(validatorIdx)}
	err = slasherDB.SaveAttestationRecordsForValidators(
		ctx,
		[]*slashertypes.IndexedAttestationWrapper{att},
	)
	require.NoError(t, err)

	// Next, we apply an attestation with (source 1, target 2) and
	// expect a slashable offense to be returned.
	source = primitives.Epoch(1)
	target = primitives.Epoch(2)
	slashableAtt := createAttestationWrapperEmptySig(t, source, target, nil, nil)
	slashing, err = srv.applyAttestationForValidator(
		ctx,
		chunksByChunkIdx,
		slashableAtt,
		slashertypes.MaxSpan,
		validatorChunkIndex,
		validatorIdx,
		currentEpoch,
	)
	require.NoError(t, err)
	require.NotNil(t, slashing)
}

func Test_checkDoubleVotes_SlashableAttestationsOnDisk(t *testing.T) {
	slasherDB := dbtest.SetupSlasherDB(t)
	ctx := context.Background()
	// For a list of input attestations, check that we can
	// indeed check there could exist a double vote offense
	// within the list with respect to previous entries in the db.
	prevAtts := []*slashertypes.IndexedAttestationWrapper{
		createAttestationWrapperEmptySig(t, 0, 1, []uint64{1, 2}, []byte{1}),
		createAttestationWrapperEmptySig(t, 0, 2, []uint64{1, 2}, []byte{1}),
	}
	srv, err := New(context.Background(),
		&ServiceConfig{
			Database:      slasherDB,
			StateNotifier: &mock.MockStateNotifier{},
			ClockWaiter:   startup.NewClockSynchronizer(),
		})
	require.NoError(t, err)

	err = slasherDB.SaveAttestationRecordsForValidators(ctx, prevAtts)
	require.NoError(t, err)

	prev1 := createAttestationWrapperEmptySig(t, 0, 2, []uint64{1, 2}, []byte{1})
	cur1 := createAttestationWrapperEmptySig(t, 0, 2, []uint64{1, 2}, []byte{2})
	prev2 := createAttestationWrapperEmptySig(t, 0, 2, []uint64{1, 2}, []byte{1})
	cur2 := createAttestationWrapperEmptySig(t, 0, 2, []uint64{1, 2}, []byte{2})
	wanted := []*ethpb.AttesterSlashing{
		{
			Attestation_1: prev1.IndexedAttestation,
			Attestation_2: cur1.IndexedAttestation,
		},
		{
			Attestation_1: prev2.IndexedAttestation,
			Attestation_2: cur2.IndexedAttestation,
		},
	}
	newAtts := []*slashertypes.IndexedAttestationWrapper{
		createAttestationWrapperEmptySig(t, 0, 2, []uint64{1, 2}, []byte{2}), // Different signing root.
	}
	slashings, err := srv.checkDoubleVotes(ctx, newAtts)
	require.NoError(t, err)
	require.DeepEqual(t, wanted, slashings)
}

func Test_loadChunks_MinSpans(t *testing.T) {
	testLoadChunks(t, slashertypes.MinSpan)
}

func Test_loadChunks_MaxSpans(t *testing.T) {
	testLoadChunks(t, slashertypes.MaxSpan)
}

func testLoadChunks(t *testing.T, kind slashertypes.ChunkKind) {
	slasherDB := dbtest.SetupSlasherDB(t)
	ctx := context.Background()

	// Check if the chunk at chunk index already exists in-memory.
	s, err := New(context.Background(),
		&ServiceConfig{
			Database:      slasherDB,
			StateNotifier: &mock.MockStateNotifier{},
			ClockWaiter:   startup.NewClockSynchronizer(),
		})
	require.NoError(t, err)

	defaultParams := s.params

	// If a chunk at a chunk index does not exist, ensure it
	// is initialized as an empty chunk.
	var emptyChunk Chunker
	if kind == slashertypes.MinSpan {
		emptyChunk = EmptyMinSpanChunksSlice(defaultParams)
	} else {
		emptyChunk = EmptyMaxSpanChunksSlice(defaultParams)
	}
	chunkIdx := uint64(2)
	received, err := s.loadChunks(ctx, 0, kind, []uint64{chunkIdx})
	require.NoError(t, err)
	wanted := map[uint64]Chunker{
		chunkIdx: emptyChunk,
	}
	require.DeepEqual(t, wanted, received)

	// Save chunks to disk, then load them properly from disk.
	var existingChunk Chunker
	if kind == slashertypes.MinSpan {
		existingChunk = EmptyMinSpanChunksSlice(defaultParams)
	} else {
		existingChunk = EmptyMaxSpanChunksSlice(defaultParams)
	}
	validatorIdx := primitives.ValidatorIndex(0)
	epochInChunk := primitives.Epoch(0)
	targetEpoch := primitives.Epoch(2)
	err = setChunkDataAtEpoch(
		defaultParams,
		existingChunk.Chunk(),
		validatorIdx,
		epochInChunk,
		targetEpoch,
	)
	require.NoError(t, err)
	require.DeepNotEqual(t, existingChunk, emptyChunk)

	updatedChunks := map[uint64]Chunker{
		2: existingChunk,
		4: existingChunk,
		6: existingChunk,
	}
	err = s.saveUpdatedChunks(
		ctx,
		updatedChunks,
		kind,
		0, // validatorChunkIndex
	)
	require.NoError(t, err)
	// Check if the retrieved chunks match what we just saved to disk.
	received, err = s.loadChunks(ctx, 0, kind, []uint64{2, 4, 6})
	require.NoError(t, err)
	require.DeepEqual(t, updatedChunks, received)
}

func TestService_processQueuedAttestations(t *testing.T) {
	hook := logTest.NewGlobal()
	slasherDB := dbtest.SetupSlasherDB(t)

	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
	slot, err := slots.EpochStart(1)
	require.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(slot))
	mockChain := &mock.ChainService{
		State: beaconState,
		Slot:  &slot,
	}

	s, err := New(context.Background(),
		&ServiceConfig{
			Database:         slasherDB,
			StateNotifier:    &mock.MockStateNotifier{},
			HeadStateFetcher: mockChain,
			ClockWaiter:      startup.NewClockSynchronizer(),
		})
	require.NoError(t, err)

	s.attsQueue.extend([]*slashertypes.IndexedAttestationWrapper{
		createAttestationWrapperEmptySig(t, 0, 1, []uint64{0, 1} /* indices */, nil /* signingRoot */),
	})
	ctx, cancel := context.WithCancel(context.Background())
	tickerChan := make(chan primitives.Slot)
	s.wg.Add(1)
	go func() {
		s.processQueuedAttestations(ctx, tickerChan)
	}()

	// Send a value over the ticker.
	tickerChan <- 1
	cancel()
	s.wg.Wait()
	assert.LogsContain(t, hook, "Processing queued")
}

func BenchmarkCheckSlashableAttestations(b *testing.B) {
	slasherDB := dbtest.SetupSlasherDB(b)

	beaconState, err := util.NewBeaconState()
	require.NoError(b, err)
	slot := primitives.Slot(0)
	mockChain := &mock.ChainService{
		State: beaconState,
		Slot:  &slot,
	}

	s, err := New(context.Background(), &ServiceConfig{
		Database:         slasherDB,
		StateNotifier:    &mock.MockStateNotifier{},
		HeadStateFetcher: mockChain,
		ClockWaiter:      startup.NewClockSynchronizer(),
	})
	require.NoError(b, err)

	b.Run("1 attestation 1 validator", func(b *testing.B) {
		b.ResetTimer()
		runAttestationsBenchmark(b, s, 1, 1 /* validator */)
	})
	b.Run("1 attestation 100 validators", func(b *testing.B) {
		b.ResetTimer()
		runAttestationsBenchmark(b, s, 1, 100 /* validator */)
	})
	b.Run("1 attestation 1000 validators", func(b *testing.B) {
		b.ResetTimer()
		runAttestationsBenchmark(b, s, 1, 1000 /* validator */)
	})

	b.Run("100 attestations 1 validator", func(b *testing.B) {
		b.ResetTimer()
		runAttestationsBenchmark(b, s, 100, 1 /* validator */)
	})
	b.Run("100 attestations 100 validators", func(b *testing.B) {
		b.ResetTimer()
		runAttestationsBenchmark(b, s, 100, 100 /* validator */)
	})
	b.Run("100 attestations 1000 validators", func(b *testing.B) {
		b.ResetTimer()
		runAttestationsBenchmark(b, s, 100, 1000 /* validator */)
	})

	b.Run("1000 attestations 1 validator", func(b *testing.B) {
		b.ResetTimer()
		runAttestationsBenchmark(b, s, 1000, 1 /* validator */)
	})
	b.Run("1000 attestations 100 validators", func(b *testing.B) {
		b.ResetTimer()
		runAttestationsBenchmark(b, s, 1000, 100 /* validator */)
	})
	b.Run("1000 attestations 1000 validators", func(b *testing.B) {
		b.ResetTimer()
		runAttestationsBenchmark(b, s, 1000, 1000 /* validator */)
	})
}

func runAttestationsBenchmark(b *testing.B, s *Service, numAtts, numValidators uint64) {
	indices := make([]uint64, numValidators)
	for i := uint64(0); i < numValidators; i++ {
		indices[i] = i
	}
	atts := make([]*slashertypes.IndexedAttestationWrapper, numAtts)
	for i := uint64(0); i < numAtts; i++ {
		source := primitives.Epoch(i)
		target := primitives.Epoch(i + 1)
		var signingRoot [32]byte
		copy(signingRoot[:], fmt.Sprintf("%d", i))
		atts[i] = createAttestationWrapperEmptySig(
			b,
			source,
			target,         /* target */
			indices,        /* indices */
			signingRoot[:], /* signingRoot */
		)
	}
	for i := 0; i < b.N; i++ {
		numEpochs := numAtts
		totalSeconds := numEpochs * uint64(params.BeaconConfig().SlotsPerEpoch) * params.BeaconConfig().SecondsPerSlot
		genesisTime := time.Now().Add(-time.Second * time.Duration(totalSeconds))
		s.genesisTime = genesisTime

		epoch := slots.EpochsSinceGenesis(genesisTime)
		_, err := s.checkSlashableAttestations(context.Background(), epoch, atts)
		require.NoError(b, err)
	}
}

// createAttestationWrapperEmptySig creates an attestation wrapper with source and target,
// for validators with indices, and a beacon block root (corresponding to the head vote).
// For source and target epochs, the corresponding root is null.
// The signature of the returned wrapped attestation is empty.
func createAttestationWrapperEmptySig(
	t testing.TB,
	source, target primitives.Epoch,
	indices []uint64,
	beaconBlockRoot []byte,
) *slashertypes.IndexedAttestationWrapper {
	data := &ethpb.AttestationData{
		BeaconBlockRoot: bytesutil.PadTo(beaconBlockRoot, 32),
		Source: &ethpb.Checkpoint{
			Epoch: source,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		Target: &ethpb.Checkpoint{
			Epoch: target,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
	}

	dataRoot, err := data.HashTreeRoot()
	require.NoError(t, err)

	return &slashertypes.IndexedAttestationWrapper{
		IndexedAttestation: &ethpb.IndexedAttestation{
			AttestingIndices: indices,
			Data:             data,
			Signature:        params.BeaconConfig().EmptySignature[:],
		},
		DataRoot: dataRoot,
	}
}

// createAttestationWrapper creates an attestation wrapper with source and target,
// for validators with indices, and a beacon block root (corresponding to the head vote).
// For source and target epochs, the corresponding root is null.
// if validatorIndice = indices[i], then the corresponding private key is privateKeys[validatorIndice].
func createAttestationWrapper(
	t testing.TB,
	domain []byte,
	privateKeys []common.SecretKey,
	source, target primitives.Epoch,
	indices []uint64,
	beaconBlockRoot []byte,
) *slashertypes.IndexedAttestationWrapper {
	// Create attestation data.
	attestationData := &ethpb.AttestationData{
		BeaconBlockRoot: bytesutil.PadTo(beaconBlockRoot, 32),
		Source: &ethpb.Checkpoint{
			Epoch: source,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		Target: &ethpb.Checkpoint{
			Epoch: target,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
	}

	// Compute attestation data root.
	attestationDataRoot, err := attestationData.HashTreeRoot()
	require.NoError(t, err)

	// Create valid signatures for all input attestations in the test.
	signingRoot, err := signing.ComputeSigningRoot(attestationData, domain)
	require.NoError(t, err)

	// For each attesting indice in the indexed attestation, create a signature.
	signatures := make([]bls.Signature, 0, len(indices))
	for _, indice := range indices {
		// Check that the indice is within the range of private keys.
		require.Equal(t, true, indice < uint64(len(privateKeys)))

		// Retrieve the corresponding private key.
		privateKey := privateKeys[indice]

		// Sign the signing root.
		signature := privateKey.Sign(signingRoot[:])

		// Append the signature to the signatures list.
		signatures = append(signatures, signature)
	}

	// Compute the aggregated signature.
	signature := bls.AggregateSignatures(signatures).Marshal()

	// Create the attestation wrapper.
	return &slashertypes.IndexedAttestationWrapper{
		IndexedAttestation: &ethpb.IndexedAttestation{
			AttestingIndices: indices,
			Data:             attestationData,
			Signature:        signature,
		},
		DataRoot: attestationDataRoot,
	}
}
