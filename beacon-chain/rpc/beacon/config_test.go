package beacon

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestServer_GetBeaconConfig(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	bs := &Server{}
	res, err := bs.GetBeaconConfig(ctx, &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	conf := params.BeaconConfig()
	numFields := reflect.TypeOf(conf).Elem().NumField()

	// Check if the result has the same number of items as our config struct.
	if len(res.Config) != numFields {
		t.Errorf("Expected %d items in config result, got %d", numFields, len(res.Config))
	}
	want := fmt.Sprintf("%d", conf.Eth1FollowDistance)

	// Check that an element is properly populated from the config.
	if res.Config["Eth1FollowDistance"] != want {
		t.Errorf("Wanted %s for eth1 follow distance, received %s", want, res.Config["Eth1FollowDistance"])
	}
}

func TestServer_GetGenesisValidatorsRoot(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	mockedGenesisRoot := [32]byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
	}

	ctx := context.Background()
	bs := &Server{
		GenesisFetcher: &mock.ChainService{
			ValidatorsRoot: mockedGenesisRoot,
		},
	}

	res, err := bs.GetGenesisValidatorsRoot(ctx, &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(mockedGenesisRoot[:], res.Root) {
		t.Errorf("Incorrect genesis validators root: expected %#v, received %#v", mockedGenesisRoot, res.Root)
	}
}
