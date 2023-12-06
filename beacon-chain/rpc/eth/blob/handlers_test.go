package blob

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	mockChain "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/filesystem"
	testDB "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/verification"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestParseIndices(t *testing.T) {
	assert.DeepEqual(t, []uint64{1, 2, 3}, parseIndices(&url.URL{RawQuery: "indices=100&indices=1&indices=2&indices=foo&indices=1&indices=3&bar=bar"}))
}

func TestBlobs(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.DenebForkEpoch = 1
	params.OverrideBeaconConfig(cfg)

	db := testDB.SetupDB(t)
	denebBlock, blobs := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 123, 4)
	require.NoError(t, db.SaveBlock(context.Background(), denebBlock))
	_, bs, err := filesystem.NewEphemeralBlobStorageWithFs(t)
	require.NoError(t, err)
	testSidecars, err := verification.BlobSidecarSliceNoop(blobs)
	require.NoError(t, err)
	for i := range testSidecars {
		require.NoError(t, bs.Save(testSidecars[i]))
	}
	blockRoot := blobs[0].BlockRoot()

	t.Run("genesis", func(t *testing.T) {
		u := "http://foo.example/genesis"
		request := httptest.NewRequest("GET", u, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}
		s := &Server{}

		s.Blobs(writer, request)

		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, "Blobs are not supported for Phase 0 fork", e.Message)
	})
	t.Run("head", func(t *testing.T) {
		u := "http://foo.example/head"
		request := httptest.NewRequest("GET", u, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}
		s := &Server{
			ChainInfoFetcher: &mockChain.ChainService{Root: blockRoot[:]},
			BeaconDB:         db,
			BlobStorage:      bs,
		}

		s.Blobs(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &SidecarsResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.Equal(t, 4, len(resp.Data))
		sidecar := resp.Data[0]
		require.NotNil(t, sidecar)
		assert.Equal(t, "0", sidecar.Index)
		assert.Equal(t, hexutil.Encode(blobs[0].Blob), sidecar.Blob)
		assert.Equal(t, hexutil.Encode(blobs[0].KzgCommitment), sidecar.KzgCommitment)
		assert.Equal(t, hexutil.Encode(blobs[0].KzgProof), sidecar.KzgProof)
		sidecar = resp.Data[1]
		require.NotNil(t, sidecar)
		assert.Equal(t, "1", sidecar.Index)
		assert.Equal(t, hexutil.Encode(blobs[1].Blob), sidecar.Blob)
		assert.Equal(t, hexutil.Encode(blobs[1].KzgCommitment), sidecar.KzgCommitment)
		assert.Equal(t, hexutil.Encode(blobs[1].KzgProof), sidecar.KzgProof)
		sidecar = resp.Data[2]
		require.NotNil(t, sidecar)
		assert.Equal(t, "2", sidecar.Index)
		assert.Equal(t, hexutil.Encode(blobs[2].Blob), sidecar.Blob)
		assert.Equal(t, hexutil.Encode(blobs[2].KzgCommitment), sidecar.KzgCommitment)
		assert.Equal(t, hexutil.Encode(blobs[2].KzgProof), sidecar.KzgProof)
		sidecar = resp.Data[3]
		require.NotNil(t, sidecar)
		assert.Equal(t, "3", sidecar.Index)
		assert.Equal(t, hexutil.Encode(blobs[3].Blob), sidecar.Blob)
		assert.Equal(t, hexutil.Encode(blobs[3].KzgCommitment), sidecar.KzgCommitment)
		assert.Equal(t, hexutil.Encode(blobs[3].KzgProof), sidecar.KzgProof)
	})
	t.Run("finalized", func(t *testing.T) {
		u := "http://foo.example/finalized"
		request := httptest.NewRequest("GET", u, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}
		s := &Server{
			ChainInfoFetcher: &mockChain.ChainService{FinalizedCheckPoint: &eth.Checkpoint{Root: blockRoot[:]}},
			BeaconDB:         db,
			BlobStorage:      bs,
		}

		s.Blobs(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &SidecarsResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.Equal(t, 4, len(resp.Data))
	})
	t.Run("justified", func(t *testing.T) {
		u := "http://foo.example/justified"
		request := httptest.NewRequest("GET", u, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}
		s := &Server{
			ChainInfoFetcher: &mockChain.ChainService{CurrentJustifiedCheckPoint: &eth.Checkpoint{Root: blockRoot[:]}},
			BeaconDB:         db,
			BlobStorage:      bs,
		}

		s.Blobs(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &SidecarsResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.Equal(t, 4, len(resp.Data))
	})
	t.Run("root", func(t *testing.T) {
		u := "http://foo.example/" + hexutil.Encode(blockRoot[:])
		request := httptest.NewRequest("GET", u, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}
		s := &Server{
			BeaconDB:    db,
			BlobStorage: bs,
		}

		s.Blobs(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &SidecarsResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.Equal(t, 4, len(resp.Data))
	})
	t.Run("slot", func(t *testing.T) {
		u := "http://foo.example/123"
		request := httptest.NewRequest("GET", u, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}
		s := &Server{
			BeaconDB:    db,
			BlobStorage: bs,
		}

		s.Blobs(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &SidecarsResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.Equal(t, 4, len(resp.Data))
	})
	t.Run("one blob only", func(t *testing.T) {
		u := "http://foo.example/123?indices=2"
		request := httptest.NewRequest("GET", u, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}
		s := &Server{
			ChainInfoFetcher: &mockChain.ChainService{FinalizedCheckPoint: &eth.Checkpoint{Root: blockRoot[:]}},
			BeaconDB:         db,
			BlobStorage:      bs,
		}

		s.Blobs(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &SidecarsResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.Equal(t, 1, len(resp.Data))
		sidecar := resp.Data[0]
		require.NotNil(t, sidecar)
		assert.Equal(t, "2", sidecar.Index)
		assert.Equal(t, hexutil.Encode(blobs[2].Blob), sidecar.Blob)
		assert.Equal(t, hexutil.Encode(blobs[2].KzgCommitment), sidecar.KzgCommitment)
		assert.Equal(t, hexutil.Encode(blobs[2].KzgProof), sidecar.KzgProof)
	})
	t.Run("slot before Deneb fork", func(t *testing.T) {
		u := "http://foo.example/31"
		request := httptest.NewRequest("GET", u, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}
		s := &Server{}

		s.Blobs(writer, request)

		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, "Blobs are not supported before Deneb fork", e.Message)
	})
	t.Run("malformed block ID", func(t *testing.T) {
		u := "http://foo.example/foo"
		request := httptest.NewRequest("GET", u, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}
		s := &Server{}

		s.Blobs(writer, request)

		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "could not parse block ID"))
	})
	t.Run("ssz", func(t *testing.T) {
		u := "http://foo.example/finalized?indices=0"
		request := httptest.NewRequest("GET", u, nil)
		request.Header.Add("Accept", "application/octet-stream")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}
		s := &Server{
			ChainInfoFetcher: &mockChain.ChainService{FinalizedCheckPoint: &eth.Checkpoint{Root: blockRoot[:]}},
			BeaconDB:         db,
			BlobStorage:      bs,
		}
		s.Blobs(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		require.Equal(t, len(writer.Body.Bytes()), 131932)
	})
}
