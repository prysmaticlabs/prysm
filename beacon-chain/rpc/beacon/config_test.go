package beacon

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
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
