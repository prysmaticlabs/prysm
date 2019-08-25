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

// FilterType defines an enum which is used as the keys in a map that tracks
// set attribute filters for data as part of the `FilterQuery` struct type.
type FilterType int

const (
	// ParentRoot defines a filter for parent roots of blocks using Simple Serialize (SSZ).
	ParentRoot FilterType = 0
	// StartSlot is used for range filters of objects by their slot (inclusive).
	StartSlot FilterType = 1
	// EndSlot is used for range filters of objects by their slot (inclusive).
	EndSlot FilterType = 2
	// StartEpoch is used for range filters of objects by their epoch (inclusive).
	StartEpoch FilterType = 3
	// EndEpoch is used for range filters of objects by their epoch (inclusive).
	EndEpoch FilterType = 4
	// Shard is used for filtering data by shard index.
	Shard FilterType = 5
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

// SetParentRoot allows for filtering by the parent root data attribute of an object.
func (q *QueryFilter) SetParentRoot(val []byte) *QueryFilter {
	q.queries[ParentRoot] = val
	return q
}

// SetStartSlot enables filtering by all the items that begin at a slot (inclusive).
func (q *QueryFilter) SetStartSlot(val uint64) *QueryFilter {
	q.queries[StartSlot] = val
	return q
}

// SetEndSlot enables filtering by all the items that end at a slot (inclusive).
func (q *QueryFilter) SetEndSlot(val uint64) *QueryFilter {
	q.queries[EndSlot] = val
	return q
}

// SetStartEpoch enables filtering by the StartEpoch attribute of an object (inclusive).
func (q *QueryFilter) SetStartEpoch(val uint64) *QueryFilter {
	q.queries[StartEpoch] = val
	return q
}

// SetEndEpoch enables filtering by the EndEpoch attribute of an object (inclusive).
func (q *QueryFilter) SetEndEpoch(val uint64) *QueryFilter {
	q.queries[EndEpoch] = val
	return q
}

// SetShard enabled filtering by the Shard attribute of an object.
func (q *QueryFilter) SetShard(val uint64) *QueryFilter {
	q.queries[Shard] = val
	return q
}
