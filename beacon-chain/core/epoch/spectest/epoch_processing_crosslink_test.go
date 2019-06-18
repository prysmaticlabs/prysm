package spectest

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
)

func TestCrosslinksProcessingYaml(t *testing.T) {
	file, err := ioutil.ReadFile("crosslinks_minimal_formatted.yaml")
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &CrosslinksMinimal{}
	if err := yaml.Unmarshal(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatal(err)
	}

	t.Logf("Running spec test vectors for %s", s.Title)
	for i, testCase := range s.TestCases {
		preState := &pb.BeaconState{}
		t.Logf("Testing testcase %s", testCase.Description)
		b, err := json.Marshal(testCase.Pre)
		if err != nil {
			t.Fatal(err)
		}
		err = jsonpb.Unmarshal(bytes.NewReader(b), preState)
		if err != nil {
			t.Fatal(err)
		}

		var postState *pb.BeaconState

		postState, err = epoch.ProcessCrosslinks(preState)
		if err != nil {
			t.Fatal(err)
		}

		genPostState := &pb.BeaconState{}
		b, err = json.Marshal(testCase.Post)
		if err != nil {
			t.Fatal(err)
		}
		err = jsonpb.Unmarshal(bytes.NewReader(b), genPostState)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(postState.Slot, genPostState.Slot) {
			t.Error("Post state slot miss matched")
		}
		if !reflect.DeepEqual(postState.GenesisTime, genPostState.GenesisTime) {
			t.Error("Post state genesis time matched")
		}
		if !reflect.DeepEqual(postState.Fork, genPostState.Fork) {
			t.Error("Post state fork miss matched")
		}
		if !reflect.DeepEqual(postState.ValidatorRegistry, genPostState.ValidatorRegistry) {
			t.Error("Post state validator registry miss matched")
		}
		if !reflect.DeepEqual(postState.Balances, genPostState.Balances) {
			t.Error("Post state balances miss matched")
		}
		if !reflect.DeepEqual(postState.LatestRandaoMixes, genPostState.LatestRandaoMixes) {
			t.Error("Post state latest randao mixes miss matched")
		}
		if !reflect.DeepEqual(postState.LatestStartShard, genPostState.LatestStartShard) {
			t.Error("Post state latest start shard miss matched")
		}
		if !reflect.DeepEqual(postState.CurrentCrosslinks, genPostState.CurrentCrosslinks) {
			t.Error("Post state current crosslinks miss matched")
		}
		if !reflect.DeepEqual(postState.PreviousCrosslinks, genPostState.PreviousCrosslinks) {
			t.Error("Post state prev crosslinks miss matched")
		}
		if !reflect.DeepEqual(postState.LatestBlockRoots, genPostState.LatestBlockRoots) {
			t.Error("Post state latest block roots miss matched")
		}
		if !reflect.DeepEqual(postState.LatestStateRoots, genPostState.LatestStateRoots) {
			t.Error("Post state latest state roots miss matched")
		}
		if !reflect.DeepEqual(postState.LatestActiveIndexRoots, genPostState.LatestActiveIndexRoots) {
			t.Error("Post state latest active indxes miss matched")
		}
		if !reflect.DeepEqual(postState.LatestSlashedBalances, genPostState.LatestSlashedBalances) {
			t.Error("Post state latest slashed balances miss matched")
		}
		if !reflect.DeepEqual(postState.LatestBlockHeader, genPostState.LatestBlockHeader) {
			t.Error("Post state latest block header miss matched")
		}
		if !reflect.DeepEqual(postState.HistoricalRoots, genPostState.HistoricalRoots) {
			t.Error("Post state historical roots miss matched")
		}
		if !reflect.DeepEqual(postState.LatestEth1Data, genPostState.LatestEth1Data) {
			t.Error("Post state latest eth1 data miss matched")
		}
		if !reflect.DeepEqual(postState.DepositIndex, genPostState.DepositIndex) {
			t.Error("Post state deposit index miss matched")
		}
	}
}
