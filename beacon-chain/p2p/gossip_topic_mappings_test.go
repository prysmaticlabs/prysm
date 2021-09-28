package p2p

import (
	"reflect"
	"testing"

	eth2types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestMappingHasNoDuplicates(t *testing.T) {
	m := make(map[reflect.Type]bool)
	for _, v := range gossipTopicMappings {
		if _, ok := m[reflect.TypeOf(v)]; ok {
			t.Errorf("%T is duplicated in the topic mapping", v)
		}
		m[reflect.TypeOf(v)] = true
	}
}

func TestGossipTopicMappings_CorrectBlockType(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	bCfg := params.BeaconConfig()
	altairForkEpoch := eth2types.Epoch(100)
	mergeForkEpoch := eth2types.Epoch(200)

	bCfg.AltairForkEpoch = altairForkEpoch
	bCfg.MergeForkEpoch = mergeForkEpoch
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.AltairForkVersion)] = eth2types.Epoch(100)
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.MergeForkVersion)] = eth2types.Epoch(200)
	params.OverrideBeaconConfig(bCfg)

	// Phase 0
	pMessage := GossipTopicMappings(BlockSubnetTopicFormat, 0)
	_, ok := pMessage.(*ethpb.SignedBeaconBlock)
	assert.Equal(t, true, ok)

	// After Fork
	pMessage = GossipTopicMappings(BlockSubnetTopicFormat, altairForkEpoch)
	_, ok = pMessage.(*ethpb.SignedBeaconBlockAltair)
	assert.Equal(t, true, ok)

	//Merge Fork
	pMessage = GossipTopicMappings(BlockSubnetTopicFormat, mergeForkEpoch)
	_, ok = pMessage.(*ethpb.SignedBeaconBlockAltair)
	assert.Equal(t, true, ok)

}
