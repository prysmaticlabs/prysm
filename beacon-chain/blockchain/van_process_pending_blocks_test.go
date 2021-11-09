package blockchain

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/proto/interfaces"
	vanTypes "github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/shared/van_mock"
	"testing"
	"time"
)

// TestService_PublishAndStorePendingBlock checks PublishAndStorePendingBlock method
func TestService_PublishBlock(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	cfg := &Config{
		BeaconDB:      beaconDB,
		StateGen:      stategen.New(beaconDB),
		BlockNotifier: &mock.MockBlockNotifier{RecordEvents: true},
		StateNotifier: &mock.MockStateNotifier{RecordEvents: true},
	}
	s, err := NewService(ctx, cfg)
	require.NoError(t, err)
	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	wrappedGenesisBlk := wrapper.WrappedPhase0SignedBeaconBlock(genesis)
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrappedGenesisBlk))
	require.NoError(t, err)
	b := testutil.NewBeaconBlock()
	wrappedBlk := wrapper.WrappedPhase0SignedBeaconBlock(b)
	s.publishBlock(wrappedBlk)
	time.Sleep(3 * time.Second)
	if recvd := len(s.blockNotifier.(*mock.MockBlockNotifier).ReceivedEvents()); recvd < 1 {
		t.Errorf("Received %d pending block notifications, expected at least 1", recvd)
	}
}

func TestService_FetchConfirmation_Ok(t *testing.T) {
	tests := []struct {
		name               string
		inputs             []interfaces.SignedBeaconBlock
		confirmationStatus []*vanTypes.ConfirmationResData
		outputs            *orcConfirmationData
		errMsg             string
	}{
		{
			name:   "Returns confirmation status from orchestrator",
			inputs: getBeaconBlocks(0, 1),
			confirmationStatus: []*vanTypes.ConfirmationResData{
				{
					Slot:   0,
					Status: vanTypes.Verified,
				},
			},
			outputs: &orcConfirmationData{
				slot:   0,
				status: vanTypes.Verified,
			},
		},
		{
			name:   "Returns error when orchestrator sends invalid response",
			inputs: getBeaconBlocks(0, 1),
			errMsg: "invalid length of orchestrator confirmation response",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			var mockedOrcClient *van_mock.MockClient
			ctrl := gomock.NewController(t)
			mockedOrcClient = van_mock.NewMockClient(ctrl)

			cfg := &Config{
				BlockNotifier:      &mock.MockBlockNotifier{},
				OrcRPCClient:       mockedOrcClient,
				EnableVanguardNode: true,
			}
			s, err := NewService(ctx, cfg)
			require.NoError(t, err)

			if tt.errMsg == "" {
				mockedOrcClient.EXPECT().ConfirmVanBlockHashes(
					gomock.Any(),
					gomock.Any(),
				).AnyTimes().Return(tt.confirmationStatus, nil)
				actualOutput, err := s.fetchConfirmations(tt.inputs[0])
				require.NoError(t, err)
				assert.DeepEqual(t, tt.outputs, actualOutput)
			} else {
				mockedOrcClient.EXPECT().ConfirmVanBlockHashes(
					gomock.Any(),
					gomock.Any(),
				).AnyTimes().Return(nil, errors.New("invalid length of orchestrator confirmation response"))
				_, err := s.fetchConfirmations(tt.inputs[0])
				require.ErrorContains(t, tt.errMsg, err)
			}
		})
	}
}

func TestService_WaitForConfirmation_PendingStatus_ReturnError(t *testing.T) {
	ctx := context.Background()
	var mockedOrcClient *van_mock.MockClient
	ctrl := gomock.NewController(t)
	mockedOrcClient = van_mock.NewMockClient(ctrl)
	cfg := &Config{
		BlockNotifier:      &mock.MockBlockNotifier{},
		OrcRPCClient:       mockedOrcClient,
		EnableVanguardNode: true,
	}
	s, err := NewService(ctx, cfg)
	require.NoError(t, err)
	orcResponse := []*vanTypes.ConfirmationResData{
		{
			Slot:   15,
			Status: vanTypes.Pending,
		},
	}
	blk := getBeaconBlock(15)
	mockedOrcClient.EXPECT().ConfirmVanBlockHashes(
		gomock.Any(),
		gomock.Any(),
	).AnyTimes().Return(orcResponse, nil)

	exitRoutine := make(chan bool)
	go func(tt *testing.T) {
		assert.ErrorContains(tt, "maximum wait is exceeded and orchestrator can not verify the block", s.waitForConfirmation(blk))
		exitRoutine <- true
	}(t)
	<-exitRoutine
}

func TestService_WaitForConfirmation_VerifiedStatus_AfterFewPendingStatus(t *testing.T) {
	ctx := context.Background()
	var mockedOrcClient *van_mock.MockClient
	ctrl := gomock.NewController(t)
	mockedOrcClient = van_mock.NewMockClient(ctrl)
	cfg := &Config{
		BlockNotifier:      &mock.MockBlockNotifier{},
		OrcRPCClient:       mockedOrcClient,
		EnableVanguardNode: true,
	}
	s, err := NewService(ctx, cfg)
	require.NoError(t, err)
	pendingRes := []*vanTypes.ConfirmationResData{
		{
			Slot:   15,
			Status: vanTypes.Pending,
		},
	}
	verifiedRes := []*vanTypes.ConfirmationResData{
		{
			Slot:   15,
			Status: vanTypes.Verified,
		},
	}
	blk := getBeaconBlock(15)
	mockedOrcClient.EXPECT().ConfirmVanBlockHashes(
		gomock.Any(),
		gomock.Any(),
	).Times(10).Return(pendingRes, nil)
	mockedOrcClient.EXPECT().ConfirmVanBlockHashes(
		gomock.Any(),
		gomock.Any(),
	).Times(1).Return(verifiedRes, nil)
	exitRoutine := make(chan bool)
	go func(tt *testing.T) {
		assert.Equal(tt, nil, s.waitForConfirmation(blk))
		exitRoutine <- true
	}(t)
	<-exitRoutine
}

// Helper method to generate pending queue with random blocks
func getBeaconBlocks(from, to int) []interfaces.SignedBeaconBlock {
	pendingBlks := make([]interfaces.SignedBeaconBlock, to-from)
	for i := 0; i < to-from; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = types.Slot(from + i)
		wrappedBlk := wrapper.WrappedPhase0SignedBeaconBlock(b)
		pendingBlks[i] = wrappedBlk
	}
	return pendingBlks
}

// Helper method to generate pending queue with random block
func getBeaconBlock(slot types.Slot) interfaces.SignedBeaconBlock {
	b := testutil.NewBeaconBlock()
	b.Block.Slot = types.Slot(slot)
	return wrapper.WrappedPhase0SignedBeaconBlock(b)
}
