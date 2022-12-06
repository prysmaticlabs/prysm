// Package blocks contains block processing libraries according to
// the Ethereum beacon chain spec.
package blocks

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
func NewGenesisBlock(stateRoot []byte) *ethpb.SignedBeaconBlock {
	zeroHash := params.BeaconConfig().ZeroHash[:]
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: zeroHash,
			StateRoot:  bytesutil.PadTo(stateRoot, 32),
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: make([]byte, fieldparams.BLSSignatureLength),
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot: make([]byte, 32),
					BlockHash:   make([]byte, 32),
				},
				Graffiti: make([]byte, 32),
			},
		},
		Signature: params.BeaconConfig().EmptySignature[:],
	}
	return block
}

var ErrUnrecognizedState = errors.New("uknonwn underlying type for state.BeaconState value")

func NewGenesisBlockForState(root [32]byte, st state.BeaconState) (interfaces.SignedBeaconBlock, error) {
	ps := st.ToProto()
	switch ps.(type) {
	case *ethpb.BeaconState:
		return blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlock{
			Block: &ethpb.BeaconBlock{
				ParentRoot: params.BeaconConfig().ZeroHash[:],
				StateRoot:  root[:],
				Body: &ethpb.BeaconBlockBody{
					RandaoReveal: make([]byte, fieldparams.BLSSignatureLength),
					Eth1Data: &ethpb.Eth1Data{
						DepositRoot: make([]byte, 32),
						BlockHash:   make([]byte, 32),
					},
					Graffiti: make([]byte, 32),
				},
			},
			Signature: params.BeaconConfig().EmptySignature[:],
		})
	case *ethpb.BeaconStateBellatrix:
		hi, err := st.LatestExecutionPayloadHeader()
		if err != nil {
			return nil, err
		}
		txr, err := hi.TransactionsRoot()
		if err != nil {
			return nil, err
		}
		h := &enginev1.ExecutionPayloadHeader{
			ParentHash:       bytesutil.SafeCopyBytes(hi.ParentHash()),
			FeeRecipient:     bytesutil.SafeCopyBytes(hi.FeeRecipient()),
			StateRoot:        bytesutil.SafeCopyBytes(hi.StateRoot()),
			ReceiptsRoot:     bytesutil.SafeCopyBytes(hi.ReceiptsRoot()),
			LogsBloom:        bytesutil.SafeCopyBytes(hi.LogsBloom()),
			PrevRandao:       bytesutil.SafeCopyBytes(hi.PrevRandao()),
			BlockNumber:      hi.BlockNumber(),
			GasLimit:         hi.GasLimit(),
			GasUsed:          hi.GasUsed(),
			Timestamp:        hi.Timestamp(),
			ExtraData:        bytesutil.SafeCopyBytes(hi.ExtraData()),
			BaseFeePerGas:    bytesutil.SafeCopyBytes(hi.BaseFeePerGas()),
			BlockHash:        bytesutil.SafeCopyBytes(hi.BlockHash()),
			TransactionsRoot: bytesutil.SafeCopyBytes(txr),
		}
		return blocks.NewSignedBeaconBlock(&ethpb.SignedBlindedBeaconBlockBellatrix{
			Block: &ethpb.BlindedBeaconBlockBellatrix{
				ParentRoot: params.BeaconConfig().ZeroHash[:],
				StateRoot:  root[:],
				Body: &ethpb.BlindedBeaconBlockBodyBellatrix{
					RandaoReveal: make([]byte, 96),
					Eth1Data:     st.Eth1Data(),
					Graffiti:     make([]byte, 32),
					SyncAggregate: &ethpb.SyncAggregate{
						SyncCommitteeBits:      make([]byte, fieldparams.SyncCommitteeLength/8),
						SyncCommitteeSignature: make([]byte, fieldparams.BLSSignatureLength),
					},
					ExecutionPayloadHeader: h,
				},
			},
			Signature: params.BeaconConfig().EmptySignature[:],
		})
	default:
		return nil, ErrUnrecognizedState
		/*
			case *ethpb.BeaconStateAltair:
			case *ethpb.BeaconStateCapella:
		*/
	}
}
