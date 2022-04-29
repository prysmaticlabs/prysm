package blockchain

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/prysmaticlabs/prysm/time/slots"
	logTest "github.com/sirupsen/logrus/hooks/test"
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
		WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
	}
	service, err := NewService(ctx, opts...)
	st, _ := util.DeterministicGenesisState(t, 1)
	service.head = &head{
		state: st,
	}
	require.NoError(t, err)
	require.NoError(t, fcs.InsertOptimisticBlock(ctx, 0, [32]byte{}, [32]byte{}, params.BeaconConfig().ZeroHash, 0, 0))

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
			newForkchoiceErr: powchain.ErrAcceptedSyncingPayloadStatus,
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
			newForkchoiceErr: powchain.ErrInvalidPayloadStatus,
			finalizedRoot:    bellatrixBlkRoot,
			errString:        ErrUndefinedExecutionEngineError.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service.cfg.ExecutionEngineCaller = &mockPOW.EngineClient{ErrForkchoiceUpdated: tt.newForkchoiceErr}
			st, _ := util.DeterministicGenesisState(t, 1)
			_, err := service.notifyForkchoiceUpdate(ctx, st, tt.blk, service.headRoot(), tt.finalizedRoot)
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
	a := &ethpb.SignedBeaconBlockAltair{
		Block: &ethpb.BeaconBlockAltair{
			Body: &ethpb.BeaconBlockBodyAltair{},
		},
	}
	altairBlk, err := wrapper.WrappedSignedBeaconBlock(a)
	require.NoError(t, err)
	blk := &ethpb.SignedBeaconBlockBellatrix{
		Block: &ethpb.BeaconBlockBellatrix{
			Slot: 1,
			Body: &ethpb.BeaconBlockBodyBellatrix{
				ExecutionPayload: &v1.ExecutionPayload{
					BlockNumber:   1,
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
	bellatrixBlk, err := wrapper.WrappedSignedBeaconBlock(util.HydrateSignedBeaconBlockBellatrix(blk))
	require.NoError(t, err)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	r, err := bellatrixBlk.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, fcs.InsertOptimisticBlock(ctx, 0, [32]byte{}, [32]byte{}, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, fcs.InsertOptimisticBlock(ctx, 1, r, [32]byte{}, params.BeaconConfig().ZeroHash, 0, 0))

	tests := []struct {
		name           string
		postState      state.BeaconState
		isValidPayload bool
		blk            block.SignedBeaconBlock
		newPayloadErr  error
		errString      string
	}{
		{
			name:           "phase 0 post state",
			postState:      phase0State,
			isValidPayload: true,
		},
		{
			name:           "altair post state",
			postState:      altairState,
			isValidPayload: true,
		},
		{
			name:           "nil beacon block",
			postState:      bellatrixState,
			errString:      "signed beacon block can't be nil",
			isValidPayload: false,
		},
		{
			name:           "new payload with optimistic block",
			postState:      bellatrixState,
			blk:            bellatrixBlk,
			newPayloadErr:  powchain.ErrAcceptedSyncingPayloadStatus,
			isValidPayload: false,
		},
		{
			name:           "new payload with invalid block",
			postState:      bellatrixState,
			blk:            bellatrixBlk,
			newPayloadErr:  powchain.ErrInvalidPayloadStatus,
			errString:      "could not validate an INVALID payload from execution engine",
			isValidPayload: false,
		},
		{
			name:           "altair pre state, altair block",
			postState:      bellatrixState,
			blk:            altairBlk,
			isValidPayload: true,
		},
		{
			name:      "altair pre state, happy case",
			postState: bellatrixState,
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
			isValidPayload: true,
		},
		{
			name:      "not at merge transition",
			postState: bellatrixState,
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
			isValidPayload: true,
		},
		{
			name:      "happy case",
			postState: bellatrixState,
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
			isValidPayload: true,
		},
		{
			name:      "undefined error from ee",
			postState: bellatrixState,
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
			newPayloadErr: ErrUndefinedExecutionEngineError,
			errString:     ErrUndefinedExecutionEngineError.Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &mockPOW.EngineClient{ErrNewPayload: tt.newPayloadErr, BlockByHashMap: map[[32]byte]*v1.ExecutionBlock{}}
			e.BlockByHashMap[[32]byte{'a'}] = &v1.ExecutionBlock{
				ParentHash:      bytesutil.PadTo([]byte{'b'}, fieldparams.RootLength),
				TotalDifficulty: "0x2",
			}
			e.BlockByHashMap[[32]byte{'b'}] = &v1.ExecutionBlock{
				ParentHash:      bytesutil.PadTo([]byte{'3'}, fieldparams.RootLength),
				TotalDifficulty: "0x1",
			}
			service.cfg.ExecutionEngineCaller = e
			root := [32]byte{'a'}
			require.NoError(t, service.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 0, root, root, params.BeaconConfig().ZeroHash, 0, 0))
			postVersion, postHeader, err := getStateVersionAndPayload(tt.postState)
			require.NoError(t, err)
			isValidPayload, err := service.notifyNewPayload(ctx, postVersion, postHeader, tt.blk)
			if tt.errString != "" {
				require.ErrorContains(t, tt.errString, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.isValidPayload, isValidPayload)
			}
		})
	}
}

func Test_NotifyNewPayload_SetOptimisticToValid(t *testing.T) {
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
	bellatrixState, _ := util.DeterministicGenesisStateBellatrix(t, 2)
	blk := &ethpb.SignedBeaconBlockBellatrix{
		Block: &ethpb.BeaconBlockBellatrix{
			Body: &ethpb.BeaconBlockBodyBellatrix{
				ExecutionPayload: &v1.ExecutionPayload{
					ParentHash: bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength),
				},
			},
		},
	}
	bellatrixBlk, err := wrapper.WrappedSignedBeaconBlock(blk)
	require.NoError(t, err)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	e := &mockPOW.EngineClient{BlockByHashMap: map[[32]byte]*v1.ExecutionBlock{}}
	e.BlockByHashMap[[32]byte{'a'}] = &v1.ExecutionBlock{
		ParentHash:      bytesutil.PadTo([]byte{'b'}, fieldparams.RootLength),
		TotalDifficulty: "0x2",
	}
	e.BlockByHashMap[[32]byte{'b'}] = &v1.ExecutionBlock{
		ParentHash:      bytesutil.PadTo([]byte{'3'}, fieldparams.RootLength),
		TotalDifficulty: "0x1",
	}
	service.cfg.ExecutionEngineCaller = e
	postVersion, postHeader, err := getStateVersionAndPayload(bellatrixState)
	require.NoError(t, err)
	validated, err := service.notifyNewPayload(ctx, postVersion, postHeader, bellatrixBlk)
	require.NoError(t, err)
	require.Equal(t, true, validated)
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

	parentBlk := util.NewBeaconBlockBellatrix()
	wrappedParentBlock, err := wrapper.WrappedSignedBeaconBlock(parentBlk)
	require.NoError(t, err)
	parentRoot, err := wrappedParentBlock.Block().HashTreeRoot()
	require.NoError(t, err)

	tests := []struct {
		name      string
		blk       block.BeaconBlock
		justified block.SignedBeaconBlock
		err       error
	}{
		{
			name: "deep block",
			blk: func(tt *testing.T) block.BeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 1
				blk.Block.ParentRoot = parentRoot[:]
				wr, err := wrapper.WrappedBeaconBlock(blk.Block)
				require.NoError(tt, err)
				return wr
			}(t),
			justified: func(tt *testing.T) block.SignedBeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 32
				blk.Block.ParentRoot = parentRoot[:]
				wr, err := wrapper.WrappedSignedBeaconBlock(blk)
				require.NoError(tt, err)
				return wr
			}(t),
			err: nil,
		},
		{
			name: "shallow block, Altair justified chkpt",
			blk: func(tt *testing.T) block.BeaconBlock {
				blk := util.NewBeaconBlockAltair()
				blk.Block.Slot = 200
				blk.Block.ParentRoot = parentRoot[:]
				wr, err := wrapper.WrappedBeaconBlock(blk.Block)
				require.NoError(tt, err)
				return wr
			}(t),
			justified: func(tt *testing.T) block.SignedBeaconBlock {
				blk := util.NewBeaconBlockAltair()
				blk.Block.Slot = 32
				blk.Block.ParentRoot = parentRoot[:]
				wr, err := wrapper.WrappedSignedBeaconBlock(blk)
				require.NoError(tt, err)
				return wr
			}(t),
			err: errNotOptimisticCandidate,
		},
		{
			name: "shallow block, Bellatrix justified chkpt without execution",
			blk: func(tt *testing.T) block.BeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 200
				blk.Block.ParentRoot = parentRoot[:]
				wr, err := wrapper.WrappedBeaconBlock(blk.Block)
				require.NoError(tt, err)
				return wr
			}(t),
			justified: func(tt *testing.T) block.SignedBeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 32
				blk.Block.ParentRoot = parentRoot[:]
				wr, err := wrapper.WrappedSignedBeaconBlock(blk)
				require.NoError(tt, err)
				return wr
			}(t),
			err: errNotOptimisticCandidate,
		},
	}
	for _, tt := range tests {
		jRoot, err := tt.justified.Block().HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, tt.justified))
		service.store.SetJustifiedCheckpt(
			&ethpb.Checkpoint{
				Root:  jRoot[:],
				Epoch: slots.ToEpoch(tt.justified.Block().Slot()),
			})
		require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrappedParentBlock))

		err = service.optimisticCandidateBlock(ctx, tt.blk)
		require.Equal(t, tt.err, err)
	}
}

func Test_IsOptimisticShallowExecutionParent(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}

	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	params.BeaconConfig().SafeSlotsToImportOptimistically = 128
	service.genesisTime = time.Now().Add(-time.Second * 12 * 2 * 128)
	payload := &v1.ExecutionPayload{
		ParentHash:    make([]byte, 32),
		FeeRecipient:  make([]byte, 20),
		StateRoot:     make([]byte, 32),
		ReceiptsRoot:  make([]byte, 32),
		LogsBloom:     make([]byte, 256),
		PrevRandao:    make([]byte, 32),
		BaseFeePerGas: bytesutil.PadTo([]byte{1, 2, 3, 4}, fieldparams.RootLength),
		BlockHash:     make([]byte, 32),
		BlockNumber:   100,
	}
	body := &ethpb.BeaconBlockBodyBellatrix{ExecutionPayload: payload}
	b := &ethpb.BeaconBlockBellatrix{Body: body, Slot: 200}
	rawSigned := &ethpb.SignedBeaconBlockBellatrix{Block: b}
	blk := util.HydrateSignedBeaconBlockBellatrix(rawSigned)
	wr, err := wrapper.WrappedSignedBeaconBlock(blk)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wr))
	blkRoot, err := wr.Block().HashTreeRoot()
	require.NoError(t, err)

	childBlock := util.NewBeaconBlockBellatrix()
	childBlock.Block.ParentRoot = blkRoot[:]
	// shallow block
	childBlock.Block.Slot = 201
	wrappedChild, err := wrapper.WrappedSignedBeaconBlock(childBlock)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrappedChild))
	err = service.optimisticCandidateBlock(ctx, wrappedChild.Block())
	require.NoError(t, err)
}

func Test_GetPayloadAttribute(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
	}

	// Cache miss
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	hasPayload, _, vId, err := service.getPayloadAttribute(ctx, nil, 0)
	require.NoError(t, err)
	require.Equal(t, false, hasPayload)
	require.Equal(t, types.ValidatorIndex(0), vId)

	// Cache hit, advance state, no fee recipient
	suggestedVid := types.ValidatorIndex(1)
	slot := types.Slot(1)
	service.cfg.ProposerSlotIndexCache.SetProposerAndPayloadIDs(slot, suggestedVid, [8]byte{})
	st, _ := util.DeterministicGenesisState(t, 1)
	hook := logTest.NewGlobal()
	hasPayload, attr, vId, err := service.getPayloadAttribute(ctx, st, slot)
	require.NoError(t, err)
	require.Equal(t, true, hasPayload)
	require.Equal(t, suggestedVid, vId)
	require.Equal(t, fieldparams.EthBurnAddressHex, common.BytesToAddress(attr.SuggestedFeeRecipient).String())
	require.LogsContain(t, hook, "Fee recipient not set. Using burn address")

	// Cache hit, advance state, has fee recipient
	suggestedAddr := common.HexToAddress("123")
	require.NoError(t, service.cfg.BeaconDB.SaveFeeRecipientsByValidatorIDs(ctx, []types.ValidatorIndex{suggestedVid}, []common.Address{suggestedAddr}))
	service.cfg.ProposerSlotIndexCache.SetProposerAndPayloadIDs(slot, suggestedVid, [8]byte{})
	hasPayload, attr, vId, err = service.getPayloadAttribute(ctx, st, slot)
	require.NoError(t, err)
	require.Equal(t, true, hasPayload)
	require.Equal(t, suggestedVid, vId)
	require.Equal(t, suggestedAddr, common.BytesToAddress(attr.SuggestedFeeRecipient))
}

func Test_UpdateLastValidatedCheckpoint(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	stateGen := stategen.New(beaconDB)
	fcs := protoarray.New(0, 0, [32]byte{})
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stateGen),
		WithForkChoiceStore(fcs),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	genesisStateRoot := [32]byte{}
	genesisBlk := blocks.NewGenesisBlock(genesisStateRoot[:])
	wr, err := wrapper.WrappedSignedBeaconBlock(genesisBlk)
	require.NoError(t, err)
	assert.NoError(t, beaconDB.SaveBlock(ctx, wr))
	genesisRoot, err := genesisBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	assert.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, genesisRoot))
	require.NoError(t, fcs.InsertOptimisticBlock(ctx, 0, genesisRoot, params.BeaconConfig().ZeroHash,
		params.BeaconConfig().ZeroHash, 0, 0))
	genesisSummary := &ethpb.StateSummary{
		Root: genesisStateRoot[:],
		Slot: 0,
	}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, genesisSummary))

	// Get last validated checkpoint
	origCheckpoint, err := service.cfg.BeaconDB.LastValidatedCheckpoint(ctx)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveLastValidatedCheckpoint(ctx, origCheckpoint))

	// Optimistic finalized checkpoint
	blk := util.NewBeaconBlock()
	blk.Block.Slot = 320
	blk.Block.ParentRoot = genesisRoot[:]
	wr, err = wrapper.WrappedSignedBeaconBlock(blk)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wr))
	opRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)

	opCheckpoint := &ethpb.Checkpoint{
		Root:  opRoot[:],
		Epoch: 10,
	}
	opStateSummary := &ethpb.StateSummary{
		Root: opRoot[:],
		Slot: 320,
	}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, opStateSummary))
	require.NoError(t, fcs.InsertOptimisticBlock(ctx, 320, opRoot, genesisRoot,
		params.BeaconConfig().ZeroHash, 10, 10))
	assert.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, opRoot))
	require.NoError(t, service.updateFinalized(ctx, opCheckpoint))
	cp, err := service.cfg.BeaconDB.LastValidatedCheckpoint(ctx)
	require.NoError(t, err)
	require.DeepEqual(t, origCheckpoint.Root, cp.Root)
	require.Equal(t, origCheckpoint.Epoch, cp.Epoch)

	// Validated finalized checkpoint
	blk = util.NewBeaconBlock()
	blk.Block.Slot = 640
	blk.Block.ParentRoot = opRoot[:]
	wr, err = wrapper.WrappedSignedBeaconBlock(blk)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wr))
	validRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)

	validCheckpoint := &ethpb.Checkpoint{
		Root:  validRoot[:],
		Epoch: 20,
	}
	validSummary := &ethpb.StateSummary{
		Root: validRoot[:],
		Slot: 640,
	}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, validSummary))
	require.NoError(t, fcs.InsertOptimisticBlock(ctx, 640, validRoot, params.BeaconConfig().ZeroHash,
		params.BeaconConfig().ZeroHash, 20, 20))
	require.NoError(t, fcs.SetOptimisticToValid(ctx, validRoot))
	assert.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, validRoot))
	require.NoError(t, service.updateFinalized(ctx, validCheckpoint))
	cp, err = service.cfg.BeaconDB.LastValidatedCheckpoint(ctx)
	require.NoError(t, err)

	optimistic, err := service.IsOptimisticForRoot(ctx, validRoot)
	require.NoError(t, err)
	require.Equal(t, false, optimistic)
	require.DeepEqual(t, validCheckpoint.Root, cp.Root)
	require.Equal(t, validCheckpoint.Epoch, cp.Epoch)
}

func TestService_removeInvalidBlockAndState(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(protoarray.New(0, 0, [32]byte{})),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	// Deleting unknown block should not error.
	require.NoError(t, service.removeInvalidBlockAndState(ctx, [][32]byte{{'a'}, {'b'}, {'c'}}))

	// Happy case
	b1 := util.NewBeaconBlock()
	b1.Block.Slot = 1
	blk1, err := wrapper.WrappedSignedBeaconBlock(b1)
	require.NoError(t, err)
	r1, err := blk1.Block().HashTreeRoot()
	require.NoError(t, err)
	st, _ := util.DeterministicGenesisStateBellatrix(t, 1)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, blk1))
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{
		Slot: 1,
		Root: r1[:],
	}))
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st, r1))

	b2 := util.NewBeaconBlock()
	b2.Block.Slot = 2
	blk2, err := wrapper.WrappedSignedBeaconBlock(b2)
	require.NoError(t, err)
	r2, err := blk2.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, blk2))
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{
		Slot: 2,
		Root: r2[:],
	}))
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st, r2))

	require.NoError(t, service.removeInvalidBlockAndState(ctx, [][32]byte{r1, r2}))

	require.Equal(t, false, service.hasBlock(ctx, r1))
	require.Equal(t, false, service.hasBlock(ctx, r2))
	require.Equal(t, false, service.cfg.BeaconDB.HasStateSummary(ctx, r1))
	require.Equal(t, false, service.cfg.BeaconDB.HasStateSummary(ctx, r2))
	has, err := service.cfg.StateGen.HasState(ctx, r1)
	require.NoError(t, err)
	require.Equal(t, false, has)
	has, err = service.cfg.StateGen.HasState(ctx, r2)
	require.NoError(t, err)
	require.Equal(t, false, has)
}
