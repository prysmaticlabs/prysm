package checkpoint

import (
	"context"
	"fmt"
	"github.com/prysmaticlabs/prysm/api/client/openapi"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/proto/sniff"
	"github.com/prysmaticlabs/prysm/time/slots"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"

	//"os"

	"github.com/pkg/errors"
	//"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	//"github.com/prysmaticlabs/prysm/config/params"
	//"github.com/prysmaticlabs/prysm/proto/sniff"
	//"github.com/prysmaticlabs/prysm/time/slots"
)

type APIInitializer struct {
	c *openapi.Client
}

func NewAPIInitializer(beaconNodeHost string) (*APIInitializer, error) {
	c, err := openapi.NewClient(beaconNodeHost)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("unable to parse beacon node url or hostname - %s", beaconNodeHost))
	}
	return &APIInitializer{c: c}, nil
}

func (dl *APIInitializer) StateReader(ctx context.Context) (io.Reader, error) {
	return nil, nil
}

func (dl *APIInitializer) BlockReader(ctx context.Context) (io.Reader, error) {
	return nil, nil
}

// - step 1: get the weak subjectivity epoch
// - step 2: get the state at the epoch
// - step 3: get the block corresponding to that state

func (dl *APIInitializer) Initialize(ctx context.Context, d db.Database) error {
	// the weak subjectivity period computation requires the active validator count and balance
	// the fastest way to get this information is to request the head state from the remote api
	headReader, err := dl.c.GetStateById(openapi.StateIdHead)
	if err != nil {
		return errors.Wrap(err, "failed api request for head state")
	}
	headBytes, err := io.ReadAll(headReader)
	if err != nil {
		return errors.Wrap(err, "failed to read response body for get head state api call")
	}
	log.Printf("state response byte len=%d", len(headBytes))
	headState, err := sniff.BeaconState(headBytes)
	if err != nil {
		return errors.Wrap(err, "error unmarshaling state to correct version")
	}

	// by confirming that the fork version found in the head state is included in the fork version schedule of
	// the currently running beacon node, we ensure that the retrieved state is from the same network
	cf, err := sniff.ConfigForkForState(headBytes)
	if err != nil {
		return errors.Wrap(err, "error detecting chain config for beacon state")
	}
	_, ok := params.BeaconConfig().ForkVersionSchedule[cf.Version]
	if !ok {
		return fmt.Errorf("config mismatch, beacon node configured to connect to %s, detected state is for %s", params.BeaconConfig().ConfigName, cf.ConfigName.String())
	}

	log.Printf("detected supported config for state & block version detection, name=%s, fork=%s", cf.ConfigName.String(), cf.Fork)
	epoch, err := helpers.LatestWeakSubjectivityEpoch(ctx, headState)
	if err != nil {
		return errors.Wrap(err, "error computing the weak subjectivity epoch from head state")
	}

	// use first slot of the epoch for the block slot
	bSlot, err := slots.EpochStart(epoch)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Error computing first slot of epoch=%d", epoch))
	}
	// assigning this variable to make it extra obvious that the state slot is different
	sSlot := bSlot + 1
	// using the state at (slot % 32 = 1) instead of the epoch boundary ensures the
	// next block applied to the state will have the block at the weak subjectivity checkpoint
	// as its parent, satisfying prysm's sync code current verification that the parent block is present in the db
	log.Printf("weak subjectivity epoch computed as %d. downloading block@slot=%d, state@slot=%d", epoch, bSlot, sSlot)

	stateReader, err := dl.c.GetStateBySlot(uint64(sSlot))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to retrieve state bytes for slot %d from api", sSlot))
	}
	serState, err := ioutil.ReadAll(stateReader)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to read state bytes for slot %d from api", bSlot))
	}

	blockReader, err := dl.c.GetBlockBySlot(uint64(bSlot))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to retrieve block bytes for slot %d from api", bSlot))
	}

	serBlock, err := ioutil.ReadAll(blockReader)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to read block bytes for slot %d from api", bSlot))
	}

	return d.SaveOrigin(ctx, serState, serBlock)
}
