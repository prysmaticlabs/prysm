package beacon

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestServer_SubmitProposerSlashing_DontBroadcast(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{DisableBroadcastSlashings: true})
	defer resetCfg()
	ctx := context.Background()
	st, privs := testutil.DeterministicGenesisState(t, 64)
	slashedVal, err := st.ValidatorAtIndex(5)
	if err != nil {
		t.Fatal(err)
	}
	// We mark the validator at index 5 as already slashed.
	slashedVal.Slashed = true
	if err := st.UpdateValidatorAtIndex(5, slashedVal); err != nil {
		t.Fatal(err)
	}

	mb := &mockp2p.MockBroadcaster{}
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: st,
		},
		SlashingsPool: slashings.NewPool(),
		Broadcaster:   mb,
	}

	// We want a proposer slashing for validator with index 2 to
	// be included in the pool.
	wanted := &ethpb.SubmitSlashingResponse{
		SlashedIndices: []uint64{2},
	}
	slashing, err := testutil.GenerateProposerSlashingForValidator(st, privs[2], uint64(2))
	if err != nil {
		t.Fatal(err)
	}

	res, err := bs.SubmitProposerSlashing(ctx, slashing)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}

	if mb.BroadcastCalled {
		t.Errorf("Expected broadcast not to be called by default")
	}

	slashing, err = testutil.GenerateProposerSlashingForValidator(st, privs[5], uint64(5))
	if err != nil {
		t.Fatal(err)
	}

	// We do not want a proposer slashing for an already slashed validator
	// (the validator at index 5) to be included in the pool.
	if _, err := bs.SubmitProposerSlashing(ctx, slashing); err == nil {
		t.Error("Expected including a proposer slashing for an already slashed validator to fail")
	}
}

func TestServer_SubmitProposerSlashing(t *testing.T) {
	ctx := context.Background()

	st, privs := testutil.DeterministicGenesisState(t, 64)
	slashedVal, err := st.ValidatorAtIndex(5)
	if err != nil {
		t.Fatal(err)
	}
	// We mark the validator at index 5 as already slashed.
	slashedVal.Slashed = true
	if err := st.UpdateValidatorAtIndex(5, slashedVal); err != nil {
		t.Fatal(err)
	}

	mb := &mockp2p.MockBroadcaster{}
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: st,
		},
		SlashingsPool: slashings.NewPool(),
		Broadcaster:   mb,
	}

	// We want a proposer slashing for validator with index 2 to
	// be included in the pool.
	slashing, err := testutil.GenerateProposerSlashingForValidator(st, privs[2], uint64(2))
	if err != nil {
		t.Fatal(err)
	}

	_, err = bs.SubmitProposerSlashing(ctx, slashing)
	if err != nil {
		t.Fatal(err)
	}

	if !mb.BroadcastCalled {
		t.Errorf("Expected broadcast to be called")
	}
}

func TestServer_SubmitAttesterSlashing_DontBroadcast(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{DisableBroadcastSlashings: true})
	defer resetCfg()
	ctx := context.Background()
	// We mark the validators at index 5, 6 as already slashed.
	st, privs := testutil.DeterministicGenesisState(t, 64)
	slashedVal, err := st.ValidatorAtIndex(5)
	if err != nil {
		t.Fatal(err)
	}

	// We mark the validator at index 5 as already slashed.
	slashedVal.Slashed = true
	if err := st.UpdateValidatorAtIndex(5, slashedVal); err != nil {
		t.Fatal(err)
	}

	mb := &mockp2p.MockBroadcaster{}
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: st,
		},
		SlashingsPool: slashings.NewPool(),
		Broadcaster:   mb,
	}

	slashing, err := testutil.GenerateAttesterSlashingForValidator(st, privs[2], uint64(2))
	if err != nil {
		t.Fatal(err)
	}

	// We want the intersection of the slashing attesting indices
	// to be slashed, so we expect validators 2 and 3 to be in the response
	// slashed indices.
	wanted := &ethpb.SubmitSlashingResponse{
		SlashedIndices: []uint64{2},
	}
	res, err := bs.SubmitAttesterSlashing(ctx, slashing)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
	if mb.BroadcastCalled {
		t.Errorf("Expected broadcast not to be called by default")
	}

	slashing, err = testutil.GenerateAttesterSlashingForValidator(st, privs[5], uint64(5))
	if err != nil {
		t.Fatal(err)
	}
	// If any of the attesting indices in the slashing object have already
	// been slashed, we should fail to insert properly into the attester slashing pool.
	if _, err := bs.SubmitAttesterSlashing(ctx, slashing); err == nil {
		t.Error("Expected including a attester slashing for an already slashed validator to fail")
	}
}

func TestServer_SubmitAttesterSlashing(t *testing.T) {
	ctx := context.Background()
	// We mark the validators at index 5, 6 as already slashed.
	st, privs := testutil.DeterministicGenesisState(t, 64)
	slashedVal, err := st.ValidatorAtIndex(5)
	if err != nil {
		t.Fatal(err)
	}

	// We mark the validator at index 5 as already slashed.
	slashedVal.Slashed = true
	if err := st.UpdateValidatorAtIndex(5, slashedVal); err != nil {
		t.Fatal(err)
	}

	mb := &mockp2p.MockBroadcaster{}
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: st,
		},
		SlashingsPool: slashings.NewPool(),
		Broadcaster:   mb,
	}

	slashing, err := testutil.GenerateAttesterSlashingForValidator(st, privs[2], uint64(2))
	if err != nil {
		t.Fatal(err)
	}

	// We want the intersection of the slashing attesting indices
	// to be slashed, so we expect validators 2 and 3 to be in the response
	// slashed indices.
	_, err = bs.SubmitAttesterSlashing(ctx, slashing)
	if err != nil {
		t.Fatal(err)
	}
	if !mb.BroadcastCalled {
		t.Errorf("Expected broadcast to be called when flag is set")
	}
}
