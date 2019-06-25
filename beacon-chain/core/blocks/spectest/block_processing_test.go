package spectest

import (
	"context"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	log "github.com/sirupsen/logrus"
)

func TestBlockProcessingYaml(t *testing.T) {
	ctx := context.Background()
	filepath, err := bazel.Runfile("/eth2_spec_tests/tests/sanity/blocks/sanity_blocks_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}

	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &BlocksMinimal{}
	if err := yaml.Unmarshal(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	log.Infof("Title: %v", s.Title)
	log.Infof("Summary: %v", s.Summary)
	log.Infof("Fork: %v", s.Forks)
	log.Infof("Config: %v", s.Config)

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatalf("Could not set config: %v", err)
	}

	for _, testCase := range s.TestCases {
		t.Logf("Description: %s", testCase.Description)

		if testCase.Description == "attestation" || testCase.Description == "voluntary_exit" {
			continue
		}

		postState := &pb.BeaconState{}
		stateConfig := state.DefaultConfig()

		for _, b := range testCase.Blocks {

			postState, err = state.ExecuteStateTransition(ctx, testCase.Pre, b, stateConfig)
			if err != nil {
				t.Fatal(err)
			}
		}

		postRoot, err := ssz.HashTreeRoot(postState)
		if err != nil {
			t.Error(err)
			continue
		}

		testRoot, err := ssz.HashTreeRoot(testCase.Post)
		if err != nil {
			t.Error(err)
			continue
		}

		if testRoot != postRoot {
			checkState(postState, testCase.Post)
		}
	}
}

func checkState(a interface{}, b interface{}) {
	if !validateStruct(a) || !validateStruct(b) {
		panic("invalid data types provided")
	}
	fieldsA := fields(a)
	fieldsB := fields(b)

	if len(fieldsA) != len(fieldsB) {
		panic("fields length different")
	}

	for i, v := range fieldsA {

		if !reflect.DeepEqual(v, fieldsB[i].Interface()) {
			log.Errorf("Field %s for struct are unequal. Got %v but wanted %v", v.Type().Name(), v, fieldsB[i])
		}
	}

}

func validateStruct(a interface{}) bool {
	valA := reflect.ValueOf(a)
	if valA.Kind() == reflect.Struct {
		return true
	}

	if valA.Kind() == reflect.Ptr {
		return valA.Elem().Kind() == reflect.Struct
	}

	return false
}

func fields(a interface{}) []reflect.Value {
	val := reflect.ValueOf(a)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	var fields []reflect.Value
	for i := 0; i < val.NumField(); i++ {
		if !strings.Contains(val.Type().Field(i).Name, "XXX") {
			fields = append(fields, val.Field(i))
		}
	}
	return fields
}
