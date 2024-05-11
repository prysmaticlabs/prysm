package p2p

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
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
	altairForkEpoch := primitives.Epoch(100)
	BellatrixForkEpoch := primitives.Epoch(200)
	CapellaForkEpoch := primitives.Epoch(300)
	DenebForkEpoch := primitives.Epoch(400)
	ElectraForkEpoch := primitives.Epoch(500)

	bCfg.AltairForkEpoch = altairForkEpoch
	bCfg.BellatrixForkEpoch = BellatrixForkEpoch
	bCfg.CapellaForkEpoch = CapellaForkEpoch
	bCfg.DenebForkEpoch = DenebForkEpoch
	bCfg.ElectraForkEpoch = ElectraForkEpoch
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.AltairForkVersion)] = primitives.Epoch(100)
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.BellatrixForkVersion)] = primitives.Epoch(200)
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.CapellaForkVersion)] = primitives.Epoch(300)
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.DenebForkVersion)] = primitives.Epoch(400)
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.ElectraForkVersion)] = primitives.Epoch(500)
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

	// Capella Fork
	pMessage = GossipTopicMappings(BlockSubnetTopicFormat, CapellaForkEpoch)
	_, ok = pMessage.(*ethpb.SignedBeaconBlockCapella)
	assert.Equal(t, true, ok)

	// Deneb Fork
	pMessage = GossipTopicMappings(BlockSubnetTopicFormat, DenebForkEpoch)
	_, ok = pMessage.(*ethpb.SignedBeaconBlockDeneb)
	assert.Equal(t, true, ok)

	// Electra Fork
	pMessage = GossipTopicMappings(BlockSubnetTopicFormat, ElectraForkEpoch)
	_, ok = pMessage.(*ethpb.SignedBeaconBlockElectra)
	assert.Equal(t, true, ok)
}
