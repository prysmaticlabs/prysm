package db

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/forkutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// createBlock This function has been copied from beacon-chain/db/block.go. Move it to shared metod?
func createBlock(enc []byte) (*pbp2p.BeaconBlock, error) {
	protoBlock := &pbp2p.BeaconBlock{}
	err := proto.Unmarshal(enc, protoBlock)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}
	return protoBlock, nil
}

// SaveProposedBlock save proposed beacon block to disk
func (db *ValidatorDB) SaveProposedBlock(fork *pbp2p.Fork, pubKey *bls.PublicKey, block *pbp2p.BeaconBlock) error {
	epoch := block.Slot / params.BeaconConfig().SlotsPerEpoch

	if lastProposedBlockEpoch, ok := db.lastProposedBlockEpoch[(*pubKey)]; !ok || lastProposedBlockEpoch < epoch {
		db.lastProposedBlockEpoch[(*pubKey)] = epoch
	}
	forkVersion := forkutil.ForkVersion(fork, epoch)
	blockEnc, err := block.Marshal()
	if err != nil {
		return fmt.Errorf("failed to encode block: %v", err)
	}

	return db.update(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, pubKey, forkVersion, proposedBlockBucket, true)
		return bucket.Put(bytesutil.Bytes8(epoch), blockEnc)
	})
}

// GetProposedBlock returns the previously proposed block from disk.
// if validator has not previously proposed a block, then it returns nil
func (db *ValidatorDB) GetProposedBlock(fork *pbp2p.Fork, pubKey *bls.PublicKey, epoch uint64) (block *pbp2p.BeaconBlock, err error) {
	// maybe it makes no sense to read to disk
	lastProposedBlockEpoch, lastProposedBlockEpochExists := db.lastProposedBlockEpoch[(*pubKey)]
	if !lastProposedBlockEpochExists {
		// lastProposedBlockEpoch not initiated for this key, initiate it
		lastProposedBlockEpoch, err = db.getMaxProposedEpoch(pubKey)
		if err != nil {
			log.WithError(err).Error("Can not init lastProposedBlockEpoch")
			return nil, err
		}
		db.lastProposedBlockEpoch[(*pubKey)] = lastProposedBlockEpoch
	}
	if lastProposedBlockEpoch < epoch {
		return
	}
	// try read block from disk
	forkVersion := forkutil.ForkVersion(fork, epoch)
	err = db.view(func(tx *bolt.Tx) error {
		bucket := getBucket(tx, pubKey, forkVersion, proposedBlockBucket, false)
		if bucket != nil {
			blockEnc := bucket.Get(bytesutil.Bytes8(epoch))
			block, err = createBlock(blockEnc)
		}
		return err
	})
	return
}

// getMaxProposedEpoch return max epoch for saved proposed block. Used for lastProposedBlockEpoch init
func (db *ValidatorDB) getMaxProposedEpoch(pubKey *bls.PublicKey) (maxProposedEpoch uint64, err error) {
	err = db.lastInAllForks(pubKey, proposedBlockBucket, func(_, lastInForkEnc []byte) error {
		if lastInForkEnc == nil {
			return nil
		}
		lastInFork, err := createBlock(lastInForkEnc)
		if err != nil {
			log.Fatalf("can't create block: %s", err)
			return err
		}

		maxProposedEpoch = lastInFork.Slot / params.BeaconConfig().SlotsPerEpoch
		return nil
	})
	return
}
