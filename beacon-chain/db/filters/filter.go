// Package filters specifies utilities for building a set of data attribute
// filters to be used  when filtering data through database queries in practice.
// For example, one can specify a filter query for data by start epoch + end epoch + shard
// for attestations, build a filter as follows, and respond to it accordingly:
//
//   f := filters.NewFilter().SetStartEpoch(3).SetEndEpoch(5).SetShard(5)
//   for k, v := range f.Filters() {
//       switch k {
//       case filters.StartEpoch:
//          // Verify data matches filter criteria...
//       case filters.EndEpoch:
//          // Verify data matches filter criteria...
//       case filters.Shard:
//          // Verify data matches filter criteria...
//       }
//   }
package filters

type FilterType int

const (
	Root       FilterType = 0
	ParentRoot FilterType = 1
	StartSlot  FilterType = 2
	EndSlot    FilterType = 3
	StartEpoch FilterType = 4
	EndEpoch   FilterType = 5
	Shard      FilterType = 6
)

// QueryFilter defines a generic interface for type-asserting
// specific filters to use in querying DB objects.
type QueryFilter struct {
	queries map[FilterType]interface{}
}

// NewFilter instantiates a new QueryFilter type used to build filters for
// certain eth2 data types by attribute.
func NewFilter() *QueryFilter {
	return &QueryFilter{
		queries: make(map[FilterType]interface{}),
	}
}

// Filters returns and underlying map of FilterType to interface{}, giving us
// a copy of the currently set filters which can then be iterated over and type
// asserted for use anywhere.
func (q *QueryFilter) Filters() map[FilterType]interface{} {
	return q.queries
}

// SetRoot --
func (q *QueryFilter) SetRoot(val [32]byte) *QueryFilter {
	q.queries[Root] = val
	return q
}

// SetParentRoot --
func (q *QueryFilter) SetParentRoot(val [32]byte) *QueryFilter {
	q.queries[ParentRoot] = val
	return q
}

// SetStartSlot --
func (q *QueryFilter) SetStartSlot(val uint64) *QueryFilter {
	q.queries[StartSlot] = val
	return q
}

// SetEndSlot --
func (q *QueryFilter) SetEndSlot(val uint64) *QueryFilter {
	q.queries[EndSlot] = val
	return q
}

// SetStartEpoch --
func (q *QueryFilter) SetStartEpoch(val uint64) *QueryFilter {
	q.queries[StartEpoch] = val
	return q
}

// SetEndEpoch --
func (q *QueryFilter) SetEndEpoch(val uint64) *QueryFilter {
	q.queries[EndEpoch] = val
	return q
}

// SetShard --
func (q *QueryFilter) SetShard(val uint64) *QueryFilter {
	q.queries[Shard] = val
	return q
}
