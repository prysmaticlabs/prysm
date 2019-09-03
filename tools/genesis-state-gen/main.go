package main

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const (
	blsWithdrawalPrefixByte = byte(0)
)

var (
	domainDeposit      = [4]byte{3, 0, 0, 0}
	genesisForkVersion = []byte{0, 0, 0, 0}
)

func main() {
	//hashes := make([][]byte, len(ks))
	//dataList := make([]*ethpb.Deposit_Data, len(ks))
	//for i, item := range ks {
	//	data, err := createDepositData()
	//	if err != nil {
	//		panic(err)
	//	}
	//	hash, err := ssz.HashTreeRoot(data)
	//	if err != nil {
	//		panic(err)
	//	}
	//	hashes[i] = hash[:]
	//	dataList[i] = data
	//}
	//trie, err := trieutil.GenerateTrieFromItems(
	//	hashes,
	//	int(params.BeaconConfig().DepositContractTreeDepth),
	//)
	//if err != nil {
	//	panic(err)
	//}
	//deposits := make([]*ethpb.Deposit, len(dataList))
	//for i, item := range dataList {
	//	proof, err := trie.MerkleProof(i)
	//	if err != nil {
	//		panic(err)
	//	}
	//	deposits[i] = &ethpb.Deposit{
	//		Proof: proof,
	//		Data:  item,
	//	}
	//}
	//genesisState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{
	//	DepositRoot:  make([]byte, 32),
	//	DepositCount: 0,
	//	BlockHash:    make([]byte, 32),
	//})
	//if err != nil {
	//	panic(err)
	//}
	//// Now we need to marshal to yaml...
	//fmt.Println(genesisState)
	GenerateKeys(10)
}

//CURVE_ORDER = 52435875175126190479447740508185965837690552500527637822603658699938581184513
//validator_index_to_pubkey = {}
//pubkey_to_privkey = {}
//privkey_to_pubkey = {}
//for index in range(N):
//privkey = int.from_bytes(
//sha256(int_to_bytes(n=index, length=32)),
//byteorder='littlej
//    ) % CURVE_ORDER
//pubkey = bls.privtopubkey(privkey)
//pubkey_to_privkey[pubkey] = privkey
//privkey_to_pubkey[privkey] = pubkey
//validator_index_to_pubkey[index] = pubkey

func GenerateKeys(n int) ([]byte, []byte) {
	for i := 0; i < n; i++ {
		enc := make([]byte, 4)
		binary.LittleEndian.PutUint32(enc, uint32(i))
		hash := hashutil.Hash(enc)
		encHash := binary.LittleEndian.Uint64(hash[:])
		order := new(big.Int)
		ok := false
		order, ok = order.SetString("52435875175126190479447740508185965837690552500527637822603658699938581184513", 10)
		if !ok {
			panic("Not ok")
		}
		num := big.NewInt(int64(encHash))
		num = num.Mod(num, order)
		fmt.Println(num.String())
	}
	return nil, nil
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
