package blocks

import (
	"testing"

	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util/tgen"
)

func Test_SignedBeaconBlock_Proto(t *testing.T) {
	f := tgen.GetBlockFields()

	t.Run("Phase0", func(t *testing.T) {
		expectedBlock := &eth.SignedBeaconBlock{
			Block: &eth.BeaconBlock{
				Slot:          128,
				ProposerIndex: 128,
				ParentRoot:    f.B32,
				StateRoot:     f.B32,
				Body:          tgen.PbBlockBodyPhase0(),
			},
			Signature: f.B96,
		}
		block := &SignedBeaconBlock{
			version: version.Phase0,
			block: &BeaconBlock{
				version:       version.Phase0,
				slot:          128,
				proposerIndex: 128,
				parentRoot:    f.B32,
				stateRoot:     f.B32,
				body:          bodyPhase0(),
			},
			signature: f.B96,
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
				ParentRoot:    f.B32,
				StateRoot:     f.B32,
				Body:          bodyPbAltair(),
			},
			Signature: f.B96,
		}
		block := &SignedBeaconBlock{
			version: version.Altair,
			block: &BeaconBlock{
				version:       version.Altair,
				slot:          128,
				proposerIndex: 128,
				parentRoot:    f.B32,
				stateRoot:     f.B32,
				body:          bodyAltair(),
			},
			signature: f.B96,
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
				ParentRoot:    f.B32,
				StateRoot:     f.B32,
				Body:          bodyPbBellatrix(),
			},
			Signature: f.B96,
		}
		block := &SignedBeaconBlock{
			version: version.Bellatrix,
			block: &BeaconBlock{
				version:       version.Bellatrix,
				slot:          128,
				proposerIndex: 128,
				parentRoot:    f.B32,
				stateRoot:     f.B32,
				body:          bodyBellatrix(),
			},
			signature: f.B96,
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
				ParentRoot:    f.B32,
				StateRoot:     f.B32,
				Body:          bodyPbBlindedBellatrix(),
			},
			Signature: f.B96,
		}
		block := &SignedBeaconBlock{
			version: version.BellatrixBlind,
			block: &BeaconBlock{
				version:       version.BellatrixBlind,
				slot:          128,
				proposerIndex: 128,
				parentRoot:    f.B32,
				stateRoot:     f.B32,
				body:          bodyBlindedBellatrix(),
			},
			signature: f.B96,
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
	f := tgen.GetBlockFields()

	t.Run("Phase0", func(t *testing.T) {
		expectedBlock := &eth.BeaconBlock{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.B32,
			StateRoot:     f.B32,
			Body:          tgen.PbBlockBodyPhase0(),
		}
		block := &BeaconBlock{
			version:       version.Phase0,
			slot:          128,
			proposerIndex: 128,
			parentRoot:    f.B32,
			stateRoot:     f.B32,
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
			ParentRoot:    f.B32,
			StateRoot:     f.B32,
			Body:          bodyPbAltair(),
		}
		block := &BeaconBlock{
			version:       version.Altair,
			slot:          128,
			proposerIndex: 128,
			parentRoot:    f.B32,
			stateRoot:     f.B32,
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
			ParentRoot:    f.B32,
			StateRoot:     f.B32,
			Body:          bodyPbBellatrix(),
		}
		block := &BeaconBlock{
			version:       version.Bellatrix,
			slot:          128,
			proposerIndex: 128,
			parentRoot:    f.B32,
			stateRoot:     f.B32,
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
			ParentRoot:    f.B32,
			StateRoot:     f.B32,
			Body:          bodyPbBlindedBellatrix(),
		}
		block := &BeaconBlock{
			version:       version.BellatrixBlind,
			slot:          128,
			proposerIndex: 128,
			parentRoot:    f.B32,
			stateRoot:     f.B32,
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
		expectedBody := tgen.PbBlockBodyPhase0()
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
	f := tgen.GetBlockFields()
	expectedBlock := &eth.SignedBeaconBlock{
		Block: &eth.BeaconBlock{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.B32,
			StateRoot:     f.B32,
			Body:          tgen.PbBlockBodyPhase0(),
		},
		Signature: f.B96,
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
	f := tgen.GetBlockFields()
	expectedBlock := &eth.SignedBeaconBlockAltair{
		Block: &eth.BeaconBlockAltair{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.B32,
			StateRoot:     f.B32,
			Body:          bodyPbAltair(),
		},
		Signature: f.B96,
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
	f := tgen.GetBlockFields()
	expectedBlock := &eth.SignedBeaconBlockBellatrix{
		Block: &eth.BeaconBlockBellatrix{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.B32,
			StateRoot:     f.B32,
			Body:          bodyPbBellatrix(),
		},
		Signature: f.B96,
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
	f := tgen.GetBlockFields()
	expectedBlock := &eth.SignedBlindedBeaconBlockBellatrix{
		Block: &eth.BlindedBeaconBlockBellatrix{
			Slot:          128,
			ProposerIndex: 128,
			ParentRoot:    f.B32,
			StateRoot:     f.B32,
			Body:          bodyPbBlindedBellatrix(),
		},
		Signature: f.B96,
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
	f := tgen.GetBlockFields()
	expectedBlock := &eth.BeaconBlock{
		Slot:          128,
		ProposerIndex: 128,
		ParentRoot:    f.B32,
		StateRoot:     f.B32,
		Body:          tgen.PbBlockBodyPhase0(),
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
	f := tgen.GetBlockFields()
	expectedBlock := &eth.BeaconBlockAltair{
		Slot:          128,
		ProposerIndex: 128,
		ParentRoot:    f.B32,
		StateRoot:     f.B32,
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
	f := tgen.GetBlockFields()
	expectedBlock := &eth.BeaconBlockBellatrix{
		Slot:          128,
		ProposerIndex: 128,
		ParentRoot:    f.B32,
		StateRoot:     f.B32,
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
	f := tgen.GetBlockFields()
	expectedBlock := &eth.BlindedBeaconBlockBellatrix{
		Slot:          128,
		ProposerIndex: 128,
		ParentRoot:    f.B32,
		StateRoot:     f.B32,
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
	expectedBody := tgen.PbBlockBodyPhase0()
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

func bodyPbAltair() *eth.BeaconBlockBodyAltair {
	f := tgen.GetBlockFields()
	return &eth.BeaconBlockBodyAltair{
		RandaoReveal: f.B96,
		Eth1Data: &eth.Eth1Data{
			DepositRoot:  f.B32,
			DepositCount: 128,
			BlockHash:    f.B32,
		},
		Graffiti:          f.B32,
		ProposerSlashings: f.ProposerSlashings,
		AttesterSlashings: f.AttesterSlashings,
		Attestations:      f.Atts,
		Deposits:          f.Deposits,
		VoluntaryExits:    f.VoluntaryExits,
		SyncAggregate:     f.SyncAggregate,
	}
}

func bodyPbBellatrix() *eth.BeaconBlockBodyBellatrix {
	f := tgen.GetBlockFields()
	return &eth.BeaconBlockBodyBellatrix{
		RandaoReveal: f.B96,
		Eth1Data: &eth.Eth1Data{
			DepositRoot:  f.B32,
			DepositCount: 128,
			BlockHash:    f.B32,
		},
		Graffiti:          f.B32,
		ProposerSlashings: f.ProposerSlashings,
		AttesterSlashings: f.AttesterSlashings,
		Attestations:      f.Atts,
		Deposits:          f.Deposits,
		VoluntaryExits:    f.VoluntaryExits,
		SyncAggregate:     f.SyncAggregate,
		ExecutionPayload:  f.ExecPayload,
	}
}

func bodyPbBlindedBellatrix() *eth.BlindedBeaconBlockBodyBellatrix {
	f := tgen.GetBlockFields()
	return &eth.BlindedBeaconBlockBodyBellatrix{
		RandaoReveal: f.B96,
		Eth1Data: &eth.Eth1Data{
			DepositRoot:  f.B32,
			DepositCount: 128,
			BlockHash:    f.B32,
		},
		Graffiti:               f.B32,
		ProposerSlashings:      f.ProposerSlashings,
		AttesterSlashings:      f.AttesterSlashings,
		Attestations:           f.Atts,
		Deposits:               f.Deposits,
		VoluntaryExits:         f.VoluntaryExits,
		SyncAggregate:          f.SyncAggregate,
		ExecutionPayloadHeader: f.ExecPayloadHeader,
	}
}

func bodyPhase0() *BeaconBlockBody {
	f := tgen.GetBlockFields()
	return &BeaconBlockBody{
		version:      version.Phase0,
		randaoReveal: f.B96,
		eth1Data: &eth.Eth1Data{
			DepositRoot:  f.B32,
			DepositCount: 128,
			BlockHash:    f.B32,
		},
		graffiti:          f.B32,
		proposerSlashings: f.ProposerSlashings,
		attesterSlashings: f.AttesterSlashings,
		attestations:      f.Atts,
		deposits:          f.Deposits,
		voluntaryExits:    f.VoluntaryExits,
	}
}

func bodyAltair() *BeaconBlockBody {
	f := tgen.GetBlockFields()
	return &BeaconBlockBody{
		version:      version.Altair,
		randaoReveal: f.B96,
		eth1Data: &eth.Eth1Data{
			DepositRoot:  f.B32,
			DepositCount: 128,
			BlockHash:    f.B32,
		},
		graffiti:          f.B32,
		proposerSlashings: f.ProposerSlashings,
		attesterSlashings: f.AttesterSlashings,
		attestations:      f.Atts,
		deposits:          f.Deposits,
		voluntaryExits:    f.VoluntaryExits,
		syncAggregate:     f.SyncAggregate,
	}
}

func bodyBellatrix() *BeaconBlockBody {
	f := tgen.GetBlockFields()
	return &BeaconBlockBody{
		version:      version.Bellatrix,
		randaoReveal: f.B96,
		eth1Data: &eth.Eth1Data{
			DepositRoot:  f.B32,
			DepositCount: 128,
			BlockHash:    f.B32,
		},
		graffiti:          f.B32,
		proposerSlashings: f.ProposerSlashings,
		attesterSlashings: f.AttesterSlashings,
		attestations:      f.Atts,
		deposits:          f.Deposits,
		voluntaryExits:    f.VoluntaryExits,
		syncAggregate:     f.SyncAggregate,
		executionPayload:  f.ExecPayload,
	}
}

func bodyBlindedBellatrix() *BeaconBlockBody {
	f := tgen.GetBlockFields()
	return &BeaconBlockBody{
		version:      version.BellatrixBlind,
		randaoReveal: f.B96,
		eth1Data: &eth.Eth1Data{
			DepositRoot:  f.B32,
			DepositCount: 128,
			BlockHash:    f.B32,
		},
		graffiti:               f.B32,
		proposerSlashings:      f.ProposerSlashings,
		attesterSlashings:      f.AttesterSlashings,
		attestations:           f.Atts,
		deposits:               f.Deposits,
		voluntaryExits:         f.VoluntaryExits,
		syncAggregate:          f.SyncAggregate,
		executionPayloadHeader: f.ExecPayloadHeader,
	}
}
