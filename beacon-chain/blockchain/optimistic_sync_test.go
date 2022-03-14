package blockchain

import (
	"context"
	"testing"
	"time"

	testing2 "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	engine "github.com/prysmaticlabs/prysm/beacon-chain/powchain/engine-api-client/v1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/prysmaticlabs/prysm/time/slots"
)

func Test_NotifyForkchoiceUpdate(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	altairBlk, err := wrapper.WrappedSignedBeaconBlock(util.NewBeaconBlockAltair())
	require.NoError(t, err)
	altairBlkRoot, err := altairBlk.Block().HashTreeRoot()
	require.NoError(t, err)
	bellatrixBlk, err := wrapper.WrappedSignedBeaconBlock(util.NewBeaconBlockBellatrix())
	require.NoError(t, err)
	bellatrixBlkRoot, err := bellatrixBlk.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, altairBlk))
	require.NoError(t, beaconDB.SaveBlock(ctx, bellatrixBlk))

	fcs := protoarray.New(0, 0, [32]byte{'a'})
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	tests := []struct {
		name             string
		blk              block.BeaconBlock
		finalizedRoot    [32]byte
		newForkchoiceErr error
		errString        string
	}{
		{
			name:      "nil block",
			errString: "nil head block",
		},
		{
			name: "phase0 block",
			blk: func() block.BeaconBlock {
				b, err := wrapper.WrappedBeaconBlock(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}})
				require.NoError(t, err)
				return b
			}(),
		},
		{
			name: "altair block",
			blk: func() block.BeaconBlock {
				b, err := wrapper.WrappedBeaconBlock(&ethpb.BeaconBlockAltair{Body: &ethpb.BeaconBlockBodyAltair{}})
				require.NoError(t, err)
				return b
			}(),
		},
		{
			name: "not execution block",
			blk: func() block.BeaconBlock {
				b, err := wrapper.WrappedBeaconBlock(&ethpb.BeaconBlockBellatrix{
					Body: &ethpb.BeaconBlockBodyBellatrix{
						ExecutionPayload: &v1.ExecutionPayload{
							ParentHash:    make([]byte, fieldparams.RootLength),
							FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
							StateRoot:     make([]byte, fieldparams.RootLength),
							ReceiptsRoot:  make([]byte, fieldparams.RootLength),
							LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
							PrevRandao:    make([]byte, fieldparams.RootLength),
							BaseFeePerGas: make([]byte, fieldparams.RootLength),
							BlockHash:     make([]byte, fieldparams.RootLength),
						},
					},
				})
				require.NoError(t, err)
				return b
			}(),
		},
		{
			name: "happy case: finalized root is altair block",
			blk: func() block.BeaconBlock {
				b, err := wrapper.WrappedBeaconBlock(&ethpb.BeaconBlockBellatrix{
					Body: &ethpb.BeaconBlockBodyBellatrix{
						ExecutionPayload: &v1.ExecutionPayload{},
					},
				})
				require.NoError(t, err)
				return b
			}(),
			finalizedRoot: altairBlkRoot,
		},
		{
			name: "happy case: finalized root is bellatrix block",
			blk: func() block.BeaconBlock {
				b, err := wrapper.WrappedBeaconBlock(&ethpb.BeaconBlockBellatrix{
					Body: &ethpb.BeaconBlockBodyBellatrix{
						ExecutionPayload: &v1.ExecutionPayload{},
					},
				})
				require.NoError(t, err)
				return b
			}(),
			finalizedRoot: bellatrixBlkRoot,
		},
		{
			name: "forkchoice updated with optimistic block",
			blk: func() block.BeaconBlock {
				b, err := wrapper.WrappedBeaconBlock(&ethpb.BeaconBlockBellatrix{
					Body: &ethpb.BeaconBlockBodyBellatrix{
						ExecutionPayload: &v1.ExecutionPayload{},
					},
				})
				require.NoError(t, err)
				return b
			}(),
			newForkchoiceErr: engine.ErrAcceptedSyncingPayloadStatus,
			finalizedRoot:    bellatrixBlkRoot,
		},
		{
			name: "forkchoice updated with invalid block",
			blk: func() block.BeaconBlock {
				b, err := wrapper.WrappedBeaconBlock(&ethpb.BeaconBlockBellatrix{
					Body: &ethpb.BeaconBlockBodyBellatrix{
						ExecutionPayload: &v1.ExecutionPayload{},
					},
				})
				require.NoError(t, err)
				return b
			}(),
			newForkchoiceErr: engine.ErrInvalidPayloadStatus,
			finalizedRoot:    bellatrixBlkRoot,
			errString:        "could not notify forkchoice update from execution engine: payload status is INVALID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &testing2.MockEngineService{ForkchoiceError: tt.newForkchoiceErr}
			service.cfg.ExecutionEngineCaller = engine
			_, err := service.notifyForkchoiceUpdate(ctx, tt.blk, tt.finalizedRoot)
			if tt.errString != "" {
				require.ErrorContains(t, tt.errString, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_NotifyNewPayload(t *testing.T) {
	cfg := params.BeaconConfig()
	cfg.TerminalTotalDifficulty = "2"
	params.OverrideBeaconConfig(cfg)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := protoarray.New(0, 0, [32]byte{'a'})
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
	phase0State, _ := util.DeterministicGenesisState(t, 1)
	altairState, _ := util.DeterministicGenesisStateAltair(t, 1)
	bellatrixState, _ := util.DeterministicGenesisStateBellatrix(t, 2)
	blk := &ethpb.SignedBeaconBlockBellatrix{
		Block: &ethpb.BeaconBlockBellatrix{
			Body: &ethpb.BeaconBlockBodyBellatrix{
				ExecutionPayload: &v1.ExecutionPayload{},
			},
		},
	}
	bellatrixBlk, err := wrapper.WrappedSignedBeaconBlock(blk)
	require.NoError(t, err)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	tests := []struct {
		name          string
		preState      state.BeaconState
		postState     state.BeaconState
		blk           block.SignedBeaconBlock
		newPayloadErr error
		errString     string
	}{
		{
			name:      "phase 0 post state",
			postState: phase0State,
			preState:  phase0State,
		},
		{
			name:      "altair post state",
			postState: altairState,
			preState:  altairState,
		},
		{
			name:      "nil post state",
			preState:  phase0State,
			errString: "pre and post states must not be nil",
		},
		{
			name:      "nil beacon block",
			postState: bellatrixState,
			preState:  bellatrixState,
			errString: "signed beacon block can't be nil",
		},
		{
			name:          "new payload with optimistic block",
			postState:     bellatrixState,
			preState:      bellatrixState,
			blk:           bellatrixBlk,
			newPayloadErr: engine.ErrAcceptedSyncingPayloadStatus,
		},
		{
			name:          "new payload with invalid block",
			postState:     bellatrixState,
			preState:      bellatrixState,
			blk:           bellatrixBlk,
			newPayloadErr: engine.ErrInvalidPayloadStatus,
			errString:     "could not validate execution payload from execution engine: payload status is INVALID",
		},
		{
			name:      "altair pre state",
			postState: bellatrixState,
			preState:  altairState,
			blk:       bellatrixBlk,
		},
		{
			name:      "could not get merge block",
			postState: bellatrixState,
			preState:  bellatrixState,
			blk:       bellatrixBlk,
			errString: "could not get merge block parent hash and total difficulty",
		},
		{
			name:      "not at merge transition",
			postState: bellatrixState,
			preState:  bellatrixState,
			blk: func() block.SignedBeaconBlock {
				blk := &ethpb.SignedBeaconBlockBellatrix{
					Block: &ethpb.BeaconBlockBellatrix{
						Body: &ethpb.BeaconBlockBodyBellatrix{
							ExecutionPayload: &v1.ExecutionPayload{
								ParentHash:    make([]byte, fieldparams.RootLength),
								FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
								StateRoot:     make([]byte, fieldparams.RootLength),
								ReceiptsRoot:  make([]byte, fieldparams.RootLength),
								LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
								PrevRandao:    make([]byte, fieldparams.RootLength),
								BaseFeePerGas: make([]byte, fieldparams.RootLength),
								BlockHash:     make([]byte, fieldparams.RootLength),
							},
						},
					},
				}
				b, err := wrapper.WrappedSignedBeaconBlock(blk)
				require.NoError(t, err)
				return b
			}(),
		},
		{
			name:      "could not get merge block",
			postState: bellatrixState,
			preState:  bellatrixState,
			blk:       bellatrixBlk,
			errString: "could not get merge block parent hash and total difficulty",
		},
		{
			name:      "happy case",
			postState: bellatrixState,
			preState:  bellatrixState,
			blk: func() block.SignedBeaconBlock {
				blk := &ethpb.SignedBeaconBlockBellatrix{
					Block: &ethpb.BeaconBlockBellatrix{
						Body: &ethpb.BeaconBlockBodyBellatrix{
							ExecutionPayload: &v1.ExecutionPayload{
								ParentHash: bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength),
							},
						},
					},
				}
				b, err := wrapper.WrappedSignedBeaconBlock(blk)
				require.NoError(t, err)
				return b
			}(),
		},
	}
	for _, tt := range tests {
		engine := &testing2.MockEngineService{NewPayloadError: tt.newPayloadErr, Blks: map[[32]byte]*v1.ExecutionBlock{}}
		engine.Blks[[32]byte{'a'}] = &v1.ExecutionBlock{
			ParentHash:      bytesutil.PadTo([]byte{'b'}, fieldparams.RootLength),
			TotalDifficulty: "0x2",
		}
		engine.Blks[[32]byte{'b'}] = &v1.ExecutionBlock{
			ParentHash:      bytesutil.PadTo([]byte{'3'}, fieldparams.RootLength),
			TotalDifficulty: "0x1",
		}
		service.cfg.ExecutionEngineCaller = engine
		var payload *ethpb.ExecutionPayloadHeader
		if tt.preState.Version() == version.Bellatrix {
			payload, err = tt.preState.LatestExecutionPayloadHeader()
			require.NoError(t, err)
		}
		err := service.notifyNewPayload(ctx, tt.preState.Version(), payload, tt.postState, tt.blk)
		if tt.errString != "" {
			require.ErrorContains(t, tt.errString, err)
		} else {
			require.NoError(t, err)
		}
	}
}

func Test_IsOptimisticCandidateBlock(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := protoarray.New(0, 0, [32]byte{'a'})
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}

	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	params.BeaconConfig().SafeSlotsToImportOptimistically = 128
	service.genesisTime = time.Now().Add(-time.Second * 12 * 2 * 128)

	tests := []struct {
		name      string
		blk       block.BeaconBlock
		justified block.SignedBeaconBlock
		want      bool
	}{
		{
			name: "deep block",
			blk: func(tt *testing.T) block.BeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 1
				wr, err := wrapper.WrappedBellatrixBeaconBlock(blk.Block)
				require.NoError(tt, err)
				return wr
			}(t),
			justified: func(tt *testing.T) block.SignedBeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 32
				wr, err := wrapper.WrappedBellatrixSignedBeaconBlock(blk)
				require.NoError(tt, err)
				return wr
			}(t),
			want: true,
		},
		{
			name: "shallow block, Altair justified chkpt",
			blk: func(tt *testing.T) block.BeaconBlock {
				blk := util.NewBeaconBlockAltair()
				blk.Block.Slot = 200
				wr, err := wrapper.WrappedAltairBeaconBlock(blk.Block)
				require.NoError(tt, err)
				return wr
			}(t),
			justified: func(tt *testing.T) block.SignedBeaconBlock {
				blk := util.NewBeaconBlockAltair()
				blk.Block.Slot = 32
				wr, err := wrapper.WrappedAltairSignedBeaconBlock(blk)
				require.NoError(tt, err)
				return wr
			}(t),
			want: false,
		},
		{
			name: "shallow block, Bellatrix justified chkpt without execution",
			blk: func(tt *testing.T) block.BeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 200
				wr, err := wrapper.WrappedBellatrixBeaconBlock(blk.Block)
				require.NoError(tt, err)
				return wr
			}(t),
			justified: func(tt *testing.T) block.SignedBeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 32
				wr, err := wrapper.WrappedBellatrixSignedBeaconBlock(blk)
				require.NoError(tt, err)
				return wr
			}(t),
			want: false,
		},
		{
			name: "shallow block, execution enabled justified chkpt",
			blk: func(tt *testing.T) block.BeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 200
				wr, err := wrapper.WrappedBellatrixBeaconBlock(blk.Block)
				require.NoError(tt, err)
				return wr
			}(t),
			justified: func(tt *testing.T) block.SignedBeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 32
				blk.Block.Body.ExecutionPayload.ParentHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				blk.Block.Body.ExecutionPayload.FeeRecipient = bytesutil.PadTo([]byte{'a'}, fieldparams.FeeRecipientLength)
				blk.Block.Body.ExecutionPayload.StateRoot = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				blk.Block.Body.ExecutionPayload.ReceiptsRoot = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				blk.Block.Body.ExecutionPayload.LogsBloom = bytesutil.PadTo([]byte{'a'}, fieldparams.LogsBloomLength)
				blk.Block.Body.ExecutionPayload.PrevRandao = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				blk.Block.Body.ExecutionPayload.BaseFeePerGas = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				blk.Block.Body.ExecutionPayload.BlockHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				wr, err := wrapper.WrappedBellatrixSignedBeaconBlock(blk)
				require.NoError(tt, err)
				return wr
			}(t),
			want: true,
		},
	}
	for _, tt := range tests {
		jroot, err := tt.justified.Block().HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, tt.justified))
		service.store.SetJustifiedCheckpt(
			&ethpb.Checkpoint{
				Root:  jroot[:],
				Epoch: slots.ToEpoch(tt.justified.Block().Slot()),
			})
		candidate, err := service.optimisticCandidateBlock(ctx, tt.blk)
		require.NoError(t, err)
		require.Equal(t, tt.want, candidate, tt.name)
	}
}
