package db

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/database"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "blockchain")

// BeaconDB manages the data layer of the beacon chain implementation.
// The exposed methods do not have an opinion of the underlying data engine,
// but instead reflect the beacon chain logic.
// For example, instead of defining get, put, remove
// This defines methods such as getBlock, saveBlocksAndAttestations, etc.
type BeaconDB struct {
	db    *database.DB
	state *beaconState
}

// Config exposes relevant config options for starting a database.
type Config struct {
	Path        string
	Name        string
	InMemory    bool
	GenesisJSON string
}

type beaconState struct {
	// aState captures the beacon state at block processing level,
	// it focuses on verifying aggregated signatures and pending attestations.
	aState *types.ActiveState
	// cState captures the beacon state at cycle transition level,
	// it focuses on changes to the validator set, processing cross links and
	// setting up FFG checkpoints.
	cState *types.CrystallizedState
}

func (db *BeaconDB) has(key []byte) (bool, error) {
	return db.db.DB().Has(key)
}

func (db *BeaconDB) get(key []byte) ([]byte, error) {
	return db.db.DB().Get(key)
}

func (db *BeaconDB) put(key []byte, val []byte) error {
	return db.db.DB().Put(key, val)
}

func (db *BeaconDB) delete(key []byte) error {
	return db.db.DB().Delete(key)
}

// Close closes the underlying leveldb database.
func (db *BeaconDB) Close() {
	db.db.Close()
}

// NewDB initializes a new DB. If the genesis block and states do not exist, this method creates it.
func NewDB(cfg Config) (*BeaconDB, error) {
	config := &database.DBConfig{DataDir: cfg.Path, Name: cfg.Name, InMemory: cfg.InMemory}
	db, err := database.NewDB(config)
	if err != nil {
		return nil, fmt.Errorf("failed to start db: %v", err)
	}

	beaconDB := &BeaconDB{
		db:    db,
		state: &beaconState{},
	}
	hasCrystallized, err := beaconDB.has(crystallizedStateLookupKey)
	if err != nil {
		return nil, err
	}
	hasGenesis, err := beaconDB.HasCanonicalBlockForSlot(0)
	if err != nil {
		return nil, err
	}

	var genesisValidators []*pb.ValidatorRecord

	if cfg.GenesisJSON != "" {
		log.Infof("Initializing Crystallized State from %s", cfg.GenesisJSON)
		genesisValidators, err = utils.InitialValidatorsFromJSON(cfg.GenesisJSON)
		if err != nil {
			return nil, err
		}
	}

	active := types.NewGenesisActiveState()
	crystallized, err := types.NewGenesisCrystallizedState(genesisValidators)
	if err != nil {
		return nil, err
	}

	beaconDB.state.aState = active

	if !hasGenesis {
		log.Info("No genesis block found on disk, initializing genesis block")

		// Active state hash is predefined so error can be safely ignored
		// #nosec G104
		activeStateHash, _ := active.Hash()
		// Crystallized state hash is predefined so error can be safely ignored
		// #nosec G104
		crystallizedStateHash, _ := crystallized.Hash()
		genesisBlock := types.NewGenesisBlock(activeStateHash, crystallizedStateHash)
		if err := beaconDB.SaveBlock(genesisBlock); err != nil {
			return nil, err
		}
		if err := beaconDB.SaveCanonicalBlock(genesisBlock); err != nil {
			return nil, err
		}
		genesisBlockHash, err := genesisBlock.Hash()
		if err != nil {
			return nil, err
		}
		if err := beaconDB.SaveCanonicalSlotNumber(0, genesisBlockHash); err != nil {
			return nil, err
		}
	}
	if !hasCrystallized {
		log.Info("No chainstate found on disk, initializing beacon from genesis")
		beaconDB.state.cState = crystallized
		return beaconDB, nil
	}

	enc, err := beaconDB.get(crystallizedStateLookupKey)
	if err != nil {
		return nil, err
	}
	crystallizedData := &pb.CrystallizedState{}
	err = proto.Unmarshal(enc, crystallizedData)
	if err != nil {
		return nil, err
	}
	beaconDB.state.cState = types.NewCrystallizedState(crystallizedData)

	return beaconDB, nil
}
