package withdrawal

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
)

type BlsToExecutionEngineMessage struct {
	ValidatorIndex     uint64 `json:"validator_index" yaml:"validator_index"`
	FromBlsPubkey      string `json:"from_bls_pubkey" yaml:"from_bls_pubkey"`
	ToExecutionAddress string `json:"to_execution_address" yaml:"to_execution_address"`
}

func (d *BlsToExecutionEngineMessage) FromBlsPubkeyAsBytes() ([]byte, error) {
	return hexutil.Decode(d.FromBlsPubkey)
}

func (d *BlsToExecutionEngineMessage) ToExecutionAddressAsBytes() ([]byte, error) {
	return hexutil.Decode(d.ToExecutionAddress)
}

type BlsToExecutionEngineFile struct {
	Version   uint64                       `json:"version" yaml:"version"`
	Message   *BlsToExecutionEngineMessage `json:"message" yaml:"message"`
	Signature string                       `json:"signature" yaml:"signature"`
}

func (b *BlsToExecutionEngineFile) SignatureAsBls() (bls.Signature, error) {
	bytevalue, err := hexutil.Decode(b.Signature)
	if err != nil {
		return nil, err
	}
	return bls.SignatureFromBytes(bytevalue)
}
