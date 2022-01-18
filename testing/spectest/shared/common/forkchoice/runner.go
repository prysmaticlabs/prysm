package forkchoice

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/snappy"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	v2 "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	v3 "github.com/prysmaticlabs/prysm/beacon-chain/state/v3"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/testing/util"
)

// Run executes "forkchoice" test.
func Run(t *testing.T, config string, fork int) {
	require.NoError(t, utils.SetConfig(t, config))
	testFolders, _ := utils.TestFolders(t, config, version.String(fork), "fork_choice")

	for _, folder := range testFolders {
		folderPath := path.Join("fork_choice", folder.Name(), "pyspec_tests")
		testFolders, testsFolderPath := utils.TestFolders(t, config, version.String(fork), folderPath)

		for _, folder := range testFolders {
			t.Run(folder.Name(), func(t *testing.T) {
				ctx := context.Background()
				preStepsFile, err := util.BazelFileBytes(testsFolderPath, folder.Name(), "steps.yaml")
				require.NoError(t, err)
				var steps []Step
				require.NoError(t, utils.UnmarshalYaml(preStepsFile, &steps))

				preBeaconStateFile, err := util.BazelFileBytes(testsFolderPath, folder.Name(), "anchor_state.ssz_snappy")
				require.NoError(t, err)
				preBeaconStateSSZ, err := snappy.Decode(nil /* dst */, preBeaconStateFile)
				require.NoError(t, err)

				blockFile, err := util.BazelFileBytes(testsFolderPath, folder.Name(), "anchor_block.ssz_snappy")
				require.NoError(t, err)
				blockSSZ, err := snappy.Decode(nil /* dst */, blockFile)
				require.NoError(t, err)

				var beaconState state.BeaconState
				var beaconBlock block.SignedBeaconBlock
				switch fork {
				case version.Phase0:
					beaconState = unmarshalPhase0State(t, preBeaconStateSSZ)
					beaconBlock = unmarshalPhase0Block(t, blockSSZ)
				case version.Altair:
					beaconState = unmarshalAltairState(t, preBeaconStateSSZ)
					t.Log("Altair fork")
					beaconBlock = unmarshalAltairBlock(t, blockSSZ)
				case version.Bellatrix:
					beaconState = unmarshalBellatrixState(t, preBeaconStateSSZ)
					beaconBlock = unmarshalBellatrixBlock(t, blockSSZ)
				default:
					t.Fatalf("unknown fork version: %v", fork)
				}

				service := startChainService(t, beaconState, beaconBlock)
				require.NoError(t, service.InitializeStore(ctx, beaconState, beaconBlock))
				for _, step := range steps {
					if step.Tick != nil {
						require.NoError(t, service.OnTick(ctx, uint64(*step.Tick)))
					}
					if step.Block != nil {
						blockFile, err := util.BazelFileBytes(testsFolderPath, folder.Name(), fmt.Sprint(*step.Block, ".ssz_snappy"))
						require.NoError(t, err)
						blockSSZ, err := snappy.Decode(nil /* dst */, blockFile)
						require.NoError(t, err)
						var beaconBlock block.SignedBeaconBlock
						switch fork {
						case version.Phase0:
							beaconBlock = unmarshalSignedPhase0Block(t, blockSSZ)
						case version.Altair:
							beaconBlock = unmarshalSignedAltairBlock(t, blockSSZ)
						case version.Bellatrix:
							beaconBlock = unmarshalSignedBellatrixBlock(t, blockSSZ)
						default:
							t.Fatalf("unknown fork version: %v", fork)
						}
						r, err := beaconBlock.Block().HashTreeRoot()
						require.NoError(t, err)
						if step.Valid != nil && *step.Valid == false {
							require.Equal(t, true, service.ReceiveBlock(ctx, beaconBlock, r) != nil)
						} else {
							require.NoError(t, service.ReceiveBlock(ctx, beaconBlock, r))
						}
					}
					if step.Attestation != nil {
						attFile, err := util.BazelFileBytes(testsFolderPath, folder.Name(), fmt.Sprint(*step.Attestation, ".ssz_snappy"))
						require.NoError(t, err)
						attSSZ, err := snappy.Decode(nil /* dst */, attFile)
						require.NoError(t, err)
						att := &ethpb.Attestation{}
						require.NoError(t, att.UnmarshalSSZ(attSSZ), "Failed to unmarshal")
						require.NoError(t, service.OnAttestation(ctx, att))
						require.NoError(t, service.UpdateHead(ctx))
					}
					if step.Check != nil {
						c := step.Check
						if c.Time != nil {
							require.Equal(t, uint64(*c.Time), service.TimeInStore())
						}
						if c.Head != nil {
							r, err := service.HeadRoot(ctx)
							require.NoError(t, err)
							require.DeepEqual(t, common.FromHex(c.Head.Root), r)
							require.Equal(t, types.Slot(c.Head.Slot), service.HeadSlot())
						}
						if c.JustifiedCheckPoint != nil {
							cp := &ethpb.Checkpoint{
								Epoch: types.Epoch(c.JustifiedCheckPoint.Epoch),
								Root:  common.FromHex(c.JustifiedCheckPoint.Root),
							}
							require.DeepEqual(t, cp, service.CurrentJustifiedCheckpt())
						}
						if c.BestJustifiedCheckPoint != nil {
							cp := &ethpb.Checkpoint{
								Epoch: types.Epoch(c.BestJustifiedCheckPoint.Epoch),
								Root:  common.FromHex(c.BestJustifiedCheckPoint.Root),
							}
							require.DeepEqual(t, cp, service.BestJustifiedCheckpt())
						}
						if c.FinalizedCheckPoint != nil {
							cp := &ethpb.Checkpoint{
								Epoch: types.Epoch(c.FinalizedCheckPoint.Epoch),
								Root:  common.FromHex(c.FinalizedCheckPoint.Root),
							}
							require.DeepSSZEqual(t, cp, service.FinalizedCheckpt())
						}
					}
				}
			})
		}
	}
}

func unmarshalPhase0State(t *testing.T, raw []byte) state.BeaconState {
	base := &ethpb.BeaconState{}
	require.NoError(t, base.UnmarshalSSZ(raw))
	st, err := v1.InitializeFromProto(base)
	require.NoError(t, err)
	return st
}

func unmarshalPhase0Block(t *testing.T, raw []byte) block.SignedBeaconBlock {
	base := &ethpb.BeaconBlock{}
	require.NoError(t, base.UnmarshalSSZ(raw))
	blk, err := wrapper.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlock{Block: base, Signature: make([]byte, fieldparams.BLSSignatureLength)})
	require.NoError(t, err)
	return blk
}

func unmarshalSignedPhase0Block(t *testing.T, raw []byte) block.SignedBeaconBlock {
	base := &ethpb.SignedBeaconBlock{}
	require.NoError(t, base.UnmarshalSSZ(raw))
	blk, err := wrapper.WrappedSignedBeaconBlock(base)
	require.NoError(t, err)
	return blk
}

func unmarshalAltairState(t *testing.T, raw []byte) state.BeaconState {
	base := &ethpb.BeaconStateAltair{}
	require.NoError(t, base.UnmarshalSSZ(raw))
	st, err := v2.InitializeFromProto(base)
	require.NoError(t, err)
	return st
}

func unmarshalAltairBlock(t *testing.T, raw []byte) block.SignedBeaconBlock {
	base := &ethpb.BeaconBlockAltair{}
	require.NoError(t, base.UnmarshalSSZ(raw))
	blk, err := wrapper.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{Block: base, Signature: make([]byte, fieldparams.BLSSignatureLength)})
	require.NoError(t, err)
	return blk
}

func unmarshalSignedAltairBlock(t *testing.T, raw []byte) block.SignedBeaconBlock {
	base := &ethpb.SignedBeaconBlockAltair{}
	require.NoError(t, base.UnmarshalSSZ(raw))
	blk, err := wrapper.WrappedSignedBeaconBlock(base)
	require.NoError(t, err)
	return blk
}

func unmarshalBellatrixState(t *testing.T, raw []byte) state.BeaconState {
	base := &ethpb.BeaconStateBellatrix{}
	require.NoError(t, base.UnmarshalSSZ(raw))
	st, err := v3.InitializeFromProto(base)
	require.NoError(t, err)
	return st
}

func unmarshalBellatrixBlock(t *testing.T, raw []byte) block.SignedBeaconBlock {
	base := &ethpb.BeaconBlockMerge{}
	require.NoError(t, base.UnmarshalSSZ(raw))
	blk, err := wrapper.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlockMerge{Block: base, Signature: make([]byte, fieldparams.BLSSignatureLength)})
	require.NoError(t, err)
	return blk
}

func unmarshalSignedBellatrixBlock(t *testing.T, raw []byte) block.SignedBeaconBlock {
	base := &ethpb.SignedBeaconBlockMerge{}
	require.NoError(t, base.UnmarshalSSZ(raw))
	blk, err := wrapper.WrappedSignedBeaconBlock(base)
	require.NoError(t, err)
	return blk
}
