package testutil

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
)

func TestSetupInitialDeposits_1024Entries(t *testing.T) {
	entries := 1
	ResetCache()
	deposits, privKeys, err := DeterministicDepositsAndKeys(uint64(entries))
	if err != nil {
		t.Fatal(err)
	}
	_, depositDataRoots, err := DeterministicDepositTrie(len(deposits))
	if err != nil {
		t.Fatal(err)
	}

	if len(deposits) != entries {
		t.Fatalf("incorrect number of deposits returned, wanted %d but received %d", entries, len(deposits))
	}
	if len(privKeys) != entries {
		t.Fatalf("incorrect number of private keys returned, wanted %d but received %d", entries, len(privKeys))
	}
	expectedPublicKeyAt0 := []byte{0xa9, 0x9a, 0x76, 0xed, 0x77, 0x96, 0xf7, 0xbe, 0x22, 0xd5, 0xb7, 0xe8, 0x5d, 0xee, 0xb7, 0xc5, 0x67, 0x7e, 0x88, 0xe5, 0x11, 0xe0, 0xb3, 0x37, 0x61, 0x8f, 0x8c, 0x4e, 0xb6, 0x13, 0x49, 0xb4, 0xbf, 0x2d, 0x15, 0x3f, 0x64, 0x9f, 0x7b, 0x53, 0x35, 0x9f, 0xe8, 0xb9, 0x4a, 0x38, 0xe4, 0x4c}
	if !bytes.Equal(deposits[0].Data.PublicKey, expectedPublicKeyAt0) {
		t.Fatalf("incorrect public key, wanted %x but received %x", expectedPublicKeyAt0, deposits[0].Data.PublicKey)
	}
	expectedWithdrawalCredentialsAt0 := []byte{0x00, 0xec, 0x7e, 0xf7, 0x78, 0x0c, 0x9d, 0x15, 0x15, 0x97, 0x92, 0x40, 0x36, 0x26, 0x2d, 0xd2, 0x8d, 0xc6, 0x0e, 0x12, 0x28, 0xf4, 0xda, 0x6f, 0xec, 0xf9, 0xd4, 0x02, 0xcb, 0x3f, 0x35, 0x94}
	if !bytes.Equal(deposits[0].Data.WithdrawalCredentials, expectedWithdrawalCredentialsAt0) {
		t.Fatalf("incorrect withdrawal credentials, wanted %x but received %x", expectedWithdrawalCredentialsAt0, deposits[0].Data.WithdrawalCredentials)
	}

	expectedDepositDataRootAt0 := []byte{0x3c, 0x8a, 0xb9, 0xf0, 0xfc, 0x9e, 0x5b, 0x2e, 0x29, 0xee, 0xf5, 0xb6, 0xa3, 0xf9, 0x06, 0x14, 0x54, 0x52, 0xf0, 0x37, 0xed, 0xdd, 0x4b, 0x20, 0x74, 0xc6, 0x66, 0xba, 0xa7, 0x29, 0x51, 0x75}
	if !bytes.Equal(depositDataRoots[0][:], expectedDepositDataRootAt0) {
		t.Fatalf("incorrect deposit data root, wanted %x but received %x", expectedDepositDataRootAt0, depositDataRoots[0])
	}

	sigAt0 := []byte("9348b36c0a48a0b7543b17b1da3c65ca6997ddc3284a7044c363d5cb81e82c0148ec861f5ae3ea0cfc136fa836bbb5360aee9f92b9c2597ceff1d759741edc8f90481b27fce59e1c7ff3c81792e9ba0ef7d67c6a8ac264eaef255b369ce433a4")
	sigAt0B := make([]byte, hex.DecodedLen(len(sigAt0)))
	_, err = hex.Decode(sigAt0B, sigAt0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(deposits[0].Data.Signature, sigAt0B) {
		t.Fatalf("incorrect signature, wanted %x but received %x", sigAt0B, deposits[0].Data.Signature)
	}

	entries = 1024
	ResetCache()
	deposits, privKeys, err = DeterministicDepositsAndKeys(uint64(entries))
	if err != nil {
		t.Fatal(err)
	}
	_, depositDataRoots, err = DeterministicDepositTrie(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	if len(deposits) != entries {
		t.Fatalf("incorrect number of deposits returned, wanted %d but received %d", entries, len(deposits))
	}
	if len(privKeys) != entries {
		t.Fatalf("incorrect number of private keys returned, wanted %d but received %d", entries, len(privKeys))
	}
	// Ensure 0  has not changed
	if !bytes.Equal(deposits[0].Data.PublicKey, expectedPublicKeyAt0) {
		t.Fatalf("incorrect public key, wanted %x but received %x", expectedPublicKeyAt0, deposits[0].Data.PublicKey)
	}
	if !bytes.Equal(deposits[0].Data.WithdrawalCredentials, expectedWithdrawalCredentialsAt0) {
		t.Fatalf("incorrect withdrawal credentials, wanted %x but received %x", expectedWithdrawalCredentialsAt0, deposits[0].Data.WithdrawalCredentials)
	}
	if !bytes.Equal(depositDataRoots[0][:], expectedDepositDataRootAt0) {
		t.Fatalf("incorrect deposit data root, wanted %x but received %x", expectedDepositDataRootAt0, depositDataRoots[0])
	}
	if !bytes.Equal(deposits[0].Data.Signature, sigAt0B) {
		t.Fatalf("incorrect signature, wanted %x but received %x", sigAt0B, deposits[0].Data.Signature)
	}
	expectedPublicKeyAt1023 := []byte{0x81, 0x2b, 0x93, 0x5e, 0xc8, 0x4b, 0x0e, 0x9a, 0x83, 0x95, 0x55, 0xaf, 0x33, 0x60, 0xca, 0xfb, 0x83, 0x1b, 0xd6, 0x12, 0xcf, 0xa2, 0x2e, 0x25, 0xea, 0xb0, 0x3c, 0xf5, 0xfd, 0xb0, 0x2a, 0xf5, 0x2b, 0xa4, 0x01, 0x7a, 0xee, 0xa8, 0x8a, 0x2f, 0x62, 0x2c, 0x78, 0x6e, 0x7f, 0x47, 0x6f, 0x4b}
	if !bytes.Equal(deposits[1023].Data.PublicKey, expectedPublicKeyAt1023) {
		t.Fatalf("incorrect public key, wanted %x but received %x", expectedPublicKeyAt1023, deposits[1023].Data.PublicKey)
	}
	expectedWithdrawalCredentialsAt1023 := []byte{0x00, 0x23, 0xd5, 0x76, 0xbc, 0x6c, 0x15, 0xdb, 0xc4, 0x34, 0x70, 0x1f, 0x3f, 0x41, 0xfd, 0x3e, 0x67, 0x59, 0xd2, 0xea, 0x7c, 0xdc, 0x64, 0x71, 0x0e, 0xe2, 0x8d, 0xde, 0xf7, 0xd2, 0xda, 0x28}
	if !bytes.Equal(deposits[1023].Data.WithdrawalCredentials, expectedWithdrawalCredentialsAt1023) {
		t.Fatalf("incorrect withdrawal credentials, wanted %x but received %x", expectedWithdrawalCredentialsAt1023, deposits[1023].Data.WithdrawalCredentials)
	}
	dRootAt1023 := []byte("259330f2a7b8db40a622f95a50492ab9bf9dac788fb9dbf6b39ddec3a555adfc")
	dRootAt1023B := make([]byte, hex.DecodedLen(len(dRootAt1023)))
	_, err = hex.Decode(dRootAt1023B, dRootAt1023)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(depositDataRoots[1023][:], dRootAt1023B) {
		t.Fatalf("incorrect deposit data root, wanted %x but received %x", dRootAt1023B, depositDataRoots[1023])
	}
	sigAt1023 := []byte("a0333abe2c75aa2e672cbb7f191a7936dd08b81dfa3e2f0b1478445c8897169bdea5a507ebf0dcf947eeddc6b23c2e4905e2c4e43fcbc6fac360c662d74e2c3af79390a79e43ef6a7731a6c2ea0999accde35bf1dc0b132487d701ca8459c63e")
	sigAt1023B := make([]byte, hex.DecodedLen(len(sigAt1023)))
	_, err = hex.Decode(sigAt1023B, sigAt1023)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(deposits[1023].Data.Signature, sigAt1023B) {
		t.Fatalf("incorrect signature, wanted %x but received %x", sigAt1023B, deposits[1023].Data.Signature)
	}
}

func TestDeterministicGenesisState_100Validators(t *testing.T) {
	validatorCount := uint64(100)
	beaconState, privKeys := DeterministicGenesisState(t, validatorCount)
	activeValidators, err := helpers.ActiveValidatorCount(beaconState, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(privKeys) != int(validatorCount) {
		t.Fatalf("expected amount of private keys %d to match requested amount of validators %d", len(privKeys), validatorCount)
	}
	if activeValidators != validatorCount {
		t.Fatalf("expected validators in state %d to match requested amount %d", activeValidators, validatorCount)
	}
}

func TestDepositTrieFromDeposits(t *testing.T) {
	deposits, _, err := DeterministicDepositsAndKeys(100)
	if err != nil {
		t.Fatal(err)
	}
	eth1Data, err := DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(err)
	}

	depositTrie, _, err := DepositTrieFromDeposits(deposits)
	if err != nil {
		t.Fatal(err)
	}

	root := depositTrie.Root()
	if !bytes.Equal(root[:], eth1Data.DepositRoot) {
		t.Fatal("expected deposit trie root to equal eth1data deposit root")
	}
}
