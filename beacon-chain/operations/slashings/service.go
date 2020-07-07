package slashings

import (
	"context"
	"fmt"
	"sort"

	"github.com/prysmaticlabs/prysm/shared/mathutil"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"go.opencensus.io/trace"
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
func (p *Pool) PendingAttesterSlashings(ctx context.Context, state *beaconstate.BeaconState) []*ethpb.AttesterSlashing {
	p.lock.RLock()
	defer p.lock.RUnlock()
	ctx, span := trace.StartSpan(ctx, "operations.PendingAttesterSlashing")
	defer span.End()

	// Update prom metric.
	numPendingAttesterSlashings.Set(float64(len(p.pendingAttesterSlashing)))

	included := make(map[uint64]bool)
	// Allocate pending slice with a capacity of min(len(p.pendingAttesterSlashing), maxAttesterSlashings)
	// since the array cannot exceed the max and is typically less than the max value.
	pending := make([]*ethpb.AttesterSlashing, 0, mathutil.Min(uint64(len(p.pendingAttesterSlashing)), params.BeaconConfig().MaxAttesterSlashings))
	for i := 0; i < len(p.pendingAttesterSlashing); i++ {
		slashing := p.pendingAttesterSlashing[i]
		if uint64(len(pending)) >= params.BeaconConfig().MaxAttesterSlashings {
			break
		}
		valid, err := p.validatorSlashingPreconditionCheck(state, slashing.validatorToSlash)
		if err != nil {
			log.WithError(err).Error("could not validate attester slashing")
			continue
		}
		if included[slashing.validatorToSlash] || !valid {
			p.pendingAttesterSlashing = append(p.pendingAttesterSlashing[:i], p.pendingAttesterSlashing[i+1:]...)
			i--
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
func (p *Pool) PendingProposerSlashings(ctx context.Context, state *beaconstate.BeaconState) []*ethpb.ProposerSlashing {
	p.lock.RLock()
	defer p.lock.RUnlock()
	ctx, span := trace.StartSpan(ctx, "operations.PendingProposerSlashing")
	defer span.End()

	// Update prom metric.
	numPendingProposerSlashings.Set(float64(len(p.pendingProposerSlashing)))

	// Allocate pending slice with a capacity of min(len(p.pendingProposerSlashing), maxProposerSlashings)
	// since the array cannot exceed the max and is typically less than the max value.
	pending := make([]*ethpb.ProposerSlashing, 0, mathutil.Min(uint64(len(p.pendingProposerSlashing)), params.BeaconConfig().MaxProposerSlashings))
	for i := 0; i < len(p.pendingProposerSlashing); i++ {
		slashing := p.pendingProposerSlashing[i]
		if uint64(len(pending)) >= params.BeaconConfig().MaxProposerSlashings {
			break
		}
		valid, err := p.validatorSlashingPreconditionCheck(state, slashing.Header_1.Header.ProposerIndex)
		if err != nil {
			log.WithError(err).Error("could not validate proposer slashing")
			continue
		}
		if !valid {
			p.pendingProposerSlashing = append(p.pendingProposerSlashing[:i], p.pendingProposerSlashing[i+1:]...)
			i--
			continue
		}

		pending = append(pending, slashing)
	}
	return pending
}

// InsertAttesterSlashing into the pool. This method is a no-op if the attester slashing already exists in the pool,
// has been included into a block recently, or the validator is already exited.
func (p *Pool) InsertAttesterSlashing(
	ctx context.Context,
	state *beaconstate.BeaconState,
	slashing *ethpb.AttesterSlashing,
) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	ctx, span := trace.StartSpan(ctx, "operations.InsertAttesterSlashing")
	defer span.End()

	if err := blocks.VerifyAttesterSlashing(ctx, state, slashing); err != nil {
		numPendingAttesterSlashingFailedSigVerify.Inc()
		return errors.Wrap(err, "could not verify attester slashing")
	}

	slashedVal := sliceutil.IntersectionUint64(slashing.Attestation_1.AttestingIndices, slashing.Attestation_2.AttestingIndices)
	cantSlash := make([]uint64, 0, len(slashedVal))
	for _, val := range slashedVal {
		// Has this validator index been included recently?
		ok, err := p.validatorSlashingPreconditionCheck(state, val)
		if err != nil {
			return err
		}
		// If the validator has already exited, has already been slashed, or if its index
		// has been recently included in the pool of slashings, skip including this indice.
		if !ok {
			attesterSlashingReattempts.Inc()
			cantSlash = append(cantSlash, val)
			continue
		}

		// Check if the validator already exists in the list of slashings.
		// Use binary search to find the answer.
		found := sort.Search(len(p.pendingAttesterSlashing), func(i int) bool {
			return p.pendingAttesterSlashing[i].validatorToSlash >= val
		})
		if found != len(p.pendingAttesterSlashing) && p.pendingAttesterSlashing[found].validatorToSlash == val {
			attesterSlashingReattempts.Inc()
			cantSlash = append(cantSlash, val)
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
		numPendingAttesterSlashings.Set(float64(len(p.pendingAttesterSlashing)))
	}
	if len(cantSlash) == len(slashedVal) {
		return fmt.Errorf("could not slash any of %d validators in submitted slashing", len(slashedVal))
	}
	return nil
}

// InsertProposerSlashing into the pool. This method is a no-op if the pending slashing already exists,
// has been included recently, the validator is already exited, or the validator was already slashed.
func (p *Pool) InsertProposerSlashing(
	ctx context.Context,
	state *beaconstate.BeaconState,
	slashing *ethpb.ProposerSlashing,
) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	ctx, span := trace.StartSpan(ctx, "operations.InsertProposerSlashing")
	defer span.End()

	if err := blocks.VerifyProposerSlashing(state, slashing); err != nil {
		numPendingProposerSlashingFailedSigVerify.Inc()
		return errors.Wrap(err, "could not verify proposer slashing")
	}

	idx := slashing.Header_1.Header.ProposerIndex
	ok, err := p.validatorSlashingPreconditionCheck(state, idx)
	if err != nil {
		return err
	}
	// If the validator has already exited, has already been slashed, or if its index
	// has been recently included in the pool of slashings, do not process this new
	// slashing.
	if !ok {
		proposerSlashingReattempts.Inc()
		return fmt.Errorf("validator at index %d cannot be slashed", idx)
	}

	// Check if the validator already exists in the list of slashings.
	// Use binary search to find the answer.
	found := sort.Search(len(p.pendingProposerSlashing), func(i int) bool {
		return p.pendingProposerSlashing[i].Header_1.Header.ProposerIndex >= slashing.Header_1.Header.ProposerIndex
	})
	if found != len(p.pendingProposerSlashing) && p.pendingProposerSlashing[found].Header_1.Header.ProposerIndex ==
		slashing.Header_1.Header.ProposerIndex {
		return errors.New("slashing object already exists in pending proposer slashings")
	}

	// Insert into pending list and sort again.
	p.pendingProposerSlashing = append(p.pendingProposerSlashing, slashing)
	sort.Slice(p.pendingProposerSlashing, func(i, j int) bool {
		return p.pendingProposerSlashing[i].Header_1.Header.ProposerIndex < p.pendingProposerSlashing[j].Header_1.Header.ProposerIndex
	})
	numPendingProposerSlashings.Set(float64(len(p.pendingProposerSlashing)))

	return nil
}

// MarkIncludedAttesterSlashing is used when an attester slashing has been included in a beacon block.
// Every block seen by this node that contains proposer slashings should call this method to include
// the proposer slashings.
func (p *Pool) MarkIncludedAttesterSlashing(as *ethpb.AttesterSlashing) {
	p.lock.Lock()
	defer p.lock.Unlock()
	slashedVal := sliceutil.IntersectionUint64(as.Attestation_1.AttestingIndices, as.Attestation_2.AttestingIndices)
	for _, val := range slashedVal {
		i := sort.Search(len(p.pendingAttesterSlashing), func(i int) bool {
			return p.pendingAttesterSlashing[i].validatorToSlash >= val
		})
		if i != len(p.pendingAttesterSlashing) && p.pendingAttesterSlashing[i].validatorToSlash == val {
			p.pendingAttesterSlashing = append(p.pendingAttesterSlashing[:i], p.pendingAttesterSlashing[i+1:]...)
		}
		p.included[val] = true
		numAttesterSlashingsIncluded.Inc()
	}
}

// MarkIncludedProposerSlashing is used when an proposer slashing has been included in a beacon block.
// Every block seen by this node that contains proposer slashings should call this method to include
// the proposer slashings.
func (p *Pool) MarkIncludedProposerSlashing(ps *ethpb.ProposerSlashing) {
	p.lock.Lock()
	defer p.lock.Unlock()
	i := sort.Search(len(p.pendingProposerSlashing), func(i int) bool {
		return p.pendingProposerSlashing[i].Header_1.Header.ProposerIndex >= ps.Header_1.Header.ProposerIndex
	})
	if i != len(p.pendingProposerSlashing) && p.pendingProposerSlashing[i].Header_1.Header.ProposerIndex == ps.Header_1.Header.ProposerIndex {
		p.pendingProposerSlashing = append(p.pendingProposerSlashing[:i], p.pendingProposerSlashing[i+1:]...)
	}
	p.included[ps.Header_1.Header.ProposerIndex] = true
	numProposerSlashingsIncluded.Inc()
}

// this function checks a few items about a validator before proceeding with inserting
// a proposer/attester slashing into the pool. First, it checks if the validator
// has been recently included in the pool, then it checks if the validator is slashable.
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
	// Checking if the validator is slashable.
	if !helpers.IsSlashableValidatorUsingTrie(validator, helpers.CurrentEpoch(state)) {
		return false, nil
	}
	return true, nil
}
