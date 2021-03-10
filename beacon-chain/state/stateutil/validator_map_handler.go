package stateutil

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	coreutils "github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
)

// ValidatorMapHandler is a container to hold the map and a reference tracker for how many
// states shared this.
type ValidatorMapHandler struct {
	valIdxMap map[[48]byte]types.ValidatorIndex
	mapRef    *Reference
}

// NewValMapHandler returns a new validator map handler.
func NewValMapHandler(vals []*ethpb.Validator) *ValidatorMapHandler {
	return &ValidatorMapHandler{
		valIdxMap: coreutils.ValidatorIndexMap(vals),
		mapRef:    &Reference{refs: 1},
	}
}

// Copy the whole map and returns a map handler with the copied map.
func (v *ValidatorMapHandler) AddRef() {
	v.mapRef.AddRef()
}

// ValidatorIndexMap returns the validator index map.
func (v *ValidatorMapHandler) ValidatorIndexMap() map[[48]byte]types.ValidatorIndex {
	return v.valIdxMap
}

// MapRef returns the map reference.
func (v *ValidatorMapHandler) MapRef() *Reference {
	return v.mapRef
}

// IsNil returns true if the underlying validator index map is nil.
func (v *ValidatorMapHandler) IsNil() bool {
	return v.mapRef == nil
}

// Copy the whole map and returns a map handler with the copied map.
func (v *ValidatorMapHandler) Copy() *ValidatorMapHandler {
	if v == nil || v.valIdxMap == nil {
		return &ValidatorMapHandler{valIdxMap: map[[48]byte]types.ValidatorIndex{}, mapRef: new(Reference)}
	}
	m := make(map[[48]byte]types.ValidatorIndex, len(v.valIdxMap))
	for k, v := range v.valIdxMap {
		m[k] = v
	}
	return &ValidatorMapHandler{
		valIdxMap: m,
		mapRef:    &Reference{refs: 1},
	}
}

// Get the validator index using the corresponding public key.
func (v *ValidatorMapHandler) Get(key [48]byte) (types.ValidatorIndex, bool) {
	idx, ok := v.valIdxMap[key]
	if !ok {
		return 0, false
	}
	return idx, true
}

// Set the validator index using the corresponding public key.
func (v *ValidatorMapHandler) Set(key [48]byte, index types.ValidatorIndex) {
	v.valIdxMap[key] = index
}
