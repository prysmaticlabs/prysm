package blocks

import (
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	consensus_types "github.com/prysmaticlabs/prysm/v5/consensus-types"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	pb "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func Test_EpbsBlock_Copy(t *testing.T) {
	signedHeader := &pb.SignedExecutionPayloadHeader{
		Message: &pb.ExecutionPayloadHeaderEPBS{
			ParentBlockHash:        bytesutil.PadTo([]byte("parentblockhash"), fieldparams.RootLength),
			ParentBlockRoot:        bytesutil.PadTo([]byte("parentblockroot"), fieldparams.RootLength),
			BlockHash:              bytesutil.PadTo([]byte("blockhash"), fieldparams.RootLength),
			BuilderIndex:           1,
			Slot:                   2,
			Value:                  3,
			BlobKzgCommitmentsRoot: bytesutil.PadTo([]byte("blobkzgcommitmentsroot"), fieldparams.RootLength),
		},
		Signature: bytesutil.PadTo([]byte("signature"), fieldparams.BLSSignatureLength),
	}
	aggregationBits := bitfield.NewBitvector512()
	aggregationBits.SetBitAt(1, true)
	aggregationBits.SetBitAt(2, true)

	payloadAttestations := []*eth.PayloadAttestation{
		{
			AggregationBits: aggregationBits,
			Data: &eth.PayloadAttestationData{
				BeaconBlockRoot: []byte("beaconblockroot"),
				Slot:            1,
				PayloadStatus:   2,
			},
			Signature: []byte("signature"),
		},
		{
			AggregationBits: aggregationBits,
			Data: &eth.PayloadAttestationData{
				BeaconBlockRoot: []byte("beaconblockroot"),
				Slot:            1,
				PayloadStatus:   1,
			},
			Signature: []byte("signature"),
		},
	}

	epbsBlockProto := &eth.BeaconBlockEpbs{
		Body: &eth.BeaconBlockBodyEpbs{
			SignedExecutionPayloadHeader: signedHeader,
			PayloadAttestations:          payloadAttestations,
		},
	}

	epbsBlock, err := NewBeaconBlock(epbsBlockProto)
	require.NoError(t, err)
	copiedEpbsBlock, err := epbsBlock.Copy()
	require.NoError(t, err)
	copiedHeader, err := copiedEpbsBlock.Body().SignedExecutionPayloadHeader()
	require.NoError(t, err)
	require.DeepEqual(t, copiedHeader, signedHeader)

	copiedPayloadAtts, err := copiedEpbsBlock.Body().PayloadAttestations()
	require.NoError(t, err)
	require.DeepEqual(t, copiedPayloadAtts, payloadAttestations)
}

func Test_EpbsBlock_Pb(t *testing.T) {
	b := &SignedBeaconBlock{version: version.Deneb, block: &BeaconBlock{version: version.Deneb, body: &BeaconBlockBody{version: version.Deneb}}}
	_, err := b.PbEPBSBlock()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	b = &SignedBeaconBlock{version: version.EPBS, block: &BeaconBlock{version: version.EPBS, body: &BeaconBlockBody{version: version.EPBS}}}
	_, err = b.PbEPBSBlock()
	require.NoError(t, err)
}

func Test_EpbsBlock_ToBlinded(t *testing.T) {
	b := &SignedBeaconBlock{version: version.EPBS}
	_, err := b.ToBlinded()
	require.ErrorIs(t, err, ErrUnsupportedVersion)
}

func Test_EpbsBlock_Unblind(t *testing.T) {
	b := &SignedBeaconBlock{version: version.EPBS}
	e, err := WrappedExecutionPayload(&pb.ExecutionPayload{})
	require.NoError(t, err)
	err = b.Unblind(e)
	require.ErrorIs(t, err, ErrAlreadyUnblinded)
}

func Test_EpbsBlock_IsBlinded(t *testing.T) {
	b := &SignedBeaconBlock{version: version.EPBS}
	require.Equal(t, false, b.IsBlinded())
	bb := &BeaconBlock{version: version.EPBS}
	require.Equal(t, false, bb.IsBlinded())
	bd := &BeaconBlockBody{version: version.EPBS}
	require.Equal(t, false, bd.IsBlinded())
}
