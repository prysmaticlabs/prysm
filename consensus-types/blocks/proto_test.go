package blocks

import (
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

type fields struct {
	b20               []byte
	b32               []byte
	b48               []byte
	b96               []byte
	b256              []byte
	deposits          []*eth.Deposit
	atts              []*eth.Attestation
	proposerSlashings []*eth.ProposerSlashing
	attesterSlashings []*eth.AttesterSlashing
	voluntaryExits    []*eth.SignedVoluntaryExit
	syncAggregate     *eth.SyncAggregate
	execPayload       *enginev1.ExecutionPayload
	execPayloadHeader *enginev1.ExecutionPayloadHeader
}

func Test_SignedBeaconBlock_Proto(t *testing.T) {
	f := getFields()

	t.Run("Phase0", func(t *testing.T) {
		expectedBlock := &eth.SignedBeaconBlock{
			Block: &eth.BeaconBlock{
				Slot:          128,
				ProposerIndex: 128,
				ParentRoot:    f.b32,
				StateRoot:     f.b32,
				Body:          bodyPbPhase0(),
			},
			Signature: f.b96,
		}
		block := &SignedBeaconBlock{
			version: version.Phase0,
			block: &BeaconBlock{
				version:       version.Phase0,
				slot:          128,
				proposerIndex: 128,
				parentRoot:    f.b32,
				stateRoot:     f.b32,
				body:          bodyPhase0(),
			},
			signature: f.b96,
		}

		result, err := block.Proto()
		require.NoError(t, err)
		resultBlock, ok := result.(*eth.SignedBeaconBlock)
		require.Equal(t, true, ok)
		resultHTR, err := resultBlock.HashTreeRoot()
		require.NoError(t, err)
		expectedHTR, err := expectedBlock.HashTreeRoot()
		require.NoError(t, err)
		assert.DeepEqual(t, expectedHTR, resultHTR)
	})
	t.Run("Altair", func(t *testing.T) {
		expectedBlock := &eth.SignedBeaconBlockAltair{
			Block: &eth.BeaconBlockAltair{
				Slot:          128,
				ProposerIndex: 128,
				ParentRoot:    f.b32,
				StateRoot:     f.b32,
				Body:          bodyPbAltair(),
			},
			Signature: f.b96,
		}
		block := &SignedBeaconBlock{
			version: version.Altair,
			block: &BeaconBlock{
				version:       version.Altair,
				slot:          128,
				proposerIndex: 128,
				parentRoot:    f.b32,
				stateRoot:     f.b32,
				body:          bodyAltair(),
			},
			signature: f.b96,
		}

		result, err := block.Proto()
		require.NoError(t, err)
		resultBlock, ok := result.(*eth.SignedBeaconBlockAltair)
		require.Equal(t, true, ok)
		resultHTR, err := resultBlock.HashTreeRoot()
		require.NoError(t, err)
		expectedHTR, err := expectedBlock.HashTreeRoot()
		require.NoError(t, err)
		assert.DeepEqual(t, expectedHTR, resultHTR)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		expectedBlock := &eth.SignedBeaconBlockBellatrix{
			Block: &eth.BeaconBlockBellatrix{
				Slot:          128,
				ProposerIndex: 128,
				ParentRoot:    f.b32,
				StateRoot:     f.b32,
				Body:          bodyPbBellatrix(),
			},
			Signature: f.b96,
		}
		block := &SignedBeaconBlock{
			version: version.Bellatrix,
			block: &BeaconBlock{
				version:       version.Bellatrix,
				slot:          128,
				proposerIndex: 128,
				parentRoot:    f.b32,
				stateRoot:     f.b32,
				body:          bodyBellatrix(),
			},
			signature: f.b96,
		}

		result, err := block.Proto()
		require.NoError(t, err)
		resultBlock, ok := result.(*eth.SignedBeaconBlockBellatrix)
		require.Equal(t, true, ok)
		resultHTR, err := resultBlock.HashTreeRoot()
		require.NoError(t, err)
		expectedHTR, err := expectedBlock.HashTreeRoot()
		require.NoError(t, err)
		assert.DeepEqual(t, expectedHTR, resultHTR)
	})
	t.Run("BellatrixBlind", func(t *testing.T) {
		expectedBlock := &eth.SignedBlindedBeaconBlockBellatrix{
			Block: &eth.BlindedBeaconBlockBellatrix{
				Slot:          128,
				ProposerIndex: 128,
				ParentRoot:    f.b32,
				StateRoot:     f.b32,
				Body:          bodyPbBlindedBellatrix(),
			},
			Signature: f.b96,
		}
		block := &SignedBeaconBlock{
			version: version.Bellatrix,
			block: &BeaconBlock{
				version:       version.Bellatrix,
				slot:          128,
				proposerIndex: 128,
				parentRoot:    f.b32,
				stateRoot:     f.b32,
				body:          bodyBlindedBellatrix(),
			},
			signature: f.b96,
		}

		result, err := block.Proto()
		require.NoError(t, err)
		resultBlock, ok := result.(*eth.SignedBlindedBeaconBlockBellatrix)
		require.Equal(t, true, ok)
		resultHTR, err := resultBlock.HashTreeRoot()
		require.NoError(t, err)
		expectedHTR, err := expectedBlock.HashTreeRoot()
		require.NoError(t, err)
		assert.DeepEqual(t, expectedHTR, resultHTR)
	})
}

func Test_BeaconBlock_Proto(t *testing.T) {
	f := getFields()

	t.Run("Phase0", func(t *testing.T) {
		expectedBlock := &eth.BeaconBlock{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.b32,
			StateRoot:     f.b32,
			Body:          bodyPbPhase0(),
		}
		block := &BeaconBlock{
			version:       version.Phase0,
			slot:          128,
			proposerIndex: 128,
			parentRoot:    f.b32,
			stateRoot:     f.b32,
			body:          bodyPhase0(),
		}

		result, err := block.Proto()
		require.NoError(t, err)
		resultBlock, ok := result.(*eth.BeaconBlock)
		require.Equal(t, true, ok)
		resultHTR, err := resultBlock.HashTreeRoot()
		require.NoError(t, err)
		expectedHTR, err := expectedBlock.HashTreeRoot()
		require.NoError(t, err)
		assert.DeepEqual(t, expectedHTR, resultHTR)
	})
	t.Run("Altair", func(t *testing.T) {
		expectedBlock := &eth.BeaconBlockAltair{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.b32,
			StateRoot:     f.b32,
			Body:          bodyPbAltair(),
		}
		block := &BeaconBlock{
			version:       version.Altair,
			slot:          128,
			proposerIndex: 128,
			parentRoot:    f.b32,
			stateRoot:     f.b32,
			body:          bodyAltair(),
		}

		result, err := block.Proto()
		require.NoError(t, err)
		resultBlock, ok := result.(*eth.BeaconBlockAltair)
		require.Equal(t, true, ok)
		resultHTR, err := resultBlock.HashTreeRoot()
		require.NoError(t, err)
		expectedHTR, err := expectedBlock.HashTreeRoot()
		require.NoError(t, err)
		assert.DeepEqual(t, expectedHTR, resultHTR)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		expectedBlock := &eth.BeaconBlockBellatrix{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.b32,
			StateRoot:     f.b32,
			Body:          bodyPbBellatrix(),
		}
		block := &BeaconBlock{
			version:       version.Bellatrix,
			slot:          128,
			proposerIndex: 128,
			parentRoot:    f.b32,
			stateRoot:     f.b32,
			body:          bodyBellatrix(),
		}

		result, err := block.Proto()
		require.NoError(t, err)
		resultBlock, ok := result.(*eth.BeaconBlockBellatrix)
		require.Equal(t, true, ok)
		resultHTR, err := resultBlock.HashTreeRoot()
		require.NoError(t, err)
		expectedHTR, err := expectedBlock.HashTreeRoot()
		require.NoError(t, err)
		assert.DeepEqual(t, expectedHTR, resultHTR)
	})
	t.Run("BellatrixBlind", func(t *testing.T) {
		expectedBlock := &eth.BlindedBeaconBlockBellatrix{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.b32,
			StateRoot:     f.b32,
			Body:          bodyPbBlindedBellatrix(),
		}
		block := &BeaconBlock{
			version:       version.Bellatrix,
			slot:          128,
			proposerIndex: 128,
			parentRoot:    f.b32,
			stateRoot:     f.b32,
			body:          bodyBlindedBellatrix(),
		}

		result, err := block.Proto()
		require.NoError(t, err)
		resultBlock, ok := result.(*eth.BlindedBeaconBlockBellatrix)
		require.Equal(t, true, ok)
		resultHTR, err := resultBlock.HashTreeRoot()
		require.NoError(t, err)
		expectedHTR, err := expectedBlock.HashTreeRoot()
		require.NoError(t, err)
		assert.DeepEqual(t, expectedHTR, resultHTR)
	})
}

func Test_BeaconBlockBody_Proto(t *testing.T) {
	t.Run("Phase0", func(t *testing.T) {
		expectedBody := bodyPbPhase0()
		body := bodyPhase0()

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
		expectedBody := bodyPbAltair()
		body := bodyAltair()
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
		expectedBody := bodyPbBellatrix()
		body := bodyBellatrix()
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
		expectedBody := bodyPbBlindedBellatrix()
		body := bodyBlindedBellatrix()
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

func Test_initSignedBlockFromProtoPhase0(t *testing.T) {
	f := getFields()
	expectedBlock := &eth.SignedBeaconBlock{
		Block: &eth.BeaconBlock{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.b32,
			StateRoot:     f.b32,
			Body:          bodyPbPhase0(),
		},
		Signature: f.b96,
	}
	resultBlock, err := initSignedBlockFromProtoPhase0(expectedBlock)
	require.NoError(t, err)
	resultHTR, err := resultBlock.block.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
	assert.DeepEqual(t, expectedBlock.Signature, resultBlock.signature)
}

func Test_initSignedBlockFromProtoAltair(t *testing.T) {
	f := getFields()
	expectedBlock := &eth.SignedBeaconBlockAltair{
		Block: &eth.BeaconBlockAltair{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.b32,
			StateRoot:     f.b32,
			Body:          bodyPbAltair(),
		},
		Signature: f.b96,
	}
	resultBlock, err := initSignedBlockFromProtoAltair(expectedBlock)
	require.NoError(t, err)
	resultHTR, err := resultBlock.block.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
	assert.DeepEqual(t, expectedBlock.Signature, resultBlock.signature)
}

func Test_initSignedBlockFromProtoBellatrix(t *testing.T) {
	f := getFields()
	expectedBlock := &eth.SignedBeaconBlockBellatrix{
		Block: &eth.BeaconBlockBellatrix{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.b32,
			StateRoot:     f.b32,
			Body:          bodyPbBellatrix(),
		},
		Signature: f.b96,
	}
	resultBlock, err := initSignedBlockFromProtoBellatrix(expectedBlock)
	require.NoError(t, err)
	resultHTR, err := resultBlock.block.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
	assert.DeepEqual(t, expectedBlock.Signature, resultBlock.signature)
}

func Test_initBlindedSignedBlockFromProtoBellatrix(t *testing.T) {
	f := getFields()
	expectedBlock := &eth.SignedBlindedBeaconBlockBellatrix{
		Block: &eth.BlindedBeaconBlockBellatrix{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.b32,
			StateRoot:     f.b32,
			Body:          bodyPbBlindedBellatrix(),
		},
		Signature: f.b96,
	}
	resultBlock, err := initBlindedSignedBlockFromProtoBellatrix(expectedBlock)
	require.NoError(t, err)
	resultHTR, err := resultBlock.block.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
	assert.DeepEqual(t, expectedBlock.Signature, resultBlock.signature)
}

func Test_initBlockFromProtoPhase0(t *testing.T) {
	f := getFields()
	expectedBlock := &eth.BeaconBlock{
		Slot:          128,
		ProposerIndex: 128,
		ParentRoot:    f.b32,
		StateRoot:     f.b32,
		Body:          bodyPbPhase0(),
	}
	resultBlock, err := initBlockFromProtoPhase0(expectedBlock)
	require.NoError(t, err)
	resultHTR, err := resultBlock.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBlock.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
}

func Test_initBlockFromProtoAltair(t *testing.T) {
	f := getFields()
	expectedBlock := &eth.BeaconBlockAltair{
		Slot:          128,
		ProposerIndex: 128,
		ParentRoot:    f.b32,
		StateRoot:     f.b32,
		Body:          bodyPbAltair(),
	}
	resultBlock, err := initBlockFromProtoAltair(expectedBlock)
	require.NoError(t, err)
	resultHTR, err := resultBlock.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBlock.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
}

func Test_initBlockFromProtoBellatrix(t *testing.T) {
	f := getFields()
	expectedBlock := &eth.BeaconBlockBellatrix{
		Slot:          128,
		ProposerIndex: 128,
		ParentRoot:    f.b32,
		StateRoot:     f.b32,
		Body:          bodyPbBellatrix(),
	}
	resultBlock, err := initBlockFromProtoBellatrix(expectedBlock)
	require.NoError(t, err)
	resultHTR, err := resultBlock.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBlock.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
}

func Test_initBlockFromProtoBlindedBellatrix(t *testing.T) {
	f := getFields()
	expectedBlock := &eth.BlindedBeaconBlockBellatrix{
		Slot:          128,
		ProposerIndex: 128,
		ParentRoot:    f.b32,
		StateRoot:     f.b32,
		Body:          bodyPbBlindedBellatrix(),
	}
	resultBlock, err := initBlindedBlockFromProtoBellatrix(expectedBlock)
	require.NoError(t, err)
	resultHTR, err := resultBlock.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBlock.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
}

func Test_initBlockBodyFromProtoPhase0(t *testing.T) {
	expectedBody := bodyPbPhase0()
	resultBody, err := initBlockBodyFromProtoPhase0(expectedBody)
	require.NoError(t, err)
	resultHTR, err := resultBody.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBody.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
}

func Test_initBlockBodyFromProtoAltair(t *testing.T) {
	expectedBody := bodyPbAltair()
	resultBody, err := initBlockBodyFromProtoAltair(expectedBody)
	require.NoError(t, err)
	resultHTR, err := resultBody.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBody.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
}

func Test_initBlockBodyFromProtoBellatrix(t *testing.T) {
	expectedBody := bodyPbBellatrix()
	resultBody, err := initBlockBodyFromProtoBellatrix(expectedBody)
	require.NoError(t, err)
	resultHTR, err := resultBody.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBody.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
}

func Test_initBlockBodyFromProtoBlindedBellatrix(t *testing.T) {
	expectedBody := bodyPbBlindedBellatrix()
	resultBody, err := initBlindedBlockBodyFromProtoBellatrix(expectedBody)
	require.NoError(t, err)
	resultHTR, err := resultBody.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBody.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
}

func bodyPbPhase0() *eth.BeaconBlockBody {
	f := getFields()
	return &eth.BeaconBlockBody{
		RandaoReveal: f.b96,
		Eth1Data: &eth.Eth1Data{
			DepositRoot:  f.b32,
			DepositCount: 128,
			BlockHash:    f.b32,
		},
		Graffiti:          f.b32,
		ProposerSlashings: f.proposerSlashings,
		AttesterSlashings: f.attesterSlashings,
		Attestations:      f.atts,
		Deposits:          f.deposits,
		VoluntaryExits:    f.voluntaryExits,
	}
}

func bodyPbAltair() *eth.BeaconBlockBodyAltair {
	f := getFields()
	return &eth.BeaconBlockBodyAltair{
		RandaoReveal: f.b96,
		Eth1Data: &eth.Eth1Data{
			DepositRoot:  f.b32,
			DepositCount: 128,
			BlockHash:    f.b32,
		},
		Graffiti:          f.b32,
		ProposerSlashings: f.proposerSlashings,
		AttesterSlashings: f.attesterSlashings,
		Attestations:      f.atts,
		Deposits:          f.deposits,
		VoluntaryExits:    f.voluntaryExits,
		SyncAggregate:     f.syncAggregate,
	}
}

func bodyPbBellatrix() *eth.BeaconBlockBodyBellatrix {
	f := getFields()
	return &eth.BeaconBlockBodyBellatrix{
		RandaoReveal: f.b96,
		Eth1Data: &eth.Eth1Data{
			DepositRoot:  f.b32,
			DepositCount: 128,
			BlockHash:    f.b32,
		},
		Graffiti:          f.b32,
		ProposerSlashings: f.proposerSlashings,
		AttesterSlashings: f.attesterSlashings,
		Attestations:      f.atts,
		Deposits:          f.deposits,
		VoluntaryExits:    f.voluntaryExits,
		SyncAggregate:     f.syncAggregate,
		ExecutionPayload:  f.execPayload,
	}
}

func bodyPbBlindedBellatrix() *eth.BlindedBeaconBlockBodyBellatrix {
	f := getFields()
	return &eth.BlindedBeaconBlockBodyBellatrix{
		RandaoReveal: f.b96,
		Eth1Data: &eth.Eth1Data{
			DepositRoot:  f.b32,
			DepositCount: 128,
			BlockHash:    f.b32,
		},
		Graffiti:               f.b32,
		ProposerSlashings:      f.proposerSlashings,
		AttesterSlashings:      f.attesterSlashings,
		Attestations:           f.atts,
		Deposits:               f.deposits,
		VoluntaryExits:         f.voluntaryExits,
		SyncAggregate:          f.syncAggregate,
		ExecutionPayloadHeader: f.execPayloadHeader,
	}
}

func bodyPhase0() *BeaconBlockBody {
	f := getFields()
	return &BeaconBlockBody{
		version:      version.Phase0,
		randaoReveal: f.b96,
		eth1Data: &eth.Eth1Data{
			DepositRoot:  f.b32,
			DepositCount: 128,
			BlockHash:    f.b32,
		},
		graffiti:          f.b32,
		proposerSlashings: f.proposerSlashings,
		attesterSlashings: f.attesterSlashings,
		attestations:      f.atts,
		deposits:          f.deposits,
		voluntaryExits:    f.voluntaryExits,
	}
}

func bodyAltair() *BeaconBlockBody {
	f := getFields()
	return &BeaconBlockBody{
		version:      version.Altair,
		randaoReveal: f.b96,
		eth1Data: &eth.Eth1Data{
			DepositRoot:  f.b32,
			DepositCount: 128,
			BlockHash:    f.b32,
		},
		graffiti:          f.b32,
		proposerSlashings: f.proposerSlashings,
		attesterSlashings: f.attesterSlashings,
		attestations:      f.atts,
		deposits:          f.deposits,
		voluntaryExits:    f.voluntaryExits,
		syncAggregate:     f.syncAggregate,
	}
}

func bodyBellatrix() *BeaconBlockBody {
	f := getFields()
	return &BeaconBlockBody{
		version:      version.Bellatrix,
		randaoReveal: f.b96,
		eth1Data: &eth.Eth1Data{
			DepositRoot:  f.b32,
			DepositCount: 128,
			BlockHash:    f.b32,
		},
		graffiti:          f.b32,
		proposerSlashings: f.proposerSlashings,
		attesterSlashings: f.attesterSlashings,
		attestations:      f.atts,
		deposits:          f.deposits,
		voluntaryExits:    f.voluntaryExits,
		syncAggregate:     f.syncAggregate,
		executionPayload:  f.execPayload,
	}
}

func bodyBlindedBellatrix() *BeaconBlockBody {
	f := getFields()
	return &BeaconBlockBody{
		version:      version.Bellatrix,
		isBlinded:    true,
		randaoReveal: f.b96,
		eth1Data: &eth.Eth1Data{
			DepositRoot:  f.b32,
			DepositCount: 128,
			BlockHash:    f.b32,
		},
		graffiti:               f.b32,
		proposerSlashings:      f.proposerSlashings,
		attesterSlashings:      f.attesterSlashings,
		attestations:           f.atts,
		deposits:               f.deposits,
		voluntaryExits:         f.voluntaryExits,
		syncAggregate:          f.syncAggregate,
		executionPayloadHeader: f.execPayloadHeader,
	}
}

func getFields() fields {
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
	execPayload := &enginev1.ExecutionPayload{
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
	execPayloadHeader := &enginev1.ExecutionPayloadHeader{
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

	return fields{
		b20:               b20,
		b32:               b32,
		b48:               b48,
		b96:               b96,
		b256:              b256,
		deposits:          deposits,
		atts:              atts,
		proposerSlashings: []*eth.ProposerSlashing{proposerSlashing},
		attesterSlashings: []*eth.AttesterSlashing{attesterSlashing},
		voluntaryExits:    []*eth.SignedVoluntaryExit{voluntaryExit},
		syncAggregate:     syncAggregate,
		execPayload:       execPayload,
		execPayloadHeader: execPayloadHeader,
	}
}
