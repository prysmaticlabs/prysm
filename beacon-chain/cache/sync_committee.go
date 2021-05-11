package cache

import (
	"errors"
	"sync"

	statealtair "github.com/prysmaticlabs/prysm/beacon-chain/state/state-altair"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"k8s.io/client-go/tools/cache"
)

// errNonExistingSyncCommitteeKey will be returned when key does not exists.
var errNonExistingSyncCommitteeKey = errors.New("does not exist sync committee key")

type SyncCommitteeCache struct {
	cache *cache.FIFO
	lock sync.RWMutex
}

type syncCommitteeItem struct {
	currentSyncCommitteeRoot [32]byte
	assignment map[[48]byte]*indexPositionInCommittee
}

type indexPositionInCommittee struct {
	currentEpoch []uint64
	nextEpoch []uint64
}

func NewSyncCommittee() *SyncCommitteeCache {
	return &SyncCommitteeCache{
		cache: cache.NewFIFO(keyFn),
	}
}

func keyFn(obj interface{}) (string, error) {
	info, ok := obj.(*syncCommitteeItem)
	if !ok {
		return "", errors.New("obj is not syncCommitteeItem")
	}

	return string(info.currentSyncCommitteeRoot[:]), nil
}


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

func (s *SyncCommitteeCache) idxPositionInCommittee(root [32]byte, pubKey [48]byte) (*indexPositionInCommittee, error) {
	obj, exists, err := s.cache.GetByKey(key(root))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errNonExistingSyncCommitteeKey
	}
	item, ok := obj.(*syncCommitteeItem)
	if !ok {
		return nil, errors.New("obj is not syncCommitteeItem")
	}
	idxInCommittee, ok := item.assignment[pubKey]
	if !ok {
		return nil, nil
	}
	return idxInCommittee, nil
}

func (s *SyncCommitteeCache) UpdateEpochAssignment(state *statealtair.BeaconState) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	csc, err := state.CurrentSyncCommittee()
	if err != nil {
		return err
	}
	newAssignment := make(map[[48]byte]*indexPositionInCommittee)
	for i, pubkey := range csc.Pubkeys {
		p := bytesutil.ToBytes48(pubkey)
		if _, ok := newAssignment[p]; !ok {
			m := &indexPositionInCommittee{currentEpoch: []uint64{uint64(i)}}
			newAssignment[p] = m
		} else {
			newAssignment[p].currentEpoch = append(newAssignment[p].currentEpoch, uint64(i))
		}
	}

	nsc, err := state.NextSyncCommittee()
	if err != nil {
		return err
	}
	for i, pubkey := range nsc.Pubkeys {
		p := bytesutil.ToBytes48(pubkey)
		if _, ok := newAssignment[p]; !ok {
			m := &indexPositionInCommittee{currentEpoch: []uint64{uint64(i)}}
			newAssignment[p] = m
		} else {
			newAssignment[p].currentEpoch = append(newAssignment[p].currentEpoch, uint64(i))
		}
	}
}