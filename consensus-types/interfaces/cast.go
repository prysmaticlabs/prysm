package interfaces

import "github.com/prysmaticlabs/prysm/v5/runtime/version"

// AsROBlockBodyElectra safely asserts the ReadOnlyBeaconBlockBody to a ROBlockBodyElectra.
// This allows the caller to access methods on the block body which are only available on values after
// the Electra hard fork. If the value is for an earlier fork (based on comparing its Version() to the electra version)
// an error will be returned. Callers that want to conditionally process electra data can check for this condition
// and safely ignore it like `if err != nil && errors.Is(interfaces.ErrInvalidCast) {`
func AsROBlockBodyElectra(in ReadOnlyBeaconBlockBody) (ROBlockBodyElectra, error) {
	if in.Version() >= version.Electra {
		return in.(ROBlockBodyElectra), nil
	}
	return nil, NewInvalidCastError(in.Version(), version.Electra)
}
