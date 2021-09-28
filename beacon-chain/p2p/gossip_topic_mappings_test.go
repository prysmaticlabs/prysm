package p2p

import (
	"reflect"
	"testing"

	eth2types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
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
	forkEpoch := eth2types.Epoch(100)
	bCfg.AltairForkEpoch = forkEpoch
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.AltairForkVersion)] = eth2types.Epoch(100)
	params.OverrideBeaconConfig(bCfg)

	// Before Fork
	pMessage := GossipTopicMappings(BlockSubnetTopicFormat, 0)
	_, ok := pMessage.(*ethpb.SignedBeaconBlock)
	assert.Equal(t, true, ok)

	// After Fork
	pMessage = GossipTopicMappings(BlockSubnetTopicFormat, forkEpoch)
	_, ok = pMessage.(*ethpb.SignedBeaconBlockAltair)
	assert.Equal(t, true, ok)
}
