package forkchoice

import (
	"sync"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/attestation"
)

// Attestations --
type Attestations struct {
	atts map[attestation.Id]ethpb.Att
	sync.RWMutex
}

// New --
func New() *Attestations {
	return &Attestations{atts: make(map[attestation.Id]ethpb.Att)}
}

// SaveForkchoiceAttestation saves a forkchoice attestation.
func (a *Attestations) SaveForkchoiceAttestation(att ethpb.Att) error {
	if att == nil {
		return nil
	}

	id, err := attestation.NewId(att, attestation.Full)
	if err != nil {
		return errors.Wrap(err, "could not create attestation ID")
	}

	a.Lock()
	defer a.Unlock()
	a.atts[id] = att

	return nil
}

// SaveForkchoiceAttestations saves a list of forkchoice attestations.
func (a *Attestations) SaveForkchoiceAttestations(atts []ethpb.Att) error {
	for _, att := range atts {
		if err := a.SaveForkchoiceAttestation(att); err != nil {
			return err
		}
	}

	return nil
}

// ForkchoiceAttestations returns all forkchoice attestations.
func (a *Attestations) ForkchoiceAttestations() []ethpb.Att {
	a.RLock()
	defer a.RUnlock()

	atts := make([]ethpb.Att, len(a.atts))
	i := 0
	for _, att := range a.atts {
		atts[i] = att.Clone()
		i++
	}

	return atts
}

// DeleteForkchoiceAttestation deletes a forkchoice attestation.
func (a *Attestations) DeleteForkchoiceAttestation(att ethpb.Att) error {
	if att == nil {
		return nil
	}

	id, err := attestation.NewId(att, attestation.Full)
	if err != nil {
		return errors.Wrap(err, "could not create attestation ID")
	}

	a.Lock()
	defer a.Unlock()
	delete(a.atts, id)

	return nil
}

// ForkchoiceAttestationCount returns the number of forkchoice attestation keys.
func (a *Attestations) ForkchoiceAttestationCount() int {
	a.RLock()
	defer a.RUnlock()
	return len(a.atts)
}
