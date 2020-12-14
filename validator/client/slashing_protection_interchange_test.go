package client

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	interchangeformat "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/standard-protection-format"
)

type test struct {
	Name                  string `json:"name"`
	GenesisValidatorsRoot string `json:"genesis_validators_root"`
	Steps                 []struct {
		ShouldSucceed      bool `json:"should_succeed"`
		AllowPartialImport bool `json:"allow_partial_import"`
		Interchange        struct {
			Metadata struct {
				InterchangeFormatVersion string `json:"interchange_format_version"`
				GenesisValidatorsRoot    string `json:"genesis_validators_root"`
			} `json:"metadata"`
			Data []struct {
				Pubkey       string `json:"pubkey"`
				SignedBlocks []struct {
					Slot        string `json:"slot"`
					SigningRoot string `json:"signing_root"`
				} `json:"signed_blocks"`
				SignedAttestations []struct {
					SourceEpoch string `json:"source_epoch"`
					TargetEpoch string `json:"target_epoch"`
					SigningRoot string `json:"signing_root"`
				} `json:"signed_attestations"`
			} `json:"data"`
		} `json:"interchange"`
		Blocks []struct {
			Pubkey        string `json:"pubkey"`
			Slot          string `json:"slot"`
			SigningRoot   string `json:"signing_root"`
			ShouldSucceed bool   `json:"should_succeed"`
		} `json:"blocks"`
		Attestations []struct {
			Pubkey        string `json:"pubkey"`
			SourceEpoch   string `json:"source_epoch"`
			TargetEpoch   string `json:"target_epoch"`
			SigningRoot   string `json:"signing_root"`
			ShouldSucceed bool   `json:"should_succeed"`
		} `json:"attestations"`
	} `json:"steps"`
}

func TestSlashingInterchangeStandard(t *testing.T) {
	config := &featureconfig.Flags{
		SlasherProtection: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()

	repo := "https://github.com/eth2-clients/slashing-protection-interchange-tests/tarball/b8413ca42dc92308019d0d4db52c87e9e125c4e9"
	resp, err := http.Get(repo)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("could not pull data from repo, status code is %d", resp.StatusCode)
	}

	gzf, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	tarReader := tar.NewReader(gzf)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		// The test configs are defined in json format.
		if strings.Contains(header.Name, "json") {
			var b []byte
			b, err := ioutil.ReadAll(tarReader)
			if err != nil {
				t.Fatal(err)
			}

			test := &test{}
			if err := json.Unmarshal(b, test); err != nil {
				t.Fatal(err)
			}

			// The test name in the config should not be empty. This is to eliminate running
			// invalid test file.
			if test.Name != "" {
				t.Run(test.Name, func(t *testing.T) {
					for _, step := range test.Steps {
						// Set up validator client, one new validator client per test.
						// This ensures we initialize a new (empty) slashing protection database.
						validator, _, _, _ := setup(t)

						if test.GenesisValidatorsRoot != "" {
							r, err := interchangeformat.RootFromHex(test.GenesisValidatorsRoot)
							require.NoError(t, validator.db.SaveGenesisValidatorsRoot(context.Background(), r[:]))
							require.NoError(t, err)
						}

						// The test config contains the interchange config in json.
						// This loads the interchange data via ImportStandardProtectionJSON.
						interchangeBytes, err := json.Marshal(step.Interchange)
						if err != nil {
							t.Fatal(err)
						}
						b := bytes.NewBuffer(interchangeBytes)
						if err := interchangeformat.ImportStandardProtectionJSON(context.Background(), validator.db, b); err != nil {
							if step.ShouldSucceed {
								t.Fatal(err)
							}
						} else if !step.ShouldSucceed {
							require.NotNil(t, err, "import standard protection json should have failed")
						}

						// This loops through a list of block signings to attempt after importing the interchange data above.
						for _, sb := range step.Blocks {
							bSlot, err := interchangeformat.Uint64FromString(sb.Slot)
							require.NoError(t, err)
							pk, err := interchangeformat.PubKeyFromHex(sb.Pubkey)
							require.NoError(t, err)
							b := testutil.NewBeaconBlock()
							b.Block.Slot = bSlot

							var signingRoot [32]byte
							if sb.SigningRoot != "" {
								signingRootBytes, err := hex.DecodeString(strings.TrimPrefix(sb.SigningRoot, "0x"))
								require.NoError(t, err)
								copy(signingRoot[:], signingRootBytes)
							}

							err = validator.preBlockSignValidations(context.Background(), pk, b.Block, signingRoot)
							if sb.ShouldSucceed {
								require.NoError(t, err)
							} else {
								require.NotEqual(t, nil, err, "pre validation should have failed for block")
							}

							// Only proceed post update if pre validation did not error.
							if err == nil {
								err = validator.postBlockSignUpdate(context.Background(), pk, b, signingRoot)
								if sb.ShouldSucceed {
									require.NoError(t, err)
								} else {
									require.NotEqual(t, nil, err, "post validation should have failed for block")
								}
							}
						}

						// This loops through a list of attestation signings to attempt after importing the interchange data above.
						for _, sa := range step.Attestations {
							target, err := interchangeformat.Uint64FromString(sa.TargetEpoch)
							require.NoError(t, err)
							source, err := interchangeformat.Uint64FromString(sa.SourceEpoch)
							require.NoError(t, err)
							pk, err := interchangeformat.PubKeyFromHex(sa.Pubkey)
							require.NoError(t, err)
							ia := &ethpb.IndexedAttestation{
								Data: &ethpb.AttestationData{
									BeaconBlockRoot: make([]byte, 32),
									Target:          &ethpb.Checkpoint{Epoch: target, Root: make([]byte, 32)},
									Source:          &ethpb.Checkpoint{Epoch: source, Root: make([]byte, 32)},
								},
								Signature: make([]byte, 96),
							}

							var signingRoot [32]byte
							if sa.SigningRoot != "" {
								signingRootBytes, err := hex.DecodeString(strings.TrimPrefix(sa.SigningRoot, "0x"))
								require.NoError(t, err)
								copy(signingRoot[:], signingRootBytes)
							}

							fmt.Println(sa)
							err = validator.slashableAttestationCheck(context.Background(), ia, pk, signingRoot)
							if sa.ShouldSucceed {
								fmt.Printf("Should succeed %v\n", err)
								require.NoError(t, err)
							} else {
								fmt.Printf("Should not succeed %v\n", err)
								require.NotNil(t, err, "pre validation should have failed for attestation")
							}
						}
						require.NoError(t, err, validator.db.Close())
					}
				})
			}
		}
	}
}
