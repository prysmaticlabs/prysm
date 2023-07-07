package multi_value_slice

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/container/multi-value-slice/interfaces"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

type testObject struct {
	id    interfaces.Id
	slice *Slice[int, *testObject]
}

func (o *testObject) Id() uint64 {
	return o.id
}

func (o *testObject) SetId(id uint64) {
	o.id = id
}

func TestLen(t *testing.T) {
	s := &Slice[int, *testObject]{}
	s.Init([]int{1, 2, 3})
	s.cachedLengths[0] = 123
	t.Run("cached", func(t *testing.T) {
		assert.Equal(t, 123, s.Len(&testObject{id: 0}))
	})
	t.Run("not cached", func(t *testing.T) {
		assert.Equal(t, 3, s.Len(&testObject{id: 1}))
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
	src := &testObject{id: 1, slice: s}
	dst := &testObject{id: 999, slice: s}

	s.Copy(src, dst)

	assert.Equal(t, (*MultiValue[int])(nil), dst.slice.individualItems[0])

	found := false
	for _, v := range dst.slice.individualItems[1].Individual {
		for _, o := range v.objs {
			if o == dst.id {
				found = true
				assert.Equal(t, 1, v.val)
			}
		}
	}
	assert.Equal(t, true, found)

	found = false
	for _, v := range s.individualItems[2].Individual {
		for _, o := range v.objs {
			if o == dst.id {
				found = true
				assert.Equal(t, 3, v.val)
			}
		}
	}
	assert.Equal(t, true, found)

	found = false
	for _, v := range s.individualItems[3].Individual {
		for _, o := range v.objs {
			if o == dst.id {
				found = true
				assert.Equal(t, 1, v.val)
			}
		}
	}
	assert.Equal(t, true, found)

	found = false
	for _, v := range s.individualItems[4].Individual {
		for _, o := range v.objs {
			if o == dst.id {
				found = true
			}
		}
	}
	assert.Equal(t, false, found)

	found = false
	for _, v := range s.appendedItems[0].Individual {
		for _, o := range v.objs {
			if o == dst.id {
				found = true
				assert.Equal(t, 1, v.val)
			}
		}
	}
	assert.Equal(t, true, found)

	found = false
	for _, v := range s.appendedItems[1].Individual {
		for _, o := range v.objs {
			if o == dst.id {
				found = true
				assert.Equal(t, 3, v.val)
			}
		}
	}
	assert.Equal(t, true, found)

	found = false
	for _, v := range s.appendedItems[2].Individual {
		for _, o := range v.objs {
			if o == dst.id {
				found = true
			}
		}
	}
	assert.Equal(t, false, found)

	l, ok := s.cachedLengths[999]
	require.Equal(t, true, ok)
	assert.Equal(t, 7, l)
}

func TestValue(t *testing.T) {
	// What we want to check:
	// - correct values are returned for first object
	// - correct values are returned for second object

	s := setup()
	first := &testObject{id: 1, slice: s}
	second := &testObject{id: 2, slice: s}

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
}

func TestAt(t *testing.T) {
	// What we want to check:
	// - correct values are returned for first object
	// - correct values are returned for second object
	// - ERROR when index too large in general
	// - ERROR when index not too large in general, but too large for an object

	s := setup()
	first := &testObject{id: 1, slice: s}
	second := &testObject{id: 2, slice: s}

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
	v, err = s.At(first, 7)
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
	v, err = s.At(second, 8)
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
	first := &testObject{id: 1, slice: s}
	second := &testObject{id: 2, slice: s}

	require.NoError(t, s.UpdateAt(first, 0, 999))
	assert.Equal(t, 123, s.sharedItems[0])
	found := false
	for _, v := range s.individualItems[0].Individual {
		for _, o := range v.objs {
			if o == first.id {
				found = true
				assert.Equal(t, 999, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
	found = false
	for _, v := range s.individualItems[0].Individual {
		for _, o := range v.objs {
			if o == second.id {
				found = true
			}
		}
	}
	assert.Equal(t, false, found)

	require.NoError(t, s.UpdateAt(first, 1, 999))
	found = false
	for _, v := range s.individualItems[1].Individual {
		for _, o := range v.objs {
			if o == first.id {
				found = true
				assert.Equal(t, 999, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
	found = false
	for _, v := range s.individualItems[1].Individual {
		for _, o := range v.objs {
			if o == second.id {
				found = true
				assert.Equal(t, 2, v.val)
			}
		}
	}
	assert.Equal(t, true, found)

	require.NoError(t, s.UpdateAt(first, 1, 2))
	found = false
	for _, v := range s.individualItems[1].Individual {
		for _, o := range v.objs {
			if o == first.id {
				found = true
				assert.Equal(t, 2, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
	found = false
	for _, v := range s.individualItems[1].Individual {
		for _, o := range v.objs {
			if o == second.id {
				found = true
				assert.Equal(t, 2, v.val)
			}
		}
	}
	assert.Equal(t, true, found)

	require.NoError(t, s.UpdateAt(first, 2, 999))
	found = false
	for _, v := range s.individualItems[2].Individual {
		for _, o := range v.objs {
			if o == first.id {
				found = true
				assert.Equal(t, 999, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
	found = false
	for _, v := range s.individualItems[2].Individual {
		for _, o := range v.objs {
			if o == second.id {
				found = true
				assert.Equal(t, 3, v.val)
			}
		}
	}
	assert.Equal(t, true, found)

	require.NoError(t, s.UpdateAt(first, 3, 999))
	found = false
	for _, v := range s.individualItems[3].Individual {
		for _, o := range v.objs {
			if o == first.id {
				found = true
				assert.Equal(t, 999, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
	found = false
	for _, v := range s.individualItems[3].Individual {
		for _, o := range v.objs {
			if o == second.id {
				found = true
			}
		}
	}
	assert.Equal(t, false, found)

	require.NoError(t, s.UpdateAt(first, 4, 999))
	found = false
	for _, v := range s.individualItems[4].Individual {
		for _, o := range v.objs {
			if o == first.id {
				found = true
				assert.Equal(t, 999, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
	found = false
	for _, v := range s.individualItems[4].Individual {
		for _, o := range v.objs {
			if o == second.id {
				found = true
				assert.Equal(t, 2, v.val)
			}
		}
	}
	assert.Equal(t, true, found)

	require.NoError(t, s.UpdateAt(first, 4, 123))
	found = false
	for _, v := range s.individualItems[4].Individual {
		for _, o := range v.objs {
			if o == first.id {
				found = true
			}
		}
	}
	assert.Equal(t, false, found)
	found = false
	for _, v := range s.individualItems[4].Individual {
		for _, o := range v.objs {
			if o == second.id {
				found = true
				assert.Equal(t, 2, v.val)
			}
		}
	}
	assert.Equal(t, true, found)

	require.NoError(t, s.UpdateAt(first, 5, 999))
	found = false
	for _, v := range s.appendedItems[0].Individual {
		for _, o := range v.objs {
			if o == first.id {
				found = true
				assert.Equal(t, 999, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
	found = false
	for _, v := range s.appendedItems[0].Individual {
		for _, o := range v.objs {
			if o == second.id {
				found = true
				assert.Equal(t, 2, v.val)
			}
		}
	}
	assert.Equal(t, true, found)

	require.NoError(t, s.UpdateAt(first, 5, 2))
	found = false
	for _, v := range s.appendedItems[0].Individual {
		for _, o := range v.objs {
			if o == first.id {
				found = true
				assert.Equal(t, 2, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
	found = false
	for _, v := range s.appendedItems[0].Individual {
		for _, o := range v.objs {
			if o == second.id {
				found = true
				assert.Equal(t, 2, v.val)
			}
		}
	}
	assert.Equal(t, true, found)

	require.NoError(t, s.UpdateAt(first, 6, 999))
	found = false
	for _, v := range s.appendedItems[1].Individual {
		for _, o := range v.objs {
			if o == first.id {
				found = true
				assert.Equal(t, 999, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
	found = false
	for _, v := range s.appendedItems[1].Individual {
		for _, o := range v.objs {
			if o == second.id {
				found = true
				assert.Equal(t, 3, v.val)
			}
		}
	}
	assert.Equal(t, true, found)

	// we update the second object because there are no more appended items for the first object
	require.NoError(t, s.UpdateAt(second, 7, 999))
	found = false
	for _, v := range s.appendedItems[2].Individual {
		for _, o := range v.objs {
			if o == first.id {
				found = true
			}
		}
	}
	assert.Equal(t, false, found)
	found = false
	for _, v := range s.appendedItems[2].Individual {
		for _, o := range v.objs {
			if o == second.id {
				found = true
				assert.Equal(t, 999, v.val)
			}
		}
	}
	assert.Equal(t, true, found)

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
	first := &testObject{id: 1, slice: s}
	second := &testObject{id: 2, slice: s}

	// append first value ever
	s.Append(first, 1)
	require.Equal(t, 1, len(s.appendedItems))
	found := false
	for _, v := range s.appendedItems[0].Individual {
		for _, o := range v.objs {
			if o == first.id {
				found = true
				assert.Equal(t, 1, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
	found = false
	for _, v := range s.appendedItems[0].Individual {
		for _, o := range v.objs {
			if o == second.id {
				found = true
			}
		}
	}
	assert.Equal(t, false, found)
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
	found = false
	for _, v := range s.appendedItems[0].Individual {
		for _, o := range v.objs {
			if o == first.id {
				found = true
				assert.Equal(t, 1, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
	found = false
	for _, v := range s.appendedItems[0].Individual {
		for _, o := range v.objs {
			if o == second.id {
				found = true
				assert.Equal(t, 1, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
	l, ok = s.cachedLengths[first.id]
	require.Equal(t, true, ok)
	assert.Equal(t, 3, l)
	l, ok = s.cachedLengths[second.id]
	assert.Equal(t, true, ok)
	assert.Equal(t, 2, l)

	// append the first value to the second object, different than the value for the first object
	s.Append(second, 2)
	require.Equal(t, 2, len(s.appendedItems))
	found = false
	for _, v := range s.appendedItems[1].Individual {
		for _, o := range v.objs {
			if o == first.id {
				found = true
				assert.Equal(t, 1, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
	found = false
	for _, v := range s.appendedItems[1].Individual {
		for _, o := range v.objs {
			if o == second.id {
				found = true
				assert.Equal(t, 2, v.val)
			}
		}
	}
	assert.Equal(t, true, found)
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
	obj := &testObject{id: 1, slice: s}

	s.Detach(obj)

	for _, item := range s.individualItems {
		found := false
		for _, v := range item.Individual {
			for _, o := range v.objs {
				if o == obj.id {
					found = true
				}
			}
		}
		assert.Equal(t, false, found)
	}
	for _, item := range s.appendedItems {
		found := false
		for _, v := range item.Individual {
			for _, o := range v.objs {
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
// Index 3: Individual value ONLY for the second object
// Index 4: Different appended value
// Index 5: Same appended value
// Index 6: Appended value ONLY for the second object
func setup() *Slice[int, *testObject] {
	s := &Slice[int, *testObject]{}
	s.Init([]int{123, 123, 123, 123, 123})
	s.individualItems[1] = &MultiValue[int]{
		Individual: []*Value[int]{
			{
				val:  1,
				objs: []interfaces.Id{1},
			},
			{
				val:  2,
				objs: []interfaces.Id{2},
			},
		},
	}
	s.individualItems[2] = &MultiValue[int]{
		Individual: []*Value[int]{
			{
				val:  3,
				objs: []interfaces.Id{1, 2},
			},
		},
	}
	s.individualItems[3] = &MultiValue[int]{
		Individual: []*Value[int]{
			{
				val:  1,
				objs: []interfaces.Id{1},
			},
		},
	}
	s.individualItems[4] = &MultiValue[int]{
		Individual: []*Value[int]{
			{
				val:  2,
				objs: []interfaces.Id{2},
			},
		},
	}
	s.appendedItems = []*MultiValue[int]{
		{
			Individual: []*Value[int]{
				{
					val:  1,
					objs: []interfaces.Id{1},
				},
				{
					val:  2,
					objs: []interfaces.Id{2},
				},
			},
		},
		{
			Individual: []*Value[int]{
				{
					val:  3,
					objs: []interfaces.Id{1, 2},
				},
			},
		},
		{
			Individual: []*Value[int]{
				{
					val:  2,
					objs: []interfaces.Id{2},
				},
			},
		},
	}
	s.cachedLengths[1] = 7
	s.cachedLengths[2] = 8

	return s
}
