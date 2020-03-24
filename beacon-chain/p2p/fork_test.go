package p2p

import (
	"bytes"
	"math/rand"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestDiscv5_AddRetrieveForkEntryENR(t *testing.T) {
	c := params.BeaconConfig()
	c.ForkVersionSchedule = map[uint64][]byte{
		0: params.BeaconConfig().GenesisForkVersion,
		1: {0, 0, 0, 1},
		2: {0, 0, 0, 2},
		3: {0, 0, 0, 3},
	}
	nextForkEpoch := uint64(2)
	nextForkVersion := []byte{0, 0, 0, 2}
	c.NextForkEpoch = nextForkEpoch
	c.NextForkVersion = nextForkVersion
	params.OverrideBeaconConfig(c)

	// We simulate being in epoch 1.
	secondsPerEpoch := params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().SecondsPerSlot
	durationPerEpoch := time.Duration(secondsPerEpoch) * time.Second
	genesisTime := time.Now().Add(-durationPerEpoch)

	// In epoch 1 of current time, the fork version should be
	// {0, 0, 0, 1} according to the configuration override above.
	temp := testutil.TempDir()
	randNum := rand.Int()
	tempPath := path.Join(temp, strconv.Itoa(randNum))
	if err := os.Mkdir(tempPath, 0700); err != nil {
		t.Fatal(err)
	}
	pkey, err := privKey(&Config{Encoding: "ssz", DataDir: tempPath})
	if err != nil {
		t.Fatalf("Could not get private key: %v", err)
	}
	db, err := enode.OpenDB("")
	if err != nil {
		t.Fatal(err)
	}
	localNode := enode.NewLocalNode(db, pkey)

	genesisValidatorsRoot := make([]byte, 32)
	localNode, err = addForkEntry(localNode, genesisTime, genesisValidatorsRoot)
	if err != nil {
		t.Fatal(err)
	}

	want, err := helpers.ComputeForkDigest([]byte{0, 0, 0, 1}, genesisValidatorsRoot)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := retrieveForkEntry(localNode.Node().Record())
	if err != nil {
		t.Fatal(err)
	}
	if resp.CurrentForkDigest != want {
		t.Errorf("Wanted fork digest: %v, received %v", want, resp.CurrentForkDigest)
	}
	if !bytes.Equal(resp.NextForkVersion[:], nextForkVersion) {
		t.Errorf("Wanted next fork version: %v, received %v", nextForkVersion, resp.NextForkVersion)
	}
	if resp.NextForkEpoch != nextForkEpoch {
		t.Errorf("Wanted next for epoch: %d, received: %d", nextForkEpoch, resp.NextForkEpoch)
	}
}
