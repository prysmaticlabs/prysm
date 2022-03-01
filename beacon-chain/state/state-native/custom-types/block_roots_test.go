package customtypes

import (
	"bytes"
	"reflect"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
)

func TestBlockRoots_Casting(t *testing.T) {
	var b [fieldparams.BlockRootsLength][32]byte
	f := SetFromSlice([][]byte{})
	f.SetFromBaseField(b)
	if !reflect.DeepEqual(f.Array(), b) {
		t.Errorf("Unequal: %v = %v", f.Array(), b)
	}
}

func TestBlockRoots_UnmarshalSSZ(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		d := SetFromSlice([][]byte{})
		var b [fieldparams.BlockRootsLength][32]byte
		b[0] = [32]byte{'f', 'o', 'o'}
		b[1] = [32]byte{'b', 'a', 'r'}
		bb := make([]byte, fieldparams.BlockRootsLength*32)
		for i, elem32 := range b {
			for j, elem := range elem32 {
				bb[i*32+j] = elem
			}
		}
		err := d.UnmarshalSSZ(bb)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !reflect.DeepEqual(b, d.Array()) {
			t.Errorf("Unequal: %v = %v", b, d.Array())
		}
	})

	t.Run("Wrong slice length", func(t *testing.T) {
		d := BlockRoots{}
		var b [fieldparams.BlockRootsLength][16]byte
		b[0] = [16]byte{'f', 'o', 'o'}
		b[1] = [16]byte{'b', 'a', 'r'}
		bb := make([]byte, fieldparams.BlockRootsLength*16)
		for i, elem16 := range b {
			for j, elem := range elem16 {
				bb[i*16+j] = elem
			}
		}
		err := d.UnmarshalSSZ(bb)
		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestBlockRoots_MarshalSSZTo(t *testing.T) {
	var d BlockRoots
	dst := []byte("foo")
	b, err := d.MarshalSSZTo(dst)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	expected := []byte{'f', 'o', 'o'}
	actual := []byte{b[0], b[1], b[2]}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Unequal: %v = %v", expected, actual)
	}
}

func TestBlockRoots_MarshalSSZ(t *testing.T) {
	d := SetFromSlice([][]byte{})
	d.IncreaseRef()
	d.SetRootAtIndex(0, [32]byte{'f', 'o', 'o'})
	d.IncreaseRef()
	d.IncreaseRef()
	d.SetRootAtIndex(1, [32]byte{'b', 'a', 'r'})
	b, err := d.MarshalSSZ()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	rt := d.RootAtIndex(0)
	if !reflect.DeepEqual(rt[:], b[0:32]) {
		t.Errorf("Unequal: %v = %v", rt, b[0:32])
	}
	rt = d.RootAtIndex(1)
	if !reflect.DeepEqual(rt[:], b[32:64]) {
		t.Errorf("Unequal: %v = %v", rt, b[32:64])
	}
	d2 := SetFromSlice([][]byte{})
	err = d2.UnmarshalSSZ(b)
	if err != nil {
		t.Error(err)
	}
	res, err := d2.MarshalSSZ()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(res, b) {
		t.Error("unequal")
	}
}

func TestBlockRoots_SizeSSZ(t *testing.T) {
	d := SetFromSlice([][]byte{})
	if d.SizeSSZ() != fieldparams.BlockRootsLength*32 {
		t.Errorf("Wrong SSZ size. Expected %v vs actual %v", fieldparams.BlockRootsLength*32, d.SizeSSZ())
	}
}

/*

func TestBlockRoots_Slice(t *testing.T) {
	a, b, c := [32]byte{'a'}, [32]byte{'b'}, [32]byte{'c'}
	roots := BlockRoots{}
	roots[1] = a
	roots[10] = b
	roots[100] = c
	slice := roots.Slice()
	assert.DeepEqual(t, a[:], slice[1])
	assert.DeepEqual(t, b[:], slice[10])
	assert.DeepEqual(t, c[:], slice[100])
}
*/
