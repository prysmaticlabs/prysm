package kv

import (
	"context"
	"runtime"
	"sync"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	attaggregation "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/attestation/aggregation/attestations"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// AggregateUnaggregatedAttestations aggregates the unaggregated attestations and saves the
// newly aggregated attestations in the pool.
// It tracks the unaggregated attestations that weren't able to aggregate to prevent
// the deletion of unaggregated attestations in the pool.
func (c *AttCaches) AggregateUnaggregatedAttestations(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "operations.attestations.kv.AggregateUnaggregatedAttestations")
	defer span.End()
	unaggregatedAtts, err := c.UnaggregatedAttestations()
	if err != nil {
		return err
	}
	return c.aggregateUnaggregatedAtts(ctx, unaggregatedAtts)
}

func (c *AttCaches) aggregateUnaggregatedAtts(ctx context.Context, unaggregatedAtts []blocks.ROAttestation) error {
	_, span := trace.StartSpan(ctx, "operations.attestations.kv.aggregateUnaggregatedAtts")
	defer span.End()

	attsByDataId := make(map[blocks.AttestationId][]blocks.ROAttestation, len(unaggregatedAtts))
	for _, att := range unaggregatedAtts {
		attsByDataId[att.DataId()] = append(attsByDataId[att.DataId()], att)
	}

	// Aggregate unaggregated attestations from the pool and save them in the pool.
	// Track the unaggregated attestations that aren't able to aggregate.
	leftOverUnaggregatedAtt := make(map[blocks.AttestationId]bool)

	leftOverUnaggregatedAtt = c.aggregateParallel(attsByDataId, leftOverUnaggregatedAtt)

	// Remove the unaggregated attestations from the pool that were successfully aggregated.
	for _, att := range unaggregatedAtts {
		if leftOverUnaggregatedAtt[att.Id()] {
			continue
		}
		if err := c.DeleteUnaggregatedAttestation(att); err != nil {
			return err
		}
	}
	return nil
}

// aggregateParallel aggregates attestations in parallel for `atts` and saves them in the pool,
// returns the unaggregated attestations that weren't able to aggregate.
// Given `n` CPU cores, it creates a channel of size `n` and spawns `n` goroutines to aggregate attestations
func (c *AttCaches) aggregateParallel(
	atts map[blocks.AttestationId][]blocks.ROAttestation,
	leftOver map[blocks.AttestationId]bool,
) map[blocks.AttestationId]bool {
	var leftoverLock sync.Mutex
	wg := sync.WaitGroup{}

	n := runtime.GOMAXPROCS(0) // defaults to the value of runtime.NumCPU
	ch := make(chan []blocks.ROAttestation, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			for as := range ch {
				aggregated, err := attaggregation.AggregateDisjointOneBitAtts(as)
				if err != nil {
					log.WithError(err).Error("could not aggregate unaggregated attestations")
					continue
				}
				if helpers.IsAggregated(aggregated.Att) {
					if err := c.SaveAggregatedAttestation(aggregated); err != nil {
						log.WithError(err).Error("could not save aggregated attestation")
						continue
					}
				} else {
					leftoverLock.Lock()
					leftOver[aggregated.Id()] = true
					leftoverLock.Unlock()
				}
			}
		}()
	}

	for _, as := range atts {
		ch <- as
	}

	close(ch)
	wg.Wait()

	return leftOver
}

// SaveAggregatedAttestation saves an aggregated attestation in cache.
func (c *AttCaches) SaveAggregatedAttestation(att blocks.ROAttestation) error {
	if err := helpers.ValidateNilAttestation(att.Att); err != nil {
		return err
	}
	if !helpers.IsAggregated(att.Att) {
		return errors.New("attestation is not aggregated")
	}
	has, err := c.HasAggregatedAttestation(att)
	if err != nil {
		return err
	}
	if has {
		return nil
	}

	seen, err := c.hasSeenBit(att)
	if err != nil {
		return err
	}
	if seen {
		return nil
	}

	copiedAtt := att.Copy()
	c.aggregatedAttLock.Lock()
	defer c.aggregatedAttLock.Unlock()
	roAtts, ok := c.aggregatedAtt[att.DataId()]
	if !ok {
		atts := []blocks.ROAttestation{copiedAtt}
		c.aggregatedAtt[att.DataId()] = atts
		return nil
	}

	roAtts, err = attaggregation.Aggregate(append(roAtts, copiedAtt))
	if err != nil {
		return err
	}
	c.aggregatedAtt[att.DataId()] = roAtts

	return nil
}

// SaveAggregatedAttestations saves a list of aggregated attestations in cache.
func (c *AttCaches) SaveAggregatedAttestations(atts []blocks.ROAttestation) error {
	for _, att := range atts {
		if err := c.SaveAggregatedAttestation(att); err != nil {
			log.WithError(err).Debug("Could not save aggregated attestation")
			if err := c.DeleteAggregatedAttestation(att); err != nil {
				log.WithError(err).Debug("Could not delete aggregated attestation")
			}
		}
	}
	return nil
}

// AggregatedAttestations returns the aggregated attestations in cache.
func (c *AttCaches) AggregatedAttestations() []blocks.ROAttestation {
	c.aggregatedAttLock.RLock()
	defer c.aggregatedAttLock.RUnlock()

	atts := make([]blocks.ROAttestation, 0)

	for _, a := range c.aggregatedAtt {
		atts = append(atts, a...)
	}

	return atts
}

// AggregatedAttestationsBySlotIndex returns the aggregated attestations in cache,
// filtered by committee index and slot.
func (c *AttCaches) AggregatedAttestationsBySlotIndex(
	ctx context.Context,
	slot primitives.Slot,
	committeeIndex primitives.CommitteeIndex,
) []ethpb.Att {
	_, span := trace.StartSpan(ctx, "operations.attestations.kv.AggregatedAttestationsBySlotIndex")
	defer span.End()

	atts := make([]ethpb.Att, 0)

	c.aggregatedAttLock.RLock()
	defer c.aggregatedAttLock.RUnlock()
	for _, as := range c.aggregatedAtt {
		if slot == as[0].GetData().Slot && committeeIndex == as[0].GetData().CommitteeIndex {
			for _, a := range as {
				att, ok := a.Att.(*ethpb.Attestation)
				if ok {
					atts = append(atts, att)
				}
			}
		}
	}

	return atts
}

// DeleteAggregatedAttestation deletes the aggregated attestations in cache.
func (c *AttCaches) DeleteAggregatedAttestation(att blocks.ROAttestation) error {
	if err := helpers.ValidateNilAttestation(att.Att); err != nil {
		return err
	}
	if !helpers.IsAggregated(att.Att) {
		return errors.New("attestation is not aggregated")
	}

	if err := c.insertSeenBit(att); err != nil {
		return err
	}

	c.aggregatedAttLock.Lock()
	defer c.aggregatedAttLock.Unlock()
	attList, ok := c.aggregatedAtt[att.DataId()]
	if !ok {
		return nil
	}

	filtered := make([]blocks.ROAttestation, 0)
	for _, a := range attList {
		if c, err := att.GetAggregationBits().Contains(a.GetAggregationBits()); err != nil {
			return err
		} else if !c {
			filtered = append(filtered, a)
		}
	}
	if len(filtered) == 0 {
		delete(c.aggregatedAtt, att.DataId())
	} else {
		c.aggregatedAtt[att.DataId()] = filtered
	}

	return nil
}

// HasAggregatedAttestation checks if the input attestations has already existed in cache.
func (c *AttCaches) HasAggregatedAttestation(att blocks.ROAttestation) (bool, error) {
	if err := helpers.ValidateNilAttestation(att.Att); err != nil {
		return false, err
	}

	c.aggregatedAttLock.RLock()
	defer c.aggregatedAttLock.RUnlock()
	if atts, ok := c.aggregatedAtt[att.DataId()]; ok {
		for _, a := range atts {
			if c, err := a.GetAggregationBits().Contains(att.GetAggregationBits()); err != nil {
				return false, err
			} else if c {
				return true, nil
			}
		}
	}

	c.blockAttLock.RLock()
	defer c.blockAttLock.RUnlock()
	if atts, ok := c.blockAtt[att.DataId()]; ok {
		for _, a := range atts {
			if c, err := a.GetAggregationBits().Contains(att.GetAggregationBits()); err != nil {
				return false, err
			} else if c {
				return true, nil
			}
		}
	}

	return false, nil
}

// AggregatedAttestationCount returns the number of aggregated attestations key in the pool.
func (c *AttCaches) AggregatedAttestationCount() int {
	c.aggregatedAttLock.RLock()
	defer c.aggregatedAttLock.RUnlock()
	return len(c.aggregatedAtt)
}
