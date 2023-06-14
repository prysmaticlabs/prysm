package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/go-bitfield"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	types "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

type roundtrip func(*http.Request) (*http.Response, error)

func (fn roundtrip) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func TestClient_Status(t *testing.T) {
	ctx := context.Background()
	statusPath := "/eth/v1/builder/status"
	hc := &http.Client{
		Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
			defer func() {
				if r.Body == nil {
					return
				}
				require.NoError(t, r.Body.Close())
			}()
			require.Equal(t, statusPath, r.URL.Path)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(nil)),
				Request:    r.Clone(ctx),
			}, nil
		}),
	}
	c := &Client{
		hc:      hc,
		baseURL: &url.URL{Host: "localhost:3500", Scheme: "http"},
	}
	require.NoError(t, c.Status(ctx))
	hc = &http.Client{
		Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
			defer func() {
				if r.Body == nil {
					return
				}
				require.NoError(t, r.Body.Close())
			}()
			require.Equal(t, statusPath, r.URL.Path)
			message := ErrorMessage{
				Code:    500,
				Message: "Internal server error",
			}
			resp, err := json.Marshal(message)
			require.NoError(t, err)
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewBuffer(resp)),
				Request:    r.Clone(ctx),
			}, nil
		}),
	}
	c = &Client{
		hc:      hc,
		baseURL: &url.URL{Host: "localhost:3500", Scheme: "http"},
	}
	require.ErrorIs(t, c.Status(ctx), ErrNotOK)
}

func TestClient_RegisterValidator(t *testing.T) {
	ctx := context.Background()
	expectedBody := `[{"message":{"fee_recipient":"0x0000000000000000000000000000000000000000","gas_limit":"23","timestamp":"42","pubkey":"0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"},"signature":"0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"}]`
	expectedPath := "/eth/v1/builder/validators"
	hc := &http.Client{
		Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(r.Body)
			defer func() {
				require.NoError(t, r.Body.Close())
			}()
			require.NoError(t, err)
			require.Equal(t, expectedBody, string(body))
			require.Equal(t, expectedPath, r.URL.Path)
			require.Equal(t, http.MethodPost, r.Method)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(nil)),
				Request:    r.Clone(ctx),
			}, nil
		}),
	}
	c := &Client{
		hc:      hc,
		baseURL: &url.URL{Host: "localhost:3500", Scheme: "http"},
	}
	reg := &eth.SignedValidatorRegistrationV1{
		Message: &eth.ValidatorRegistrationV1{
			FeeRecipient: ezDecode(t, params.BeaconConfig().EthBurnAddressHex),
			GasLimit:     23,
			Timestamp:    42,
			Pubkey:       ezDecode(t, "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"),
		},
		Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
	}
	require.NoError(t, c.RegisterValidator(ctx, []*eth.SignedValidatorRegistrationV1{reg}))
}

func TestClient_GetHeader(t *testing.T) {
	ctx := context.Background()
	expectedPath := "/eth/v1/builder/header/23/0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2/0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"
	var slot types.Slot = 23
	parentHash := ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	pubkey := ezDecode(t, "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a")
	t.Run("server error", func(t *testing.T) {
		hc := &http.Client{
			Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, expectedPath, r.URL.Path)
				message := ErrorMessage{
					Code:    500,
					Message: "Internal server error",
				}
				resp, err := json.Marshal(message)
				require.NoError(t, err)
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBuffer(resp)),
					Request:    r.Clone(ctx),
				}, nil
			}),
		}
		c := &Client{
			hc:      hc,
			baseURL: &url.URL{Host: "localhost:3500", Scheme: "http"},
		}

		_, err := c.GetHeader(ctx, slot, bytesutil.ToBytes32(parentHash), bytesutil.ToBytes48(pubkey))
		require.ErrorIs(t, err, ErrNotOK)
	})
	t.Run("header not available", func(t *testing.T) {
		hc := &http.Client{
			Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, expectedPath, r.URL.Path)
				return &http.Response{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(bytes.NewBuffer([]byte("No header is available."))),
					Request:    r.Clone(ctx),
				}, nil
			}),
		}
		c := &Client{
			hc:      hc,
			baseURL: &url.URL{Host: "localhost:3500", Scheme: "http"},
		}
		_, err := c.GetHeader(ctx, slot, bytesutil.ToBytes32(parentHash), bytesutil.ToBytes48(pubkey))
		require.ErrorIs(t, err, ErrNoContent)
	})
	t.Run("bellatrix", func(t *testing.T) {
		hc := &http.Client{
			Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, expectedPath, r.URL.Path)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(testExampleHeaderResponse)),
					Request:    r.Clone(ctx),
				}, nil
			}),
		}
		c := &Client{
			hc:      hc,
			baseURL: &url.URL{Host: "localhost:3500", Scheme: "http"},
		}
		h, err := c.GetHeader(ctx, slot, bytesutil.ToBytes32(parentHash), bytesutil.ToBytes48(pubkey))
		require.NoError(t, err)
		expectedSig := ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505")
		require.Equal(t, true, bytes.Equal(expectedSig, h.Signature()))
		expectedTxRoot := ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
		bid, err := h.Message()
		require.NoError(t, err)
		bidHeader, err := bid.Header()
		require.NoError(t, err)
		withdrawalsRoot, err := bidHeader.TransactionsRoot()
		require.NoError(t, err)
		require.Equal(t, true, bytes.Equal(expectedTxRoot, withdrawalsRoot))
		require.Equal(t, uint64(1), bidHeader.GasUsed())
		value, err := stringToUint256("652312848583266388373324160190187140051835877600158453279131187530910662656")
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%#x", value.SSZBytes()), fmt.Sprintf("%#x", bid.Value()))
		bidValue := bytesutil.ReverseByteOrder(bid.Value())
		require.DeepEqual(t, bidValue, value.Bytes())
		require.DeepEqual(t, big.NewInt(0).SetBytes(bidValue), value.Int)
	})
	t.Run("capella", func(t *testing.T) {
		hc := &http.Client{
			Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, expectedPath, r.URL.Path)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(testExampleHeaderResponseCapella)),
					Request:    r.Clone(ctx),
				}, nil
			}),
		}
		c := &Client{
			hc:      hc,
			baseURL: &url.URL{Host: "localhost:3500", Scheme: "http"},
		}
		h, err := c.GetHeader(ctx, slot, bytesutil.ToBytes32(parentHash), bytesutil.ToBytes48(pubkey))
		require.NoError(t, err)
		expectedWithdrawalsRoot := ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
		bid, err := h.Message()
		require.NoError(t, err)
		bidHeader, err := bid.Header()
		require.NoError(t, err)
		withdrawalsRoot, err := bidHeader.WithdrawalsRoot()
		require.NoError(t, err)
		require.Equal(t, true, bytes.Equal(expectedWithdrawalsRoot, withdrawalsRoot))
		value, err := stringToUint256("652312848583266388373324160190187140051835877600158453279131187530910662656")
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%#x", value.SSZBytes()), fmt.Sprintf("%#x", bid.Value()))
		bidValue := bytesutil.ReverseByteOrder(bid.Value())
		require.DeepEqual(t, bidValue, value.Bytes())
		require.DeepEqual(t, big.NewInt(0).SetBytes(bidValue), value.Int)
	})
	t.Run("deneb", func(t *testing.T) {
		hc := &http.Client{
			Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, expectedPath, r.URL.Path)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(testExampleHeaderResponseDeneb)),
					Request:    r.Clone(ctx),
				}, nil
			}),
		}
		c := &Client{
			hc:      hc,
			baseURL: &url.URL{Host: "localhost:3500", Scheme: "http"},
		}
		h, err := c.GetHeader(ctx, slot, bytesutil.ToBytes32(parentHash), bytesutil.ToBytes48(pubkey))
		require.NoError(t, err)
		expectedWithdrawalsRoot := ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
		bid, err := h.Message()
		require.NoError(t, err)
		bidHeader, err := bid.Header()
		require.NoError(t, err)
		withdrawalsRoot, err := bidHeader.WithdrawalsRoot()
		require.NoError(t, err)
		require.Equal(t, true, bytes.Equal(expectedWithdrawalsRoot, withdrawalsRoot))
		value, err := stringToUint256("652312848583266388373324160190187140051835877600158453279131187530910662656")
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%#x", value.SSZBytes()), fmt.Sprintf("%#x", bid.Value()))
		bidValue := bytesutil.ReverseByteOrder(bid.Value())
		require.DeepEqual(t, bidValue, value.Bytes())
		require.DeepEqual(t, big.NewInt(0).SetBytes(bidValue), value.Int)
		bundle, err := bid.BlindedBlobsBundle()
		require.NoError(t, err)
		require.Equal(t, len(bundle.BlobRoots) <= fieldparams.MaxBlobsPerBlock && len(bundle.BlobRoots) > 0, true)
		for i := range bundle.BlobRoots {
			require.Equal(t, len(bundle.BlobRoots[i]) == fieldparams.RootLength, true)
		}
		require.Equal(t, len(bundle.KzgCommitments) > 0, true)
		for i := range bundle.KzgCommitments {
			require.Equal(t, len(bundle.KzgCommitments[i]) == 48, true)
		}
		require.Equal(t, len(bundle.Proofs) > 0, true)
		for i := range bundle.Proofs {
			require.Equal(t, len(bundle.Proofs[i]) == 48, true)
		}
	})
	t.Run("unsupported version", func(t *testing.T) {
		hc := &http.Client{
			Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, expectedPath, r.URL.Path)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(testExampleHeaderResponseUnknownVersion)),
					Request:    r.Clone(ctx),
				}, nil
			}),
		}
		c := &Client{
			hc:      hc,
			baseURL: &url.URL{Host: "localhost:3500", Scheme: "http"},
		}
		_, err := c.GetHeader(ctx, slot, bytesutil.ToBytes32(parentHash), bytesutil.ToBytes48(pubkey))
		require.ErrorContains(t, "unsupported header version", err)
	})
}

func TestSubmitBlindedBlock(t *testing.T) {
	ctx := context.Background()

	t.Run("bellatrix", func(t *testing.T) {
		hc := &http.Client{
			Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, postBlindedBeaconBlockPath, r.URL.Path)
				require.Equal(t, "bellatrix", r.Header.Get("Eth-Consensus-Version"))
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(testExampleExecutionPayload)),
					Request:    r.Clone(ctx),
				}, nil
			}),
		}
		c := &Client{
			hc:      hc,
			baseURL: &url.URL{Host: "localhost:3500", Scheme: "http"},
		}
		sbbb, err := blocks.NewSignedBeaconBlock(testSignedBlindedBeaconBlockBellatrix(t))
		require.NoError(t, err)
		ep, _, err := c.SubmitBlindedBlock(ctx, sbbb, nil)
		require.NoError(t, err)
		require.Equal(t, true, bytes.Equal(ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"), ep.ParentHash()))
		bfpg, err := stringToUint256("452312848583266388373324160190187140051835877600158453279131187530910662656")
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%#x", bfpg.SSZBytes()), fmt.Sprintf("%#x", ep.BaseFeePerGas()))
		require.Equal(t, uint64(1), ep.GasLimit())
	})
	t.Run("capella", func(t *testing.T) {
		hc := &http.Client{
			Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, postBlindedBeaconBlockPath, r.URL.Path)
				require.Equal(t, "capella", r.Header.Get("Eth-Consensus-Version"))
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(testExampleExecutionPayloadCapella)),
					Request:    r.Clone(ctx),
				}, nil
			}),
		}
		c := &Client{
			hc:      hc,
			baseURL: &url.URL{Host: "localhost:3500", Scheme: "http"},
		}
		sbb, err := blocks.NewSignedBeaconBlock(testSignedBlindedBeaconBlockCapella(t))
		require.NoError(t, err)
		ep, _, err := c.SubmitBlindedBlock(ctx, sbb, nil)
		require.NoError(t, err)
		withdrawals, err := ep.Withdrawals()
		require.NoError(t, err)
		require.Equal(t, 1, len(withdrawals))
		assert.Equal(t, uint64(1), withdrawals[0].Index)
		assert.Equal(t, types.ValidatorIndex(1), withdrawals[0].ValidatorIndex)
		assert.DeepEqual(t, ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943"), withdrawals[0].Address)
		assert.Equal(t, uint64(1), withdrawals[0].Amount)
	})
	t.Run("deneb", func(t *testing.T) {
		hc := &http.Client{
			Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, postBlindedBeaconBlockPath, r.URL.Path)
				require.Equal(t, "deneb", r.Header.Get("Eth-Consensus-Version"))
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(testExampleExecutionPayloadDeneb)),
					Request:    r.Clone(ctx),
				}, nil
			}),
		}
		c := &Client{
			hc:      hc,
			baseURL: &url.URL{Host: "localhost:3500", Scheme: "http"},
		}
		test := testSignedBlindedBeaconBlockAndBlobsDeneb(t)
		sbb, err := blocks.NewSignedBeaconBlock(test.Block)
		require.NoError(t, err)

		ep, blobBundle, err := c.SubmitBlindedBlock(ctx, sbb, test.Blobs)
		require.NoError(t, err)
		withdrawals, err := ep.Withdrawals()
		require.NoError(t, err)
		require.Equal(t, 1, len(withdrawals))
		assert.Equal(t, uint64(1), withdrawals[0].Index)
		assert.Equal(t, types.ValidatorIndex(1), withdrawals[0].ValidatorIndex)
		assert.DeepEqual(t, ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943"), withdrawals[0].Address)
		assert.Equal(t, uint64(1), withdrawals[0].Amount)
		require.NotNil(t, blobBundle)
		require.Equal(t, hexutil.Encode(blobBundle.Blobs[0]), hexutil.Encode(make([]byte, fieldparams.BlobLength)))
		require.Equal(t, hexutil.Encode(blobBundle.KzgCommitments[0]), "0x8dab030c51e16e84be9caab84ee3d0b8bbec1db4a0e4de76439da8424d9b957370a10a78851f97e4b54d2ce1ab0d686f")
		require.Equal(t, hexutil.Encode(blobBundle.Proofs[0]), "0xb4021b0de10f743893d4f71e1bf830c019e832958efd6795baf2f83b8699a9eccc5dc99015d8d4d8ec370d0cc333c06a")
	})
	t.Run("mismatched versions, expected bellatrix got capella", func(t *testing.T) {
		hc := &http.Client{
			Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, postBlindedBeaconBlockPath, r.URL.Path)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(testExampleExecutionPayloadCapella)), // send a Capella payload
					Request:    r.Clone(ctx),
				}, nil
			}),
		}
		c := &Client{
			hc:      hc,
			baseURL: &url.URL{Host: "localhost:3500", Scheme: "http"},
		}
		sbbb, err := blocks.NewSignedBeaconBlock(testSignedBlindedBeaconBlockBellatrix(t))
		require.NoError(t, err)
		_, _, err = c.SubmitBlindedBlock(ctx, sbbb, nil)
		require.ErrorContains(t, "not a bellatrix payload", err)
	})
	t.Run("not blinded", func(t *testing.T) {
		sbb, err := blocks.NewSignedBeaconBlock(&eth.SignedBeaconBlockBellatrix{Block: &eth.BeaconBlockBellatrix{Body: &eth.BeaconBlockBodyBellatrix{}}})
		require.NoError(t, err)
		_, _, err = (&Client{}).SubmitBlindedBlock(ctx, sbb, nil)
		require.ErrorIs(t, err, errNotBlinded)
	})
}

func testSignedBlindedBeaconBlockBellatrix(t *testing.T) *eth.SignedBlindedBeaconBlockBellatrix {
	return &eth.SignedBlindedBeaconBlockBellatrix{
		Block: &eth.BlindedBeaconBlockBellatrix{
			Slot:          1,
			ProposerIndex: 1,
			ParentRoot:    ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
			StateRoot:     ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
			Body: &eth.BlindedBeaconBlockBodyBellatrix{
				RandaoReveal: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
				Eth1Data: &eth.Eth1Data{
					DepositRoot:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					DepositCount: 1,
					BlockHash:    ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
				},
				Graffiti: ezDecode(t, "0xdeadbeefc0ffee"),
				ProposerSlashings: []*eth.ProposerSlashing{
					{
						Header_1: &eth.SignedBeaconBlockHeader{
							Header: &eth.BeaconBlockHeader{
								Slot:          1,
								ProposerIndex: 1,
								ParentRoot:    ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								StateRoot:     ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								BodyRoot:      ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
							},
							Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
						},
						Header_2: &eth.SignedBeaconBlockHeader{
							Header: &eth.BeaconBlockHeader{
								Slot:          1,
								ProposerIndex: 1,
								ParentRoot:    ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								StateRoot:     ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								BodyRoot:      ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
							},
							Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
						},
					},
				},
				AttesterSlashings: []*eth.AttesterSlashing{
					{
						Attestation_1: &eth.IndexedAttestation{
							AttestingIndices: []uint64{1},
							Data: &eth.AttestationData{
								Slot:            1,
								CommitteeIndex:  1,
								BeaconBlockRoot: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								Source: &eth.Checkpoint{
									Epoch: 1,
									Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								},
								Target: &eth.Checkpoint{
									Epoch: 1,
									Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								},
							},
							Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
						},
						Attestation_2: &eth.IndexedAttestation{
							AttestingIndices: []uint64{1},
							Data: &eth.AttestationData{
								Slot:            1,
								CommitteeIndex:  1,
								BeaconBlockRoot: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								Source: &eth.Checkpoint{
									Epoch: 1,
									Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								},
								Target: &eth.Checkpoint{
									Epoch: 1,
									Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								},
							},
							Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
						},
					},
				},
				Attestations: []*eth.Attestation{
					{
						AggregationBits: bitfield.Bitlist{0x01},
						Data: &eth.AttestationData{
							Slot:            1,
							CommitteeIndex:  1,
							BeaconBlockRoot: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
							Source: &eth.Checkpoint{
								Epoch: 1,
								Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
							},
							Target: &eth.Checkpoint{
								Epoch: 1,
								Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
							},
						},
						Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
					},
				},
				Deposits: []*eth.Deposit{
					{
						Proof: [][]byte{ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")},
						Data: &eth.Deposit_Data{
							PublicKey:             ezDecode(t, "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"),
							WithdrawalCredentials: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
							Amount:                1,
							Signature:             ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
						},
					},
				},
				VoluntaryExits: []*eth.SignedVoluntaryExit{
					{
						Exit: &eth.VoluntaryExit{
							Epoch:          1,
							ValidatorIndex: 1,
						},
						Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
					},
				},
				SyncAggregate: &eth.SyncAggregate{
					SyncCommitteeSignature: make([]byte, 48),
					SyncCommitteeBits:      bitfield.Bitvector512{0x01},
				},
				ExecutionPayloadHeader: &v1.ExecutionPayloadHeader{
					ParentHash:       ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					FeeRecipient:     ezDecode(t, "0xabcf8e0d4e9587369b2301d0790347320302cc09"),
					StateRoot:        ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					ReceiptsRoot:     ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					LogsBloom:        ezDecode(t, "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
					PrevRandao:       ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					BlockNumber:      1,
					GasLimit:         1,
					GasUsed:          1,
					Timestamp:        1,
					ExtraData:        ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					BaseFeePerGas:    []byte(strconv.FormatUint(1, 10)),
					BlockHash:        ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					TransactionsRoot: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
				},
			},
		},
		Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
	}
}

func testSignedBlindedBeaconBlockCapella(t *testing.T) *eth.SignedBlindedBeaconBlockCapella {
	return &eth.SignedBlindedBeaconBlockCapella{
		Block: &eth.BlindedBeaconBlockCapella{
			Slot:          1,
			ProposerIndex: 1,
			ParentRoot:    ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
			StateRoot:     ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
			Body: &eth.BlindedBeaconBlockBodyCapella{
				RandaoReveal: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
				Eth1Data: &eth.Eth1Data{
					DepositRoot:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					DepositCount: 1,
					BlockHash:    ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
				},
				Graffiti: ezDecode(t, "0xdeadbeefc0ffee"),
				ProposerSlashings: []*eth.ProposerSlashing{
					{
						Header_1: &eth.SignedBeaconBlockHeader{
							Header: &eth.BeaconBlockHeader{
								Slot:          1,
								ProposerIndex: 1,
								ParentRoot:    ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								StateRoot:     ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								BodyRoot:      ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
							},
							Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
						},
						Header_2: &eth.SignedBeaconBlockHeader{
							Header: &eth.BeaconBlockHeader{
								Slot:          1,
								ProposerIndex: 1,
								ParentRoot:    ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								StateRoot:     ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								BodyRoot:      ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
							},
							Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
						},
					},
				},
				AttesterSlashings: []*eth.AttesterSlashing{
					{
						Attestation_1: &eth.IndexedAttestation{
							AttestingIndices: []uint64{1},
							Data: &eth.AttestationData{
								Slot:            1,
								CommitteeIndex:  1,
								BeaconBlockRoot: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								Source: &eth.Checkpoint{
									Epoch: 1,
									Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								},
								Target: &eth.Checkpoint{
									Epoch: 1,
									Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								},
							},
							Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
						},
						Attestation_2: &eth.IndexedAttestation{
							AttestingIndices: []uint64{1},
							Data: &eth.AttestationData{
								Slot:            1,
								CommitteeIndex:  1,
								BeaconBlockRoot: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								Source: &eth.Checkpoint{
									Epoch: 1,
									Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								},
								Target: &eth.Checkpoint{
									Epoch: 1,
									Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								},
							},
							Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
						},
					},
				},
				Attestations: []*eth.Attestation{
					{
						AggregationBits: bitfield.Bitlist{0x01},
						Data: &eth.AttestationData{
							Slot:            1,
							CommitteeIndex:  1,
							BeaconBlockRoot: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
							Source: &eth.Checkpoint{
								Epoch: 1,
								Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
							},
							Target: &eth.Checkpoint{
								Epoch: 1,
								Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
							},
						},
						Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
					},
				},
				Deposits: []*eth.Deposit{
					{
						Proof: [][]byte{ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")},
						Data: &eth.Deposit_Data{
							PublicKey:             ezDecode(t, "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"),
							WithdrawalCredentials: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
							Amount:                1,
							Signature:             ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
						},
					},
				},
				VoluntaryExits: []*eth.SignedVoluntaryExit{
					{
						Exit: &eth.VoluntaryExit{
							Epoch:          1,
							ValidatorIndex: 1,
						},
						Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
					},
				},
				SyncAggregate: &eth.SyncAggregate{
					SyncCommitteeSignature: make([]byte, 48),
					SyncCommitteeBits:      bitfield.Bitvector512{0x01},
				},
				ExecutionPayloadHeader: &v1.ExecutionPayloadHeaderCapella{
					ParentHash:       ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					FeeRecipient:     ezDecode(t, "0xabcf8e0d4e9587369b2301d0790347320302cc09"),
					StateRoot:        ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					ReceiptsRoot:     ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					LogsBloom:        ezDecode(t, "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
					PrevRandao:       ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					BlockNumber:      1,
					GasLimit:         1,
					GasUsed:          1,
					Timestamp:        1,
					ExtraData:        ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					BaseFeePerGas:    []byte(strconv.FormatUint(1, 10)),
					BlockHash:        ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					TransactionsRoot: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					WithdrawalsRoot:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
				},
			},
		},
		Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
	}
}

func testSignedBlindedBeaconBlockAndBlobsDeneb(t *testing.T) *eth.SignedBlindedBeaconBlockAndBlobsDeneb {
	return &eth.SignedBlindedBeaconBlockAndBlobsDeneb{
		Block: &eth.SignedBlindedBeaconBlockDeneb{
			Block: &eth.BlindedBeaconBlockDeneb{
				Slot:          1,
				ProposerIndex: 1,
				ParentRoot:    ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
				StateRoot:     ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
				Body: &eth.BlindedBeaconBlockBodyDeneb{
					RandaoReveal: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
					Eth1Data: &eth.Eth1Data{
						DepositRoot:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
						DepositCount: 1,
						BlockHash:    ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					},
					Graffiti: ezDecode(t, "0xdeadbeefc0ffee"),
					ProposerSlashings: []*eth.ProposerSlashing{
						{
							Header_1: &eth.SignedBeaconBlockHeader{
								Header: &eth.BeaconBlockHeader{
									Slot:          1,
									ProposerIndex: 1,
									ParentRoot:    ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
									StateRoot:     ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
									BodyRoot:      ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								},
								Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
							},
							Header_2: &eth.SignedBeaconBlockHeader{
								Header: &eth.BeaconBlockHeader{
									Slot:          1,
									ProposerIndex: 1,
									ParentRoot:    ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
									StateRoot:     ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
									BodyRoot:      ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								},
								Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
							},
						},
					},
					AttesterSlashings: []*eth.AttesterSlashing{
						{
							Attestation_1: &eth.IndexedAttestation{
								AttestingIndices: []uint64{1},
								Data: &eth.AttestationData{
									Slot:            1,
									CommitteeIndex:  1,
									BeaconBlockRoot: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
									Source: &eth.Checkpoint{
										Epoch: 1,
										Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
									},
									Target: &eth.Checkpoint{
										Epoch: 1,
										Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
									},
								},
								Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
							},
							Attestation_2: &eth.IndexedAttestation{
								AttestingIndices: []uint64{1},
								Data: &eth.AttestationData{
									Slot:            1,
									CommitteeIndex:  1,
									BeaconBlockRoot: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
									Source: &eth.Checkpoint{
										Epoch: 1,
										Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
									},
									Target: &eth.Checkpoint{
										Epoch: 1,
										Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
									},
								},
								Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
							},
						},
					},
					Attestations: []*eth.Attestation{
						{
							AggregationBits: bitfield.Bitlist{0x01},
							Data: &eth.AttestationData{
								Slot:            1,
								CommitteeIndex:  1,
								BeaconBlockRoot: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								Source: &eth.Checkpoint{
									Epoch: 1,
									Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								},
								Target: &eth.Checkpoint{
									Epoch: 1,
									Root:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								},
							},
							Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
						},
					},
					Deposits: []*eth.Deposit{
						{
							Proof: [][]byte{ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")},
							Data: &eth.Deposit_Data{
								PublicKey:             ezDecode(t, "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"),
								WithdrawalCredentials: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
								Amount:                1,
								Signature:             ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
							},
						},
					},
					VoluntaryExits: []*eth.SignedVoluntaryExit{
						{
							Exit: &eth.VoluntaryExit{
								Epoch:          1,
								ValidatorIndex: 1,
							},
							Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
						},
					},
					SyncAggregate: &eth.SyncAggregate{
						SyncCommitteeSignature: make([]byte, 48),
						SyncCommitteeBits:      bitfield.Bitvector512{0x01},
					},
					ExecutionPayloadHeader: &v1.ExecutionPayloadHeaderDeneb{
						ParentHash:       ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
						FeeRecipient:     ezDecode(t, "0xabcf8e0d4e9587369b2301d0790347320302cc09"),
						StateRoot:        ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
						ReceiptsRoot:     ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
						LogsBloom:        ezDecode(t, "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
						PrevRandao:       ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
						BlockNumber:      1,
						GasLimit:         1,
						GasUsed:          1,
						Timestamp:        1,
						ExtraData:        ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
						BaseFeePerGas:    []byte(strconv.FormatUint(1, 10)),
						BlockHash:        ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
						TransactionsRoot: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
						WithdrawalsRoot:  ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
						DataGasUsed:      1,
						ExcessDataGas:    1,
					},
				},
			},
			Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
		},
		Blobs: []*eth.SignedBlindedBlobSidecar{
			{
				Message: &eth.BlindedBlobSidecar{
					BlockRoot:       ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					Index:           0,
					Slot:            1,
					BlockParentRoot: ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"),
					ProposerIndex:   1,
					BlobRoot:        ezDecode(t, "0x24564723180fcb3d994104538d351c8dcbde12d541676bb736cf678018ca4739"),
					KzgCommitment:   ezDecode(t, "0x8dab030c51e16e84be9caab84ee3d0b8bbec1db4a0e4de76439da8424d9b957370a10a78851f97e4b54d2ce1ab0d686f"),
					KzgProof:        ezDecode(t, "0xb4021b0de10f743893d4f71e1bf830c019e832958efd6795baf2f83b8699a9eccc5dc99015d8d4d8ec370d0cc333c06a"),
				},
				Signature: ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"),
			},
		},
	}
}

func TestRequestLogger(t *testing.T) {
	wo := WithObserver(&requestLogger{})
	c, err := NewClient("localhost:3500", wo)
	require.NoError(t, err)

	ctx := context.Background()
	hc := &http.Client{
		Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
			require.Equal(t, getStatus, r.URL.Path)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(testExampleExecutionPayload)),
				Request:    r.Clone(ctx),
			}, nil
		}),
	}
	c.hc = hc
	err = c.Status(ctx)
	require.NoError(t, err)
}
