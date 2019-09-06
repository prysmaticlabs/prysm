package kv

import (
	"bytes"

	"github.com/boltdb/bolt"
)

// lookupValuesForIndices takes in a list of indices and looks up
// their corresponding values in the DB, returning a list of
// roots which can then be used for batch lookups of their corresponding
// objects from the DB. For example, if we are fetching
// attestations and we have an index `[]byte("5")` under the shard indices bucket,
// we might find roots `0x23` and `0x45` stored under that index. We can then
// do a batch read for attestations corresponding to those roots.
func lookupValuesForIndices(indicesByBucket map[string][]byte, tx *bolt.Tx) [][][]byte {
	values := make([][][]byte, 0)
	for k, v := range indicesByBucket {
		bkt := tx.Bucket([]byte(k))
		roots := bkt.Get(v)
		splitRoots := make([][]byte, 0)
		for i := 0; i < len(roots); i += 32 {
			splitRoots = append(splitRoots, roots[i:i+32])
		}
		values = append(values, splitRoots)
	}
	return values
}

// updateValueForIndices updates the value for each index by appending it to the previous
// values stored at said index. Typically, indices are roots of data that can then
// be used for reads or batch reads from the DB.
func updateValueForIndices(indicesByBucket map[string][]byte, root []byte, tx *bolt.Tx) error {
	for k, idx := range indicesByBucket {
		bkt := tx.Bucket([]byte(k))
		valuesAtIndex := bkt.Get(idx)
		if valuesAtIndex == nil {
			if err := bkt.Put(idx, root); err != nil {
				return err
			}
		} else {
			if err := bkt.Put(idx, append(valuesAtIndex, root...)); err != nil {
				return err
			}
		}
	}
	return nil
}

// deleteValueForIndices clears a root stored at each index.
func deleteValueForIndices(indicesByBucket map[string][]byte, root []byte, tx *bolt.Tx) error {
	for k, idx := range indicesByBucket {
		bkt := tx.Bucket([]byte(k))
		valuesAtIndex := bkt.Get(idx)
		if valuesAtIndex != nil {
			start := bytes.Index(valuesAtIndex, root)
			// if the root was not found inside the values at index slice, we continue.
			if start == -1 {
				continue
			}
			// We clear out the root from the values at index slice. For example,
			// If we had [0x32, 0x33, 0x45] and we wanted to clear out 0x33, the code below
			// updates the slice to [0x32, 0x45].

			valuesStart := make([]byte, len(valuesAtIndex[:start]))
			copy(valuesStart, valuesAtIndex[:start])

			valuesEnd := make([]byte, len(valuesAtIndex[start+len(root):]))
			copy(valuesEnd, valuesAtIndex[start+len(root):])

			valuesAtIndex = append(valuesStart, valuesEnd...)
			if err := bkt.Put(idx, valuesAtIndex); err != nil {
				return err
			}
		}
	}
	return nil
}
