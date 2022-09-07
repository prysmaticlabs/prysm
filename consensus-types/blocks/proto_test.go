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
	root              [32]byte
	sig               [96]byte
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
				ParentRoot:    f.root[:],
				StateRoot:     f.root[:],
				Body:          bodyPbPhase0(),
			},
			Signature: f.sig[:],
		}
		block := &SignedBeaconBlock{
			version: version.Phase0,
			block: &BeaconBlock{
				version:       version.Phase0,
				slot:          128,
				proposerIndex: 128,
				parentRoot:    f.root,
				stateRoot:     f.root,
				body:          bodyPhase0(),
			},
			signature: f.sig,
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
				ParentRoot:    f.root[:],
				StateRoot:     f.root[:],
				Body:          bodyPbAltair(),
			},
			Signature: f.sig[:],
		}
		block := &SignedBeaconBlock{
			version: version.Altair,
			block: &BeaconBlock{
				version:       version.Altair,
				slot:          128,
				proposerIndex: 128,
				parentRoot:    f.root,
				stateRoot:     f.root,
				body:          bodyAltair(),
			},
			signature: f.sig,
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
				ParentRoot:    f.root[:],
				StateRoot:     f.root[:],
				Body:          bodyPbBellatrix(),
			},
			Signature: f.sig[:],
		}
		block := &SignedBeaconBlock{
			version: version.Bellatrix,
			block: &BeaconBlock{
				version:       version.Bellatrix,
				slot:          128,
				proposerIndex: 128,
				parentRoot:    f.root,
				stateRoot:     f.root,
				body:          bodyBellatrix(),
			},
			signature: f.sig,
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
				ParentRoot:    f.root[:],
				StateRoot:     f.root[:],
				Body:          bodyPbBlindedBellatrix(),
			},
			Signature: f.sig[:],
		}
		block := &SignedBeaconBlock{
			version: version.Bellatrix,
			block: &BeaconBlock{
				version:       version.Bellatrix,
				slot:          128,
				proposerIndex: 128,
				parentRoot:    f.root,
				stateRoot:     f.root,
				body:          bodyBlindedBellatrix(),
			},
			signature: f.sig,
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
			ParentRoot:    f.root[:],
			StateRoot:     f.root[:],
			Body:          bodyPbPhase0(),
		}
		block := &BeaconBlock{
			version:       version.Phase0,
			slot:          128,
			proposerIndex: 128,
			parentRoot:    f.root,
			stateRoot:     f.root,
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
			ParentRoot:    f.root[:],
			StateRoot:     f.root[:],
			Body:          bodyPbAltair(),
		}
		block := &BeaconBlock{
			version:       version.Altair,
			slot:          128,
			proposerIndex: 128,
			parentRoot:    f.root,
			stateRoot:     f.root,
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
			ParentRoot:    f.root[:],
			StateRoot:     f.root[:],
			Body:          bodyPbBellatrix(),
		}
		block := &BeaconBlock{
			version:       version.Bellatrix,
			slot:          128,
			proposerIndex: 128,
			parentRoot:    f.root,
			stateRoot:     f.root,
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
			ParentRoot:    f.root[:],
			StateRoot:     f.root[:],
			Body:          bodyPbBlindedBellatrix(),
		}
		block := &BeaconBlock{
			version:       version.Bellatrix,
			slot:          128,
			proposerIndex: 128,
			parentRoot:    f.root,
			stateRoot:     f.root,
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
			ParentRoot:    f.root[:],
			StateRoot:     f.root[:],
			Body:          bodyPbPhase0(),
		},
		Signature: f.sig[:],
	}
	resultBlock, err := initSignedBlockFromProtoPhase0(expectedBlock)
	require.NoError(t, err)
	resultHTR, err := resultBlock.block.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
	assert.DeepEqual(t, expectedBlock.Signature, resultBlock.signature[:])
}

func Test_initSignedBlockFromProtoAltair(t *testing.T) {
	f := getFields()
	expectedBlock := &eth.SignedBeaconBlockAltair{
		Block: &eth.BeaconBlockAltair{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.root[:],
			StateRoot:     f.root[:],
			Body:          bodyPbAltair(),
		},
		Signature: f.sig[:],
	}
	resultBlock, err := initSignedBlockFromProtoAltair(expectedBlock)
	require.NoError(t, err)
	resultHTR, err := resultBlock.block.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
	assert.DeepEqual(t, expectedBlock.Signature, resultBlock.signature[:])
}

func Test_initSignedBlockFromProtoBellatrix(t *testing.T) {
	f := getFields()
	expectedBlock := &eth.SignedBeaconBlockBellatrix{
		Block: &eth.BeaconBlockBellatrix{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.root[:],
			StateRoot:     f.root[:],
			Body:          bodyPbBellatrix(),
		},
		Signature: f.sig[:],
	}
	resultBlock, err := initSignedBlockFromProtoBellatrix(expectedBlock)
	require.NoError(t, err)
	resultHTR, err := resultBlock.block.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
	assert.DeepEqual(t, expectedBlock.Signature, resultBlock.signature[:])
}

func Test_initBlindedSignedBlockFromProtoBellatrix(t *testing.T) {
	f := getFields()
	expectedBlock := &eth.SignedBlindedBeaconBlockBellatrix{
		Block: &eth.BlindedBeaconBlockBellatrix{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.root[:],
			StateRoot:     f.root[:],
			Body:          bodyPbBlindedBellatrix(),
		},
		Signature: f.sig[:],
	}
	resultBlock, err := initBlindedSignedBlockFromProtoBellatrix(expectedBlock)
	require.NoError(t, err)
	resultHTR, err := resultBlock.block.HashTreeRoot()
	require.NoError(t, err)
	expectedHTR, err := expectedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, resultHTR)
	assert.DeepEqual(t, expectedBlock.Signature, resultBlock.signature[:])
}

func Test_initBlockFromProtoPhase0(t *testing.T) {
	f := getFields()
	expectedBlock := &eth.BeaconBlock{
		Slot:          128,
		ProposerIndex: 128,
		ParentRoot:    f.root[:],
		StateRoot:     f.root[:],
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
		ParentRoot:    f.root[:],
		StateRoot:     f.root[:],
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
		ParentRoot:    f.root[:],
		StateRoot:     f.root[:],
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
		ParentRoot:    f.root[:],
		StateRoot:     f.root[:],
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
		RandaoReveal: f.sig[:],
		Eth1Data: &eth.Eth1Data{
			DepositRoot:  f.root[:],
			DepositCount: 128,
			BlockHash:    f.root[:],
		},
		Graffiti:          f.root[:],
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
		RandaoReveal: f.sig[:],
		Eth1Data: &eth.Eth1Data{
			DepositRoot:  f.root[:],
			DepositCount: 128,
			BlockHash:    f.root[:],
		},
		Graffiti:          f.root[:],
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
		RandaoReveal: f.sig[:],
		Eth1Data: &eth.Eth1Data{
			DepositRoot:  f.root[:],
			DepositCount: 128,
			BlockHash:    f.root[:],
		},
		Graffiti:          f.root[:],
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
		RandaoReveal: f.sig[:],
		Eth1Data: &eth.Eth1Data{
			DepositRoot:  f.root[:],
			DepositCount: 128,
			BlockHash:    f.root[:],
		},
		Graffiti:               f.root[:],
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
		randaoReveal: f.sig,
		eth1Data: &eth.Eth1Data{
			DepositRoot:  f.root[:],
			DepositCount: 128,
			BlockHash:    f.root[:],
		},
		graffiti:          f.root,
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
		randaoReveal: f.sig,
		eth1Data: &eth.Eth1Data{
			DepositRoot:  f.root[:],
			DepositCount: 128,
			BlockHash:    f.root[:],
		},
		graffiti:          f.root,
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
		randaoReveal: f.sig,
		eth1Data: &eth.Eth1Data{
			DepositRoot:  f.root[:],
			DepositCount: 128,
			BlockHash:    f.root[:],
		},
		graffiti:          f.root,
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
		randaoReveal: f.sig,
		eth1Data: &eth.Eth1Data{
			DepositRoot:  f.root[:],
			DepositCount: 128,
			BlockHash:    f.root[:],
		},
		graffiti:               f.root,
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
	b48 := make([]byte, 48)
	b256 := make([]byte, 256)
	var root [32]byte
	var sig [96]byte
	b20[0], b20[5], b20[10] = 'q', 'u', 'x'
	b48[0], b48[5], b48[10] = 'b', 'a', 'r'
	b256[0], b256[5], b256[10] = 'x', 'y', 'z'
	root[0], root[5], root[10] = 'a', 'b', 'c'
	sig[0], sig[5], sig[10] = 'd', 'e', 'f'
	deposits := make([]*eth.Deposit, 16)
	for i := range deposits {
		deposits[i] = &eth.Deposit{}
		deposits[i].Proof = make([][]byte, 33)
		for j := range deposits[i].Proof {
			deposits[i].Proof[j] = root[:]
		}
		deposits[i].Data = &eth.Deposit_Data{
			PublicKey:             b48,
			WithdrawalCredentials: root[:],
			Amount:                128,
			Signature:             sig[:],
		}
	}
	atts := make([]*eth.Attestation, 128)
	for i := range atts {
		atts[i] = &eth.Attestation{}
		atts[i].Signature = sig[:]
		atts[i].AggregationBits = bitfield.NewBitlist(1)
		atts[i].Data = &eth.AttestationData{
			Slot:            128,
			CommitteeIndex:  128,
			BeaconBlockRoot: root[:],
			Source: &eth.Checkpoint{
				Epoch: 128,
				Root:  root[:],
			},
			Target: &eth.Checkpoint{
				Epoch: 128,
				Root:  root[:],
			},
		}
	}
	proposerSlashing := &eth.ProposerSlashing{
		Header_1: &eth.SignedBeaconBlockHeader{
			Header: &eth.BeaconBlockHeader{
				Slot:          128,
				ProposerIndex: 128,
				ParentRoot:    root[:],
				StateRoot:     root[:],
				BodyRoot:      root[:],
			},
			Signature: sig[:],
		},
		Header_2: &eth.SignedBeaconBlockHeader{
			Header: &eth.BeaconBlockHeader{
				Slot:          128,
				ProposerIndex: 128,
				ParentRoot:    root[:],
				StateRoot:     root[:],
				BodyRoot:      root[:],
			},
			Signature: sig[:],
		},
	}
	attesterSlashing := &eth.AttesterSlashing{
		Attestation_1: &eth.IndexedAttestation{
			AttestingIndices: []uint64{1, 2, 8},
			Data: &eth.AttestationData{
				Slot:            128,
				CommitteeIndex:  128,
				BeaconBlockRoot: root[:],
				Source: &eth.Checkpoint{
					Epoch: 128,
					Root:  root[:],
				},
				Target: &eth.Checkpoint{
					Epoch: 128,
					Root:  root[:],
				},
			},
			Signature: sig[:],
		},
		Attestation_2: &eth.IndexedAttestation{
			AttestingIndices: []uint64{1, 2, 8},
			Data: &eth.AttestationData{
				Slot:            128,
				CommitteeIndex:  128,
				BeaconBlockRoot: root[:],
				Source: &eth.Checkpoint{
					Epoch: 128,
					Root:  root[:],
				},
				Target: &eth.Checkpoint{
					Epoch: 128,
					Root:  root[:],
				},
			},
			Signature: sig[:],
		},
	}
	voluntaryExit := &eth.SignedVoluntaryExit{
		Exit: &eth.VoluntaryExit{
			Epoch:          128,
			ValidatorIndex: 128,
		},
		Signature: sig[:],
	}
	syncCommitteeBits := bitfield.NewBitvector512()
	syncCommitteeBits.SetBitAt(1, true)
	syncCommitteeBits.SetBitAt(2, true)
	syncCommitteeBits.SetBitAt(8, true)
	syncAggregate := &eth.SyncAggregate{
		SyncCommitteeBits:      syncCommitteeBits,
		SyncCommitteeSignature: sig[:],
	}
	execPayload := &enginev1.ExecutionPayload{
		ParentHash:    root[:],
		FeeRecipient:  b20,
		StateRoot:     root[:],
		ReceiptsRoot:  root[:],
		LogsBloom:     b256,
		PrevRandao:    root[:],
		BlockNumber:   128,
		GasLimit:      128,
		GasUsed:       128,
		Timestamp:     128,
		ExtraData:     root[:],
		BaseFeePerGas: root[:],
		BlockHash:     root[:],
		Transactions: [][]byte{
			[]byte("transaction1"),
			[]byte("transaction2"),
			[]byte("transaction8"),
		},
	}
	execPayloadHeader := &enginev1.ExecutionPayloadHeader{
		ParentHash:       root[:],
		FeeRecipient:     b20,
		StateRoot:        root[:],
		ReceiptsRoot:     root[:],
		LogsBloom:        b256,
		PrevRandao:       root[:],
		BlockNumber:      128,
		GasLimit:         128,
		GasUsed:          128,
		Timestamp:        128,
		ExtraData:        root[:],
		BaseFeePerGas:    root[:],
		BlockHash:        root[:],
		TransactionsRoot: root[:],
	}

	return fields{
		root:              root,
		sig:               sig,
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
