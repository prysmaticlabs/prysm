package cache

import (
	"testing"
)

var aInfo300K, _ = createIndices(300000)
var aInfo1M, _ = createIndices(1000000)
var epoch = uint64(1)
var _, treeIndices300K = createIndices(300000)

func createIndices(count int) (*ActiveIndicesByEpoch, []uint64) {
	indices := make([]uint64, count, count)
	for i := 0; i < count; i++ {
		indices = append(indices, uint64(i))
	}
	return &ActiveIndicesByEpoch{
		Epoch:         epoch,
		ActiveIndices: indices,
	}, indices
}

func BenchmarkCachingAddRetrieve(b *testing.B) {

	c := NewActiveIndicesCache()

	b.Run("ADD300K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := c.AddActiveIndicesList(aInfo300K); err != nil {
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

func BenchmarkTreeAddRetrieve(b *testing.B) {
	t := NewActiveIndicesTree()

	b.Run("TREEADD300K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			t.InsertReplaceActiveIndicesTree(treeIndices300K)
		}
	})

	b.Run("RETRTREE300K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := t.RetrieveActiveIndicesTree(); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("TREEADDNOREPL300K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			t.InsertNoReplaceActiveIndicesTree(treeIndices300K)
		}
	})

	b.Run("RETRTREE300K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := t.RetrieveActiveIndicesTree(); err != nil {
				b.Fatal(err)
			}
		}
	})
}
