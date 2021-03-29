package beaconclient

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
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
	require.NoError(t, err, "Could not create new cache")
	bs := Service{
		cfg:            &Config{BeaconClient: client},
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
	res, err := bs.FindOrGetPublicKeys(context.Background(), []types.ValidatorIndex{0, 1})
	require.NoError(t, err)
	for i, v := range wanted.ValidatorList {
		assert.DeepEqual(t, wanted.ValidatorList[i].Validator.PublicKey, res[v.Index])
	}

	require.LogsContain(t, hook, "Retrieved validators id public key map:")
	require.LogsDoNotContain(t, hook, "Retrieved validators public keys from cache:")
	// We expect public key of validator id 0 to be in cache.
	res, err = bs.FindOrGetPublicKeys(context.Background(), []types.ValidatorIndex{0, 3})
	require.NoError(t, err)

	for i, v := range wanted2.ValidatorList {
		assert.DeepEqual(t, wanted2.ValidatorList[i].Validator.PublicKey, res[v.Index])
	}
	require.LogsContain(t, hook, "Retrieved validators public keys from cache: map[0:[1 2 3]]")
}
