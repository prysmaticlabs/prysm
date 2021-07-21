package attestations

import ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"

// NaiveAttestationAggregation aggregates naively, without any complex algorithms or optimizations.
// Note: this is currently a naive implementation to the order of O(mn^2).
func NaiveAttestationAggregation(atts []*ethpb.Attestation) ([]*ethpb.Attestation, error) {
	if len(atts) <= 1 {
		return atts, nil
	}

	// Naive aggregation. O(n^2) time.
	for i, a := range atts {
		if i >= len(atts) {
			break
		}
		for j := i + 1; j < len(atts); j++ {
			b := atts[j]
			if o, err := a.AggregationBits.Overlaps(b.AggregationBits); err != nil {
				return nil, err
			} else if !o {
				var err error
				a, err = AggregatePair(a, b)
				if err != nil {
					return nil, err
				}
				// Delete b
				atts = append(atts[:j], atts[j+1:]...)
				j--
				atts[i] = a
			}
		}
	}

	// Naive deduplication of identical aggregations. O(n^2) time.
	for i, a := range atts {
		for j := i + 1; j < len(atts); j++ {
			b := atts[j]

			if a.AggregationBits.Len() != b.AggregationBits.Len() {
				continue
			}

			if c, err := a.AggregationBits.Contains(b.AggregationBits); err != nil {
				return nil, err
			} else if c {
				// If b is fully contained in a, then b can be removed.
				atts = append(atts[:j], atts[j+1:]...)
				j--
			} else if c, err := b.AggregationBits.Contains(a.AggregationBits); err != nil {
				return nil, err
			} else if c {
				// if a is fully contained in b, then a can be removed.
				atts = append(atts[:i], atts[i+1:]...)
				break // Stop the inner loop, advance a.
			}
		}
	}

	return atts, nil
}
