package testing

import (
	"context"

	"github.com/prysmaticlabs/prysm/validator/db/kv"
	slashingprotection "github.com/prysmaticlabs/prysm/validator/slashing-protection"
)

var _ = slashingprotection.AttestingHistoryManager(MockAttestingHistoryManager{})

type MockAttestingHistoryManager struct{}

func (MockAttestingHistoryManager) SaveAttestingHistoryForPubKey(ctx context.Context, pubKey [48]byte) error {
	return nil
}

func (MockAttestingHistoryManager) LoadAttestingHistoryForPubKeys(ctx context.Context, attestingPubKeys [][48]byte) error {
	return nil
}

func (MockAttestingHistoryManager) AttestingHistoryForPubKey(ctx context.Context, pubKey [48]byte) (kv.EncHistoryData, error) {
	return kv.EncHistoryData{}, nil
}

func (MockAttestingHistoryManager) ResetAttestingHistoryForEpoch(ctx context.Context) {}
