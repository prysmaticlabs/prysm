package testutil

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/testing"
)

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

	if output.Foo != "foo" {
		t.Error("Expected output.Foo to be foo")
	}
	if output.Bar != "baz" {
		t.Errorf("Expected output.Bar to be baz")
	}
}
