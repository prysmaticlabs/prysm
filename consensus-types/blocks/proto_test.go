package blocks

import (
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func Test_SignedBeaconBlock_Proto(t *testing.T) {

}

func Test_BeaconBlock_Proto(t *testing.T) {

}

func Test_BeaconBlockBody_Proto(t *testing.T) {
	b20 := make([]byte, 20)
	b32 := make([]byte, 32)
	b48 := make([]byte, 48)
	b96 := make([]byte, 96)
	b256 := make([]byte, 256)
	b20[0], b20[5], b20[10] = 'q', 'u', 'x'
	b32[0], b32[5], b32[10] = 'f', 'o', 'o'
	b48[0], b48[5], b48[10] = 'b', 'a', 'r'
	b96[0], b96[5], b96[10] = 'b', 'a', 'z'
	b256[0], b256[5], b256[10] = 'x', 'y', 'z'
	deposits := make([]*eth.Deposit, 16)
	for i := range deposits {
		deposits[i] = &eth.Deposit{}
		deposits[i].Proof = make([][]byte, 33)
		for j := range deposits[i].Proof {
			deposits[i].Proof[j] = b32
		}
		deposits[i].Data = &eth.Deposit_Data{
			PublicKey:             b48,
			WithdrawalCredentials: b32,
			Amount:                128,
			Signature:             b96,
		}
	}
	atts := make([]*eth.Attestation, 128)
	for i := range atts {
		atts[i] = &eth.Attestation{}
		atts[i].Signature = b96
		atts[i].AggregationBits = bitfield.NewBitlist(1)
		atts[i].Data = &eth.AttestationData{
			Slot:            128,
			CommitteeIndex:  128,
			BeaconBlockRoot: b32,
			Source: &eth.Checkpoint{
				Epoch: 128,
				Root:  b32,
			},
			Target: &eth.Checkpoint{
				Epoch: 128,
				Root:  b32,
			},
		}
	}
	proposerSlashing := &eth.ProposerSlashing{
		Header_1: &eth.SignedBeaconBlockHeader{
			Header: &eth.BeaconBlockHeader{
				Slot:          128,
				ProposerIndex: 128,
				ParentRoot:    b32,
				StateRoot:     b32,
				BodyRoot:      b32,
			},
			Signature: b96,
		},
		Header_2: &eth.SignedBeaconBlockHeader{
			Header: &eth.BeaconBlockHeader{
				Slot:          128,
				ProposerIndex: 128,
				ParentRoot:    b32,
				StateRoot:     b32,
				BodyRoot:      b32,
			},
			Signature: b96,
		},
	}
	attesterSlashing := &eth.AttesterSlashing{
		Attestation_1: &eth.IndexedAttestation{
			AttestingIndices: []uint64{1, 2, 8},
			Data: &eth.AttestationData{
				Slot:            128,
				CommitteeIndex:  128,
				BeaconBlockRoot: b32,
				Source: &eth.Checkpoint{
					Epoch: 128,
					Root:  b32,
				},
				Target: &eth.Checkpoint{
					Epoch: 128,
					Root:  b32,
				},
			},
			Signature: b96,
		},
		Attestation_2: &eth.IndexedAttestation{
			AttestingIndices: []uint64{1, 2, 8},
			Data: &eth.AttestationData{
				Slot:            128,
				CommitteeIndex:  128,
				BeaconBlockRoot: b32,
				Source: &eth.Checkpoint{
					Epoch: 128,
					Root:  b32,
				},
				Target: &eth.Checkpoint{
					Epoch: 128,
					Root:  b32,
				},
			},
			Signature: b96,
		},
	}
	voluntaryExit := &eth.SignedVoluntaryExit{
		Exit: &eth.VoluntaryExit{
			Epoch:          128,
			ValidatorIndex: 128,
		},
		Signature: b96,
	}
	syncCommitteeBits := bitfield.NewBitvector512()
	syncCommitteeBits.SetBitAt(1, true)
	syncCommitteeBits.SetBitAt(2, true)
	syncCommitteeBits.SetBitAt(8, true)
	syncAggregate := &eth.SyncAggregate{
		SyncCommitteeBits:      syncCommitteeBits,
		SyncCommitteeSignature: b96,
	}
	executionPayload := &enginev1.ExecutionPayload{
		ParentHash:    b32,
		FeeRecipient:  b20,
		StateRoot:     b32,
		ReceiptsRoot:  b32,
		LogsBloom:     b256,
		PrevRandao:    b32,
		BlockNumber:   128,
		GasLimit:      128,
		GasUsed:       128,
		Timestamp:     128,
		ExtraData:     b32,
		BaseFeePerGas: b32,
		BlockHash:     b32,
		Transactions: [][]byte{
			[]byte("transaction1"),
			[]byte("transaction2"),
			[]byte("transaction8"),
		},
	}
	executionPayloadHeader := &eth.ExecutionPayloadHeader{
		ParentHash:       b32,
		FeeRecipient:     b20,
		StateRoot:        b32,
		ReceiptsRoot:     b32,
		LogsBloom:        b256,
		PrevRandao:       b32,
		BlockNumber:      128,
		GasLimit:         128,
		GasUsed:          128,
		Timestamp:        128,
		ExtraData:        b32,
		BaseFeePerGas:    b32,
		BlockHash:        b32,
		TransactionsRoot: b32,
	}

	t.Run("Phase0", func(t *testing.T) {
		expectedBody := &eth.BeaconBlockBody{
			RandaoReveal: b96,
			Eth1Data: &eth.Eth1Data{
				DepositRoot:  b32,
				DepositCount: 128,
				BlockHash:    b32,
			},
			Graffiti:          b32,
			ProposerSlashings: []*eth.ProposerSlashing{proposerSlashing},
			AttesterSlashings: []*eth.AttesterSlashing{attesterSlashing},
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    []*eth.SignedVoluntaryExit{voluntaryExit},
		}
		body := &BeaconBlockBody{
			version:      version.Phase0,
			randaoReveal: b96,
			eth1Data: &eth.Eth1Data{
				DepositRoot:  b32,
				DepositCount: 128,
				BlockHash:    b32,
			},
			graffiti:          b32,
			proposerSlashings: []*eth.ProposerSlashing{proposerSlashing},
			attesterSlashings: []*eth.AttesterSlashing{attesterSlashing},
			attestations:      atts,
			deposits:          deposits,
			voluntaryExits:    []*eth.SignedVoluntaryExit{voluntaryExit},
		}

		result, err := body.Proto()
		require.NoError(t, err)
		resultBlock, ok := result.(*eth.BeaconBlockBody)
		require.Equal(t, true, ok)
		resultHTR, err := resultBlock.HashTreeRoot()
		require.NoError(t, err)
		expectedHTR, err := expectedBody.HashTreeRoot()
		require.NoError(t, err)
		assert.DeepEqual(t, expectedHTR, resultHTR)
	})
	t.Run("Altair", func(t *testing.T) {
		expectedBody := &eth.BeaconBlockBodyAltair{
			RandaoReveal: b96,
			Eth1Data: &eth.Eth1Data{
				DepositRoot:  b32,
				DepositCount: 128,
				BlockHash:    b32,
			},
			Graffiti:          b32,
			ProposerSlashings: []*eth.ProposerSlashing{proposerSlashing},
			AttesterSlashings: []*eth.AttesterSlashing{attesterSlashing},
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    []*eth.SignedVoluntaryExit{voluntaryExit},
			SyncAggregate:     syncAggregate,
		}
		body := &BeaconBlockBody{
			version:      version.Altair,
			randaoReveal: b96,
			eth1Data: &eth.Eth1Data{
				DepositRoot:  b32,
				DepositCount: 128,
				BlockHash:    b32,
			},
			graffiti:          b32,
			proposerSlashings: []*eth.ProposerSlashing{proposerSlashing},
			attesterSlashings: []*eth.AttesterSlashing{attesterSlashing},
			attestations:      atts,
			deposits:          deposits,
			voluntaryExits:    []*eth.SignedVoluntaryExit{voluntaryExit},
			syncAggregate:     syncAggregate,
		}
		result, err := body.Proto()
		require.NoError(t, err)
		resultBlock, ok := result.(*eth.BeaconBlockBodyAltair)
		require.Equal(t, true, ok)
		resultHTR, err := resultBlock.HashTreeRoot()
		require.NoError(t, err)
		expectedHTR, err := expectedBody.HashTreeRoot()
		require.NoError(t, err)
		assert.DeepEqual(t, expectedHTR, resultHTR)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		expectedBody := &eth.BeaconBlockBodyBellatrix{
			RandaoReveal: b96,
			Eth1Data: &eth.Eth1Data{
				DepositRoot:  b32,
				DepositCount: 128,
				BlockHash:    b32,
			},
			Graffiti:          b32,
			ProposerSlashings: []*eth.ProposerSlashing{proposerSlashing},
			AttesterSlashings: []*eth.AttesterSlashing{attesterSlashing},
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    []*eth.SignedVoluntaryExit{voluntaryExit},
			SyncAggregate:     syncAggregate,
			ExecutionPayload:  executionPayload,
		}
		body := &BeaconBlockBody{
			version:      version.Bellatrix,
			randaoReveal: b96,
			eth1Data: &eth.Eth1Data{
				DepositRoot:  b32,
				DepositCount: 128,
				BlockHash:    b32,
			},
			graffiti:          b32,
			proposerSlashings: []*eth.ProposerSlashing{proposerSlashing},
			attesterSlashings: []*eth.AttesterSlashing{attesterSlashing},
			attestations:      atts,
			deposits:          deposits,
			voluntaryExits:    []*eth.SignedVoluntaryExit{voluntaryExit},
			syncAggregate:     syncAggregate,
			executionPayload:  executionPayload,
		}
		result, err := body.Proto()
		require.NoError(t, err)
		resultBlock, ok := result.(*eth.BeaconBlockBodyBellatrix)
		require.Equal(t, true, ok)
		resultHTR, err := resultBlock.HashTreeRoot()
		require.NoError(t, err)
		expectedHTR, err := expectedBody.HashTreeRoot()
		require.NoError(t, err)
		assert.DeepEqual(t, expectedHTR, resultHTR)
	})
	t.Run("BellatrixBlind", func(t *testing.T) {
		expectedBody := &eth.BlindedBeaconBlockBodyBellatrix{
			RandaoReveal: b96,
			Eth1Data: &eth.Eth1Data{
				DepositRoot:  b32,
				DepositCount: 128,
				BlockHash:    b32,
			},
			Graffiti:               b32,
			ProposerSlashings:      []*eth.ProposerSlashing{proposerSlashing},
			AttesterSlashings:      []*eth.AttesterSlashing{attesterSlashing},
			Attestations:           atts,
			Deposits:               deposits,
			VoluntaryExits:         []*eth.SignedVoluntaryExit{voluntaryExit},
			SyncAggregate:          syncAggregate,
			ExecutionPayloadHeader: executionPayloadHeader,
		}
		body := &BeaconBlockBody{
			version:      version.BellatrixBlind,
			randaoReveal: b96,
			eth1Data: &eth.Eth1Data{
				DepositRoot:  b32,
				DepositCount: 128,
				BlockHash:    b32,
			},
			graffiti:               b32,
			proposerSlashings:      []*eth.ProposerSlashing{proposerSlashing},
			attesterSlashings:      []*eth.AttesterSlashing{attesterSlashing},
			attestations:           atts,
			deposits:               deposits,
			voluntaryExits:         []*eth.SignedVoluntaryExit{voluntaryExit},
			syncAggregate:          syncAggregate,
			executionPayloadHeader: executionPayloadHeader,
		}
		result, err := body.Proto()
		require.NoError(t, err)
		resultBlock, ok := result.(*eth.BlindedBeaconBlockBodyBellatrix)
		require.Equal(t, true, ok)
		resultHTR, err := resultBlock.HashTreeRoot()
		require.NoError(t, err)
		expectedHTR, err := expectedBody.HashTreeRoot()
		require.NoError(t, err)
		assert.DeepEqual(t, expectedHTR, resultHTR)
	})
}
