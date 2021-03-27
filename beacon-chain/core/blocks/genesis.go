// Package blocks contains block processing libraries according to
// the eth2spec.
package blocks

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
func NewGenesisBlock(stateRoot []byte) *ethpb.SignedBeaconBlock {
	zeroHash := params.BeaconConfig().ZeroHash[:]
	lb := make([][]byte, 8)
	for i := range lb {
		lb[i] = make([]byte, 32)
	}
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: zeroHash,
			StateRoot:  bytesutil.PadTo(stateRoot, 32),
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: make([]byte, 96),
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot: make([]byte, 32),
					BlockHash:   make([]byte, 32),
				},
				Graffiti: make([]byte, 32),
				ApplicationPayload: &ethpb.ApplicationPayload{
					BlockHash:    make([]byte, 32),
					Coinbase:     make([]byte, 20),
					StateRoot:    make([]byte, 32),
					GasLimit:     0,
					GasUsed:      0,
					ReceiptRoot:  make([]byte, 32),
					LogsBloom:    lb,
					Transactions: make([]*ethpb.Transaction, 0),
				},
			},
		},
		Signature: params.BeaconConfig().EmptySignature[:],
	}
	return block
}
