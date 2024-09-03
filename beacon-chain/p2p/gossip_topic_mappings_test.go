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
		if _, ok := m[reflect.TypeOf(v())]; ok {
			t.Errorf("%T is duplicated in the topic mapping", v)
		}
		m[reflect.TypeOf(v)] = true
	}
}

func TestGossipTopicMappings_CorrectType(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	bCfg := params.BeaconConfig().Copy()
	altairForkEpoch := primitives.Epoch(100)
	bellatrixForkEpoch := primitives.Epoch(200)
	capellaForkEpoch := primitives.Epoch(300)
	denebForkEpoch := primitives.Epoch(400)
	electraForkEpoch := primitives.Epoch(500)

	bCfg.AltairForkEpoch = altairForkEpoch
	bCfg.BellatrixForkEpoch = bellatrixForkEpoch
	bCfg.CapellaForkEpoch = capellaForkEpoch
	bCfg.DenebForkEpoch = denebForkEpoch
	bCfg.ElectraForkEpoch = electraForkEpoch
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.AltairForkVersion)] = primitives.Epoch(100)
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.BellatrixForkVersion)] = primitives.Epoch(200)
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.CapellaForkVersion)] = primitives.Epoch(300)
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.DenebForkVersion)] = primitives.Epoch(400)
	bCfg.ForkVersionSchedule[bytesutil.ToBytes4(bCfg.ElectraForkVersion)] = primitives.Epoch(500)
	params.OverrideBeaconConfig(bCfg)

	// Phase 0
	pMessage := GossipTopicMappings(BeaconBlockSubnetTopicFormat, 0)
	_, ok := pMessage.(*ethpb.SignedBeaconBlock)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(BeaconAttestationSubnetTopicFormat, 0)
	_, ok = pMessage.(*ethpb.Attestation)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(AttesterSlashingSubnetTopicFormat, 0)
	_, ok = pMessage.(*ethpb.AttesterSlashing)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(BeaconAggregateAndProofSubnetTopicFormat, 0)
	_, ok = pMessage.(*ethpb.SignedAggregateAttestationAndProof)
	assert.Equal(t, true, ok)

	// Altair Fork
	pMessage = GossipTopicMappings(BeaconBlockSubnetTopicFormat, altairForkEpoch)
	_, ok = pMessage.(*ethpb.SignedBeaconBlockAltair)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(BeaconAttestationSubnetTopicFormat, altairForkEpoch)
	_, ok = pMessage.(*ethpb.Attestation)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(AttesterSlashingSubnetTopicFormat, altairForkEpoch)
	_, ok = pMessage.(*ethpb.AttesterSlashing)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(BeaconAggregateAndProofSubnetTopicFormat, altairForkEpoch)
	_, ok = pMessage.(*ethpb.SignedAggregateAttestationAndProof)
	assert.Equal(t, true, ok)

	// Bellatrix Fork
	pMessage = GossipTopicMappings(BeaconBlockSubnetTopicFormat, bellatrixForkEpoch)
	_, ok = pMessage.(*ethpb.SignedBeaconBlockBellatrix)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(BeaconAttestationSubnetTopicFormat, bellatrixForkEpoch)
	_, ok = pMessage.(*ethpb.Attestation)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(AttesterSlashingSubnetTopicFormat, bellatrixForkEpoch)
	_, ok = pMessage.(*ethpb.AttesterSlashing)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(BeaconAggregateAndProofSubnetTopicFormat, bellatrixForkEpoch)
	_, ok = pMessage.(*ethpb.SignedAggregateAttestationAndProof)
	assert.Equal(t, true, ok)

	// Capella Fork
	pMessage = GossipTopicMappings(BeaconBlockSubnetTopicFormat, capellaForkEpoch)
	_, ok = pMessage.(*ethpb.SignedBeaconBlockCapella)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(BeaconAttestationSubnetTopicFormat, capellaForkEpoch)
	_, ok = pMessage.(*ethpb.Attestation)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(AttesterSlashingSubnetTopicFormat, capellaForkEpoch)
	_, ok = pMessage.(*ethpb.AttesterSlashing)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(BeaconAggregateAndProofSubnetTopicFormat, capellaForkEpoch)
	_, ok = pMessage.(*ethpb.SignedAggregateAttestationAndProof)
	assert.Equal(t, true, ok)

	// Deneb Fork
	pMessage = GossipTopicMappings(BeaconBlockSubnetTopicFormat, denebForkEpoch)
	_, ok = pMessage.(*ethpb.SignedBeaconBlockDeneb)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(BeaconAttestationSubnetTopicFormat, denebForkEpoch)
	_, ok = pMessage.(*ethpb.Attestation)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(AttesterSlashingSubnetTopicFormat, denebForkEpoch)
	_, ok = pMessage.(*ethpb.AttesterSlashing)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(BeaconAggregateAndProofSubnetTopicFormat, denebForkEpoch)
	_, ok = pMessage.(*ethpb.SignedAggregateAttestationAndProof)
	assert.Equal(t, true, ok)

	// Electra Fork
	pMessage = GossipTopicMappings(BeaconBlockSubnetTopicFormat, electraForkEpoch)
	_, ok = pMessage.(*ethpb.SignedBeaconBlockElectra)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(BeaconAttestationSubnetTopicFormat, electraForkEpoch)
	_, ok = pMessage.(*ethpb.AttestationElectra)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(AttesterSlashingSubnetTopicFormat, electraForkEpoch)
	_, ok = pMessage.(*ethpb.AttesterSlashingElectra)
	assert.Equal(t, true, ok)
	pMessage = GossipTopicMappings(BeaconAggregateAndProofSubnetTopicFormat, electraForkEpoch)
	_, ok = pMessage.(*ethpb.SignedAggregateAttestationAndProofElectra)
	assert.Equal(t, true, ok)
}
