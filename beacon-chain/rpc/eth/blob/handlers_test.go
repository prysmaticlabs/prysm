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
	testDB "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestParseIndices(t *testing.T) {
	assert.DeepEqual(t, []uint64{1, 2, 3}, parseIndices(&url.URL{RawQuery: "indices=1,2,foo,1&indices=3,1&bar=bar"}))
}

func TestBlobs(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.DenebForkEpoch = 1
	params.OverrideBeaconConfig(cfg)

	db := testDB.SetupDB(t)
	blockroot := bytesutil.PadTo([]byte("blockroot"), 32)
	require.NoError(t, db.SaveBlobSidecar(context.Background(), []*eth.BlobSidecar{
		{
			BlockRoot:       blockroot,
			Index:           0,
			Slot:            123,
			BlockParentRoot: []byte("blockparentroot"),
			ProposerIndex:   123,
			Blob:            []byte("blob0"),
			KzgCommitment:   []byte("kzgcommitment0"),
			KzgProof:        []byte("kzgproof0"),
		},
		{
			BlockRoot:       blockroot,
			Index:           1,
			Slot:            123,
			BlockParentRoot: []byte("blockparentroot"),
			ProposerIndex:   123,
			Blob:            []byte("blob1"),
			KzgCommitment:   []byte("kzgcommitment1"),
			KzgProof:        []byte("kzgproof1"),
		},
		{
			BlockRoot:       blockroot,
			Index:           2,
			Slot:            123,
			BlockParentRoot: []byte("blockparentroot"),
			ProposerIndex:   123,
			Blob:            []byte("blob2"),
			KzgCommitment:   []byte("kzgcommitment2"),
			KzgProof:        []byte("kzgproof2"),
		},
		{
			BlockRoot:       blockroot,
			Index:           3,
			Slot:            123,
			BlockParentRoot: []byte("blockparentroot"),
			ProposerIndex:   123,
			Blob:            []byte("blob3"),
			KzgCommitment:   []byte("kzgcommitment3"),
			KzgProof:        []byte("kzgproof3"),
		},
	}))

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
		assert.Equal(t, "blobs are not supported for Phase 0 fork", e.Message)
	})
	t.Run("head", func(t *testing.T) {
		u := "http://foo.example/head"
		request := httptest.NewRequest("GET", u, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}
		s := &Server{
			ChainInfoFetcher: &mockChain.ChainService{Root: blockroot},
			BeaconDB:         db,
		}

		s.Blobs(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &SidecarsResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.Equal(t, 4, len(resp.Data))
		sidecar := resp.Data[0]
		require.NotNil(t, sidecar)
		assert.Equal(t, "0x626c6f636b726f6f740000000000000000000000000000000000000000000000", sidecar.BlockRoot)
		assert.Equal(t, "0", sidecar.Index)
		assert.Equal(t, "123", sidecar.Slot)
		assert.Equal(t, "0x626c6f636b706172656e74726f6f74", sidecar.BlockParentRoot)
		assert.Equal(t, "123", sidecar.ProposerIndex)
		assert.Equal(t, "0x626c6f6230", sidecar.Blob)
		assert.Equal(t, "0x6b7a67636f6d6d69746d656e7430", sidecar.KZGCommitment)
		assert.Equal(t, "0x6b7a6770726f6f6630", sidecar.KZGProof)
		sidecar = resp.Data[1]
		require.NotNil(t, sidecar)
		assert.Equal(t, "0x626c6f636b726f6f740000000000000000000000000000000000000000000000", sidecar.BlockRoot)
		assert.Equal(t, "1", sidecar.Index)
		assert.Equal(t, "123", sidecar.Slot)
		assert.Equal(t, "0x626c6f636b706172656e74726f6f74", sidecar.BlockParentRoot)
		assert.Equal(t, "123", sidecar.ProposerIndex)
		assert.Equal(t, "0x626c6f6231", sidecar.Blob)
		assert.Equal(t, "0x6b7a67636f6d6d69746d656e7431", sidecar.KZGCommitment)
		assert.Equal(t, "0x6b7a6770726f6f6631", sidecar.KZGProof)
		sidecar = resp.Data[2]
		require.NotNil(t, sidecar)
		assert.Equal(t, "0x626c6f636b726f6f740000000000000000000000000000000000000000000000", sidecar.BlockRoot)
		assert.Equal(t, "2", sidecar.Index)
		assert.Equal(t, "123", sidecar.Slot)
		assert.Equal(t, "0x626c6f636b706172656e74726f6f74", sidecar.BlockParentRoot)
		assert.Equal(t, "123", sidecar.ProposerIndex)
		assert.Equal(t, "0x626c6f6232", sidecar.Blob)
		assert.Equal(t, "0x6b7a67636f6d6d69746d656e7432", sidecar.KZGCommitment)
		assert.Equal(t, "0x6b7a6770726f6f6632", sidecar.KZGProof)
		sidecar = resp.Data[3]
		require.NotNil(t, sidecar)
		assert.Equal(t, "0x626c6f636b726f6f740000000000000000000000000000000000000000000000", sidecar.BlockRoot)
		assert.Equal(t, "3", sidecar.Index)
		assert.Equal(t, "123", sidecar.Slot)
		assert.Equal(t, "0x626c6f636b706172656e74726f6f74", sidecar.BlockParentRoot)
		assert.Equal(t, "123", sidecar.ProposerIndex)
		assert.Equal(t, "0x626c6f6233", sidecar.Blob)
		assert.Equal(t, "0x6b7a67636f6d6d69746d656e7433", sidecar.KZGCommitment)
		assert.Equal(t, "0x6b7a6770726f6f6633", sidecar.KZGProof)
	})
	t.Run("finalized", func(t *testing.T) {
		u := "http://foo.example/finalized"
		request := httptest.NewRequest("GET", u, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}
		s := &Server{
			ChainInfoFetcher: &mockChain.ChainService{FinalizedCheckPoint: &eth.Checkpoint{Root: blockroot}},
			BeaconDB:         db,
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
			ChainInfoFetcher: &mockChain.ChainService{CurrentJustifiedCheckPoint: &eth.Checkpoint{Root: blockroot}},
			BeaconDB:         db,
		}

		s.Blobs(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &SidecarsResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.Equal(t, 4, len(resp.Data))
	})
	t.Run("root", func(t *testing.T) {
		u := "http://foo.example/" + hexutil.Encode(blockroot)
		request := httptest.NewRequest("GET", u, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}
		s := &Server{
			BeaconDB: db,
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
			BeaconDB: db,
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
			BeaconDB: db,
		}

		s.Blobs(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &SidecarsResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.Equal(t, 1, len(resp.Data))
		sidecar := resp.Data[0]
		require.NotNil(t, sidecar)
		assert.Equal(t, "0x626c6f636b726f6f740000000000000000000000000000000000000000000000", sidecar.BlockRoot)
		assert.Equal(t, "2", sidecar.Index)
		assert.Equal(t, "123", sidecar.Slot)
		assert.Equal(t, "0x626c6f636b706172656e74726f6f74", sidecar.BlockParentRoot)
		assert.Equal(t, "123", sidecar.ProposerIndex)
		assert.Equal(t, "0x626c6f6232", sidecar.Blob)
		assert.Equal(t, "0x6b7a67636f6d6d69746d656e7432", sidecar.KZGCommitment)
		assert.Equal(t, "0x6b7a6770726f6f6632", sidecar.KZGProof)
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
		assert.Equal(t, "blobs are not supported before Deneb fork", e.Message)
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
		require.NoError(t, db.SaveBlobSidecar(context.Background(), []*eth.BlobSidecar{
			{
				BlockRoot:       blockroot,
				Index:           0,
				Slot:            3,
				BlockParentRoot: make([]byte, fieldparams.RootLength),
				ProposerIndex:   123,
				Blob:            make([]byte, fieldparams.BlobLength),
				KzgCommitment:   make([]byte, fieldparams.BLSPubkeyLength),
				KzgProof:        make([]byte, fieldparams.BLSPubkeyLength),
			},
		}))
		u := "http://foo.example/finalized?indices=0"
		request := httptest.NewRequest("GET", u, nil)
		request.Header.Add("Accept", "application/octet-stream")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}
		s := &Server{
			ChainInfoFetcher: &mockChain.ChainService{FinalizedCheckPoint: &eth.Checkpoint{Root: blockroot}},
			BeaconDB:         db,
		}

		s.Blobs(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, len(writer.Body.Bytes()), 131260)
		assert.Equal(t, true, strings.HasPrefix(hexutil.Encode(writer.Body.Bytes()), "0x04000000626c6f636b726f6f7400000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000000000000000000007b"))
	})
}
