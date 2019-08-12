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

func NewFilter() *QueryFilter {
	return &QueryFilter{
		queries: make(map[FilterType]interface{}),
	}
}

func (q *QueryFilter) Filters() map[FilterType]interface{} {
	return q.queries
}

func (q *QueryFilter) SetRoot(val [32]byte) *QueryFilter {
	q.queries[Root] = val
	return q
}

func (q *QueryFilter) SetParentRoot(val [32]byte) *QueryFilter {
	q.queries[ParentRoot] = val
	return q
}

func (q *QueryFilter) SetStartSlot(val uint64) *QueryFilter {
	q.queries[StartSlot] = val
	return q
}

func (q *QueryFilter) SetEndSlot(val uint64) *QueryFilter {
	q.queries[EndSlot] = val
	return q
}

func (q *QueryFilter) SetStartEpoch(val uint64) *QueryFilter {
	q.queries[StartEpoch] = val
	return q
}

func (q *QueryFilter) SetEndEpoch(val uint64) *QueryFilter {
	q.queries[EndEpoch] = val
	return q
}
