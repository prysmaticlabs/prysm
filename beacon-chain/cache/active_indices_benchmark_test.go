package cache

import (
	"testing"
)



var aInfo300K = createAInfo(300000)
var aInfo1M = createAInfo(1000000)



func createAInfo(count int) *ActiveIndicesByEpoch {
	indices := make([]uint64, count, count)
    for i := 0; i < count; i++ {
		indices = append(indices, uint64(i))
	}
    return &ActiveIndicesByEpoch{
		Epoch:         1,
		ActiveIndices: indices,
	}
}



func BenchmarkAddActiveIndicesList(b *testing.B) {

    c := NewActiveIndicesCache()
	b.Run("300K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := c.AddActiveIndicesList(aInfo300K);err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("1M", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := c.AddActiveIndicesList(aInfo1M);err != nil {
				b.Fatal(err)
			}
		}
	})
    
	
}


