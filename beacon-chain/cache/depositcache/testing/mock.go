package testing

import (
	"context"
	"math/big"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/cache"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

type MockDepositFetcher struct {
	Snap *ethpb.DepositSnapshot
}

func (MockDepositFetcher) AllDeposits(_ context.Context, _ *big.Int) []*ethpb.Deposit {
	panic("implement me")
}

func (MockDepositFetcher) AllDepositContainers(_ context.Context) []*ethpb.DepositContainer {
	panic("implement me")
}

func (MockDepositFetcher) DepositByPubkey(_ context.Context, _ []byte) (*ethpb.Deposit, *big.Int) {
	panic("implement me")
}

func (MockDepositFetcher) DepositsNumberAndRootAtHeight(_ context.Context, _ *big.Int) (uint64, [32]byte) {
	panic("implement me")
}

func (MockDepositFetcher) InsertPendingDeposit(_ context.Context, _ *ethpb.Deposit, _ uint64, _ int64, _ [32]byte) {
	panic("implement me")
}

func (MockDepositFetcher) PendingDeposits(_ context.Context, _ *big.Int) []*ethpb.Deposit {
	panic("implement me")
}

func (MockDepositFetcher) PendingContainers(_ context.Context, _ *big.Int) []*ethpb.DepositContainer {
	panic("implement me")
}

func (MockDepositFetcher) PrunePendingDeposits(_ context.Context, _ int64) {
	panic("implement me")
}

func (MockDepositFetcher) PruneProofs(_ context.Context, _ int64) error {
	panic("implement me")
}

func (m MockDepositFetcher) Snapshot() (*ethpb.DepositSnapshot, error) {
	return m.Snap, nil
}

func (MockDepositFetcher) FinalizedDeposits(_ context.Context) (cache.FinalizedDeposits, error) {
	panic("implement me")
}

func (MockDepositFetcher) NonFinalizedDeposits(_ context.Context, _ int64, _ *big.Int) []*ethpb.Deposit {
	panic("implement me")
}
