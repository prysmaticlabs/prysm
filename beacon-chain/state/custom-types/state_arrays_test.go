package custom_types

import (
	"reflect"
	"testing"
)

func TestByte32_Casting(t *testing.T) {
	var b [32]byte
	d := Byte32(b)
	if !reflect.DeepEqual([32]byte(d), b) {
		t.Errorf("Unequal: %v = %v", d, b)
	}
}

func TestByte32_UnmarshalSSZ(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		d := Byte32{}
		var b = [32]byte{'f', 'o', 'o'}
		err := d.UnmarshalSSZ(b[:])
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !reflect.DeepEqual(b, [32]byte(d)) {
			t.Errorf("Unequal: %v = %v", b, [32]byte(d))
		}
	})

	t.Run("Wrong slice length", func(t *testing.T) {
		d := Byte32{}
		var b = [16]byte{'f', 'o', 'o'}
		err := d.UnmarshalSSZ(b[:])
		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestByte32_MarshalSSZTo(t *testing.T) {
	d := Byte32{'f', 'o', 'o'}
	dst := []byte("bar")
	b, err := d.MarshalSSZTo(dst)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	rest := [29]byte{}
	expected := append([]byte("barfoo"), rest[:]...)
	if !reflect.DeepEqual(expected, b) {
		t.Errorf("Unequal: %v = %v", expected, b)
	}
}

func TestByte32_MarshalSSZ(t *testing.T) {
	d := Byte32{'f', 'o', 'o'}
	b, err := d.MarshalSSZ()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	actual := [32]byte(d)
	if !reflect.DeepEqual(b, actual[:]) {
		t.Errorf("Unequal: %v = %v", b, [32]byte(d))
	}
}

func TestByte32_SizeSSZ(t *testing.T) {
	d := Byte32{}
	if d.SizeSSZ() != 32 {
		t.Errorf("Wrong SSZ size. Expected %v vs actual %v", 32, d.SizeSSZ())
	}
}

func TestStateRoots_Casting(t *testing.T) {
	var b [stateRootsSize][32]byte
	d := StateRoots(b)
	if !reflect.DeepEqual([stateRootsSize][32]byte(d), b) {
		t.Errorf("Unequal: %v = %v", d, b)
	}
}

func TestStateRoots_UnmarshalSSZ(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		d := StateRoots{}
		var b [stateRootsSize][32]byte
		b[0] = [32]byte{'f', 'o', 'o'}
		b[1] = [32]byte{'b', 'a', 'r'}
		bb := make([]byte, stateRootsSize*32)
		for i, elem32 := range b {
			for j, elem := range elem32 {
				bb[i*32+j] = elem
			}
		}
		err := d.UnmarshalSSZ(bb)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !reflect.DeepEqual(b, [stateRootsSize][32]byte(d)) {
			t.Errorf("Unequal: %v = %v", b, [stateRootsSize][32]byte(d))
		}
	})

	t.Run("Wrong slice length", func(t *testing.T) {
		d := StateRoots{}
		var b [stateRootsSize][16]byte
		b[0] = [16]byte{'f', 'o', 'o'}
		b[1] = [16]byte{'b', 'a', 'r'}
		bb := make([]byte, stateRootsSize*16)
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

func TestStateRoots_MarshalSSZTo(t *testing.T) {
	var d StateRoots
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

func TestStateRoots_MarshalSSZ(t *testing.T) {
	d := StateRoots{}
	d[0] = [32]byte{'f', 'o', 'o'}
	d[1] = [32]byte{'b', 'a', 'r'}
	b, err := d.MarshalSSZ()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d[0][:], b[0:32]) {
		t.Errorf("Unequal: %v = %v", d[0], b[0:32])
	}
	if !reflect.DeepEqual(d[1][:], b[32:64]) {
		t.Errorf("Unequal: %v = %v", d[0], b[32:64])
	}
}

func TestStateRoots_SizeSSZ(t *testing.T) {
	d := StateRoots{}
	if d.SizeSSZ() != stateRootsSize*32 {
		t.Errorf("Wrong SSZ size. Expected %v vs actual %v", stateRootsSize*32, d.SizeSSZ())
	}
}

func TestRandaoMixes_Casting(t *testing.T) {
	b := make([][32]byte, 4)
	d := RandaoMixes(b)
	if !reflect.DeepEqual([][32]byte(d), b) {
		t.Errorf("Unequal: %v = %v", d, b)
	}
}

func TestRandaoMixes_UnmarshalSSZ(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		d := RandaoMixes(make([][32]byte, 2))
		b := make([][32]byte, 2)
		b[0] = [32]byte{'f', 'o', 'o'}
		b[1] = [32]byte{'b', 'a', 'r'}
		bb := make([]byte, 2*32)
		for i, elem32 := range b {
			for j, elem := range elem32 {
				bb[i*32+j] = elem
			}
		}
		err := d.UnmarshalSSZ(bb)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !reflect.DeepEqual(b, [][32]byte(d)) {
			t.Errorf("Unequal: %v = %v", b, [][32]byte(d))
		}
	})

	t.Run("Wrong slice length", func(t *testing.T) {
		d := RandaoMixes(make([][32]byte, 2))
		b := make([][16]byte, 2)
		b[0] = [16]byte{'f', 'o', 'o'}
		b[1] = [16]byte{'b', 'a', 'r'}
		bb := make([]byte, 2*16)
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

func TestRandaoMixes_MarshalSSZTo(t *testing.T) {
	d := RandaoMixes(make([][32]byte, 1))
	d[0] = [32]byte{'f', 'o', 'o'}
	dst := []byte("bar")
	b, err := d.MarshalSSZTo(dst)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	expected := []byte{'b', 'a', 'r', 'f', 'o', 'o'}
	actual := []byte{b[0], b[1], b[2], b[3], b[4], b[5]}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Unequal: %v = %v", expected, actual)
	}
}

func TestRandaoMixes_MarshalSSZ(t *testing.T) {
	d := RandaoMixes(make([][32]byte, 2))
	d[0] = [32]byte{'f', 'o', 'o'}
	d[1] = [32]byte{'b', 'a', 'r'}
	b, err := d.MarshalSSZ()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d[0][:], b[0:32]) {
		t.Errorf("Unequal: %v = %v", d[0], b[0:32])
	}
	if !reflect.DeepEqual(d[1][:], b[32:64]) {
		t.Errorf("Unequal: %v = %v", d[0], b[32:64])
	}
}

func TestRandaoMixes_SizeSSZ(t *testing.T) {
	d := RandaoMixes(make([][32]byte, 2))
	if d.SizeSSZ() != 2*32 {
		t.Errorf("Wrong SSZ size. Expected %v vs actual %v", 2*32, d.SizeSSZ())
	}
}
