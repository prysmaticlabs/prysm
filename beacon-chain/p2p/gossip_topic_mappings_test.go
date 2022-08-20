package p2p

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	eth2types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
)

func TestMappingHasNoDuplicates(t *testing.T) {
	params.SetupTestConfigCleanup(t)
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
	bCfg := params.BeaconConfig().Copy()
	altairForkEpoch := eth2types.Epoch(100)
	BellatrixForkEpoch := eth2types.Epoch(200)

	bCfg.AltairForkEpoch = altairForkEpoch
	bCfg.BellatrixForkEpoch = BellatrixForkEpoch
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.AltairForkVersion)] = eth2types.Epoch(100)
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.BellatrixForkVersion)] = eth2types.Epoch(200)
	params.OverrideBeaconConfig(bCfg)

	// Phase 0
	pMessage := GossipTopicMappings(BlockSubnetTopicFormat, 0)
	_, ok := pMessage.(*ethpb.SignedBeaconBlock)
	assert.Equal(t, true, ok)

	// Altair Fork
	pMessage = GossipTopicMappings(BlockSubnetTopicFormat, altairForkEpoch)
	_, ok = pMessage.(*ethpb.SignedBeaconBlockAltair)
	assert.Equal(t, true, ok)

	// Bellatrix Fork
	pMessage = GossipTopicMappings(BlockSubnetTopicFormat, BellatrixForkEpoch)
	_, ok = pMessage.(*ethpb.SignedBeaconBlockBellatrix)
	assert.Equal(t, true, ok)

}
