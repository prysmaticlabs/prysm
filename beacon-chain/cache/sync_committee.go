package cache

import (
	"errors"
	"sync"

	statealtair "github.com/prysmaticlabs/prysm/beacon-chain/state/state-altair"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"k8s.io/client-go/tools/cache"
)

var errNonExistingSyncCommitteeKey = errors.New("does not exist sync committee key")
var errNotSyncCommitteeIndexPosition = errors.New("obj is not syncCommitteeIndexPosition struct")
var maxSyncCommitteeSize = uint64(3)

type SyncCommitteeCache struct {
	cache *cache.FIFO
	lock sync.RWMutex
}

type syncCommitteeIndexPosition struct {
	currentSyncCommitteeRoot [32]byte
	vIndexToPositionMap map[[48]byte]*positionInCommittee
}

type positionInCommittee struct {
	currentEpoch []uint64
	nextEpoch []uint64
}

func NewSyncCommittee() *SyncCommitteeCache {
	return &SyncCommitteeCache{
		cache: cache.NewFIFO(keyFn),
	}
}

func keyFn(obj interface{}) (string, error) {
	info, ok := obj.(*syncCommitteeIndexPosition)
	if !ok {
		return "", errNotSyncCommitteeIndexPosition
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

func (s *SyncCommitteeCache) idxPositionInCommittee(root [32]byte, pubKey [48]byte) (*positionInCommittee, error) {
	obj, exists, err := s.cache.GetByKey(key(root))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errNonExistingSyncCommitteeKey
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

func (s *SyncCommitteeCache) UpdatePositionsInCommittee(state *statealtair.BeaconState) error {
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

	s.cache.Add(&syncCommitteeIndexPosition{
		currentSyncCommitteeRoot: r,
		vIndexToPositionMap: positionsMap,
	})
	trim(s.cache, maxSyncCommitteeSize)

	return nil
}
