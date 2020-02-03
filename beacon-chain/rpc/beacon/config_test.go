package beacon

import (
	"context"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
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
	t.Log(res)
}