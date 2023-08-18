package validator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	dbutil "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	state_native "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/v4/time/slots"

	"github.com/ethereum/go-ethereum/common/hexutil"
	mockChain "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/synccommittee"
	p2pmock "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/core"
	mockSync "github.com/prysmaticlabs/prysm/v4/beacon-chain/sync/initial-sync/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestGetAggregateAttestation(t *testing.T) {
	root1 := bytesutil.PadTo([]byte("root1"), 32)
	sig1 := bytesutil.PadTo([]byte("sig1"), fieldparams.BLSSignatureLength)
	attSlot1 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            1,
			CommitteeIndex:  1,
			BeaconBlockRoot: root1,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root1,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root1,
			},
		},
		Signature: sig1,
	}
	root21 := bytesutil.PadTo([]byte("root2_1"), 32)
	sig21 := bytesutil.PadTo([]byte("sig2_1"), fieldparams.BLSSignatureLength)
	attslot21 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 1, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            2,
			CommitteeIndex:  2,
			BeaconBlockRoot: root21,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root21,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root21,
			},
		},
		Signature: sig21,
	}
	root22 := bytesutil.PadTo([]byte("root2_2"), 32)
	sig22 := bytesutil.PadTo([]byte("sig2_2"), fieldparams.BLSSignatureLength)
	attslot22 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 1, 1, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            2,
			CommitteeIndex:  3,
			BeaconBlockRoot: root22,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root22,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root22,
			},
		},
		Signature: sig22,
	}
	root33 := bytesutil.PadTo([]byte("root3_3"), 32)
	sig33 := bls.NewAggregateSignature().Marshal()
	attslot33 := &ethpbalpha.Attestation{
		AggregationBits: []byte{1, 0, 0, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            2,
			CommitteeIndex:  3,
			BeaconBlockRoot: root33,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root33,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root33,
			},
		},
		Signature: sig33,
	}
	pool := attestations.NewPool()
	err := pool.SaveAggregatedAttestations([]*ethpbalpha.Attestation{attSlot1, attslot21, attslot22})
	assert.NoError(t, err)
	s := &Server{
		AttestationsPool: pool,
	}

	t.Run("ok", func(t *testing.T) {
		reqRoot, err := attslot22.Data.HashTreeRoot()
		require.NoError(t, err)
		attDataRoot := hexutil.Encode(reqRoot[:])
		url := "http://example.com?attestation_data_root=" + attDataRoot + "&slot=2"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAggregateAttestation(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &AggregateAttestationResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		assert.DeepEqual(t, "0x00010101", resp.Data.AggregationBits)
		assert.DeepEqual(t, hexutil.Encode(sig22), resp.Data.Signature)
		assert.Equal(t, "2", resp.Data.Data.Slot)
		assert.Equal(t, "3", resp.Data.Data.CommitteeIndex)
		assert.DeepEqual(t, hexutil.Encode(root22), resp.Data.Data.BeaconBlockRoot)
		require.NotNil(t, resp.Data.Data.Source)
		assert.Equal(t, "1", resp.Data.Data.Source.Epoch)
		assert.DeepEqual(t, hexutil.Encode(root22), resp.Data.Data.Source.Root)
		require.NotNil(t, resp.Data.Data.Target)
		assert.Equal(t, "1", resp.Data.Data.Target.Epoch)
		assert.DeepEqual(t, hexutil.Encode(root22), resp.Data.Data.Target.Root)
	})

	t.Run("aggregate beforehand", func(t *testing.T) {
		err = s.AttestationsPool.SaveUnaggregatedAttestation(attslot33)
		require.NoError(t, err)
		newAtt := ethpbalpha.CopyAttestation(attslot33)
		newAtt.AggregationBits = []byte{0, 1, 0, 1}
		err = s.AttestationsPool.SaveUnaggregatedAttestation(newAtt)
		require.NoError(t, err)

		reqRoot, err := attslot33.Data.HashTreeRoot()
		require.NoError(t, err)
		attDataRoot := hexutil.Encode(reqRoot[:])
		url := "http://example.com?attestation_data_root=" + attDataRoot + "&slot=2"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAggregateAttestation(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &AggregateAttestationResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.NotNil(t, resp)
		assert.DeepEqual(t, "0x01010001", resp.Data.AggregationBits)
	})
	t.Run("no matching attestation", func(t *testing.T) {
		attDataRoot := hexutil.Encode(bytesutil.PadTo([]byte("foo"), 32))
		url := "http://example.com?attestation_data_root=" + attDataRoot + "&slot=2"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAggregateAttestation(writer, request)
		assert.Equal(t, http.StatusNotFound, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusNotFound, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "No matching attestation found"))
	})
	t.Run("no attestation_data_root provided", func(t *testing.T) {
		url := "http://example.com?slot=2"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAggregateAttestation(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "Attestation data root is required"))
	})
	t.Run("invalid attestation_data_root provided", func(t *testing.T) {
		url := "http://example.com?attestation_data_root=foo&slot=2"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAggregateAttestation(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "Attestation data root is invalid"))
	})
	t.Run("no slot provided", func(t *testing.T) {
		attDataRoot := hexutil.Encode(bytesutil.PadTo([]byte("foo"), 32))
		url := "http://example.com?attestation_data_root=" + attDataRoot
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAggregateAttestation(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "Slot is required"))
	})
	t.Run("invalid slot provided", func(t *testing.T) {
		attDataRoot := hexutil.Encode(bytesutil.PadTo([]byte("foo"), 32))
		url := "http://example.com?attestation_data_root=" + attDataRoot + "&slot=foo"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAggregateAttestation(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "Slot is invalid"))
	})
}

func TestGetAggregateAttestation_SameSlotAndRoot_ReturnMostAggregationBits(t *testing.T) {
	root := bytesutil.PadTo([]byte("root"), 32)
	sig := bytesutil.PadTo([]byte("sig"), fieldparams.BLSSignatureLength)
	att1 := &ethpbalpha.Attestation{
		AggregationBits: []byte{3, 0, 0, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            1,
			CommitteeIndex:  1,
			BeaconBlockRoot: root,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
		},
		Signature: sig,
	}
	att2 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 3, 0, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            1,
			CommitteeIndex:  1,
			BeaconBlockRoot: root,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
		},
		Signature: sig,
	}
	pool := attestations.NewPool()
	err := pool.SaveAggregatedAttestations([]*ethpbalpha.Attestation{att1, att2})
	assert.NoError(t, err)
	s := &Server{
		AttestationsPool: pool,
	}
	reqRoot, err := att1.Data.HashTreeRoot()
	require.NoError(t, err)
	attDataRoot := hexutil.Encode(reqRoot[:])
	url := "http://example.com?attestation_data_root=" + attDataRoot + "&slot=1"
	request := httptest.NewRequest(http.MethodGet, url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetAggregateAttestation(writer, request)
	assert.Equal(t, http.StatusOK, writer.Code)
	resp := &AggregateAttestationResponse{}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
	require.NotNil(t, resp)
	assert.DeepEqual(t, "0x03000001", resp.Data.AggregationBits)
}

func TestSubmitContributionAndProofs(t *testing.T) {
	c := &core.Service{
		OperationNotifier: (&mockChain.ChainService{}).OperationNotifier(),
	}

	s := &Server{CoreService: c}

	t.Run("single", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		c.Broadcaster = broadcaster
		c.SyncCommitteePool = synccommittee.NewStore()

		var body bytes.Buffer
		_, err := body.WriteString(singleContribution)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitContributionAndProofs(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, 1, len(broadcaster.BroadcastMessages))
		contributions, err := c.SyncCommitteePool.SyncCommitteeContributions(1)
		require.NoError(t, err)
		assert.Equal(t, 1, len(contributions))
	})
	t.Run("multiple", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		c.Broadcaster = broadcaster
		c.SyncCommitteePool = synccommittee.NewStore()

		var body bytes.Buffer
		_, err := body.WriteString(multipleContributions)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitContributionAndProofs(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, 2, len(broadcaster.BroadcastMessages))
		contributions, err := c.SyncCommitteePool.SyncCommitteeContributions(1)
		require.NoError(t, err)
		assert.Equal(t, 2, len(contributions))
	})
	t.Run("no body", func(t *testing.T) {
		s.SyncCommitteePool = synccommittee.NewStore()

		request := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitContributionAndProofs(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "No data submitted"))
	})
	t.Run("empty", func(t *testing.T) {
		var body bytes.Buffer
		_, err := body.WriteString("[]")
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitContributionAndProofs(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "No data submitted"))
	})
	t.Run("invalid", func(t *testing.T) {
		c.SyncCommitteePool = synccommittee.NewStore()

		var body bytes.Buffer
		_, err := body.WriteString(invalidContribution)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitContributionAndProofs(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
	})

	t.Run("no body", func(t *testing.T) {
		c.SyncCommitteePool = synccommittee.NewStore()

		request := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitContributionAndProofs(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "No data submitted"))
	})
}

func TestSubmitAggregateAndProofs(t *testing.T) {
	c := &core.Service{
		GenesisTimeFetcher: &mockChain.ChainService{},
	}

	s := &Server{
		CoreService: c,
	}

	t.Run("single", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		c.Broadcaster = broadcaster

		var body bytes.Buffer
		_, err := body.WriteString(singleAggregate)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitAggregateAndProofs(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, 1, len(broadcaster.BroadcastMessages))
	})
	t.Run("multiple", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		c.Broadcaster = broadcaster
		c.SyncCommitteePool = synccommittee.NewStore()

		var body bytes.Buffer
		_, err := body.WriteString(multipleAggregates)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitAggregateAndProofs(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, 2, len(broadcaster.BroadcastMessages))
	})
	t.Run("no body", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitAggregateAndProofs(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "No data submitted"))
	})
	t.Run("empty", func(t *testing.T) {
		var body bytes.Buffer
		_, err := body.WriteString("[]")
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitAggregateAndProofs(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "No data submitted"))
	})
	t.Run("invalid", func(t *testing.T) {
		var body bytes.Buffer
		_, err := body.WriteString(invalidAggregate)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitAggregateAndProofs(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
	})
}

func TestSubmitSyncCommitteeSubscription(t *testing.T) {
	genesis := util.NewBeaconBlock()
	deposits, _, err := util.DeterministicDepositsAndKeys(64)
	require.NoError(t, err)
	eth1Data, err := util.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bs, err := util.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
	require.NoError(t, err, "Could not set up genesis state")
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	roots := make([][]byte, fieldparams.BlockRootsLength)
	roots[0] = genesisRoot[:]
	require.NoError(t, bs.SetBlockRoots(roots))

	pubkeys := make([][]byte, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubkeys[i] = deposits[i].Data.PublicKey
	}

	chainSlot := primitives.Slot(0)
	chain := &mockChain.ChainService{
		State: bs, Root: genesisRoot[:], Slot: &chainSlot,
	}
	s := &Server{
		HeadFetcher: chain,
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}

	t.Run("single", func(t *testing.T) {
		cache.SyncSubnetIDs.EmptyAllCaches()

		var body bytes.Buffer
		_, err := body.WriteString(singleSyncCommitteeSubscription)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitSyncCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		subnets, _, _, _ := cache.SyncSubnetIDs.GetSyncCommitteeSubnets(pubkeys[1], 0)
		require.Equal(t, 2, len(subnets))
		assert.Equal(t, uint64(0), subnets[0])
		assert.Equal(t, uint64(2), subnets[1])
	})
	t.Run("multiple", func(t *testing.T) {
		cache.SyncSubnetIDs.EmptyAllCaches()

		var body bytes.Buffer
		_, err := body.WriteString(multipleSyncCommitteeSubscription)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitSyncCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		subnets, _, _, _ := cache.SyncSubnetIDs.GetSyncCommitteeSubnets(pubkeys[0], 0)
		require.Equal(t, 1, len(subnets))
		assert.Equal(t, uint64(0), subnets[0])
		subnets, _, _, _ = cache.SyncSubnetIDs.GetSyncCommitteeSubnets(pubkeys[1], 0)
		require.Equal(t, 1, len(subnets))
		assert.Equal(t, uint64(2), subnets[0])
	})
	t.Run("no body", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitSyncCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "No data submitted"))
	})
	t.Run("empty", func(t *testing.T) {
		var body bytes.Buffer
		_, err := body.WriteString("[]")
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitSyncCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "No data submitted"))
	})
	t.Run("invalid", func(t *testing.T) {
		var body bytes.Buffer
		_, err := body.WriteString(invalidSyncCommitteeSubscription)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitSyncCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
	})
	t.Run("epoch in the past", func(t *testing.T) {
		var body bytes.Buffer
		_, err := body.WriteString(singleSyncCommitteeSubscription2)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitSyncCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "Epoch for subscription at index 0 is in the past"))
	})
	t.Run("first epoch after the next sync committee is valid", func(t *testing.T) {
		var body bytes.Buffer
		_, err := body.WriteString(singleSyncCommitteeSubscription3)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitSyncCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
	})
	t.Run("epoch too far in the future", func(t *testing.T) {
		var body bytes.Buffer
		_, err := body.WriteString(singleSyncCommitteeSubscription4)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitSyncCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "Epoch for subscription at index 0 is too far in the future"))
	})
	t.Run("sync not ready", func(t *testing.T) {
		st, err := util.NewBeaconState()
		require.NoError(t, err)
		chainService := &mockChain.ChainService{State: st}
		s := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: true},
			HeadFetcher:           chainService,
			TimeFetcher:           chainService,
			OptimisticModeFetcher: chainService,
		}

		request := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitSyncCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusServiceUnavailable, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusServiceUnavailable, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "Beacon node is currently syncing"))
	})
}

func TestSubmitBeaconCommitteeSubscription(t *testing.T) {
	genesis := util.NewBeaconBlock()
	depChainStart := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, err := util.DeterministicDepositsAndKeys(depChainStart)
	require.NoError(t, err)
	eth1Data, err := util.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bs, err := transition.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
	require.NoError(t, err, "Could not set up genesis state")
	// Set state to non-epoch start slot.
	require.NoError(t, bs.SetSlot(5))
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	roots := make([][]byte, fieldparams.BlockRootsLength)
	roots[0] = genesisRoot[:]
	require.NoError(t, bs.SetBlockRoots(roots))

	pubkeys := make([][]byte, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubkeys[i] = deposits[i].Data.PublicKey
	}

	chainSlot := primitives.Slot(0)
	chain := &mockChain.ChainService{
		State: bs, Root: genesisRoot[:], Slot: &chainSlot,
	}
	s := &Server{
		HeadFetcher: chain,
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}

	t.Run("single", func(t *testing.T) {
		cache.SubnetIDs.EmptyAllCaches()

		var body bytes.Buffer
		_, err := body.WriteString(singleBeaconCommitteeContribution)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitBeaconCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		subnets := cache.SubnetIDs.GetAttesterSubnetIDs(1)
		require.Equal(t, 1, len(subnets))
		assert.Equal(t, uint64(5), subnets[0])
	})
	t.Run("multiple", func(t *testing.T) {
		cache.SubnetIDs.EmptyAllCaches()

		var body bytes.Buffer
		_, err := body.WriteString(multipleBeaconCommitteeContribution)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitBeaconCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		subnets := cache.SubnetIDs.GetAttesterSubnetIDs(1)
		require.Equal(t, 2, len(subnets))
		assert.Equal(t, uint64(5), subnets[0])
		assert.Equal(t, uint64(4), subnets[1])
	})
	t.Run("is aggregator", func(t *testing.T) {
		cache.SubnetIDs.EmptyAllCaches()

		var body bytes.Buffer
		_, err := body.WriteString(singleBeaconCommitteeContribution2)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitBeaconCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		subnets := cache.SubnetIDs.GetAggregatorSubnetIDs(1)
		require.Equal(t, 1, len(subnets))
		assert.Equal(t, uint64(5), subnets[0])
	})
	t.Run("validators assigned to subnets", func(t *testing.T) {
		cache.SubnetIDs.EmptyAllCaches()

		var body bytes.Buffer
		_, err := body.WriteString(multipleBeaconCommitteeContribution2)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitBeaconCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		subnets, ok, _ := cache.SubnetIDs.GetPersistentSubnets(pubkeys[1])
		require.Equal(t, true, ok, "subnet for validator 1 not found")
		assert.Equal(t, 1, len(subnets))
		subnets, ok, _ = cache.SubnetIDs.GetPersistentSubnets(pubkeys[2])
		require.Equal(t, true, ok, "subnet for validator 2 not found")
		assert.Equal(t, 1, len(subnets))
	})
	t.Run("no body", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitBeaconCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "No data submitted"))
	})
	t.Run("empty", func(t *testing.T) {
		var body bytes.Buffer
		_, err := body.WriteString("[]")
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitBeaconCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "No data submitted"))
	})
	t.Run("invalid", func(t *testing.T) {
		var body bytes.Buffer
		_, err := body.WriteString(invalidBeaconCommitteeContribution)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitBeaconCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
	})
	t.Run("sync not ready", func(t *testing.T) {
		st, err := util.NewBeaconState()
		require.NoError(t, err)
		chainService := &mockChain.ChainService{State: st}
		s := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: true},
			HeadFetcher:           chainService,
			TimeFetcher:           chainService,
			OptimisticModeFetcher: chainService,
		}

		request := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitBeaconCommitteeSubscription(writer, request)
		assert.Equal(t, http.StatusServiceUnavailable, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusServiceUnavailable, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "Beacon node is currently syncing"))
	})
}

func TestGetAttestationData(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		block := util.NewBeaconBlock()
		block.Block.Slot = 3*params.BeaconConfig().SlotsPerEpoch + 1
		targetBlock := util.NewBeaconBlock()
		targetBlock.Block.Slot = 1 * params.BeaconConfig().SlotsPerEpoch
		justifiedBlock := util.NewBeaconBlock()
		justifiedBlock.Block.Slot = 2 * params.BeaconConfig().SlotsPerEpoch
		blockRoot, err := block.Block.HashTreeRoot()
		require.NoError(t, err, "Could not hash beacon block")
		justifiedRoot, err := justifiedBlock.Block.HashTreeRoot()
		require.NoError(t, err, "Could not get signing root for justified block")
		targetRoot, err := targetBlock.Block.HashTreeRoot()
		require.NoError(t, err, "Could not get signing root for target block")
		slot := 3*params.BeaconConfig().SlotsPerEpoch + 1
		beaconState, err := util.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, beaconState.SetSlot(slot))
		err = beaconState.SetCurrentJustifiedCheckpoint(&ethpbalpha.Checkpoint{
			Epoch: 2,
			Root:  justifiedRoot[:],
		})
		require.NoError(t, err)

		blockRoots := beaconState.BlockRoots()
		blockRoots[1] = blockRoot[:]
		blockRoots[1*params.BeaconConfig().SlotsPerEpoch] = targetRoot[:]
		blockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedRoot[:]
		require.NoError(t, beaconState.SetBlockRoots(blockRoots))
		offset := int64(slot.Mul(params.BeaconConfig().SecondsPerSlot))
		chain := &mockChain.ChainService{
			Optimistic: false,
			Genesis:    time.Now().Add(time.Duration(-1*offset) * time.Second),
			State:      beaconState,
			Root:       blockRoot[:],
		}

		s := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			HeadFetcher:           chain,
			TimeFetcher:           chain,
			OptimisticModeFetcher: chain,
			CoreService: &core.Service{
				AttestationCache:   cache.NewAttestationCache(),
				HeadFetcher:        chain,
				GenesisTimeFetcher: chain,
			},
		}

		url := fmt.Sprintf("http://example.com?slot=%d&committee_index=%d", slot, 0)
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAttestationData(writer, request)

		expectedResponse := &GetAttestationDataResponse{
			Data: &shared.AttestationData{
				Slot:            strconv.FormatUint(uint64(slot), 10),
				BeaconBlockRoot: hexutil.Encode(blockRoot[:]),
				CommitteeIndex:  strconv.FormatUint(0, 10),
				Source: &shared.Checkpoint{
					Epoch: strconv.FormatUint(2, 10),
					Root:  hexutil.Encode(justifiedRoot[:]),
				},
				Target: &shared.Checkpoint{
					Epoch: strconv.FormatUint(3, 10),
					Root:  hexutil.Encode(blockRoot[:]),
				},
			},
		}

		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &GetAttestationDataResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.NotNil(t, resp)
		assert.DeepEqual(t, expectedResponse, resp)
	})

	t.Run("syncing", func(t *testing.T) {
		beaconState, err := util.NewBeaconState()
		require.NoError(t, err)
		chain := &mockChain.ChainService{
			Optimistic: false,
			State:      beaconState,
			Genesis:    time.Now(),
		}

		s := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: true},
			HeadFetcher:           chain,
			TimeFetcher:           chain,
			OptimisticModeFetcher: chain,
		}

		url := fmt.Sprintf("http://example.com?slot=%d&committee_index=%d", 1, 2)
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAttestationData(writer, request)

		assert.Equal(t, http.StatusServiceUnavailable, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusServiceUnavailable, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "syncing"))
	})

	t.Run("optimistic", func(t *testing.T) {
		beaconState, err := util.NewBeaconState()
		require.NoError(t, err)
		chain := &mockChain.ChainService{
			Optimistic: true,
			State:      beaconState,
			Genesis:    time.Now(),
		}

		s := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			HeadFetcher:           chain,
			TimeFetcher:           chain,
			OptimisticModeFetcher: chain,
			CoreService: &core.Service{
				AttestationCache:   cache.NewAttestationCache(),
				GenesisTimeFetcher: chain,
				HeadFetcher:        chain,
			},
		}

		url := fmt.Sprintf("http://example.com?slot=%d&committee_index=%d", 0, 0)
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAttestationData(writer, request)

		assert.Equal(t, http.StatusServiceUnavailable, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusServiceUnavailable, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "optimistic"))

		chain.Optimistic = false

		writer = httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAttestationData(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
	})

	t.Run("handles in progress request", func(t *testing.T) {
		state, err := state_native.InitializeFromProtoPhase0(&ethpbalpha.BeaconState{Slot: 100})
		require.NoError(t, err)
		ctx := context.Background()
		slot := primitives.Slot(2)
		offset := int64(slot.Mul(params.BeaconConfig().SecondsPerSlot))
		chain := &mockChain.ChainService{
			Optimistic: false,
			Genesis:    time.Now().Add(time.Duration(-1*offset) * time.Second),
			State:      state,
		}

		s := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			HeadFetcher:           chain,
			TimeFetcher:           chain,
			OptimisticModeFetcher: chain,
			CoreService: &core.Service{
				AttestationCache:   cache.NewAttestationCache(),
				HeadFetcher:        chain,
				GenesisTimeFetcher: chain,
			},
		}

		expectedResponse := &GetAttestationDataResponse{
			Data: &shared.AttestationData{
				Slot:            strconv.FormatUint(uint64(slot), 10),
				CommitteeIndex:  strconv.FormatUint(1, 10),
				BeaconBlockRoot: hexutil.Encode(make([]byte, 32)),
				Source: &shared.Checkpoint{
					Epoch: strconv.FormatUint(42, 10),
					Root:  hexutil.Encode(make([]byte, 32)),
				},
				Target: &shared.Checkpoint{
					Epoch: strconv.FormatUint(55, 10),
					Root:  hexutil.Encode(make([]byte, 32)),
				},
			},
		}

		expectedResponsePb := &ethpbalpha.AttestationData{
			Slot:            slot,
			CommitteeIndex:  1,
			BeaconBlockRoot: make([]byte, 32),
			Source:          &ethpbalpha.Checkpoint{Epoch: 42, Root: make([]byte, 32)},
			Target:          &ethpbalpha.Checkpoint{Epoch: 55, Root: make([]byte, 32)},
		}

		url := fmt.Sprintf("http://example.com?slot=%d&committee_index=%d", slot, 1)
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		requestPb := &ethpbalpha.AttestationDataRequest{
			CommitteeIndex: 1,
			Slot:           slot,
		}

		require.NoError(t, s.CoreService.AttestationCache.MarkInProgress(requestPb))

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			s.GetAttestationData(writer, request)

			assert.Equal(t, http.StatusOK, writer.Code)
			resp := &GetAttestationDataResponse{}
			require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
			require.NotNil(t, resp)
			assert.DeepEqual(t, expectedResponse, resp)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			assert.NoError(t, s.CoreService.AttestationCache.Put(ctx, requestPb, expectedResponsePb))
			assert.NoError(t, s.CoreService.AttestationCache.MarkNotInProgress(requestPb))
		}()

		wg.Wait()
	})

	t.Run("invalid slot", func(t *testing.T) {
		slot := 3*params.BeaconConfig().SlotsPerEpoch + 1
		offset := int64(slot.Mul(params.BeaconConfig().SecondsPerSlot))
		chain := &mockChain.ChainService{
			Optimistic: false,
			Genesis:    time.Now().Add(time.Duration(-1*offset) * time.Second),
		}

		s := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			HeadFetcher:           chain,
			TimeFetcher:           chain,
			OptimisticModeFetcher: chain,
			CoreService: &core.Service{
				GenesisTimeFetcher: chain,
			},
		}

		url := fmt.Sprintf("http://example.com?slot=%d&committee_index=%d", 1000000000000, 2)
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAttestationData(writer, request)

		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "invalid request"))
	})

	t.Run("head state slot greater than request slot", func(t *testing.T) {
		ctx := context.Background()
		db := dbutil.SetupDB(t)

		slot := 3*params.BeaconConfig().SlotsPerEpoch + 1
		block := util.NewBeaconBlock()
		block.Block.Slot = slot
		block2 := util.NewBeaconBlock()
		block2.Block.Slot = slot - 1
		targetBlock := util.NewBeaconBlock()
		targetBlock.Block.Slot = 1 * params.BeaconConfig().SlotsPerEpoch
		justifiedBlock := util.NewBeaconBlock()
		justifiedBlock.Block.Slot = 2 * params.BeaconConfig().SlotsPerEpoch
		blockRoot, err := block.Block.HashTreeRoot()
		require.NoError(t, err, "Could not hash beacon block")
		blockRoot2, err := block2.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, block2)
		justifiedRoot, err := justifiedBlock.Block.HashTreeRoot()
		require.NoError(t, err, "Could not get signing root for justified block")
		targetRoot, err := targetBlock.Block.HashTreeRoot()
		require.NoError(t, err, "Could not get signing root for target block")

		beaconState, err := util.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, beaconState.SetSlot(slot))
		offset := int64(slot.Mul(params.BeaconConfig().SecondsPerSlot))
		require.NoError(t, beaconState.SetGenesisTime(uint64(time.Now().Unix()-offset)))
		err = beaconState.SetLatestBlockHeader(util.HydrateBeaconHeader(&ethpbalpha.BeaconBlockHeader{
			ParentRoot: blockRoot2[:],
		}))
		require.NoError(t, err)
		err = beaconState.SetCurrentJustifiedCheckpoint(&ethpbalpha.Checkpoint{
			Epoch: 2,
			Root:  justifiedRoot[:],
		})
		require.NoError(t, err)
		blockRoots := beaconState.BlockRoots()
		blockRoots[1] = blockRoot[:]
		blockRoots[1*params.BeaconConfig().SlotsPerEpoch] = targetRoot[:]
		blockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedRoot[:]
		blockRoots[3*params.BeaconConfig().SlotsPerEpoch] = blockRoot2[:]
		require.NoError(t, beaconState.SetBlockRoots(blockRoots))

		beaconstate := beaconState.Copy()
		require.NoError(t, beaconstate.SetSlot(beaconstate.Slot()-1))
		require.NoError(t, db.SaveState(ctx, beaconstate, blockRoot2))
		chain := &mockChain.ChainService{
			State:   beaconState,
			Root:    blockRoot[:],
			Genesis: time.Now().Add(time.Duration(-1*offset) * time.Second),
		}

		s := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			HeadFetcher:           chain,
			TimeFetcher:           chain,
			OptimisticModeFetcher: chain,
			CoreService: &core.Service{
				AttestationCache:   cache.NewAttestationCache(),
				HeadFetcher:        chain,
				GenesisTimeFetcher: chain,
				StateGen:           stategen.New(db, doublylinkedtree.New()),
			},
		}

		require.NoError(t, db.SaveState(ctx, beaconState, blockRoot))
		util.SaveBlock(t, ctx, db, block)
		require.NoError(t, db.SaveHeadBlockRoot(ctx, blockRoot))

		url := fmt.Sprintf("http://example.com?slot=%d&committee_index=%d", slot-1, 0)
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAttestationData(writer, request)

		expectedResponse := &GetAttestationDataResponse{
			Data: &shared.AttestationData{
				Slot:            strconv.FormatUint(uint64(slot-1), 10),
				CommitteeIndex:  strconv.FormatUint(0, 10),
				BeaconBlockRoot: hexutil.Encode(blockRoot2[:]),
				Source: &shared.Checkpoint{
					Epoch: strconv.FormatUint(2, 10),
					Root:  hexutil.Encode(justifiedRoot[:]),
				},
				Target: &shared.Checkpoint{
					Epoch: strconv.FormatUint(3, 10),
					Root:  hexutil.Encode(blockRoot2[:]),
				},
			},
		}

		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &GetAttestationDataResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.NotNil(t, resp)
		assert.DeepEqual(t, expectedResponse, resp)
	})

	t.Run("succeeds in first epoch", func(t *testing.T) {
		slot := primitives.Slot(5)
		block := util.NewBeaconBlock()
		block.Block.Slot = slot
		targetBlock := util.NewBeaconBlock()
		targetBlock.Block.Slot = 0
		justifiedBlock := util.NewBeaconBlock()
		justifiedBlock.Block.Slot = 0
		blockRoot, err := block.Block.HashTreeRoot()
		require.NoError(t, err, "Could not hash beacon block")
		justifiedRoot, err := justifiedBlock.Block.HashTreeRoot()
		require.NoError(t, err, "Could not get signing root for justified block")
		targetRoot, err := targetBlock.Block.HashTreeRoot()
		require.NoError(t, err, "Could not get signing root for target block")

		beaconState, err := util.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, beaconState.SetSlot(slot))
		err = beaconState.SetCurrentJustifiedCheckpoint(&ethpbalpha.Checkpoint{
			Epoch: 0,
			Root:  justifiedRoot[:],
		})
		require.NoError(t, err)
		blockRoots := beaconState.BlockRoots()
		blockRoots[1] = blockRoot[:]
		blockRoots[1*params.BeaconConfig().SlotsPerEpoch] = targetRoot[:]
		blockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedRoot[:]
		require.NoError(t, beaconState.SetBlockRoots(blockRoots))
		offset := int64(slot.Mul(params.BeaconConfig().SecondsPerSlot))
		chain := &mockChain.ChainService{
			State:   beaconState,
			Root:    blockRoot[:],
			Genesis: time.Now().Add(time.Duration(-1*offset) * time.Second),
		}

		s := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			HeadFetcher:           chain,
			TimeFetcher:           chain,
			OptimisticModeFetcher: chain,
			CoreService: &core.Service{
				AttestationCache:   cache.NewAttestationCache(),
				HeadFetcher:        chain,
				GenesisTimeFetcher: chain,
			},
		}

		url := fmt.Sprintf("http://example.com?slot=%d&committee_index=%d", slot, 0)
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAttestationData(writer, request)

		expectedResponse := &GetAttestationDataResponse{
			Data: &shared.AttestationData{
				Slot:            strconv.FormatUint(uint64(slot), 10),
				BeaconBlockRoot: hexutil.Encode(blockRoot[:]),
				CommitteeIndex:  strconv.FormatUint(0, 10),
				Source: &shared.Checkpoint{
					Epoch: strconv.FormatUint(0, 10),
					Root:  hexutil.Encode(justifiedRoot[:]),
				},
				Target: &shared.Checkpoint{
					Epoch: strconv.FormatUint(0, 10),
					Root:  hexutil.Encode(blockRoot[:]),
				},
			},
		}

		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &GetAttestationDataResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.NotNil(t, resp)
		assert.DeepEqual(t, expectedResponse, resp)
	})

	t.Run("handles far away justified epoch", func(t *testing.T) {
		// Scenario:
		//
		// State slot = 10000
		// Last justified slot = epoch start of 1500
		// HistoricalRootsLimit = 8192
		//
		// More background: https://github.com/prysmaticlabs/prysm/issues/2153
		// This test breaks if it doesn't use mainnet config

		// Ensure HistoricalRootsLimit matches scenario
		params.SetupTestConfigCleanup(t)
		cfg := params.MainnetConfig().Copy()
		cfg.HistoricalRootsLimit = 8192
		params.OverrideBeaconConfig(cfg)

		block := util.NewBeaconBlock()
		block.Block.Slot = 10000
		epochBoundaryBlock := util.NewBeaconBlock()
		var err error
		epochBoundaryBlock.Block.Slot, err = slots.EpochStart(slots.ToEpoch(10000))
		require.NoError(t, err)
		justifiedBlock := util.NewBeaconBlock()
		justifiedBlock.Block.Slot, err = slots.EpochStart(slots.ToEpoch(1500))
		require.NoError(t, err)
		justifiedBlock.Block.Slot -= 2 // Imagine two skip block
		blockRoot, err := block.Block.HashTreeRoot()
		require.NoError(t, err, "Could not hash beacon block")
		justifiedBlockRoot, err := justifiedBlock.Block.HashTreeRoot()
		require.NoError(t, err, "Could not hash justified block")
		epochBoundaryRoot, err := epochBoundaryBlock.Block.HashTreeRoot()
		require.NoError(t, err, "Could not hash justified block")
		slot := primitives.Slot(10000)

		beaconState, err := util.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, beaconState.SetSlot(slot))
		err = beaconState.SetCurrentJustifiedCheckpoint(&ethpbalpha.Checkpoint{
			Epoch: slots.ToEpoch(1500),
			Root:  justifiedBlockRoot[:],
		})
		require.NoError(t, err)
		blockRoots := beaconState.BlockRoots()
		blockRoots[1] = blockRoot[:]
		blockRoots[1*params.BeaconConfig().SlotsPerEpoch] = epochBoundaryRoot[:]
		blockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedBlockRoot[:]
		require.NoError(t, beaconState.SetBlockRoots(blockRoots))
		offset := int64(slot.Mul(params.BeaconConfig().SecondsPerSlot))
		chain := &mockChain.ChainService{
			State:   beaconState,
			Root:    blockRoot[:],
			Genesis: time.Now().Add(time.Duration(-1*offset) * time.Second),
		}

		s := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			HeadFetcher:           chain,
			TimeFetcher:           chain,
			OptimisticModeFetcher: chain,
			CoreService: &core.Service{
				AttestationCache:   cache.NewAttestationCache(),
				HeadFetcher:        chain,
				GenesisTimeFetcher: chain,
			},
		}

		url := fmt.Sprintf("http://example.com?slot=%d&committee_index=%d", slot, 0)
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAttestationData(writer, request)

		expectedResponse := &GetAttestationDataResponse{
			Data: &shared.AttestationData{
				Slot:            strconv.FormatUint(uint64(slot), 10),
				BeaconBlockRoot: hexutil.Encode(blockRoot[:]),
				CommitteeIndex:  strconv.FormatUint(0, 10),
				Source: &shared.Checkpoint{
					Epoch: strconv.FormatUint(uint64(slots.ToEpoch(1500)), 10),
					Root:  hexutil.Encode(justifiedBlockRoot[:]),
				},
				Target: &shared.Checkpoint{
					Epoch: strconv.FormatUint(312, 10),
					Root:  hexutil.Encode(blockRoot[:]),
				},
			},
		}

		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &GetAttestationDataResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.NotNil(t, resp)
		assert.DeepEqual(t, expectedResponse, resp)
	})
}

func TestProduceSyncCommitteeContribution(t *testing.T) {
	root := bytesutil.PadTo([]byte("0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"), 32)
	sig := bls.NewAggregateSignature().Marshal()
	messsage := &ethpbalpha.SyncCommitteeMessage{
		Slot:           1,
		BlockRoot:      root,
		ValidatorIndex: 0,
		Signature:      sig,
	}
	syncCommitteePool := synccommittee.NewStore()
	require.NoError(t, syncCommitteePool.SaveSyncCommitteeMessage(messsage))
	server := Server{
		CoreService: &core.Service{
			HeadFetcher: &mockChain.ChainService{
				SyncCommitteeIndices: []primitives.CommitteeIndex{0},
			},
		},
		SyncCommitteePool: syncCommitteePool,
	}
	t.Run("ok", func(t *testing.T) {
		url := "http://example.com?slot=1&subcommittee_index=1&beacon_block_root=0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		server.ProduceSyncCommitteeContribution(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &ProduceSyncCommitteeContributionResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.NotNil(t, resp.Data)
		require.Equal(t, resp.Data.Slot, "1")
		require.Equal(t, resp.Data.SubcommitteeIndex, "1")
		require.Equal(t, resp.Data.BeaconBlockRoot, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	})
	t.Run("no slot provided", func(t *testing.T) {
		url := "http://example.com?subcommittee_index=1&beacon_block_root=0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		server.ProduceSyncCommitteeContribution(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		resp := &ProduceSyncCommitteeContributionResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.ErrorContains(t, "Slot is required", errors.New(writer.Body.String()))
	})
	t.Run("no subcommittee_index provided", func(t *testing.T) {
		url := "http://example.com?slot=1&beacon_block_root=0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		server.ProduceSyncCommitteeContribution(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		resp := &ProduceSyncCommitteeContributionResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.ErrorContains(t, "Subcommittee Index is required", errors.New(writer.Body.String()))
	})
	t.Run("no beacon_block_root provided", func(t *testing.T) {
		url := "http://example.com?slot=1&subcommittee_index=1"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		server.ProduceSyncCommitteeContribution(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		resp := &ProduceSyncCommitteeContributionResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.ErrorContains(t, "Invalid Beacon Block Root: empty hex string", errors.New(writer.Body.String()))
	})
	t.Run("invalid block root", func(t *testing.T) {
		url := "http://example.com?slot=1&subcommittee_index=1&beacon_block_root=0"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		server.ProduceSyncCommitteeContribution(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		resp := &ProduceSyncCommitteeContributionResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.ErrorContains(t, "Invalid Beacon Block Root: hex string without 0x prefix", errors.New(writer.Body.String()))
	})
	t.Run("no committee messages", func(t *testing.T) {
		url := "http://example.com?slot=1&subcommittee_index=1&beacon_block_root=0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		server.ProduceSyncCommitteeContribution(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &ProduceSyncCommitteeContributionResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)

		request = httptest.NewRequest(http.MethodGet, url, nil)
		writer = httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}
		syncCommitteePool = synccommittee.NewStore()
		server = Server{
			CoreService: &core.Service{
				HeadFetcher: &mockChain.ChainService{
					SyncCommitteeIndices: []primitives.CommitteeIndex{0},
				},
			},
			SyncCommitteePool: syncCommitteePool,
		}
		server.ProduceSyncCommitteeContribution(writer, request)
		assert.Equal(t, http.StatusNotFound, writer.Code)
		resp2 := &ProduceSyncCommitteeContributionResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp2))
		require.ErrorContains(t, "No subcommittee messages found", errors.New(writer.Body.String()))
	})
}

var (
	singleContribution = `[
  {
    "message": {
      "aggregator_index": "1",
      "selection_proof": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
      "contribution": {
        "slot": "1",
        "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "subcommittee_index": "1",
        "aggregation_bits": "0x01",
        "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
      }
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }
]`
	multipleContributions = `[
  {
    "message": {
      "aggregator_index": "1",
      "selection_proof": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
      "contribution": {
        "slot": "1",
        "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "subcommittee_index": "1",
        "aggregation_bits": "0x01",
        "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
      }
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  },
  {
    "message": {
      "aggregator_index": "1",
      "selection_proof": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
      "contribution": {
        "slot": "1",
        "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "subcommittee_index": "1",
        "aggregation_bits": "0x01",
        "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
      }
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }
]`
	// aggregator_index is invalid
	invalidContribution = `[
  {
    "message": {
      "aggregator_index": "foo",
      "selection_proof": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
      "contribution": {
        "slot": "1",
        "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "subcommittee_index": "1",
        "aggregation_bits": "0x01",
        "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
      }
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }
]`
	singleAggregate = `[
  {
    "message": {
      "aggregator_index": "1",
      "aggregate": {
        "aggregation_bits": "0x01",
        "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
        "data": {
          "slot": "1",
          "index": "1",
          "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "source": {
            "epoch": "1",
            "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
          },
          "target": {
            "epoch": "1",
            "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
          }
        }
      },
      "selection_proof": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }
]`
	multipleAggregates = `[
  {
    "message": {
      "aggregator_index": "1",
      "aggregate": {
        "aggregation_bits": "0x01",
        "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
        "data": {
          "slot": "1",
          "index": "1",
          "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "source": {
            "epoch": "1",
            "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
          },
          "target": {
            "epoch": "1",
            "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
          }
        }
      },
      "selection_proof": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  },
{
    "message": {
      "aggregator_index": "1",
      "aggregate": {
        "aggregation_bits": "0x01",
        "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
        "data": {
          "slot": "1",
          "index": "1",
          "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "source": {
            "epoch": "1",
            "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
          },
          "target": {
            "epoch": "1",
            "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
          }
        }
      },
      "selection_proof": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }
]
`
	// aggregator_index is invalid
	invalidAggregate = `[
  {
    "message": {
      "aggregator_index": "foo",
      "aggregate": {
        "aggregation_bits": "0x01",
        "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
        "data": {
          "slot": "1",
          "index": "1",
          "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
          "source": {
            "epoch": "1",
            "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
          },
          "target": {
            "epoch": "1",
            "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
          }
        }
      },
      "selection_proof": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }
]`
	singleSyncCommitteeSubscription = `[
  {
    "validator_index": "1",
    "sync_committee_indices": [
      "0",
      "2"
    ],
    "until_epoch": "1"
  }
]`
	singleSyncCommitteeSubscription2 = `[
  {
    "validator_index": "0",
    "sync_committee_indices": [
      "0",
      "2"
    ],
    "until_epoch": "0"
  }
]`
	singleSyncCommitteeSubscription3 = fmt.Sprintf(`[
  {
    "validator_index": "0",
    "sync_committee_indices": [
      "0",
      "2"
    ],
    "until_epoch": "%d"
  }
]`, 2*params.BeaconConfig().EpochsPerSyncCommitteePeriod)
	singleSyncCommitteeSubscription4 = fmt.Sprintf(`[
  {
    "validator_index": "0",
    "sync_committee_indices": [
      "0",
      "2"
    ],
    "until_epoch": "%d"
  }
]`, 2*params.BeaconConfig().EpochsPerSyncCommitteePeriod+1)
	multipleSyncCommitteeSubscription = `[
  {
    "validator_index": "0",
    "sync_committee_indices": [
      "0"
    ],
    "until_epoch": "1"
  },
  {
    "validator_index": "1",
    "sync_committee_indices": [
      "2"
    ],
    "until_epoch": "1"
  }
]`
	// validator_index is invalid
	invalidSyncCommitteeSubscription = `[
  {
    "validator_index": "foo",
    "sync_committee_indices": [
      "0",
      "2"
    ],
    "until_epoch": "1"
  }
]`
	singleBeaconCommitteeContribution = `[
  {
    "validator_index": "1",
    "committee_index": "1",
    "committees_at_slot": "2",
    "slot": "1",
    "is_aggregator": false
  }
]`
	singleBeaconCommitteeContribution2 = `[
  {
    "validator_index": "1",
    "committee_index": "1",
    "committees_at_slot": "2",
    "slot": "1",
    "is_aggregator": true
  }
]`
	multipleBeaconCommitteeContribution = `[
  {
    "validator_index": "1",
    "committee_index": "1",
    "committees_at_slot": "2",
    "slot": "1",
    "is_aggregator": false
  },
  {
    "validator_index": "2",
    "committee_index": "0",
    "committees_at_slot": "2",
    "slot": "1",
    "is_aggregator": false
  }
]`
	multipleBeaconCommitteeContribution2 = `[
  {
    "validator_index": "1",
    "committee_index": "1",
    "committees_at_slot": "2",
    "slot": "1",
    "is_aggregator": true
  },
  {
    "validator_index": "2",
    "committee_index": "1",
    "committees_at_slot": "2",
    "slot": "1",
    "is_aggregator": false
  }
]`
	// validator_index is invalid
	invalidBeaconCommitteeContribution = `[
  {
    "validator_index": "foo",
    "committee_index": "1",
    "committees_at_slot": "2",
    "slot": "1",
    "is_aggregator": false
  }
]`
)
