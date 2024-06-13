package blocks

import (
	"bytes"
	"encoding/binary"
	"sync"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	lruwrpr "github.com/prysmaticlabs/prysm/v5/cache/lru"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
)

func makeAtts(b *testing.B) []*ethpb.Attestation {
	attData := make([]*ethpb.AttestationData, 0, 400)
	for i := 0; i < 400; i++ {
		bRoot := bytesutil.Bytes32(20)
		sRoot := bytesutil.Bytes32(1000)
		tRoot := bytesutil.Bytes32(2000)
		ad := &ethpb.AttestationData{
			Slot:            3600,
			CommitteeIndex:  primitives.CommitteeIndex(i),
			BeaconBlockRoot: bRoot,
			Source: &ethpb.Checkpoint{
				Epoch: primitives.Epoch(1000),
				Root:  sRoot,
			},
			Target: &ethpb.Checkpoint{
				Epoch: primitives.Epoch(2000),
				Root:  tRoot,
			},
		}
		attData = append(attData, ad)
	}
	allAtts := make([]*ethpb.Attestation, 0)

	for _, ad := range attData {
		copiedD := ad
		for i := 0; i < 30; i++ {
			att := &ethpb.Attestation{
				AggregationBits: bitfield.NewBitlist(3),
				Data:            copiedD,
				Signature:       make([]byte, 96),
			}
			allAtts = append(allAtts, att)
		}
	}
	return allAtts
}

func BenchmarkAttestationDataRoot(b *testing.B) {
	allAtts := makeAtts(b)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, a := range allAtts {
			_, err := a.Data.HashTreeRoot()
			assert.NoError(b, err)
		}
	}
}

var lc = lruwrpr.New(100)

func getRoot(attData *ethpb.AttestationData) ([32]byte, error) {
	var key [128]byte

	binary.LittleEndian.PutUint64(key[:8], uint64(attData.Slot))
	binary.LittleEndian.PutUint64(key[8:16], uint64(attData.CommitteeIndex))
	copy(key[16:48], attData.BeaconBlockRoot)
	binary.LittleEndian.PutUint64(key[48:56], uint64(attData.Target.Epoch))
	copy(key[56:88], attData.Target.Root)
	binary.LittleEndian.PutUint64(key[88:96], uint64(attData.Source.Epoch))
	copy(key[96:128], attData.Source.Root)

	rt, ok := lc.Get(key)
	if ok {
		return rt.([32]byte), nil
	}
	htr, err := attData.HashTreeRoot()
	if err != nil {
		return [32]byte{}, err
	}
	lc.Add(key, htr)
	return htr, nil
}

func BenchmarkAttestationDataRootWithCache(b *testing.B) {
	allAtts := makeAtts(b)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, a := range allAtts {
			rt, err := getRoot(a.Data)
			_ = rt
			assert.NoError(b, err)
		}
	}
}

var rtMap = map[uint64]map[uint64]map[[32]byte]map[uint64]map[[32]byte]map[uint64]map[[32]byte][32]byte{}
var sltMap = map[uint64][]*ethpb.AttestationData{}
var attMap = map[*ethpb.AttestationData][32]byte{}

var mt = new(sync.RWMutex)

func getRootInMap(attData *ethpb.AttestationData) ([32]byte, error) {
	mt.RLock()
	mpA, ok := rtMap[uint64(attData.Slot)]
	mt.RUnlock()
	if !ok {
		return recoverRtInMap(attData)
	}
	mt.RLock()
	mpB, ok := mpA[uint64(attData.CommitteeIndex)]
	mt.RUnlock()
	if !ok {
		return recoverRtInMap(attData)

	}
	mt.RLock()
	mpC, ok := mpB[[32]byte(attData.BeaconBlockRoot)]
	mt.RUnlock()
	if !ok {
		return recoverRtInMap(attData)

	}
	mt.RLock()
	mpD, ok := mpC[uint64(attData.Target.Epoch)]
	mt.RUnlock()
	if !ok {
		return recoverRtInMap(attData)

	}
	mt.RLock()
	mpE, ok := mpD[[32]byte(attData.Target.Root)]
	mt.RUnlock()
	if !ok {
		return recoverRtInMap(attData)

	}
	mt.RLock()
	mpF, ok := mpE[uint64(attData.Target.Epoch)]
	mt.RUnlock()
	if !ok {
		return recoverRtInMap(attData)
	}
	mt.RLock()
	htr, ok := mpF[[32]byte(attData.Target.Root)]
	mt.RUnlock()
	if !ok {
		return recoverRtInMap(attData)
	}
	return htr, nil
}

func recoverRtInMap(attData *ethpb.AttestationData) ([32]byte, error) {
	htr, err := attData.HashTreeRoot()
	if err != nil {
		return [32]byte{}, err
	}
	mt.Lock()
	rtMap[uint64(attData.Slot)] = map[uint64]map[[32]byte]map[uint64]map[[32]byte]map[uint64]map[[32]byte][32]byte{}
	rtMap[uint64(attData.Slot)][uint64(attData.CommitteeIndex)] = map[[32]byte]map[uint64]map[[32]byte]map[uint64]map[[32]byte][32]byte{}
	rtMap[uint64(attData.Slot)][uint64(attData.CommitteeIndex)][[32]byte(attData.BeaconBlockRoot)] = map[uint64]map[[32]byte]map[uint64]map[[32]byte][32]byte{}
	rtMap[uint64(attData.Slot)][uint64(attData.CommitteeIndex)][[32]byte(attData.BeaconBlockRoot)][uint64(attData.Target.Epoch)] = map[[32]byte]map[uint64]map[[32]byte][32]byte{}
	rtMap[uint64(attData.Slot)][uint64(attData.CommitteeIndex)][[32]byte(attData.BeaconBlockRoot)][uint64(attData.Target.Epoch)][[32]byte(attData.Target.Root)] = map[uint64]map[[32]byte][32]byte{}
	rtMap[uint64(attData.Slot)][uint64(attData.CommitteeIndex)][[32]byte(attData.BeaconBlockRoot)][uint64(attData.Target.Epoch)][[32]byte(attData.Target.Root)][uint64(attData.Source.Epoch)] = map[[32]byte][32]byte{}
	rtMap[uint64(attData.Slot)][uint64(attData.CommitteeIndex)][[32]byte(attData.BeaconBlockRoot)][uint64(attData.Target.Epoch)][[32]byte(attData.Target.Root)][uint64(attData.Source.Epoch)][[32]byte(attData.Source.Root)] = htr
	mt.Unlock()
	return htr, nil
}

func BenchmarkAttestationDataRootWithMap(b *testing.B) {
	allAtts := makeAtts(b)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, a := range allAtts {
			rt, err := getRootInMap(a.Data)
			_ = rt
			assert.NoError(b, err)
		}
	}
}

func recoverRtInMapNew(attData *ethpb.AttestationData) ([32]byte, error) {
	htr, err := attData.HashTreeRoot()
	if err != nil {
		return [32]byte{}, err
	}
	mt.Lock()
	sltMap[uint64(attData.Slot)] = append(sltMap[uint64(attData.Slot)], attData)
	attMap[attData] = htr
	mt.Unlock()
	return htr, nil
}

func getRootInMapNew(attData *ethpb.AttestationData) ([32]byte, error) {
	mt.RLock()
	attList, ok := sltMap[uint64(attData.Slot)]
	mt.RUnlock()
	if !ok {
		return recoverRtInMapNew(attData)
	}
	for _, a := range attList {
		if attData.CommitteeIndex != a.CommitteeIndex {
			continue
		}
		if !bytes.Equal(attData.BeaconBlockRoot, a.BeaconBlockRoot) {
			continue
		}
		if attData.Target.Epoch != a.Target.Epoch {
			continue
		}
		if !bytes.Equal(attData.Target.Root, a.Target.Root) {
			continue
		}
		if attData.Source.Epoch != a.Source.Epoch {
			continue
		}
		if !bytes.Equal(attData.Source.Root, a.Source.Root) {
			continue
		}
		mt.RLock()
		val, ok := attMap[a]
		mt.RUnlock()
		if !ok {
			return recoverRtInMapNew(attData)
		}
		return val, nil
	}
	return recoverRtInMapNew(attData)
}

func BenchmarkAttestationDataRootWithMapNew(b *testing.B) {
	allAtts := makeAtts(b)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, a := range allAtts {
			rt, err := getRootInMapNew(a.Data)
			_ = rt
			assert.NoError(b, err)
		}
	}
}
