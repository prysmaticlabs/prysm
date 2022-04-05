package customtypes

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
