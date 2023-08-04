package validator

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	mockChain "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/synccommittee"
	v1alpha1validator "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/prysm/v1alpha1/validator"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestProduceSyncCommitteeContribution(t *testing.T) {
	root := bytesutil.PadTo([]byte("root"), 32)
	sig := bls.NewAggregateSignature().Marshal()
	messsage := &ethpbalpha.SyncCommitteeMessage{
		Slot:           0,
		BlockRoot:      root,
		ValidatorIndex: 0,
		Signature:      sig,
	}
	syncCommitteePool := synccommittee.NewStore()
	require.NoError(t, syncCommitteePool.SaveSyncCommitteeMessage(messsage))

	vs := &Server{
		SyncCommitteePool: syncCommitteePool,
		V1Alpha1Server: &v1alpha1validator.Server{
			SyncCommitteePool: syncCommitteePool,
			HeadFetcher: &mockChain.ChainService{
				SyncCommitteeIndices: []primitives.CommitteeIndex{0},
			},
		},
	}

	var buf bytes.Buffer
	srv := httptest.NewServer(http.HandlerFunc(vs.ProduceSyncCommitteeContribution))
	client := &http.Client{}
	request := &ProduceSyncCommitteeContributionRequest{
		Slot:              0,
		SubcommitteeIndex: 0,
		BeaconBlockRoot:   root,
	}
	err := json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/foo", &buf)
	rawResp, err := client.Post(srv.URL, "application/json", req.Body)
	require.NoError(t, err)
	defer func() {
		if err := rawResp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	body, err := io.ReadAll(rawResp.Body)
	require.NoError(t, err)

	resp := &ProduceSyncCommitteeContributionResponse{}
	require.NoError(t, json.Unmarshal(body, resp))

	assert.Equal(t, primitives.Slot(0), resp.Data.Slot)
	assert.Equal(t, uint64(0), resp.Data.SubcommitteeIndex)
	assert.DeepEqual(t, root, resp.Data.BeaconBlockRoot)
	aggregationBits := resp.Data.AggregationBits
	assert.Equal(t, true, aggregationBits.BitAt(0))
	assert.DeepEqual(t, sig, resp.Data.Signature)
}
