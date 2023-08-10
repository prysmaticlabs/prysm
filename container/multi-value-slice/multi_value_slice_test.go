package mvslice

import (
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

var (
	id1   = uuid.New()
	id2   = uuid.New()
	id999 = uuid.New()
)

type testObject struct {
	id uuid.UUID
}

func (o *testObject) Id() uuid.UUID {
	return o.id
}

func (o *testObject) SetId(id uuid.UUID) {
	o.id = id
}

func TestLen(t *testing.T) {
	s := &Slice[int, *testObject]{}
	s.Init([]int{1, 2, 3})
	id := uuid.New()
	s.cachedLengths[id] = 123
	t.Run("cached", func(t *testing.T) {
		assert.Equal(t, 123, s.Len(&testObject{id: id}))
	})
	t.Run("not cached", func(t *testing.T) {
		assert.Equal(t, 3, s.Len(&testObject{id: uuid.New()}))
	})
}

func TestCopy(t *testing.T) {
	// What we want to check:
	// - shared value is copied
	// - when the source object has an individual value, it is copied
	// - when the source object does not have an individual value, the shared value is copied
	// - when the source object has an appended value, it is copied
	// - when the source object does not have an appended value, nothing is copied
	// - length of destination object is cached

	s := setup()
	src := &testObject{id: id1}
	dst := &testObject{id: id999}

	s.Copy(src, dst)

	assert.Equal(t, (*MultiValueItem[int])(nil), s.individualItems[0])
	assertIndividualFound(t, s, dst.id, 1, 1)
	assertIndividualFound(t, s, dst.id, 2, 3)
	assertIndividualFound(t, s, dst.id, 3, 1)
	assertIndividualNotFound(t, s, dst.id, 4)
	assertAppendedFound(t, s, dst.id, 0, 1)
	assertAppendedFound(t, s, dst.id, 1, 3)
	assertAppendedNotFound(t, s, dst.id, 2)
	l, ok := s.cachedLengths[id999]
	require.Equal(t, true, ok)
	assert.Equal(t, 7, l)
}

func TestValue(t *testing.T) {
	// What we want to check:
	// - correct values are returned for first object
	// - correct values are returned for second object
	// - correct values are returned for an object without appended items

	s := setup()
	first := &testObject{id: id1}
	second := &testObject{id: id2}

	v := s.Value(first)

	require.Equal(t, 7, len(v))
	assert.Equal(t, 123, v[0])
	assert.Equal(t, 1, v[1])
	assert.Equal(t, 3, v[2])
	assert.Equal(t, 1, v[3])
	assert.Equal(t, 123, v[4])
	assert.Equal(t, 1, v[5])
	assert.Equal(t, 3, v[6])

	v = s.Value(second)

	require.Equal(t, 8, len(v))
	assert.Equal(t, 123, v[0])
	assert.Equal(t, 2, v[1])
	assert.Equal(t, 3, v[2])
	assert.Equal(t, 123, v[3])
	assert.Equal(t, 2, v[4])
	assert.Equal(t, 2, v[5])
	assert.Equal(t, 3, v[6])
	assert.Equal(t, 2, v[7])

	s = &Slice[int, *testObject]{}
	s.Init([]int{1, 2, 3})
	id := uuid.New()

	v = s.Value(&testObject{id: id})

	require.Equal(t, 3, len(v))
	assert.Equal(t, 1, v[0])
	assert.Equal(t, 2, v[1])
	assert.Equal(t, 3, v[2])
}

func TestAt(t *testing.T) {
	// What we want to check:
	// - correct values are returned for first object
	// - correct values are returned for second object
	// - ERROR when index too large in general
	// - ERROR when index not too large in general, but too large for an object

	s := setup()
	first := &testObject{id: id1}
	second := &testObject{id: id2}

	v, err := s.At(first, 0)
	require.NoError(t, err)
	assert.Equal(t, 123, v)
	v, err = s.At(first, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, v)
	v, err = s.At(first, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, v)
	v, err = s.At(first, 3)
	require.NoError(t, err)
	assert.Equal(t, 1, v)
	v, err = s.At(first, 4)
	require.NoError(t, err)
	assert.Equal(t, 123, v)
	v, err = s.At(first, 5)
	require.NoError(t, err)
	assert.Equal(t, 1, v)
	v, err = s.At(first, 6)
	require.NoError(t, err)
	assert.Equal(t, 3, v)
	_, err = s.At(first, 7)
	assert.ErrorContains(t, "index 7 out of bounds", err)

	v, err = s.At(second, 0)
	require.NoError(t, err)
	assert.Equal(t, 123, v)
	v, err = s.At(second, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, v)
	v, err = s.At(second, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, v)
	v, err = s.At(second, 3)
	require.NoError(t, err)
	assert.Equal(t, 123, v)
	v, err = s.At(second, 4)
	require.NoError(t, err)
	assert.Equal(t, 2, v)
	v, err = s.At(second, 5)
	require.NoError(t, err)
	assert.Equal(t, 2, v)
	v, err = s.At(second, 6)
	require.NoError(t, err)
	assert.Equal(t, 3, v)
	v, err = s.At(second, 7)
	require.NoError(t, err)
	assert.Equal(t, 2, v)
	_, err = s.At(second, 8)
	assert.ErrorContains(t, "index 8 out of bounds", err)
}

func TestUpdateAt(t *testing.T) {
	// What we want to check:
	// - shared value is updated only for the updated object, creating a new individual value (shared value remains the same)
	// - individual value (different for both objects) is updated to a third value
	// - individual value (different for both objects) is updated to the other object's value
	// - individual value (equal for both objects) is updated
	// - individual value existing only for the updated object is updated
	// - individual value existing only for the other-object appends an item to the individual value
	// - individual value updated to the original shared value removes that individual value
	// - appended value (different for both objects) is updated to a third value
	// - appended value (different for both objects) is updated to the other object's value
	// - appended value (equal for both objects) is updated
	// - appended value existing for one object is updated
	// - ERROR when index too large in general
	// - ERROR when index not too large in general, but too large for an object

	s := setup()
	first := &testObject{id: id1}
	second := &testObject{id: id2}

	require.NoError(t, s.UpdateAt(first, 0, 999))
	assert.Equal(t, 123, s.sharedItems[0])
	assertIndividualFound(t, s, first.id, 0, 999)
	assertIndividualNotFound(t, s, second.id, 0)

	require.NoError(t, s.UpdateAt(first, 1, 999))
	assertIndividualFound(t, s, first.id, 1, 999)
	assertIndividualFound(t, s, second.id, 1, 2)

	require.NoError(t, s.UpdateAt(first, 1, 2))
	assertIndividualFound(t, s, first.id, 1, 2)
	assertIndividualFound(t, s, second.id, 1, 2)

	require.NoError(t, s.UpdateAt(first, 2, 999))
	assertIndividualFound(t, s, first.id, 2, 999)
	assertIndividualFound(t, s, second.id, 2, 3)

	require.NoError(t, s.UpdateAt(first, 3, 999))
	assertIndividualFound(t, s, first.id, 3, 999)
	assertIndividualNotFound(t, s, second.id, 3)

	require.NoError(t, s.UpdateAt(first, 4, 999))
	assertIndividualFound(t, s, first.id, 4, 999)
	assertIndividualFound(t, s, second.id, 4, 2)

	require.NoError(t, s.UpdateAt(first, 4, 123))
	assertIndividualNotFound(t, s, first.id, 4)
	assertIndividualFound(t, s, second.id, 4, 2)

	require.NoError(t, s.UpdateAt(first, 5, 999))
	assertAppendedFound(t, s, first.id, 0, 999)
	assertAppendedFound(t, s, second.id, 0, 2)

	require.NoError(t, s.UpdateAt(first, 5, 2))
	assertAppendedFound(t, s, first.id, 0, 2)
	assertAppendedFound(t, s, second.id, 0, 2)

	require.NoError(t, s.UpdateAt(first, 6, 999))
	assertAppendedFound(t, s, first.id, 1, 999)
	assertAppendedFound(t, s, second.id, 1, 3)

	// we update the second object because there are no more appended items for the first object
	require.NoError(t, s.UpdateAt(second, 7, 999))
	assertAppendedNotFound(t, s, first.id, 2)
	assertAppendedFound(t, s, second.id, 2, 999)

	assert.ErrorContains(t, "index 7 out of bounds", s.UpdateAt(first, 7, 999))
	assert.ErrorContains(t, "index 8 out of bounds", s.UpdateAt(second, 8, 999))
}

func TestAppend(t *testing.T) {
	// What we want to check:
	// - appending first item ever to the slice
	// - appending an item to an object when there is no corresponding item for the other object
	// - appending an item to an object when there is a corresponding item with same value for the other object
	// - appending an item to an object when there is a corresponding item with different value for the other object
	// - we also want to check that cached length is properly updated after every append

	// we want to start with the simplest slice possible
	s := &Slice[int, *testObject]{}
	s.Init([]int{0})
	first := &testObject{id: id1}
	second := &testObject{id: id2}

	// append first value ever
	s.Append(first, 1)
	require.Equal(t, 1, len(s.appendedItems))
	assertAppendedFound(t, s, first.id, 0, 1)
	assertAppendedNotFound(t, s, second.id, 0)
	l, ok := s.cachedLengths[first.id]
	require.Equal(t, true, ok)
	assert.Equal(t, 2, l)
	_, ok = s.cachedLengths[second.id]
	assert.Equal(t, false, ok)

	// append one more value to the first object, so that we can test two append scenarios for the second object
	s.Append(first, 1)

	// append the first value to the second object, equal to the value for the first object
	s.Append(second, 1)
	require.Equal(t, 2, len(s.appendedItems))
	assertAppendedFound(t, s, first.id, 0, 1)
	assertAppendedFound(t, s, second.id, 0, 1)
	l, ok = s.cachedLengths[first.id]
	require.Equal(t, true, ok)
	assert.Equal(t, 3, l)
	l, ok = s.cachedLengths[second.id]
	assert.Equal(t, true, ok)
	assert.Equal(t, 2, l)

	// append the first value to the second object, different than the value for the first object
	s.Append(second, 2)
	require.Equal(t, 2, len(s.appendedItems))
	assertAppendedFound(t, s, first.id, 1, 1)
	assertAppendedFound(t, s, second.id, 1, 2)
	l, ok = s.cachedLengths[first.id]
	require.Equal(t, true, ok)
	assert.Equal(t, 3, l)
	l, ok = s.cachedLengths[second.id]
	assert.Equal(t, true, ok)
	assert.Equal(t, 3, l)
}

func TestDetach(t *testing.T) {
	// What we want to check:
	// - no individual or appended items left after detaching an object
	// - length removed from cache

	s := setup()
	obj := &testObject{id: id1}

	s.Detach(obj)

	for _, item := range s.individualItems {
		found := false
		for _, v := range item.Values {
			for _, o := range v.ids {
				if o == obj.id {
					found = true
				}
			}
		}
		assert.Equal(t, false, found)
	}
	for _, item := range s.appendedItems {
		found := false
		for _, v := range item.Values {
			for _, o := range v.ids {
				if o == obj.id {
					found = true
				}
			}
		}
		assert.Equal(t, false, found)
	}
	_, ok := s.cachedLengths[obj.id]
	assert.Equal(t, false, ok)
}

// Share the slice between 2 objects.
// Index 0: Shared value
// Index 1: Different individual value
// Index 2: Same individual value
// Index 3: Individual value ONLY for the first object
// Index 4: Individual value ONLY for the second object
// Index 5: Different appended value
// Index 6: Same appended value
// Index 7: Appended value ONLY for the second object
func setup() *Slice[int, *testObject] {
	s := &Slice[int, *testObject]{}
	s.Init([]int{123, 123, 123, 123, 123})
	s.individualItems[1] = &MultiValueItem[int]{
		Values: []*Value[int]{
			{
				val: 1,
				ids: []uuid.UUID{id1},
			},
			{
				val: 2,
				ids: []uuid.UUID{id2},
			},
		},
	}
	s.individualItems[2] = &MultiValueItem[int]{
		Values: []*Value[int]{
			{
				val: 3,
				ids: []uuid.UUID{id1, id2},
			},
		},
	}
	s.individualItems[3] = &MultiValueItem[int]{
		Values: []*Value[int]{
			{
				val: 1,
				ids: []uuid.UUID{id1},
			},
		},
	}
	s.individualItems[4] = &MultiValueItem[int]{
		Values: []*Value[int]{
			{
				val: 2,
				ids: []uuid.UUID{id2},
			},
		},
	}
	s.appendedItems = []*MultiValueItem[int]{
		{
			Values: []*Value[int]{
				{
					val: 1,
					ids: []uuid.UUID{id1},
				},
				{
					val: 2,
					ids: []uuid.UUID{id2},
				},
			},
		},
		{
			Values: []*Value[int]{
				{
					val: 3,
					ids: []uuid.UUID{id1, id2},
				},
			},
		},
		{
			Values: []*Value[int]{
				{
					val: 2,
					ids: []uuid.UUID{id2},
				},
			},
		},
	}
	s.cachedLengths[id1] = 7
	s.cachedLengths[id2] = 8

	return s
}

func assertIndividualFound(t *testing.T, slice *Slice[int, *testObject], id uuid.UUID, itemIndex uint64, expected int) {
	found := false
	for _, v := range slice.individualItems[itemIndex].Values {
		for _, o := range v.ids {
			if o == id {
				found = true
				assert.Equal(t, expected, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
}

func assertIndividualNotFound(t *testing.T, slice *Slice[int, *testObject], id uuid.UUID, itemIndex uint64) {
	found := false
	for _, v := range slice.individualItems[itemIndex].Values {
		for _, o := range v.ids {
			if o == id {
				found = true
			}
		}
	}
	assert.Equal(t, false, found)
}

func assertAppendedFound(t *testing.T, slice *Slice[int, *testObject], id uuid.UUID, itemIndex uint64, expected int) {
	found := false
	for _, v := range slice.appendedItems[itemIndex].Values {
		for _, o := range v.ids {
			if o == id {
				found = true
				assert.Equal(t, expected, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
}

func assertAppendedNotFound(t *testing.T, slice *Slice[int, *testObject], id uuid.UUID, itemIndex uint64) {
	found := false
	for _, v := range slice.appendedItems[itemIndex].Values {
		for _, o := range v.ids {
			if o == id {
				found = true
			}
		}
	}
	assert.Equal(t, false, found)
}

func BenchmarkValue(b *testing.B) {
	const _100k = 100000
	const _1m = 1000000
	const _10m = 10000000

	b.Run("100,000 shared items", func(b *testing.B) {
		s := &Slice[int, *testObject]{}
		s.Init(make([]int, _100k))
		for i := 0; i < b.N; i++ {
			s.Value(&testObject{})
		}
	})
	b.Run("100,000 equal individual items", func(b *testing.B) {
		s := &Slice[int, *testObject]{}
		s.Init(make([]int, _100k))
		s.individualItems[0] = &MultiValueItem[int]{Values: []*Value[int]{{val: 999, ids: []uuid.UUID{}}}}
		objs := make([]*testObject, _100k)
		for i := 0; i < len(objs); i++ {
			id := uuid.New()
			objs[i] = &testObject{id: id}
			s.individualItems[0].Values[0].ids = append(s.individualItems[0].Values[0].ids, id)
		}
		for i := 0; i < b.N; i++ {
			s.Value(objs[rand.Intn(_100k)])
		}
	})
	b.Run("100,000 different individual items", func(b *testing.B) {
		s := &Slice[int, *testObject]{}
		s.Init(make([]int, _100k))
		objs := make([]*testObject, _100k)
		for i := 0; i < len(objs); i++ {
			id := uuid.New()
			objs[i] = &testObject{id: id}
			s.individualItems[uint64(i)] = &MultiValueItem[int]{Values: []*Value[int]{{val: i, ids: []uuid.UUID{id}}}}
		}
		for i := 0; i < b.N; i++ {
			s.Value(objs[rand.Intn(_100k)])
		}
	})
	b.Run("100,000 shared items and 100,000 equal appended items", func(b *testing.B) {
		s := &Slice[int, *testObject]{}
		s.Init(make([]int, _100k))
		s.appendedItems = []*MultiValueItem[int]{{Values: []*Value[int]{{val: 999, ids: []uuid.UUID{}}}}}
		objs := make([]*testObject, _100k)
		for i := 0; i < len(objs); i++ {
			id := uuid.New()
			objs[i] = &testObject{id: id}
			s.appendedItems[0].Values[0].ids = append(s.appendedItems[0].Values[0].ids, id)
		}
		for i := 0; i < b.N; i++ {
			s.Value(objs[rand.Intn(_100k)])
		}
	})
	b.Run("100,000 shared items and 100,000 different appended items", func(b *testing.B) {
		s := &Slice[int, *testObject]{}
		s.Init(make([]int, _100k))
		s.appendedItems = []*MultiValueItem[int]{}
		objs := make([]*testObject, _100k)
		for i := 0; i < len(objs); i++ {
			id := uuid.New()
			objs[i] = &testObject{id: id}
			s.appendedItems = append(s.appendedItems, &MultiValueItem[int]{Values: []*Value[int]{{val: i, ids: []uuid.UUID{id}}}})
		}
		for i := 0; i < b.N; i++ {
			s.Value(objs[rand.Intn(_100k)])
		}
	})
	b.Run("1,000,000 shared items", func(b *testing.B) {
		s := &Slice[int, *testObject]{}
		s.Init(make([]int, _1m))
		for i := 0; i < b.N; i++ {
			s.Value(&testObject{})
		}
	})
	b.Run("1,000,000 equal individual items", func(b *testing.B) {
		s := &Slice[int, *testObject]{}
		s.Init(make([]int, _1m))
		s.individualItems[0] = &MultiValueItem[int]{Values: []*Value[int]{{val: 999, ids: []uuid.UUID{}}}}
		objs := make([]*testObject, _1m)
		for i := 0; i < len(objs); i++ {
			id := uuid.New()
			objs[i] = &testObject{id: id}
			s.individualItems[0].Values[0].ids = append(s.individualItems[0].Values[0].ids, id)
		}
		for i := 0; i < b.N; i++ {
			s.Value(objs[rand.Intn(_1m)])
		}
	})
	b.Run("1,000,000 different individual items", func(b *testing.B) {
		s := &Slice[int, *testObject]{}
		s.Init(make([]int, _1m))
		objs := make([]*testObject, _1m)
		for i := 0; i < len(objs); i++ {
			id := uuid.New()
			objs[i] = &testObject{id: id}
			s.individualItems[uint64(i)] = &MultiValueItem[int]{Values: []*Value[int]{{val: i, ids: []uuid.UUID{id}}}}
		}
		for i := 0; i < b.N; i++ {
			s.Value(objs[rand.Intn(_1m)])
		}
	})
	b.Run("1,000,000 shared items and 1,000,000 equal appended items", func(b *testing.B) {
		s := &Slice[int, *testObject]{}
		s.Init(make([]int, _1m))
		s.appendedItems = []*MultiValueItem[int]{{Values: []*Value[int]{{val: 999, ids: []uuid.UUID{}}}}}
		objs := make([]*testObject, _1m)
		for i := 0; i < len(objs); i++ {
			id := uuid.New()
			objs[i] = &testObject{id: id}
			s.appendedItems[0].Values[0].ids = append(s.appendedItems[0].Values[0].ids, id)
		}
		for i := 0; i < b.N; i++ {
			s.Value(objs[rand.Intn(_1m)])
		}
	})
	b.Run("1,000,000 shared items and 1,000,000 different appended items", func(b *testing.B) {
		s := &Slice[int, *testObject]{}
		s.Init(make([]int, _1m))
		s.appendedItems = []*MultiValueItem[int]{}
		objs := make([]*testObject, _1m)
		for i := 0; i < len(objs); i++ {
			id := uuid.New()
			objs[i] = &testObject{id: id}
			s.appendedItems = append(s.appendedItems, &MultiValueItem[int]{Values: []*Value[int]{{val: i, ids: []uuid.UUID{id}}}})
		}
		for i := 0; i < b.N; i++ {
			s.Value(objs[rand.Intn(_1m)])
		}
	})
	b.Run("10,000,000 shared items", func(b *testing.B) {
		s := &Slice[int, *testObject]{}
		s.Init(make([]int, _10m))
		for i := 0; i < b.N; i++ {
			s.Value(&testObject{})
		}
	})
	b.Run("10,000,000 equal individual items", func(b *testing.B) {
		s := &Slice[int, *testObject]{}
		s.Init(make([]int, _10m))
		s.individualItems[0] = &MultiValueItem[int]{Values: []*Value[int]{{val: 999, ids: []uuid.UUID{}}}}
		objs := make([]*testObject, _10m)
		for i := 0; i < len(objs); i++ {
			id := uuid.New()
			objs[i] = &testObject{id: id}
			s.individualItems[0].Values[0].ids = append(s.individualItems[0].Values[0].ids, id)
		}
		for i := 0; i < b.N; i++ {
			s.Value(objs[rand.Intn(_10m)])
		}
	})
	b.Run("10,000,000 different individual items", func(b *testing.B) {
		s := &Slice[int, *testObject]{}
		s.Init(make([]int, _10m))
		objs := make([]*testObject, _10m)
		for i := 0; i < len(objs); i++ {
			id := uuid.New()
			objs[i] = &testObject{id: id}
			s.individualItems[uint64(i)] = &MultiValueItem[int]{Values: []*Value[int]{{val: i, ids: []uuid.UUID{id}}}}
		}
		for i := 0; i < b.N; i++ {
			s.Value(objs[rand.Intn(_10m)])
		}
	})
	b.Run("10,000,000 shared items and 10,000,000 equal appended items", func(b *testing.B) {
		s := &Slice[int, *testObject]{}
		s.Init(make([]int, _10m))
		s.appendedItems = []*MultiValueItem[int]{{Values: []*Value[int]{{val: 999, ids: []uuid.UUID{}}}}}
		objs := make([]*testObject, _10m)
		for i := 0; i < len(objs); i++ {
			id := uuid.New()
			objs[i] = &testObject{id: id}
			s.appendedItems[0].Values[0].ids = append(s.appendedItems[0].Values[0].ids, id)
		}
		for i := 0; i < b.N; i++ {
			s.Value(objs[rand.Intn(_10m)])
		}
	})
	b.Run("10,000,000 shared items and 10,000,000 different appended items", func(b *testing.B) {
		s := &Slice[int, *testObject]{}
		s.Init(make([]int, _10m))
		s.appendedItems = []*MultiValueItem[int]{}
		objs := make([]*testObject, _10m)
		for i := 0; i < len(objs); i++ {
			id := uuid.New()
			objs[i] = &testObject{id: id}
			s.appendedItems = append(s.appendedItems, &MultiValueItem[int]{Values: []*Value[int]{{val: i, ids: []uuid.UUID{id}}}})
		}
		for i := 0; i < b.N; i++ {
			s.Value(objs[rand.Intn(_10m)])
		}
	})
}

func Test_FuzzAt(t *testing.T) {
	objectIds := make([]uuid.UUID, 100)
	for i := range objectIds {
		objectIds[i] = uuid.New()
	}

	for i := 0; i < 100000; i++ {
		s := randomSlice(objectIds)
		sliceIndex := uint64(rand.Intn(len(s.sharedItems)))
		objIndex := rand.Intn(len(objectIds))

		var expected int
		individual, ok := s.individualItems[sliceIndex]
		if ok {
			found := false
			for _, v := range individual.Values {
				for _, id := range v.ids {
					if id == objectIds[objIndex] {
						expected = v.val
						found = true
					}
				}
			}
			if !found {
				expected = s.sharedItems[int(sliceIndex)]
			}
		} else {
			expected = s.sharedItems[int(sliceIndex)]
		}

		actual, err := s.At(&testObject{objectIds[objIndex]}, sliceIndex)
		require.NoError(t, err)
		if expected != actual {
			t.Fail()
		}
	}
}

func randomSlice(objectIds []uuid.UUID) *Slice[int, *testObject] {
	s := &Slice[int, *testObject]{}

	nShared := rand.Intn(99) + 1 // 1 to 100
	shared := make([]int, nShared)
	for i := 0; i < nShared; i++ {
		shared[i] = rand.Intn(20)
	}
	individual := make(map[uint64]*MultiValueItem[int])
	for index := uint64(0); index < uint64(nShared); index++ {
		insertIndividual := rand.Intn(100)%2 == 0
		if !insertIndividual {
			continue
		}

		nValues := rand.Intn(10)
		for i := 0; i < nValues; i++ {
			mv, ok := individual[index]
			if !ok {
				individual[index] = &MultiValueItem[int]{}
				mv = individual[index]
			}
			v, id := rand.Intn(20), rand.Intn(100)
			if v == shared[index] {
				break
			}
			valueFound := false
			for j := 0; j < len(mv.Values); j++ {
				if v == mv.Values[j].val {
					idFound := false
					ids := mv.Values[j].ids
					for k := 0; k < len(ids); k++ {
						if objectIds[id] == ids[k] {
							idFound = true
							break
						}
					}
					if !idFound {
						mv.Values[j].ids = append(ids, objectIds[id])
					}
					valueFound = true
					break
				}
			}
			if !valueFound {
				mv.Values = append(mv.Values, &Value[int]{val: v, ids: []uuid.UUID{objectIds[id]}})
			}
		}
	}

	s.sharedItems = shared
	s.individualItems = individual

	return s
}
