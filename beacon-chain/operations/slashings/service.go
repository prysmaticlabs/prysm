package slashings

import (
	"errors"
	"fmt"
	"sort"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

// NewPool returns an initialized attester slashing and proposer slashing pool.
func NewPool() *Pool {
	return &Pool{
		pendingProposerSlashing: make([]*ethpb.ProposerSlashing, 0),
		pendingAttesterSlashing: make([]*PendingAttesterSlashing, 0),
		included:                make(map[uint64]bool),
	}
}

// PendingAttesterSlashings returns attester slashings that are able to be included into a block.
// This method will not return more than the block enforced MaxAttesterSlashings.
func (p *Pool) PendingAttesterSlashings() []*ethpb.AttesterSlashing {
	p.lock.RLock()
	defer p.lock.RUnlock()

	included := make(map[uint64]bool)
	pending := make([]*ethpb.AttesterSlashing, 0, params.BeaconConfig().MaxAttesterSlashings)
	for i, slashing := range p.pendingAttesterSlashing {
		if i >= int(params.BeaconConfig().MaxAttesterSlashings) {
			break
		}
		if included[slashing.validatorToSlash] {
			continue
		}
		attSlashing := slashing.attesterSlashing
		slashedVal := sliceutil.IntersectionUint64(attSlashing.Attestation_1.AttestingIndices, attSlashing.Attestation_2.AttestingIndices)
		for _, idx := range slashedVal {
			included[idx] = true
		}
		pending = append(pending, attSlashing)
	}

	return pending
}

// PendingProposerSlashings returns proposer slashings that are able to be included into a block.
// This method will not return more than the block enforced MaxProposerSlashings.
func (p *Pool) PendingProposerSlashings() []*ethpb.ProposerSlashing {
	p.lock.RLock()
	defer p.lock.RUnlock()
	pending := make([]*ethpb.ProposerSlashing, 0, params.BeaconConfig().MaxProposerSlashings)
	for i, slashing := range p.pendingProposerSlashing {
		if i >= int(params.BeaconConfig().MaxProposerSlashings) {
			break
		}
		pending = append(pending, slashing)
	}
	return pending
}

// InsertAttesterSlashing into the pool. This method is a no-op if the attester slashing already exists in the pool,
// has been included into a block recently, or the validator is already exited.
func (p *Pool) InsertAttesterSlashing(state *beaconstate.BeaconState, slashing *ethpb.AttesterSlashing) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	slashedVal := sliceutil.IntersectionUint64(slashing.Attestation_1.AttestingIndices, slashing.Attestation_2.AttestingIndices)
	for _, val := range slashedVal {
		// Has this validator index been included recently?
		ok, err := p.validatorSlashingPreconditionCheck(state, val)
		if err != nil {
			return err
		}
		// If the validator has already exited, has already been slashed, or if its index
		// has been recently included in the pool of slashings, do not process this new
		// slashing.
		if !ok {
			return fmt.Errorf("validator at index %d cannot be slashed", val)
		}

		// Check if the validator already exists in the list of slashings.
		// Use binary search to find the answer.
		found := sort.Search(len(p.pendingAttesterSlashing), func(i int) bool {
			return p.pendingAttesterSlashing[i].validatorToSlash == val
		})
		if found != len(p.pendingAttesterSlashing) {
			continue
		}

		pendingSlashing := &PendingAttesterSlashing{
			attesterSlashing: slashing,
			validatorToSlash: val,
		}

		// Insert into pending list and sort again.
		p.pendingAttesterSlashing = append(p.pendingAttesterSlashing, pendingSlashing)
		sort.Slice(p.pendingAttesterSlashing, func(i, j int) bool {
			return p.pendingAttesterSlashing[i].validatorToSlash < p.pendingAttesterSlashing[j].validatorToSlash
		})
	}
	return nil
}

// InsertProposerSlashing into the pool. This method is a no-op if the pending slashing already exists,
// has been included recently, the validator is already exited, or the validator was already slashed.
func (p *Pool) InsertProposerSlashing(state *beaconstate.BeaconState, slashing *ethpb.ProposerSlashing) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	idx := slashing.ProposerIndex
	ok, err := p.validatorSlashingPreconditionCheck(state, idx)
	if err != nil {
		return err
	}
	// If the validator has already exited, has already been slashed, or if its index
	// has been recently included in the pool of slashings, do not process this new
	// slashing.
	if !ok {
		return fmt.Errorf("validator at index %d cannot be slashed", idx)
	}

	// Check if the validator already exists in the list of slashings.
	// Use binary search to find the answer.
	found := sort.Search(len(p.pendingProposerSlashing), func(i int) bool {
		return p.pendingProposerSlashing[i].ProposerIndex == slashing.ProposerIndex
	})
	if found != len(p.pendingProposerSlashing) {
		return errors.New("slashing object already exists in pending proposer slashings")
	}

	// Insert into pending list and sort again.
	p.pendingProposerSlashing = append(p.pendingProposerSlashing, slashing)
	sort.Slice(p.pendingProposerSlashing, func(i, j int) bool {
		return p.pendingProposerSlashing[i].ProposerIndex < p.pendingProposerSlashing[j].ProposerIndex
	})
	return nil
}

// MarkIncludedAttesterSlashing is used when an attester slashing has been included in a beacon block.
// Every block seen by this node that contains proposer slashings should call this method to include
// the proposer slashings.
func (p *Pool) MarkIncludedAttesterSlashing(as *ethpb.AttesterSlashing) {
	p.lock.Lock()
	defer p.lock.Unlock()
	slashedVal := sliceutil.IntersectionUint64(as.Attestation_1.AttestingIndices, as.Attestation_2.AttestingIndices)
	sort.Slice(slashedVal, func(i, j int) bool {
		return slashedVal[i] < slashedVal[j]
	})
	for _, val := range slashedVal {
		i := sort.Search(len(p.pendingAttesterSlashing), func(i int) bool {
			return p.pendingAttesterSlashing[i].validatorToSlash == val
		})
		if i != len(p.pendingAttesterSlashing) {
			p.pendingAttesterSlashing = append(p.pendingAttesterSlashing[:i], p.pendingAttesterSlashing[i+1:]...)
		}
		p.included[val] = true
	}
}

// MarkIncludedProposerSlashing is used when an proposer slashing has been included in a beacon block.
// Every block seen by this node that contains proposer slashings should call this method to include
// the proposer slashings.
func (p *Pool) MarkIncludedProposerSlashing(ps *ethpb.ProposerSlashing) {
	p.lock.Lock()
	defer p.lock.Unlock()
	i := sort.Search(len(p.pendingProposerSlashing), func(i int) bool {
		return p.pendingProposerSlashing[i].ProposerIndex == ps.ProposerIndex
	})
	if i != len(p.pendingProposerSlashing) {
		p.pendingProposerSlashing = append(p.pendingProposerSlashing[:i], p.pendingProposerSlashing[i+1:]...)
	}
	p.included[ps.ProposerIndex] = true
}

// this function checks a few items about a validator before proceeding with inserting
// a proposer/attester slashing into the pool. First, it checks if the validator
// has been recently included in the pool, then it checks if the validator has exited,
// finally, it ensures the validator has not yet been slashed.
func (p *Pool) validatorSlashingPreconditionCheck(
	state *beaconstate.BeaconState,
	valIdx uint64,
) (bool, error) {
	// Check if the validator index has been included recently.
	if p.included[valIdx] {
		return false, nil
	}
	validator, err := state.ValidatorAtIndexReadOnly(valIdx)
	if err != nil {
		return false, err
	}
	// Checking if the validator has already exited.
	if validator.ExitEpoch() < helpers.CurrentEpoch(state) {
		return false, nil
	}
	// Checking if the validator has already been slashed.
	if validator.Slashed() {
		return false, nil
	}
	return true, nil
}
