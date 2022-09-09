package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
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
	expectedBody := `[{"message":{"fee_recipient":"0x0000000000000000000000000000000000000000","gas_limit":"23","timestamp":"42","pubkey":"0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"}}]`
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
	}
	require.NoError(t, c.RegisterValidator(ctx, []*eth.SignedValidatorRegistrationV1{reg}))
}

func TestClient_GetHeader(t *testing.T) {
	ctx := context.Background()
	expectedPath := "/eth/v1/builder/header/23/0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2/0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"
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
	var slot types.Slot = 23
	parentHash := ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	pubkey := ezDecode(t, "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a")
	_, err := c.GetHeader(ctx, slot, bytesutil.ToBytes32(parentHash), bytesutil.ToBytes48(pubkey))
	require.ErrorIs(t, err, ErrNotOK)

	hc = &http.Client{
		Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
			require.Equal(t, expectedPath, r.URL.Path)
			return &http.Response{
				StatusCode: http.StatusNoContent,
				Body:       io.NopCloser(bytes.NewBuffer([]byte("No header is available."))),
				Request:    r.Clone(ctx),
			}, nil
		}),
	}
	c = &Client{
		hc:      hc,
		baseURL: &url.URL{Host: "localhost:3500", Scheme: "http"},
	}
	_, err = c.GetHeader(ctx, slot, bytesutil.ToBytes32(parentHash), bytesutil.ToBytes48(pubkey))
	require.ErrorIs(t, err, ErrNoContent)

	hc = &http.Client{
		Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
			require.Equal(t, expectedPath, r.URL.Path)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(testExampleHeaderResponse)),
				Request:    r.Clone(ctx),
			}, nil
		}),
	}
	c = &Client{
		hc:      hc,
		baseURL: &url.URL{Host: "localhost:3500", Scheme: "http"},
	}
	h, err := c.GetHeader(ctx, slot, bytesutil.ToBytes32(parentHash), bytesutil.ToBytes48(pubkey))
	require.NoError(t, err)
	expectedSig := ezDecode(t, "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505")
	require.Equal(t, true, bytes.Equal(expectedSig, h.Signature))
	expectedTxRoot := ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	require.Equal(t, true, bytes.Equal(expectedTxRoot, h.Message.Header.TransactionsRoot))
	require.Equal(t, uint64(1), h.Message.Header.GasUsed)
	value, err := stringToUint256("652312848583266388373324160190187140051835877600158453279131187530910662656")
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%#x", value.SSZBytes()), fmt.Sprintf("%#x", h.Message.Value))
}

func TestSubmitBlindedBlock(t *testing.T) {
	ctx := context.Background()
	hc := &http.Client{
		Transport: roundtrip(func(r *http.Request) (*http.Response, error) {
			require.Equal(t, postBlindedBeaconBlockPath, r.URL.Path)
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
	sbbb := testSignedBlindedBeaconBlockBellatrix(t)
	ep, err := c.SubmitBlindedBlock(ctx, sbbb)
	require.NoError(t, err)
	require.Equal(t, true, bytes.Equal(ezDecode(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"), ep.ParentHash))
	bfpg, err := stringToUint256("452312848583266388373324160190187140051835877600158453279131187530910662656")
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%#x", bfpg.SSZBytes()), fmt.Sprintf("%#x", ep.BaseFeePerGas))
	require.Equal(t, uint64(1), ep.GasLimit)
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
