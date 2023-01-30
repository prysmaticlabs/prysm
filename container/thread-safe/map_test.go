package threadsafe

import (
	"sort"
	"sync"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestMap(t *testing.T) {
	m := map[int]string{
		1:     "foo",
		200:   "bar",
		10000: "baz",
	}

	tMap := NewThreadSafeMap(m)
	keys := tMap.Keys()
	sort.IntSlice(keys).Sort()

	require.DeepEqual(t, []int{1, 200, 10000}, keys)
	require.Equal(t, 3, tMap.Len())

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(w *sync.WaitGroup, scopedMap *Map[int, string]) {
			defer w.Done()
			v, ok := scopedMap.Get(1)
			require.Equal(t, true, ok)
			require.Equal(t, "foo", v)

			scopedMap.Put(3, "nyan")

			v, ok = scopedMap.Get(3)
			require.Equal(t, true, ok)
			require.Equal(t, "nyan", v)

		}(&wg, tMap)
	}
	wg.Wait()

	tMap.Delete(3)

	_, ok := tMap.Get(3)
	require.Equal(t, false, ok)
}
