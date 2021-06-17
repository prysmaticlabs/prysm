package client

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestSubmitSyncCommitteeMessage_ValidatorDutiesRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, validatorKey, finish := setup(t)
	validator.duties = &eth.DutiesResponse{Duties: []*eth.DutiesResponse_Duty{}}
	defer finish()

	m.validatorClient.EXPECT().GetSyncMessageBlockRoot(
		gomock.Any(), // ctx
		&emptypb.Empty{},
	).Return(&eth.SyncMessageBlockRootResponse{
		Root: bytesutil.PadTo([]byte{}, 32),
	}, nil)

	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	validator.SubmitSyncCommitteeMessage(context.Background(), 1, pubKey)
	require.LogsContain(t, hook, "Could not fetch validator assignment")
}

func TestSubmitSyncCommitteeMessage_BadDomainData(t *testing.T) {
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	hook := logTest.NewGlobal()
	validatorIndex := types.ValidatorIndex(7)
	committee := []types.ValidatorIndex{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.duties = &eth.DutiesResponse{Duties: []*eth.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey().Marshal(),
			Committee:      committee,
			ValidatorIndex: validatorIndex,
		},
	}}

	r := []byte{'a'}
	m.validatorClient.EXPECT().GetSyncMessageBlockRoot(
		gomock.Any(), // ctx
		&emptypb.Empty{},
	).Return(&eth.SyncMessageBlockRootResponse{
		Root: bytesutil.PadTo(r, 32),
	}, nil)

	m.validatorClient.EXPECT().
		DomainData(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("uh oh"))

	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	validator.SubmitSyncCommitteeMessage(context.Background(), 1, pubKey)
	require.LogsContain(t, hook, "Could not get sync committee domain data")
}

func TestSubmitSyncCommitteeMessage_CouldNotSubmit(t *testing.T) {
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	hook := logTest.NewGlobal()
	validatorIndex := types.ValidatorIndex(7)
	committee := []types.ValidatorIndex{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.duties = &eth.DutiesResponse{Duties: []*eth.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey().Marshal(),
			Committee:      committee,
			ValidatorIndex: validatorIndex,
		},
	}}

	r := []byte{'a'}
	m.validatorClient.EXPECT().GetSyncMessageBlockRoot(
		gomock.Any(), // ctx
		&emptypb.Empty{},
	).Return(&eth.SyncMessageBlockRootResponse{
		Root: bytesutil.PadTo(r, 32),
	}, nil)

	m.validatorClient.EXPECT().
		DomainData(gomock.Any(), // ctx
			gomock.Any()). // epoch
		Return(&eth.DomainResponse{
			SignatureDomain: make([]byte, 32),
		}, nil)

	m.validatorClient.EXPECT().SubmitSyncMessage(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&eth.SyncCommitteeMessage{}),
	).Return(&emptypb.Empty{}, errors.New("uh oh") /* error */)

	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	validator.SubmitSyncCommitteeMessage(context.Background(), 1, pubKey)

	require.LogsContain(t, hook, "Could not submit sync committee message")
}

func TestSubmitSyncCommitteeMessage_OK(t *testing.T) {
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	hook := logTest.NewGlobal()
	validatorIndex := types.ValidatorIndex(7)
	committee := []types.ValidatorIndex{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.duties = &eth.DutiesResponse{Duties: []*eth.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey().Marshal(),
			Committee:      committee,
			ValidatorIndex: validatorIndex,
		},
	}}

	r := []byte{'a'}
	m.validatorClient.EXPECT().GetSyncMessageBlockRoot(
		gomock.Any(), // ctx
		&emptypb.Empty{},
	).Return(&eth.SyncMessageBlockRootResponse{
		Root: bytesutil.PadTo(r, 32),
	}, nil)

	m.validatorClient.EXPECT().
		DomainData(gomock.Any(), // ctx
			gomock.Any()). // epoch
		Return(&eth.DomainResponse{
			SignatureDomain: make([]byte, 32),
		}, nil)

	var generatedMsg *eth.SyncCommitteeMessage
	m.validatorClient.EXPECT().SubmitSyncMessage(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&eth.SyncCommitteeMessage{}),
	).Do(func(_ context.Context, msg *eth.SyncCommitteeMessage) {
		generatedMsg = msg
	}).Return(&emptypb.Empty{}, nil /* error */)

	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	validator.SubmitSyncCommitteeMessage(context.Background(), 1, pubKey)

	require.LogsDoNotContain(t, hook, "Could not")
	require.Equal(t, types.Slot(1), generatedMsg.Slot)
	require.Equal(t, validatorIndex, generatedMsg.ValidatorIndex)
	require.DeepEqual(t, bytesutil.PadTo(r, 32), generatedMsg.BlockRoot)
}
