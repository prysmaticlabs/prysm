package testing

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func TestSSZTagSize(t *testing.T) {
	sigSize := 96
	pubKeySize := 48
	rootSize := 32

	sizes, err := sszTagSizes(pb.Attestation{}, "Signature")
	if err != nil {
		t.Fatal(err)
	}
	if sizes[0] != sigSize {
		t.Errorf("wanted signature size: %d, got: %d", sigSize, sizes[0])
	}

	sizes, err = sszTagSizes(pb.BeaconBlock{}, "Signature")
	if err != nil {
		t.Fatal(err)
	}
	if sizes[0] != sigSize {
		t.Errorf("wanted signature size: %d, got: %d", sigSize, sizes[0])
	}

	sizes, err = sszTagSizes(pb.Checkpoint{}, "Root")
	if err != nil {
		t.Fatal(err)
	}
	if sizes[0] != rootSize {
		t.Errorf("wanted signature size: %d, got: %d", rootSize, sizes[0])
	}

	sizes, err = sszTagSizes(pb.Validator{}, "PublicKey")
	if err != nil {
		t.Fatal(err)
	}
	if sizes[0] != pubKeySize {
		t.Errorf("wanted signature size: %d, got: %d", pubKeySize, sizes[0])
	}
}

func sszTagSizes(i interface{}, fName string) ([]int, error) {
	v := reflect.ValueOf(i)
	field, exists := v.Type().FieldByName(fName)
	if !exists {
		return nil, errors.New("wanted field does not exist")
	}
	tag, exists := field.Tag.Lookup("ssz-size")
	if !exists {
		return nil, errors.New("wanted field does not contain ssz-size tag")
	}
	start := strings.IndexRune(tag, '=')
	items := strings.Split(tag[start+1:], ",")
	sizes := make([]int, len(items))
	var err error
	for i := 0; i < len(items); i++ {
		if items[i] == "?" {
			sizes[i] = 0
			continue
		}
		sizes[i], err = strconv.Atoi(items[i])
		if err != nil {
			return nil, err
		}
	}
	return sizes, nil
}
