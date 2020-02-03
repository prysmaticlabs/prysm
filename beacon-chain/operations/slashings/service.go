package slashings

import (
	"context"
	"sort"
	"sync"

	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Pool implements a struct to maintain pending and recently included voluntary exits. This pool
// is used by proposers to insert into new blocks.
type Pool struct {
	lock                    sync.RWMutex
	pendingProposerSlashing []*ethpb.ProposerSlashing
	pendingAttesterSlashing []*PendingAttesterSlashing
	included                map[uint64]bool
}

type PendingAttesterSlashing struct {
	attesterSlashing *ethpb.AttesterSlashing
	validatorToSlash uint64
}

// NewPool accepts a head fetcher (for reading the validator set) and returns an initialized
// slashed validator pool.
func NewPool() *Pool {
	return &Pool{
		pendingProposerSlashing: make([]*ethpb.ProposerSlashing, 0),
		pendingAttesterSlashing: make([]*PendingAttesterSlashing, 0),
		included:                make(map[uint64]bool),
	}
}

// PendingAttesterSlashings returns exits that are ready for inclusion at the given slot. This method will not
// return more than the block enforced MaxAttesterSlashings.
func (p *Pool) PendingAttesterSlashings() []*ethpb.AttesterSlashing {
	p.lock.RLock()
	defer p.lock.RUnlock()
	pending := make([]*ethpb.AttesterSlashing, 0)
	for _, e := range p.pendingAttesterSlashing {
		pending = append(pending, e.attesterSlashing)
	}
	if len(pending) > int(params.BeaconConfig().MaxAttesterSlashings) {
		pending = pending[:params.BeaconConfig().MaxAttesterSlashings]
	}
	return pending
}

// PendingAttesterSlashings returns exits that are ready for inclusion at the given slot. This method will not
// return more than the block enforced MaxProposerSlashings.
func (p *Pool) PendingProposerSlashings() []*ethpb.ProposerSlashing {
	p.lock.RLock()
	defer p.lock.RUnlock()
	pending := make([]*ethpb.ProposerSlashing, 0)
	for _, e := range p.pendingProposerSlashing {
		pending = append(pending, e)
	}
	if len(pending) > int(params.BeaconConfig().MaxProposerSlashings) {
		pending = pending[:params.BeaconConfig().MaxProposerSlashings]
	}
	return pending
}

// InsertAttesterSlashing into the pool. This method is a no-op if the pending exit already exists,
// has been included recently, or the validator is already exited.
func (p *Pool) InsertAttesterSlashing(ctx context.Context, state *beaconstate.BeaconState, slashing *ethpb.AttesterSlashing) {
	p.lock.Lock()
	defer p.lock.Unlock()

	slashedVal := sliceutil.IntersectionUint64(slashing.Attestation_1.AttestingIndices, slashing.Attestation_2.AttestingIndices)
	sort.Slice(slashedVal, func(i, j int) bool {
		return slashedVal[i] < slashedVal[j]
	})
	for i, val := range slashedVal {
		// Has this validator index been included recently?
		if p.included[val] {
			slashedVal = append(slashedVal[:i], slashedVal[i+1:]...)
		}
		stateValidators := state.Validators()
		// Has the validators been exited already?
		if len(stateValidators) <= int(val) || stateValidators[val].ExitEpoch < helpers.CurrentEpoch(state) {
			{
				slashedVal = append(slashedVal[:i], slashedVal[i+1:]...)
			}
		}
		//Has the validator been slashed already?
		slashedValidators := state.Slashings()
		if found := sort.Search(len(slashedValidators), func(i int) bool {
			return slashedValidators[i] == val
		}); found != len(slashedValidators) {
			slashedVal = append(slashedVal[:i], slashedVal[i+1:]...)
		}

		// Has the list of slashed validators been left empty?
		if len(slashedVal) == 0 {
			return
		}

		// Does this validator exist in the list already? Use binary search to find the answer.
		if found := sort.Search(len(p.pendingAttesterSlashing), func(i int) bool {
			e := p.pendingAttesterSlashing[i]
			return e.validatorToSlash == val
		}); found != len(p.pendingAttesterSlashing) {
			return
		}

		// Insert into pending list and sort again.
		p.pendingAttesterSlashing = append(p.pendingAttesterSlashing, &PendingAttesterSlashing{slashing, val})
		sort.Slice(p.pendingAttesterSlashing, func(i, j int) bool {
			return p.pendingAttesterSlashing[i].validatorToSlash < p.pendingAttesterSlashing[j].validatorToSlash
		})
	}
}

// InsertAttesterSlashing into the pool. This method is a no-op if the pending exit already exists,
// has been included recently, or the validator is already exited.
func (p *Pool) InsertProposerSlashing(ctx context.Context, state *beaconstate.BeaconState, slashing *ethpb.ProposerSlashing) {
	p.lock.Lock()
	defer p.lock.Unlock()

	// Has this validator index been included recently?
	if p.included[slashing.ProposerIndex] {
		return
	}
	// Has the validators been exited already?
	stateValidators := state.Validators()
	if len(stateValidators) <= int(slashing.ProposerIndex) || stateValidators[slashing.ProposerIndex].ExitEpoch < helpers.CurrentEpoch(state) {
		return
	}
	//Has the validator been slashed already?
	slashedValidators := state.Slashings()
	if found := sort.Search(len(slashedValidators), func(i int) bool {
		return slashedValidators[i] == slashing.ProposerIndex
	}); found != len(slashedValidators) {
		return
	}

	// Does this validator exist in the list already? Use binary search to find the answer.
	if found := sort.Search(len(p.pendingProposerSlashing), func(i int) bool {
		e := p.pendingProposerSlashing[i]
		return e.ProposerIndex == slashing.ProposerIndex
	}); found != len(p.pendingAttesterSlashing) {
		return
	}

	// Insert into pending list and sort again.
	p.pendingProposerSlashing = append(p.pendingProposerSlashing, slashing)
	sort.Slice(p.pendingProposerSlashing, func(i, j int) bool {
		return p.pendingProposerSlashing[i].ProposerIndex < p.pendingProposerSlashing[j].ProposerIndex
	})

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

// MarkIncludedProposerSlashing is used when an proposer slashing has been included in a beacon block.
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
		if i != len(p.pendingProposerSlashing) {
			p.pendingProposerSlashing = append(p.pendingProposerSlashing[:i], p.pendingProposerSlashing[i+1:]...)
		}
		p.included[val] = true
	}
}
