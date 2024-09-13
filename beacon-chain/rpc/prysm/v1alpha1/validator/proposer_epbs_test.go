package validator

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	mockChain "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	dbutil "github.com/prysmaticlabs/prysm/v5/beacon-chain/db/testing"
	mockExecution "github.com/prysmaticlabs/prysm/v5/beacon-chain/execution/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v5/beacon-chain/forkchoice/doubly-linked-tree"
	p2ptest "github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/testing"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestServer_SubmitSignedExecutionPayloadEnvelope(t *testing.T) {
	env := &enginev1.SignedExecutionPayloadEnvelope{
		Message: &enginev1.ExecutionPayloadEnvelope{
			Payload:            &enginev1.ExecutionPayloadElectra{},
			BeaconBlockRoot:    make([]byte, 32),
			BlobKzgCommitments: [][]byte{},
			StateRoot:          make([]byte, 32),
		},
		Signature: make([]byte, 96),
	}
	t.Run("Happy case", func(t *testing.T) {
		st, _ := util.DeterministicGenesisStateEpbs(t, 1)
		s := &Server{
			P2P:                      p2ptest.NewTestP2P(t),
			ExecutionPayloadReceiver: &mockChain.ChainService{State: st},
		}
		_, err := s.SubmitSignedExecutionPayloadEnvelope(context.Background(), env)
		require.NoError(t, err)
	})

	t.Run("Receive failed", func(t *testing.T) {
		s := &Server{
			P2P:                      p2ptest.NewTestP2P(t),
			ExecutionPayloadReceiver: &mockChain.ChainService{ReceiveBlockMockErr: errors.New("receive failed")},
		}
		_, err := s.SubmitSignedExecutionPayloadEnvelope(context.Background(), env)
		require.ErrorContains(t, "failed to receive execution payload envelope: receive failed", err)
	})
}

func TestProposer_ComputePostPayloadStateRoot(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()

	proposerServer := &Server{
		ChainStartFetcher: &mockExecution.Chain{},
		Eth1InfoFetcher:   &mockExecution.Chain{},
		Eth1BlockFetcher:  &mockExecution.Chain{},
		StateGen:          stategen.New(db, doublylinkedtree.New()),
	}

	bh := [32]byte{'h'}
	root := [32]byte{'r'}
	p := &enginev1.ExecutionPayloadEnvelope{
		Payload: &enginev1.ExecutionPayloadElectra{},
	}
	p.Payload.BlockHash = bh[:]
	p.BeaconBlockRoot = root[:]
	e, err := blocks.WrappedROExecutionPayloadEnvelope(p)
	require.NoError(t, err)
	validators := make([]*ethpb.Validator, 0)
	stpb := &ethpb.BeaconStateEPBS{Slot: 3, Validators: validators}
	st, err := state_native.InitializeFromProtoUnsafeEpbs(stpb)
	require.NoError(t, err)

	require.NoError(t, db.SaveState(ctx, st, e.BeaconBlockRoot()))
	_, err = proposerServer.computePostPayloadStateRoot(ctx, e)
	require.NoError(t, err)
	require.DeepEqual(t, e.StateRoot(), root[:])
}
