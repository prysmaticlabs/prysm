package sync

import (
	"context"
	initialsync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync"
)

type SyncService struct {
	RegularSync *RegularSync
	InitialSync *initialsync.InitialSync
	Querier     *SyncQuerier
}

func NewSyncService(ctx context.Context, syncCfg *Config, queryCfg *QuerierConfig,
	initCfg *initialsync.Config) *SyncService {

	sq := NewSyncQuerierService(ctx, queryCfg)
	is := initialsync.NewInitialSyncService(ctx, initCfg)
	rs := NewRegularSyncService(ctx, syncCfg)

	return &SyncService{
		RegularSync: rs,
		InitialSync: is,
		Querier:     sq,
	}

}

func (ss *SyncService) Start() {

}

func (ss *SyncService) Stop() {

}
