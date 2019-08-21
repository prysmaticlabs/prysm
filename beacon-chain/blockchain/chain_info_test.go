package blockchain

import (
	"testing"

	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
)

// Ensure ChainService implements chain info interface.
var _ = ChainInfoRetriever(&ChainService{})

func TestFinalizedCheckpt(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	c := setupBeaconChain(t, db, nil)

	t.Log(c.FinalizedCheckpt())
}
