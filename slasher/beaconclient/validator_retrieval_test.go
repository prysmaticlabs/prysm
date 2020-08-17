package beaconclient

import (
	"bytes"
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/slasher/cache"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestService_RequestValidator(t *testing.T) {
	hook := logTest.NewGlobal()
	logrus.SetLevel(logrus.TraceLevel)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconChainClient(ctrl)
	validatorCache, err := cache.NewPublicKeyCache(0, nil)
	if err != nil {
		t.Fatalf("could not create new cache: %v", err)
	}
	bs := Service{
		beaconClient:   client,
		publicKeyCache: validatorCache,
	}
	wanted := &ethpb.Validators{
		ValidatorList: []*ethpb.Validators_ValidatorContainer{
			{
				Index: 0, Validator: &ethpb.Validator{PublicKey: []byte{1, 2, 3}},
			},
			{
				Index: 1, Validator: &ethpb.Validator{PublicKey: []byte{2, 4, 5}},
			},
		},
	}
	wanted2 := &ethpb.Validators{
		ValidatorList: []*ethpb.Validators_ValidatorContainer{
			{
				Index: 3, Validator: &ethpb.Validator{PublicKey: []byte{3, 4, 5}},
			},
		},
	}
	client.EXPECT().ListValidators(
		gomock.Any(),
		gomock.Any(),
	).Return(wanted, nil)

	client.EXPECT().ListValidators(
		gomock.Any(),
		gomock.Any(),
	).Return(wanted2, nil)

	// We request public key of validator id 0,1.
	res, err := bs.FindOrGetPublicKeys(context.Background(), []uint64{0, 1})
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range wanted.ValidatorList {
		if !bytes.Equal(res[v.Index], wanted.ValidatorList[i].Validator.PublicKey) {
			t.Errorf("Wanted %v, received %v", wanted, res)
		}
	}

	require.LogsContain(t, hook, "Retrieved validators id public key map:")
	require.LogsDoNotContain(t, hook, "Retrieved validators public keys from cache:")
	// We expect public key of validator id 0 to be in cache.
	res, err = bs.FindOrGetPublicKeys(context.Background(), []uint64{0, 3})
	if err != nil {
		t.Fatal(err)
	}

	for i, v := range wanted2.ValidatorList {
		if !bytes.Equal(res[v.Index], wanted2.ValidatorList[i].Validator.PublicKey) {
			t.Errorf("Wanted %v, received %v", wanted2, res)
		}
	}
	require.LogsContain(t, hook, "Retrieved validators public keys from cache: map[0:[1 2 3]]")
}
