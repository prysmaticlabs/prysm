package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"gopkg.in/yaml.v2"
)

const (
	blsWithdrawalPrefixByte = byte(0)
)

var (
	inputFile          = flag.String("validator-keys-yaml", "", "Input validator keys YAML file")
	domainDeposit      = [4]byte{3, 0, 0, 0}
	genesisForkVersion = []byte{0, 0, 0, 0}
)

type keyPair struct {
	PrivateKey string `yaml:"privkey"`
	PublicKey  string `yaml:"pubkey"`
}

func main() {
	flag.Parse()
	f, err := os.Open(*inputFile)
	if err != nil {
		panic(err)
	}
	enc, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	var ks []*keyPair
	if err := yaml.Unmarshal(enc, &ks); err != nil {
		panic(err)
	}
	deps := make([]*ethpb.Deposit, len(ks))
	for i, item := range ks {
		dp, err := createDepositData([]byte(item.PrivateKey), []byte(item.PublicKey))
		if err != nil {
			panic(err)
		}
		deps[i] = &ethpb.Deposit{
			Data: dp,
		}
	}
	fmt.Println(deps)
}

func createDepositData(privKey []byte, pubKey []byte) (*ethpb.Deposit_Data, error) {
	sk1, err := bls.SecretKeyFromBytes(privKey)
	if err != nil {
		return nil, err
	}
	di := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		WithdrawalCredentials: withdrawalCredentialsHash(pubKey),
		Amount:                params.BeaconConfig().MaxEffectiveBalance,
	}

	sr, err := ssz.HashTreeRoot(di)
	if err != nil {
		return nil, err
	}

	domain := bls.Domain(domainDeposit[:], genesisForkVersion)
	di.Signature = sk1.Sign(sr[:], domain).Marshal()
	return di, nil
}

// withdrawalCredentialsHash forms a 32 byte hash of the withdrawal public
// address.
//
// The specification is as follows:
//   withdrawal_credentials[:1] == BLS_WITHDRAWAL_PREFIX_BYTE
//   withdrawal_credentials[1:] == hash(withdrawal_pubkey)[1:]
// where withdrawal_credentials is of type bytes32.
func withdrawalCredentialsHash(pubKey []byte) []byte {
	h := hashutil.HashKeccak256(pubKey)
	return append([]byte{blsWithdrawalPrefixByte}, h[0:]...)[:32]
}
