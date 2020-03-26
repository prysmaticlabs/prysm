package beaconclient

import (
	"bytes"
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/testutil"
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
	validatorCache, err := cache.NewValidatorsCache(0, nil)
	if err != nil {
		t.Fatalf("could not create new cache: %v", err)
	}
	bs := Service{
		beaconClient:   client,
		validatorCache: validatorCache,
	}
	wanted := &ethpb.Validator{PublicKey: []byte{1, 2, 3}}
	client.EXPECT().GetValidator(
		gomock.Any(),
		gomock.Any(),
	).Return(wanted, nil)

	// We request public key of validator id 0.
	res, err := bs.RequestValidator(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(res, wanted.PublicKey) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
	testutil.AssertLogsContain(t, hook, "Retrieved validator id: 0")

	// We expect public key of validator id 0 to be in cache.
	res, err = bs.RequestValidator(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(res, wanted.PublicKey) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
	testutil.AssertLogsContain(t, hook, "Retrieved validator id: 0 from cache")
}
