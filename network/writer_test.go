package network

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestWriteSsz(t *testing.T) {
	name := "test.ssz"
	sidecar := &eth.BlobSidecar{
		BlockRoot:       make([]byte, fieldparams.RootLength),
		Index:           0,
		Slot:            0,
		BlockParentRoot: make([]byte, fieldparams.RootLength),
		ProposerIndex:   123,
		Blob:            make([]byte, fieldparams.BlobLength),
		KzgCommitment:   make([]byte, fieldparams.BLSPubkeyLength),
		KzgProof:        make([]byte, fieldparams.BLSPubkeyLength),
	}
	respSsz, err := sidecar.MarshalSSZ()
	require.NoError(t, err)
	writer := httptest.NewRecorder()
	WriteSsz(writer, respSsz, name)
	require.Equal(t, writer.Header().Get("Content-Type"), "application/octet-stream")
	require.Equal(t, true, strings.Contains(writer.Header().Get("Content-Disposition"), name))
	require.Equal(t, writer.Header().Get("Content-Length"), fmt.Sprintf("%v", len(respSsz)))
	require.Equal(t, hexutil.Encode(writer.Body.Bytes()), hexutil.Encode(respSsz))
}
