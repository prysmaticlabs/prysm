package random

import (
	"crypto/rand"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// SignedBeaconBlock creates a random SignedBeaconBlockEPBS for testing purposes.
func SignedBeaconBlock(t *testing.T) *ethpb.SignedBeaconBlockEpbs {
	return &ethpb.SignedBeaconBlockEpbs{
		Block:     BeaconBlock(t),
		Signature: randomBytes(96, t),
	}
}

// BeaconBlock creates a random BeaconBlockEPBS for testing purposes.
func BeaconBlock(t *testing.T) *ethpb.BeaconBlockEpbs {
	return &ethpb.BeaconBlockEpbs{
		Slot:          primitives.Slot(randomUint64(t)),
		ProposerIndex: primitives.ValidatorIndex(randomUint64(t)),
		ParentRoot:    randomBytes(32, t),
		StateRoot:     randomBytes(32, t),
		Body:          BeaconBlockBody(t),
	}
}

// BeaconBlockBody creates a random BeaconBlockBodyEPBS for testing purposes.
func BeaconBlockBody(t *testing.T) *ethpb.BeaconBlockBodyEpbs {
	return &ethpb.BeaconBlockBodyEpbs{
		RandaoReveal:                 randomBytes(96, t),
		Eth1Data:                     nil,
		Graffiti:                     nil,
		ProposerSlashings:            nil,
		AttesterSlashings:            nil,
		Attestations:                 nil,
		Deposits:                     nil,
		VoluntaryExits:               nil,
		SyncAggregate:                nil,
		BlsToExecutionChanges:        nil,
		SignedExecutionPayloadHeader: SignedExecutionPayloadHeader(t),
		PayloadAttestations: []*ethpb.PayloadAttestation{
			PayloadAttestation(t), PayloadAttestation(t), PayloadAttestation(t), PayloadAttestation(t),
		},
	}
}

// BeaconState creates a random BeaconStateEPBS for testing purposes.
func BeaconState(t *testing.T) *ethpb.BeaconStateEPBS {
	return &ethpb.BeaconStateEPBS{
		GenesisTime:           randomUint64(t),
		GenesisValidatorsRoot: randomBytes(32, t),
		Slot:                  primitives.Slot(randomUint64(t)),
		Fork:                  nil,
		LatestBlockHeader:     nil,
		BlockRoots:            nil,
		StateRoots:            nil,
		HistoricalRoots:       nil,
		Eth1Data:              &ethpb.Eth1Data{},
		Eth1DataVotes:         []*ethpb.Eth1Data{},
		Eth1DepositIndex:      randomUint64(t),
		Validators: []*ethpb.Validator{
			{
				PublicKey:                  randomBytes(48, t),
				WithdrawalCredentials:      randomBytes(32, t),
				EffectiveBalance:           randomUint64(t),
				ActivationEligibilityEpoch: primitives.Epoch(randomUint64(t)),
				ActivationEpoch:            primitives.Epoch(randomUint64(t)),
				ExitEpoch:                  primitives.Epoch(randomUint64(t)),
				WithdrawableEpoch:          primitives.Epoch(randomUint64(t)),
			},
		},
		Balances:                      []uint64{randomUint64(t)},
		RandaoMixes:                   nil,
		Slashings:                     nil,
		PreviousEpochParticipation:    nil,
		CurrentEpochParticipation:     nil,
		JustificationBits:             nil,
		PreviousJustifiedCheckpoint:   nil,
		CurrentJustifiedCheckpoint:    nil,
		FinalizedCheckpoint:           nil,
		InactivityScores:              nil,
		CurrentSyncCommittee:          nil,
		NextSyncCommittee:             nil,
		NextWithdrawalIndex:           randomUint64(t),
		NextWithdrawalValidatorIndex:  primitives.ValidatorIndex(randomUint64(t)),
		HistoricalSummaries:           nil,
		PreviousInclusionListProposer: primitives.ValidatorIndex(randomUint64(t)),
		PreviousInclusionListSlot:     primitives.Slot(randomUint64(t)),
		LatestInclusionListProposer:   primitives.ValidatorIndex(randomUint64(t)),
		LatestInclusionListSlot:       primitives.Slot(randomUint64(t)),
		LatestBlockHash:               randomBytes(32, t),
		LatestFullSlot:                primitives.Slot(randomUint64(t)),
		ExecutionPayloadHeader: &enginev1.ExecutionPayloadHeaderEPBS{
			ParentBlockHash:        randomBytes(32, t),
			ParentBlockRoot:        randomBytes(32, t),
			BlockHash:              randomBytes(32, t),
			BuilderIndex:           primitives.ValidatorIndex(randomUint64(t)),
			Slot:                   primitives.Slot(randomUint64(t)),
			Value:                  randomUint64(t),
			BlobKzgCommitmentsRoot: randomBytes(32, t),
		},
		LatestWithdrawalsRoot: randomBytes(32, t),
	}
}

// SignedExecutionPayloadHeader creates a random SignedExecutionPayloadHeader for testing purposes.
func SignedExecutionPayloadHeader(t *testing.T) *enginev1.SignedExecutionPayloadHeader {
	return &enginev1.SignedExecutionPayloadHeader{
		Message:   ExecutionPayloadHeader(t),
		Signature: randomBytes(96, t),
	}
}

// ExecutionPayloadHeader creates a random ExecutionPayloadHeaderEPBS for testing.
func ExecutionPayloadHeader(t *testing.T) *enginev1.ExecutionPayloadHeaderEPBS {
	return &enginev1.ExecutionPayloadHeaderEPBS{
		ParentBlockHash:        randomBytes(32, t),
		ParentBlockRoot:        randomBytes(32, t),
		BlockHash:              randomBytes(32, t),
		BuilderIndex:           primitives.ValidatorIndex(randomUint64(t)),
		Slot:                   primitives.Slot(randomUint64(t)),
		Value:                  randomUint64(t),
		BlobKzgCommitmentsRoot: randomBytes(32, t),
	}
}

// PayloadAttestation creates a random PayloadAttestation for testing purposes.
func PayloadAttestation(t *testing.T) *ethpb.PayloadAttestation {
	bv := bitfield.NewBitvector512()
	b := randomBytes(64, t)
	copy(bv[:], b)
	return &ethpb.PayloadAttestation{
		AggregationBits: bv,
		Data:            PayloadAttestationData(t),
		Signature:       randomBytes(96, t),
	}
}

// PayloadAttestationData generates a random PayloadAttestationData for testing purposes.
func PayloadAttestationData(t *testing.T) *ethpb.PayloadAttestationData {
	// Generate a random BeaconBlockRoot
	randomBytes := make([]byte, fieldparams.RootLength)
	_, err := rand.Read(randomBytes)
	if err != nil {
		t.Fatalf("Failed to generate random BeaconBlockRoot: %v", err)
	}

	// Generate a random Slot value
	randomSlot, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		t.Fatalf("Failed to generate random Slot: %v", err)
	}

	payloadStatuses := []primitives.PTCStatus{
		primitives.PAYLOAD_ABSENT,
		primitives.PAYLOAD_PRESENT,
		primitives.PAYLOAD_WITHHELD,
	}
	// Select a random PayloadStatus
	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(payloadStatuses))))
	if err != nil {
		t.Fatalf("Failed to select random PayloadStatus: %v", err)
	}
	randomPayloadStatus := payloadStatuses[index.Int64()]

	return &ethpb.PayloadAttestationData{
		BeaconBlockRoot: randomBytes,
		Slot:            primitives.Slot(randomSlot.Uint64()),
		PayloadStatus:   randomPayloadStatus,
	}
}

// SignedExecutionPayloadEnvelope creates a random SignedExecutionPayloadEnvelope for testing purposes.
func SignedExecutionPayloadEnvelope(t *testing.T) *enginev1.SignedExecutionPayloadEnvelope {
	return &enginev1.SignedExecutionPayloadEnvelope{
		Message:   ExecutionPayloadEnvelope(t),
		Signature: randomBytes(96, t),
	}
}

// ExecutionPayloadEnvelope creates a random ExecutionPayloadEnvelope for testing purposes.
func ExecutionPayloadEnvelope(t *testing.T) *enginev1.ExecutionPayloadEnvelope {
	withheld := randomUint64(t)%2 == 0
	return &enginev1.ExecutionPayloadEnvelope{
		Payload:                    ExecutionPayload(t),
		BuilderIndex:               primitives.ValidatorIndex(randomUint64(t)),
		BeaconBlockRoot:            randomBytes(32, t),
		BlobKzgCommitments:         [][]byte{randomBytes(48, t), randomBytes(48, t), randomBytes(48, t)},
		InclusionListProposerIndex: primitives.ValidatorIndex(randomUint64(t)),
		InclusionListSlot:          primitives.Slot(randomUint64(t)),
		InclusionListSignature:     randomBytes(96, t),
		PayloadWithheld:            withheld,
		StateRoot:                  randomBytes(32, t),
	}
}

// ExecutionPayload creates a random ExecutionPayloadEPBS for testing purposes.
func ExecutionPayload(t *testing.T) *enginev1.ExecutionPayloadEPBS {
	return &enginev1.ExecutionPayloadEPBS{
		ParentHash:    randomBytes(32, t),
		FeeRecipient:  randomBytes(20, t),
		StateRoot:     randomBytes(32, t),
		ReceiptsRoot:  randomBytes(32, t),
		LogsBloom:     randomBytes(256, t),
		PrevRandao:    randomBytes(32, t),
		BlockNumber:   randomUint64(t),
		GasLimit:      randomUint64(t),
		GasUsed:       randomUint64(t),
		Timestamp:     randomUint64(t),
		ExtraData:     randomBytes(32, t),
		BaseFeePerGas: randomBytes(32, t),
		BlockHash:     randomBytes(32, t),
		Transactions:  [][]byte{randomBytes(32, t), randomBytes(32, t), randomBytes(32, t)},
		Withdrawals: []*enginev1.Withdrawal{
			{
				Index:          randomUint64(t),
				ValidatorIndex: primitives.ValidatorIndex(randomUint64(t)),
				Address:        randomBytes(20, t),
				Amount:         randomUint64(t),
			},
		},
		BlobGasUsed:          randomUint64(t),
		ExcessBlobGas:        randomUint64(t),
		InclusionListSummary: [][]byte{randomBytes(20, t), randomBytes(20, t), randomBytes(20, t)},
	}
}

// InclusionList creates a random InclusionList for testing purposes.
func InclusionList(t *testing.T) *enginev1.InclusionList {
	return &enginev1.InclusionList{
		SignedSummary: &enginev1.SignedInclusionListSummary{
			Message:   InclusionSummary(t),
			Signature: randomBytes(96, t),
		},
		ParentBlockHash: randomBytes(32, t),
		Transactions: [][]byte{
			randomBytes(123, t),
			randomBytes(456, t),
			randomBytes(789, t),
			randomBytes(1011, t),
		},
	}
}

// InclusionSummary creates a random InclusionListSummary for testing purposes.
func InclusionSummary(t *testing.T) *enginev1.InclusionListSummary {
	return &enginev1.InclusionListSummary{
		ProposerIndex: primitives.ValidatorIndex(randomUint64(t)),
		Slot:          primitives.Slot(randomUint64(t)),
		Summary: [][]byte{
			randomBytes(20, t),
			randomBytes(20, t),
			randomBytes(20, t),
			randomBytes(20, t),
		},
	}
}

func randomBytes(n int, t *testing.T) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		t.Fatalf("Failed to generate random bytes: %v", err)
	}
	return b
}

func randomUint64(t *testing.T) uint64 {
	var num uint64
	b := randomBytes(8, t)
	num = binary.BigEndian.Uint64(b)
	return num
}
