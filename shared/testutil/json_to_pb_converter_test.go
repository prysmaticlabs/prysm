package testutil

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/testing"
)

const baz = "baz"
const foo = "foo"

func TestOverrideSpecName(t *testing.T) {
	input := struct {
		Foo string `json:"foo"`
		Baz string `json:"baz"`
	}{
		Foo: "foo",
		Baz: "baz",
	}
	output := &pb.TestMessage{}
	if err := ConvertToPb(input, output); err != nil {
		t.Fatal(err)
	}

	if output.Foo != foo {
		t.Error("Expected output.Foo to be foo")
	}
	if output.Bar != baz {
		t.Errorf("Expected output.Bar to be baz")
	}
}

func TestOverrideSpecName_NestedProto(t *testing.T) {
	input := struct {
		Fuzz string `json:"fuzz"`
		Msg  struct {
			Foo string `json:"foo"`
			Baz string `json:"baz"`
		}
	}{
		Fuzz: "fuzz",
		Msg: struct {
			Foo string `json:"foo"`
			Baz string `json:"baz"`
		}{
			Foo: "foo",
			Baz: "baz",
		},
	}

	output := &pb.TestNestedMessage{}
	if err := ConvertToPb(input, output); err != nil {
		t.Fatal(err)
	}

	if output.Fuzz != "fuzz" {
		t.Error("Expected output.fuzz to be fuzz")
	}
	if output.Msg.Foo != foo {
		t.Error("Expected output.Msg.Foo to be foo")
	}
	if output.Msg.Bar != baz {
		t.Errorf("Expected output.Msg.Bar to be baz")
	}
}
