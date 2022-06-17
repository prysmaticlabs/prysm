package blocks

import (
	"testing"

	ssz "github.com/ferranbt/fastssz"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func Test_SignedBeaconBlock_Signature(t *testing.T) {
	sb := &SignedBeaconBlock{signature: []byte("signature")}
	assert.DeepEqual(t, []byte("signature"), sb.Signature())
}

func Test_SignedBeaconBlock_Block(t *testing.T) {
	b := &BeaconBlock{}
	sb := &SignedBeaconBlock{block: b}
	assert.Equal(t, b, sb.Block())
}

func Test_SignedBeaconBlock_IsNil(t *testing.T) {
	t.Run("nil signed block", func(t *testing.T) {
		var sb *SignedBeaconBlock
		assert.Equal(t, true, sb.IsNil())
	})
	t.Run("nil block", func(t *testing.T) {
		sb := &SignedBeaconBlock{}
		assert.Equal(t, true, sb.IsNil())
	})
	t.Run("nil body", func(t *testing.T) {
		sb := &SignedBeaconBlock{block: &BeaconBlock{}}
		assert.Equal(t, true, sb.IsNil())
	})
	t.Run("not nil", func(t *testing.T) {
		sb := &SignedBeaconBlock{block: &BeaconBlock{body: &BeaconBlockBody{}}}
		assert.Equal(t, false, sb.IsNil())
	})
}

func Test_SignedBeaconBlock_Copy(t *testing.T) {
	bb := &BeaconBlockBody{}
	b := &BeaconBlock{body: bb}
	sb := &SignedBeaconBlock{block: b}
	cp, err := sb.Copy()
	require.NoError(t, err)
	assert.NotEqual(t, cp, sb)
	assert.NotEqual(t, cp.block, sb.block)
	assert.NotEqual(t, cp.block.body, sb.block.body)
}

func Test_SignedBeaconBlock_Version(t *testing.T) {
	sb := &SignedBeaconBlock{version: 128}
	assert.Equal(t, 128, sb.Version())
}

func Test_SignedBeaconBlock_Header(t *testing.T) {
	bb := &BeaconBlockBody{
		version:      version.Phase0,
		randaoReveal: make([]byte, 96),
		eth1Data: &eth.Eth1Data{
			DepositRoot: make([]byte, 32),
			BlockHash:   make([]byte, 32),
		},
		graffiti: make([]byte, 32),
	}
	sb := &SignedBeaconBlock{
		version: version.Phase0,
		block: &BeaconBlock{
			version:       version.Phase0,
			slot:          128,
			proposerIndex: 128,
			parentRoot:    []byte("parentroot"),
			stateRoot:     []byte("stateroot"),
			body:          bb,
		},
		signature: []byte("signature"),
	}
	h, err := sb.Header()
	require.NoError(t, err)
	assert.DeepEqual(t, sb.signature, h.Signature)
	assert.Equal(t, sb.block.slot, h.Header.Slot)
	assert.Equal(t, sb.block.proposerIndex, h.Header.ProposerIndex)
	assert.DeepEqual(t, sb.block.parentRoot, h.Header.ParentRoot)
	assert.DeepEqual(t, sb.block.stateRoot, h.Header.StateRoot)
	expectedHTR, err := bb.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR[:], h.Header.BodyRoot)
}

func Test_SignedBeaconBlock_UnmarshalSSZ(t *testing.T) {
	pb := util.HydrateSignedBeaconBlock(&eth.SignedBeaconBlock{})
	buf, err := pb.MarshalSSZ()
	require.NoError(t, err)
	expectedHTR, err := pb.HashTreeRoot()
	require.NoError(t, err)
	sb := &SignedBeaconBlock{}
	require.NoError(t, sb.UnmarshalSSZ(buf))
	msg, err := sb.Proto()
	require.NoError(t, err)
	actualPb, ok := msg.(*eth.SignedBeaconBlock)
	require.Equal(t, true, ok)
	actualHTR, err := actualPb.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, actualHTR)
}

func Test_BeaconBlock_Slot(t *testing.T) {
	b := &BeaconBlock{slot: 128}
	assert.Equal(t, types.Slot(128), b.Slot())
}

func Test_BeaconBlock_ProposerIndex(t *testing.T) {
	b := &BeaconBlock{proposerIndex: 128}
	assert.Equal(t, types.ValidatorIndex(128), b.ProposerIndex())
}

func Test_BeaconBlock_ParentRoot(t *testing.T) {
	b := &BeaconBlock{parentRoot: []byte("parentroot")}
	assert.DeepEqual(t, []byte("parentroot"), b.ParentRoot())
}

func Test_BeaconBlock_StateRoot(t *testing.T) {
	b := &BeaconBlock{stateRoot: []byte("stateroot")}
	assert.DeepEqual(t, []byte("stateroot"), b.StateRoot())
}

func Test_BeaconBlock_Body(t *testing.T) {
	bb := &BeaconBlockBody{}
	b := &BeaconBlock{body: bb}
	assert.Equal(t, bb, b.Body())
}

func Test_BeaconBlock_IsNil(t *testing.T) {
	t.Run("nil block", func(t *testing.T) {
		var b *BeaconBlock
		assert.Equal(t, true, b.IsNil())
	})
	t.Run("nil block body", func(t *testing.T) {
		b := &BeaconBlock{}
		assert.Equal(t, true, b.IsNil())
	})
	t.Run("not nil", func(t *testing.T) {
		b := &BeaconBlock{body: &BeaconBlockBody{}}
		assert.Equal(t, false, b.IsNil())
	})
}

func Test_BeaconBlock_IsBlinded(t *testing.T) {
	assert.Equal(t, false, (&BeaconBlock{version: version.Phase0}).IsBlinded())
	assert.Equal(t, false, (&BeaconBlock{version: version.Altair}).IsBlinded())
	assert.Equal(t, false, (&BeaconBlock{version: version.Bellatrix}).IsBlinded())
	assert.Equal(t, true, (&BeaconBlock{version: version.BellatrixBlind}).IsBlinded())
	assert.Equal(t, false, (&BeaconBlock{version: 128}).IsBlinded())
}

func Test_BeaconBlock_Version(t *testing.T) {
	b := &BeaconBlock{version: 128}
	assert.Equal(t, 128, b.Version())
}

func Test_BeaconBlock_HashTreeRoot(t *testing.T) {
	pb := util.HydrateBeaconBlock(&eth.BeaconBlock{})
	expectedHTR, err := pb.HashTreeRoot()
	require.NoError(t, err)
	b, err := initBlockFromProtoPhase0(pb)
	require.NoError(t, err)
	actualHTR, err := b.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, actualHTR)
}

func Test_BeaconBlock_HashTreeRootWith(t *testing.T) {
	pb := util.HydrateBeaconBlock(&eth.BeaconBlock{})
	expectedHTR, err := pb.HashTreeRoot()
	require.NoError(t, err)
	b, err := initBlockFromProtoPhase0(pb)
	require.NoError(t, err)
	h := ssz.DefaultHasherPool.Get()
	require.NoError(t, b.HashTreeRootWith(h))
	actualHTR, err := h.HashRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, actualHTR)
}

func Test_BeaconBlock_UnmarshalSSZ(t *testing.T) {
	pb := util.HydrateBeaconBlock(&eth.BeaconBlock{})
	buf, err := pb.MarshalSSZ()
	require.NoError(t, err)
	expectedHTR, err := pb.HashTreeRoot()
	require.NoError(t, err)
	b := &BeaconBlock{}
	require.NoError(t, b.UnmarshalSSZ(buf))
	msg, err := b.Proto()
	require.NoError(t, err)
	actualPb, ok := msg.(*eth.BeaconBlock)
	require.Equal(t, true, ok)
	actualHTR, err := actualPb.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, actualHTR)
}

func Test_BeaconBlock_AsSignRequestObject(t *testing.T) {
	pb := util.HydrateBeaconBlock(&eth.BeaconBlock{})
	expectedHTR, err := pb.HashTreeRoot()
	require.NoError(t, err)
	b, err := initBlockFromProtoPhase0(pb)
	require.NoError(t, err)
	signRequestObj, err := b.AsSignRequestObject()
	require.NoError(t, err)
	actualSignRequestObj, ok := signRequestObj.(*validatorpb.SignRequest_Block)
	require.Equal(t, true, ok)
	actualHTR, err := actualSignRequestObj.Block.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, actualHTR)
}

func Test_BeaconBlockBody_IsNil(t *testing.T) {
	t.Run("nil block body", func(t *testing.T) {
		var bb *BeaconBlockBody
		assert.Equal(t, true, bb.IsNil())
	})
	t.Run("not nil", func(t *testing.T) {
		bb := &BeaconBlockBody{}
		assert.Equal(t, false, bb.IsNil())
	})
}

func Test_BeaconBlockBody_RandaoReveal(t *testing.T) {
	bb := &BeaconBlockBody{randaoReveal: []byte("randaoreveal")}
	assert.DeepEqual(t, []byte("randaoreveal"), bb.RandaoReveal())
}

func Test_BeaconBlockBody_Eth1Data(t *testing.T) {
	e := &eth.Eth1Data{}
	bb := &BeaconBlockBody{eth1Data: e}
	assert.Equal(t, e, bb.Eth1Data())
}

func Test_BeaconBlockBody_Graffiti(t *testing.T) {
	bb := &BeaconBlockBody{graffiti: []byte("graffiti")}
	assert.DeepEqual(t, []byte("graffiti"), bb.Graffiti())
}

func Test_BeaconBlockBody_ProposerSlashings(t *testing.T) {
	ps := make([]*eth.ProposerSlashing, 0)
	bb := &BeaconBlockBody{proposerSlashings: ps}
	assert.DeepSSZEqual(t, ps, bb.ProposerSlashings())
}

func Test_BeaconBlockBody_AttesterSlashings(t *testing.T) {
	as := make([]*eth.AttesterSlashing, 0)
	bb := &BeaconBlockBody{attesterSlashings: as}
	assert.DeepSSZEqual(t, as, bb.AttesterSlashings())
}

func Test_BeaconBlockBody_Attestations(t *testing.T) {
	a := make([]*eth.Attestation, 0)
	bb := &BeaconBlockBody{attestations: a}
	assert.DeepSSZEqual(t, a, bb.Attestations())
}

func Test_BeaconBlockBody_Deposits(t *testing.T) {
	d := make([]*eth.Deposit, 0)
	bb := &BeaconBlockBody{deposits: d}
	assert.DeepSSZEqual(t, d, bb.Deposits())
}

func Test_BeaconBlockBody_VoluntaryExits(t *testing.T) {
	ve := make([]*eth.SignedVoluntaryExit, 0)
	bb := &BeaconBlockBody{voluntaryExits: ve}
	assert.DeepSSZEqual(t, ve, bb.VoluntaryExits())
}

func Test_BeaconBlockBody_SyncAggregate(t *testing.T) {
	sa := &eth.SyncAggregate{}
	bb := &BeaconBlockBody{version: version.Altair, syncAggregate: sa}
	result, err := bb.SyncAggregate()
	require.NoError(t, err)
	assert.Equal(t, result, sa)
}

func Test_BeaconBlockBody_ExecutionPayload(t *testing.T) {
	ep := &enginev1.ExecutionPayload{}
	bb := &BeaconBlockBody{version: version.Bellatrix, executionPayload: ep}
	result, err := bb.ExecutionPayload()
	require.NoError(t, err)
	assert.Equal(t, result, ep)
}

func Test_BeaconBlockBody_ExecutionPayloadHeader(t *testing.T) {
	eph := &eth.ExecutionPayloadHeader{}
	bb := &BeaconBlockBody{version: version.BellatrixBlind, executionPayloadHeader: eph}
	result, err := bb.ExecutionPayloadHeader()
	require.NoError(t, err)
	assert.Equal(t, result, eph)
}

func Test_BeaconBlockBody_HashTreeRoot(t *testing.T) {
	pb := util.HydrateBeaconBlockBody(&eth.BeaconBlockBody{})
	expectedHTR, err := pb.HashTreeRoot()
	require.NoError(t, err)
	b, err := initBlockBodyFromProtoPhase0(pb)
	require.NoError(t, err)
	actualHTR, err := b.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHTR, actualHTR)
}
