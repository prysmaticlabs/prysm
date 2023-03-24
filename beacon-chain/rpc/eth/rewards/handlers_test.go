package rewards

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/testutil"
	mockstategen "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stategen/mock"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/blst"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/network"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestBlockRewards(t *testing.T) {
	valCount := 64

	st, err := util.NewBeaconStateAltair()
	require.NoError(t, st.SetSlot(1))
	require.NoError(t, err)
	validators := make([]*eth.Validator, 0, valCount)
	balances := make([]uint64, 0, valCount)
	secretKeys := make([]bls.SecretKey, 0, valCount)
	for i := 0; i < valCount; i++ {
		blsKey, err := bls.RandKey()
		require.NoError(t, err)
		secretKeys = append(secretKeys, blsKey)
		validators = append(validators, &eth.Validator{
			PublicKey:         blsKey.PublicKey().Marshal(),
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
		})
		balances = append(balances, params.BeaconConfig().MaxEffectiveBalance)
	}
	require.NoError(t, st.SetValidators(validators))
	require.NoError(t, st.SetBalances(balances))
	require.NoError(t, st.SetCurrentParticipationBits(make([]byte, valCount)))
	syncCommittee, err := altair.NextSyncCommittee(context.Background(), st)
	require.NoError(t, err)
	require.NoError(t, st.SetCurrentSyncCommittee(syncCommittee))
	slot0bRoot := bytesutil.PadTo([]byte("slot0root"), 32)
	bRoots := make([][]byte, fieldparams.BlockRootsLength)
	bRoots[0] = slot0bRoot
	require.NoError(t, st.SetBlockRoots(bRoots))

	b := util.HydrateSignedBeaconBlockAltair(util.NewBeaconBlockAltair())
	b.Block.Slot = 2
	// we have to set the proposer index to the value that will be randomly chosen (fortunately it's deterministic)
	b.Block.ProposerIndex = 12
	b.Block.Body.Attestations = []*eth.Attestation{
		{
			AggregationBits: bitfield.Bitlist{0b00000111},
			Data:            util.HydrateAttestationData(&eth.AttestationData{}),
			Signature:       make([]byte, fieldparams.BLSSignatureLength),
		},
		{
			AggregationBits: bitfield.Bitlist{0b00000111},
			Data:            util.HydrateAttestationData(&eth.AttestationData{}),
			Signature:       make([]byte, fieldparams.BLSSignatureLength),
		},
	}
	attData1 := util.HydrateAttestationData(&eth.AttestationData{BeaconBlockRoot: bytesutil.PadTo([]byte("root1"), 32)})
	attData2 := util.HydrateAttestationData(&eth.AttestationData{BeaconBlockRoot: bytesutil.PadTo([]byte("root2"), 32)})
	domain, err := signing.Domain(st.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, st.GenesisValidatorsRoot())
	require.NoError(t, err)
	sigRoot1, err := signing.ComputeSigningRoot(attData1, domain)
	require.NoError(t, err)
	sigRoot2, err := signing.ComputeSigningRoot(attData2, domain)
	require.NoError(t, err)
	b.Block.Body.AttesterSlashings = []*eth.AttesterSlashing{
		{
			Attestation_1: &eth.IndexedAttestation{
				AttestingIndices: []uint64{0},
				Data:             attData1,
				Signature:        secretKeys[0].Sign(sigRoot1[:]).Marshal(),
			},
			Attestation_2: &eth.IndexedAttestation{
				AttestingIndices: []uint64{0},
				Data:             attData2,
				Signature:        secretKeys[0].Sign(sigRoot2[:]).Marshal(),
			},
		},
	}
	header1 := &eth.BeaconBlockHeader{
		Slot:          0,
		ProposerIndex: 1,
		ParentRoot:    bytesutil.PadTo([]byte("root1"), 32),
		StateRoot:     bytesutil.PadTo([]byte("root1"), 32),
		BodyRoot:      bytesutil.PadTo([]byte("root1"), 32),
	}
	header2 := &eth.BeaconBlockHeader{
		Slot:          0,
		ProposerIndex: 1,
		ParentRoot:    bytesutil.PadTo([]byte("root2"), 32),
		StateRoot:     bytesutil.PadTo([]byte("root2"), 32),
		BodyRoot:      bytesutil.PadTo([]byte("root2"), 32),
	}
	domain, err = signing.Domain(st.Fork(), 0, params.BeaconConfig().DomainBeaconProposer, st.GenesisValidatorsRoot())
	require.NoError(t, err)
	sigRoot1, err = signing.ComputeSigningRoot(header1, domain)
	require.NoError(t, err)
	sigRoot2, err = signing.ComputeSigningRoot(header2, domain)
	require.NoError(t, err)
	b.Block.Body.ProposerSlashings = []*eth.ProposerSlashing{
		{
			Header_1: &eth.SignedBeaconBlockHeader{
				Header:    header1,
				Signature: secretKeys[1].Sign(sigRoot1[:]).Marshal(),
			},
			Header_2: &eth.SignedBeaconBlockHeader{
				Header:    header2,
				Signature: secretKeys[1].Sign(sigRoot2[:]).Marshal(),
			},
		},
	}
	scBits := bitfield.NewBitvector512()
	scBits.SetBitAt(10, true)
	scBits.SetBitAt(100, true)
	domain, err = signing.Domain(st.Fork(), 0, params.BeaconConfig().DomainSyncCommittee, st.GenesisValidatorsRoot())
	require.NoError(t, err)
	sszBytes := primitives.SSZBytes(slot0bRoot)
	r, err := signing.ComputeSigningRoot(&sszBytes, domain)
	require.NoError(t, err)
	// Bits set in sync committee bits determine which validators will be treated as participating in sync committee.
	// These validators have to sign the message.
	sig1, err := blst.SignatureFromBytes(secretKeys[47].Sign(r[:]).Marshal())
	require.NoError(t, err)
	sig2, err := blst.SignatureFromBytes(secretKeys[19].Sign(r[:]).Marshal())
	require.NoError(t, err)
	aggSig := bls.AggregateSignatures([]bls.Signature{sig1, sig2}).Marshal()
	b.Block.Body.SyncAggregate = &eth.SyncAggregate{SyncCommitteeBits: scBits, SyncCommitteeSignature: aggSig}

	sbb, err := blocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	phase0block, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
	require.NoError(t, err)
	mockChainService := &mock.ChainService{Optimistic: true}
	s := &Server{
		Blocker: &testutil.MockBlocker{SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			0: phase0block,
			2: sbb,
		}},
		OptimisticModeFetcher: mockChainService,
		FinalizationFetcher:   mockChainService,
		ReplayerBuilder:       mockstategen.NewMockReplayerBuilder(mockstategen.WithMockState(st)),
	}

	t.Run("ok", func(t *testing.T) {
		url := "http://only.the.slot.number.at.the.end.is.important/2"
		request := httptest.NewRequest("GET", url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.BlockRewards(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &BlockRewardsResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		assert.Equal(t, "12", resp.Data.ProposerIndex)
		assert.Equal(t, "125089490", resp.Data.Total)
		assert.Equal(t, "89442", resp.Data.Attestations)
		assert.Equal(t, "48", resp.Data.SyncAggregate)
		assert.Equal(t, "62500000", resp.Data.AttesterSlashings)
		assert.Equal(t, "62500000", resp.Data.ProposerSlashings)
		assert.Equal(t, true, resp.ExecutionOptimistic)
		assert.Equal(t, false, resp.Finalized)
	})
	t.Run("phase 0", func(t *testing.T) {
		url := "http://only.the.slot.number.at.the.end.is.important/0"
		request := httptest.NewRequest("GET", url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.BlockRewards(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &network.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, "block rewards are not supported for Phase 0 blocks", e.Message)
	})
}
