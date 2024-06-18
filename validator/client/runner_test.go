package client

import (
	"context"
	"math/bits"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api/client/beacon/mock"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/config/proposer"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/validator/client/iface"
	"github.com/prysmaticlabs/prysm/v5/validator/client/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"
)

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestCancelledContext_CleansUpValidator(t *testing.T) {
	v := &testutil.FakeValidator{
		Km: &mockKeymanager{},
	}
	run(cancelledContext(), v)
	assert.Equal(t, true, v.DoneCalled, "Expected Done() to be called")
}

func TestCancelledContext_WaitsForChainStart(t *testing.T) {
	v := &testutil.FakeValidator{
		Km: &mockKeymanager{},
	}
	run(cancelledContext(), v)
	assert.Equal(t, 1, v.WaitForChainStartCalled, "Expected WaitForChainStart() to be called")
}

func TestRetry_On_ConnectionError(t *testing.T) {
	retry := 10
	v := &testutil.FakeValidator{
		Km:               &mockKeymanager{},
		RetryTillSuccess: retry,
	}
	backOffPeriod = 10 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	go run(ctx, v)
	// each step will fail (retry times)=10 this sleep times will wait more then
	// the time it takes for all steps to succeed before main loop.
	time.Sleep(time.Duration(retry*6) * backOffPeriod)
	cancel()
	// every call will fail retry=10 times so first one will be called 4 * retry=10.
	assert.Equal(t, retry*3, v.WaitForChainStartCalled, "Expected WaitForChainStart() to be called")
	assert.Equal(t, retry*2, v.WaitForSyncCalled, "Expected WaitForSync() to be called")
	assert.Equal(t, retry, v.WaitForActivationCalled, "Expected WaitForActivation() to be called")
	assert.Equal(t, retry, v.CanonicalHeadSlotCalled, "Expected CanonicalHeadSlotCalled() to be called")
}

func TestCancelledContext_WaitsForActivation(t *testing.T) {
	v := &testutil.FakeValidator{
		Km: &mockKeymanager{},
	}
	run(cancelledContext(), v)
	assert.Equal(t, 1, v.WaitForActivationCalled, "Expected WaitForActivation() to be called")
}

func TestUpdateDuties_NextSlot(t *testing.T) {
	v := &testutil.FakeValidator{Km: &mockKeymanager{}}
	ctx, cancel := context.WithCancel(context.Background())

	slot := primitives.Slot(55)
	ticker := make(chan primitives.Slot)
	v.NextSlotRet = ticker
	go func() {
		ticker <- slot

		cancel()
	}()

	run(ctx, v)

	require.Equal(t, true, v.UpdateDutiesCalled, "Expected UpdateAssignments(%d) to be called", slot)
	assert.Equal(t, uint64(slot), v.UpdateDutiesArg1, "UpdateAssignments was called with wrong argument")
}

func TestUpdateDuties_HandlesError(t *testing.T) {
	hook := logTest.NewGlobal()
	v := &testutil.FakeValidator{Km: &mockKeymanager{}}
	ctx, cancel := context.WithCancel(context.Background())

	slot := primitives.Slot(55)
	ticker := make(chan primitives.Slot)
	v.NextSlotRet = ticker
	go func() {
		ticker <- slot

		cancel()
	}()
	v.UpdateDutiesRet = errors.New("bad")

	run(ctx, v)

	require.LogsContain(t, hook, "Failed to update assignments")
}

func TestRoleAt_NextSlot(t *testing.T) {
	v := &testutil.FakeValidator{Km: &mockKeymanager{}}
	ctx, cancel := context.WithCancel(context.Background())

	slot := primitives.Slot(55)
	ticker := make(chan primitives.Slot)
	v.NextSlotRet = ticker
	go func() {
		ticker <- slot

		cancel()
	}()

	run(ctx, v)

	require.Equal(t, true, v.RoleAtCalled, "Expected RoleAt(%d) to be called", slot)
	assert.Equal(t, uint64(slot), v.RoleAtArg1, "RoleAt called with the wrong arg")
}

func TestAttests_NextSlot(t *testing.T) {
	attSubmitted := make(chan interface{})
	v := &testutil.FakeValidator{Km: &mockKeymanager{}, AttSubmitted: attSubmitted}
	ctx, cancel := context.WithCancel(context.Background())

	slot := primitives.Slot(55)
	ticker := make(chan primitives.Slot)
	v.NextSlotRet = ticker
	v.RolesAtRet = []iface.ValidatorRole{iface.RoleAttester}
	go func() {
		ticker <- slot

		cancel()
	}()
	run(ctx, v)
	<-attSubmitted
	require.Equal(t, true, v.AttestToBlockHeadCalled, "SubmitAttestation(%d) was not called", slot)
	assert.Equal(t, uint64(slot), v.AttestToBlockHeadArg1, "SubmitAttestation was called with wrong arg")
}

func TestProposes_NextSlot(t *testing.T) {
	blockProposed := make(chan interface{})
	v := &testutil.FakeValidator{Km: &mockKeymanager{}, BlockProposed: blockProposed}
	ctx, cancel := context.WithCancel(context.Background())

	slot := primitives.Slot(55)
	ticker := make(chan primitives.Slot)
	v.NextSlotRet = ticker
	v.RolesAtRet = []iface.ValidatorRole{iface.RoleProposer}
	go func() {
		ticker <- slot

		cancel()
	}()
	run(ctx, v)
	<-blockProposed

	require.Equal(t, true, v.ProposeBlockCalled, "ProposeBlock(%d) was not called", slot)
	assert.Equal(t, uint64(slot), v.ProposeBlockArg1, "ProposeBlock was called with wrong arg")
}

func TestBothProposesAndAttests_NextSlot(t *testing.T) {
	blockProposed := make(chan interface{})
	attSubmitted := make(chan interface{})
	v := &testutil.FakeValidator{Km: &mockKeymanager{}, BlockProposed: blockProposed, AttSubmitted: attSubmitted}
	ctx, cancel := context.WithCancel(context.Background())

	slot := primitives.Slot(55)
	ticker := make(chan primitives.Slot)
	v.NextSlotRet = ticker
	v.RolesAtRet = []iface.ValidatorRole{iface.RoleAttester, iface.RoleProposer}
	go func() {
		ticker <- slot

		cancel()
	}()
	run(ctx, v)
	<-blockProposed
	<-attSubmitted
	require.Equal(t, true, v.AttestToBlockHeadCalled, "SubmitAttestation(%d) was not called", slot)
	assert.Equal(t, uint64(slot), v.AttestToBlockHeadArg1, "SubmitAttestation was called with wrong arg")
	require.Equal(t, true, v.ProposeBlockCalled, "ProposeBlock(%d) was not called", slot)
	assert.Equal(t, uint64(slot), v.ProposeBlockArg1, "ProposeBlock was called with wrong arg")
}

func TestKeyReload_ActiveKey(t *testing.T) {
	ctx := context.Background()
	km := &mockKeymanager{}
	v := &testutil.FakeValidator{Km: km}
	ac := make(chan [][fieldparams.BLSPubkeyLength]byte)
	current := [][fieldparams.BLSPubkeyLength]byte{testutil.ActiveKey}
	onAccountsChanged(ctx, v, current, ac)
	assert.Equal(t, true, v.HandleKeyReloadCalled)
	// HandleKeyReloadCalled in the FakeValidator returns true if one of the keys is equal to the
	// ActiveKey. WaitForActivation is only called if none of the keys are active, so it shouldn't be called at all.
	assert.Equal(t, 0, v.WaitForActivationCalled)
}

func TestKeyReload_NoActiveKey(t *testing.T) {
	na := notActive(t)
	ctx := context.Background()
	km := &mockKeymanager{}
	v := &testutil.FakeValidator{Km: km}
	ac := make(chan [][fieldparams.BLSPubkeyLength]byte)
	current := [][fieldparams.BLSPubkeyLength]byte{na}
	onAccountsChanged(ctx, v, current, ac)
	assert.Equal(t, true, v.HandleKeyReloadCalled)
	// HandleKeyReloadCalled in the FakeValidator returns true if one of the keys is equal to the
	// ActiveKey. Since we are using a key we know is not active, it should return false, which
	// should cause the account change handler to call WaitForActivationCalled.
	assert.Equal(t, 1, v.WaitForActivationCalled)
}

func notActive(t *testing.T) [fieldparams.BLSPubkeyLength]byte {
	var r [fieldparams.BLSPubkeyLength]byte
	copy(r[:], testutil.ActiveKey[:])
	for i := 0; i < len(r); i++ {
		r[i] = bits.Reverse8(r[i])
	}
	require.DeepNotEqual(t, r, testutil.ActiveKey)
	return r
}

func TestUpdateProposerSettingsAt_EpochStart(t *testing.T) {
	v := &testutil.FakeValidator{Km: &mockKeymanager{}}
	err := v.SetProposerSettings(context.Background(), &proposer.Settings{
		DefaultConfig: &proposer.Option{
			FeeRecipientConfig: &proposer.FeeRecipientConfig{
				FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9"),
			},
		},
	})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	hook := logTest.NewGlobal()
	slot := params.BeaconConfig().SlotsPerEpoch
	ticker := make(chan primitives.Slot)
	v.NextSlotRet = ticker
	go func() {
		ticker <- slot

		cancel()
	}()

	run(ctx, v)
	assert.LogsContain(t, hook, "updated proposer settings")
}

func TestUpdateProposerSettingsAt_EpochEndOk(t *testing.T) {
	v := &testutil.FakeValidator{
		Km:                  &mockKeymanager{},
		ProposerSettingWait: time.Duration(params.BeaconConfig().SecondsPerSlot-1) * time.Second,
	}
	err := v.SetProposerSettings(context.Background(), &proposer.Settings{
		DefaultConfig: &proposer.Option{
			FeeRecipientConfig: &proposer.FeeRecipientConfig{
				FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9"),
			},
		},
	})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	hook := logTest.NewGlobal()
	slot := params.BeaconConfig().SlotsPerEpoch - 1 //have it set close to the end of epoch
	ticker := make(chan primitives.Slot)
	v.NextSlotRet = ticker
	go func() {
		ticker <- slot
		cancel()
	}()

	run(ctx, v)
	// can't test "Failed to update proposer settings" because of log.fatal
	assert.LogsContain(t, hook, "Mock updated proposer settings")
}

func TestUpdateProposerSettings_ContinuesAfterValidatorRegistrationFails(t *testing.T) {
	errSomeOtherError := errors.New("some internal error")
	v := &testutil.FakeValidator{
		ProposerSettingsErr: errors.Wrap(ErrBuilderValidatorRegistration, errSomeOtherError.Error()),
		Km:                  &mockKeymanager{},
	}
	err := v.SetProposerSettings(context.Background(), &proposer.Settings{
		DefaultConfig: &proposer.Option{
			FeeRecipientConfig: &proposer.FeeRecipientConfig{
				FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9"),
			},
		},
	})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	hook := logTest.NewGlobal()
	slot := params.BeaconConfig().SlotsPerEpoch
	ticker := make(chan primitives.Slot)
	v.NextSlotRet = ticker
	go func() {
		ticker <- slot

		cancel()
	}()
	run(ctx, v)
	assert.LogsContain(t, hook, ErrBuilderValidatorRegistration.Error())
}

func TestLastSecondOfSlot_HealthyAgain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTracker := mock.NewMockHealthTracker(ctrl)
	mockTracker.EXPECT().IsHealthy().Times(2).Return(true).Return(false)

	ticker := make(chan primitives.Slot)
	v := &testutil.FakeValidator{Km: &mockKeymanager{}, LastSecondOfSlotRet: ticker, Tracker: mockTracker}
	go func() {
		ticker <- 0
		ticker <- 1
		cancel()
	}()
	run(ctx, v)
	assert.Equal(t, true, v.UpdateDutiesCalled)
}

func TestClosure(t *testing.T) {
	var b bool
	go func() {
		b = true
	}()
	time.Sleep(1000)
	go func() {
		if b == true {
			log.Info("true")
		}
	}()
	time.Sleep(1000)
}
