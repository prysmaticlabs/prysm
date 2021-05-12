package cache

import (
	"errors"
	"sync"

	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"k8s.io/client-go/tools/cache"
)

var ErrNonExistingSyncCommitteeKey = errors.New("does not exist sync committee key")
var errNotSyncCommitteeIndexPosition = errors.New("not syncCommitteeIndexPosition struct")
var maxSyncCommitteeSize = uint64(3) // Allows 3 forks to happen around `EPOCHS_PER_SYNC_COMMITTEE_PERIOD` boundary.

// SyncCommitteeCache utilizes a FIFO cache to sufficiently cache validator position within sync committee.
// It is thread safe with concurrent read write.
type SyncCommitteeCache struct {
	cache *cache.FIFO
	lock  sync.RWMutex
}

// Index position of all validators in sync committee where `currentSyncCommitteeRoot` is the key and `vIndexToPositionMap` is value.
// Inside `vIndexToPositionMap`, validator positions are cached where key is the 48byte public key and the value is the `positionInCommittee` struct.
type syncCommitteeIndexPosition struct {
	currentSyncCommitteeRoot [32]byte
	vIndexToPositionMap      map[[48]byte]*positionInCommittee
}

// Index position of individual validator of current epoch and previous epoch sync committee.
type positionInCommittee struct {
	currentEpoch []uint64
	nextEpoch    []uint64
}

// NewSyncCommittee initializes and returns a new SyncCommitteeCache.
func NewSyncCommittee() *SyncCommitteeCache {
	return &SyncCommitteeCache{
		cache: cache.NewFIFO(keyFn),
	}
}

// CurrentEpochIndexPosition returns current epoch index position of validator pubKey with respect with sync committee.
// If the input pubKey has no assignment, an empty list will be returned.
// If the input root does not exist in cache, ErrNonExistingSyncCommitteeKey is returned. Then performing manual checking of state for index position in state is recommended.
func (s *SyncCommitteeCache) CurrentEpochIndexPosition(root [32]byte, pubKey [48]byte) ([]uint64, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	pos, err := s.idxPositionInCommittee(root, pubKey)
	if err != nil {
		return nil, err
	}
	if pos == nil {
		return []uint64{}, nil
	}

	return pos.currentEpoch, nil
}

// NextEpochIndexPosition returns next epoch index position of validator pubKey in respect with sync committee.
// If the input pubKey has no assignment, an empty list will be returned.
// If the input root does not exist in cache, ErrNonExistingSyncCommitteeKey is returned. Then performing manual checking of state for index position in state is recommended.
func (s *SyncCommitteeCache) NextEpochIndexPosition(root [32]byte, pubKey [48]byte) ([]uint64, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	pos, err := s.idxPositionInCommittee(root, pubKey)
	if err != nil {
		return nil, err
	}
	if pos == nil {
		return []uint64{}, nil
	}
	return pos.nextEpoch, nil
}

// Helper function for `CurrentEpochIndexPosition` and `NextEpochIndexPosition` to return a mapping
// of validator pubKey to its index(s) position.
func (s *SyncCommitteeCache) idxPositionInCommittee(root [32]byte, pubKey [48]byte) (*positionInCommittee, error) {
	obj, exists, err := s.cache.GetByKey(key(root))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNonExistingSyncCommitteeKey
	}
	item, ok := obj.(*syncCommitteeIndexPosition)
	if !ok {
		return nil, errNotSyncCommitteeIndexPosition
	}
	idxInCommittee, ok := item.vIndexToPositionMap[pubKey]
	if !ok {
		return nil, nil
	}
	return idxInCommittee, nil
}

// UpdatePositionsInCommittee updates caching of validators position in sync committee in respect to
// current epoch and next epoch. This should be called when `current_sync_committee` and `next_sync_committee`
// change and that happens every `EPOCHS_PER_SYNC_COMMITTEE_PERIOD`.
func (s *SyncCommitteeCache) UpdatePositionsInCommittee(state iface.BeaconStateAltair) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	csc, err := state.CurrentSyncCommittee()
	if err != nil {
		return err
	}
	positionsMap := make(map[[48]byte]*positionInCommittee)
	for i, pubkey := range csc.Pubkeys {
		p := bytesutil.ToBytes48(pubkey)
		if _, ok := positionsMap[p]; !ok {
			m := &positionInCommittee{currentEpoch: []uint64{uint64(i)}, nextEpoch: []uint64{}}
			positionsMap[p] = m
		} else {
			positionsMap[p].currentEpoch = append(positionsMap[p].currentEpoch, uint64(i))
		}
	}

	nsc, err := state.NextSyncCommittee()
	if err != nil {
		return err
	}
	for i, pubkey := range nsc.Pubkeys {
		p := bytesutil.ToBytes48(pubkey)
		if _, ok := positionsMap[p]; !ok {
			m := &positionInCommittee{nextEpoch: []uint64{uint64(i)}, currentEpoch: []uint64{}}
			positionsMap[p] = m
		} else {
			positionsMap[p].nextEpoch = append(positionsMap[p].nextEpoch, uint64(i))
		}
	}

	r, err := csc.HashTreeRoot()
	if err != nil {
		return err
	}

	if err := s.cache.Add(&syncCommitteeIndexPosition{
		currentSyncCommitteeRoot: r,
		vIndexToPositionMap:      positionsMap,
	}); err != nil {
		return err
	}
	trim(s.cache, maxSyncCommitteeSize)

	return nil
}

// Given the `syncCommitteeIndexPosition` object, this returns the key of the object.
// The key is the `currentSyncCommitteeRoot` within the field.
// Error gets returned if input does not comply with `currentSyncCommitteeRoot` object.
func keyFn(obj interface{}) (string, error) {
	info, ok := obj.(*syncCommitteeIndexPosition)
	if !ok {
		return "", errNotSyncCommitteeIndexPosition
	}

	return string(info.currentSyncCommitteeRoot[:]), nil
}
