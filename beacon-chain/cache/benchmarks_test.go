package cache

import (
	"testing"
)

var aInfo300K, _ = createIndices(300000)
var aInfo1M, _ = createIndices(1000000)
var epoch = uint64(1)
var _, treeIndices300K = createIndices(300000)
var byteIndices300k = convertUint64ToByteSlice(treeIndices300K);


func createIndices(count int) (*ActiveIndicesByEpoch, []uint64) {
	indices := make([]uint64, 0, count)
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

func BenchmarkBloomFilter(b *testing.B) {
	bf := NewBloomFilter()


	b.Run("BLOOMADD300K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bf.AddActiveIndicesBloomFilter(byteIndices300k)
		}
	})


	b.Run("BLOOMTEST300K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if !bf.TestActiveIndicesBloomFilter(byteIndices300k) {
				break;
			}
		}
		bf.ClearBloomFilter()
	})
}

func BenchmarkCFilter(b *testing.B) {
	cf := NewCFilter()


	b.Run("CFILTERADD300K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cf.InsertActiveIndicesCFilter(byteIndices300k)
		}
	})


	b.Run("CFILTERLOOKUP300K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if !cf.LookupActiveIndicesCFilter(byteIndices300k) {
				break;
			}
		}
	})
}


