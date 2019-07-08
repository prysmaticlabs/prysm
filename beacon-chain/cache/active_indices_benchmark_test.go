package cache

import (
	"testing"

)


var aInfos16K = createAInfos(16384);
var aInfos300K = createAInfos(300000);


func createAInfos(count int) []*ActiveIndicesByEpoch {
	aInfos := make([]*ActiveIndicesByEpoch, count)
    for i := 0; i < count; i++ {
		aInfo := &ActiveIndicesByEpoch{
			Epoch:         999,
			ActiveIndices: []uint64{1, 2, 3, 4, 5},
		}
		aInfos = append(aInfos, aInfo)
	}
	return aInfos;
}



func BenchmarkActiveIndicesKeyFn_OK(b *testing.B) {
	var err error
	
	b.Run("16K", func(b *testing.B) {
		//b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key, err = activeIndicesKeyFn(aInfos16K[i])
			if err != nil {
				b.Fatal(err)
			}
			if key != strconv.Itoa(int(aInfos16K[i].Epoch)) {
				b.Errorf("Incorrect hash key: %s, expected %s", key, strconv.Itoa(int(aInfo.Epoch)))
			}

		}
	})
	b.Run("300K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
		     _, err = activeIndicesKeyFn(aInfos300K[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})
    
	
}


// func BenchmarkActiveIndicesKeyFn_InvalidObj(b *testing.B) {


// }


// func BenchmarkActiveIndicesCache_ActiveIndicesByEpoch(b *testing.B) {




// }


// func BenchmarkActiveIndices_MaxSize(b *testing.B) {

// }









