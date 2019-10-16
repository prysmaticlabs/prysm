package cache

import (
	"testing"
)

var indices300k = createIndices(300000)
var epoch = uint64(1)

func createIndices(count int) *ActiveIndicesByEpoch {
	indices := make([]uint64, 0, count)
	for i := 0; i < count; i++ {
		indices = append(indices, uint64(i))
	}
	return &ActiveIndicesByEpoch{
		Epoch:         epoch,
		ActiveIndices: indices,
	}
}

func BenchmarkCachingAddRetrieve(b *testing.B) {

	c := NewActiveIndicesCache()

	b.Run("ADD300K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := c.AddActiveIndicesList(indices300k); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("RETR300K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := c.ActiveIndicesInEpoch(epoch); err != nil {
				b.Fatal(err)
			}
		}
	})

}
