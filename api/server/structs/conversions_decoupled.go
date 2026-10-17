package structs

import (
	"fmt"

	beaconState "github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	eth "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// SignedBeaconBlockDecoupledFromConsensus converts a signed Decoupled block.
func SignedBeaconBlockDecoupledFromConsensus(b *eth.SignedBeaconBlockDecoupled) (*SignedBeaconBlockDecoupled, error) {
	block, err := BeaconBlockDecoupledFromConsensus(b.Block)
	if err != nil {
		return nil, err
	}
	return &SignedBeaconBlockDecoupled{
		Message:   block,
		Signature: hexutil.Encode(b.Signature),
	}, nil
}

// BeaconBlockDecoupledFromConsensus converts a Decoupled block.
func BeaconBlockDecoupledFromConsensus(b *eth.BeaconBlockDecoupled) (*BeaconBlockDecoupled, error) {
	payloadAttestations := make([]*PayloadAttestation, len(b.Body.PayloadAttestations))
	for i, pa := range b.Body.PayloadAttestations {
		payloadAttestations[i] = PayloadAttestationFromConsensus(pa)
	}
	return &BeaconBlockDecoupled{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BeaconBlockBodyDecoupled{
			RandaoReveal:              hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data:                  Eth1DataFromConsensus(b.Body.Eth1Data),
			Graffiti:                  hexutil.Encode(b.Body.Graffiti),
			ProposerSlashings:         ProposerSlashingsFromConsensus(b.Body.ProposerSlashings),
			AttesterSlashings:         AttesterSlashingsGloasFromConsensus(b.Body.AttesterSlashings),
			FinalityAttestations:      IndexedAttsDecoupledFromConsensus(b.Body.Attestations),
			Deposits:                  DepositsFromConsensus(b.Body.Deposits),
			VoluntaryExits:            SignedExitsFromConsensus(b.Body.VoluntaryExits),
			SyncAggregate:             SyncAggregateFromConsensus(b.Body.SyncAggregate),
			BLSToExecutionChanges:     SignedBLSChangesFromConsensus(b.Body.BlsToExecutionChanges),
			SignedExecutionPayloadBid: SignedExecutionPayloadBidFromConsensus(b.Body.SignedExecutionPayloadBid),
			PayloadAttestations:       payloadAttestations,
			ParentExecutionRequests:   ExecutionRequestsGloasFromConsensus(b.Body.ParentExecutionRequests),
			AvailableAttestations:     AvailableAttsFromConsensus(b.Body.AvailableAttestations),
			SupportVotes:              hexutil.Encode(b.Body.SupportVotes),
		},
	}, nil
}

// IndexedAttsDecoupledFromConsensus converts FG rows.
func IndexedAttsDecoupledFromConsensus(src []*eth.IndexedAttestationDecoupled) []*IndexedAttestationDecoupled {
	atts := make([]*IndexedAttestationDecoupled, len(src))
	for i, a := range src {
		indices := make([]string, len(a.AttestingIndices))
		for j, idx := range a.AttestingIndices {
			indices[j] = fmt.Sprintf("%d", idx)
		}
		atts[i] = &IndexedAttestationDecoupled{
			AttestingIndices: indices,
			Data: &AttestationDataDecoupled{
				Height:         fmt.Sprintf("%d", a.Data.Height),
				Target:         hexutil.Encode(a.Data.Target),
				Timeout:        a.Data.Timeout,
				FinalityHeight: fmt.Sprintf("%d", a.Data.FinalityHeight),
				FinalityTarget: hexutil.Encode(a.Data.FinalityTarget),
			},
			Signature: hexutil.Encode(a.Signature),
		}
	}
	return atts
}

// AvailableAttsFromConsensus converts AC aggregates.
func AvailableAttsFromConsensus(src []*eth.AvailableAttestation) []*AvailableAttestation {
	atts := make([]*AvailableAttestation, len(src))
	for i, a := range src {
		atts[i] = &AvailableAttestation{
			AggregationBits: hexutil.Encode(a.AggregationBits),
			Data: &AvailableAttestationData{
				Slot:            fmt.Sprintf("%d", a.Data.Slot),
				BeaconBlockRoot: hexutil.Encode(a.Data.BeaconBlockRoot),
			},
			Signature: hexutil.Encode(a.Signature),
		}
	}
	return atts
}

// CheckpointDecoupledFromConsensus converts a checkpoint.
func CheckpointDecoupledFromConsensus(c *eth.CheckpointDecoupled) *CheckpointDecoupled {
	if c == nil {
		return nil
	}
	return &CheckpointDecoupled{
		Slot: fmt.Sprintf("%d", c.Slot),
		Root: hexutil.Encode(c.Root),
	}
}

// PendingFinalityFromConsensus converts pending finality pairs.
func PendingFinalityFromConsensus(src []*eth.PendingFinality) []*PendingFinality {
	rows := make([]*PendingFinality, len(src))
	for i, p := range src {
		rows[i] = &PendingFinality{
			Height:        fmt.Sprintf("%d", p.Height),
			Target:        CheckpointDecoupledFromConsensus(p.Target),
			Participation: hexutil.Encode(p.Participation),
		}
	}
	return rows
}

// BeaconStateDecoupledFromConsensus converts a Decoupled state.
func BeaconStateDecoupledFromConsensus(st beaconState.BeaconState) (*BeaconStateDecoupled, error) {
	gloas, err := BeaconStateGloasFromConsensus(st)
	if err != nil {
		return nil, err
	}
	height, err := st.Height()
	if err != nil {
		return nil, err
	}
	entry, err := st.Entry()
	if err != nil {
		return nil, err
	}
	targetParticipation, err := st.TargetParticipation()
	if err != nil {
		return nil, err
	}
	progress, err := st.Progress()
	if err != nil {
		return nil, err
	}
	justifiedHeight, err := st.JustifiedHeight()
	if err != nil {
		return nil, err
	}
	justified, err := st.Justified()
	if err != nil {
		return nil, err
	}
	finalizedHeight, err := st.FinalizedHeight()
	if err != nil {
		return nil, err
	}
	finalized, err := st.Finalized()
	if err != nil {
		return nil, err
	}
	pending, err := st.PendingFinality()
	if err != nil {
		return nil, err
	}
	return &BeaconStateDecoupled{
		BeaconStateGloas:    *gloas,
		Height:              fmt.Sprintf("%d", height),
		Entry:               CheckpointDecoupledFromConsensus(entry),
		TargetParticipation: hexutil.Encode(targetParticipation),
		Progress:            hexutil.Encode(progress),
		JustifiedHeight:     fmt.Sprintf("%d", justifiedHeight),
		Justified:           CheckpointDecoupledFromConsensus(justified),
		FinalizedHeight:     fmt.Sprintf("%d", finalizedHeight),
		Finalized:           CheckpointDecoupledFromConsensus(finalized),
		PendingFinality:     PendingFinalityFromConsensus(pending),
	}, nil
}
