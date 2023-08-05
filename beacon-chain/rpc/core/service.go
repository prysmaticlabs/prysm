package core

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	opfeed "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/operation"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/synccommittee"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/sync"
)

type Service struct {
	HeadFetcher        blockchain.HeadFetcher
	GenesisTimeFetcher blockchain.TimeFetcher
	SyncChecker        sync.Checker
	Broadcaster        p2p.Broadcaster
	SyncCommitteePool  synccommittee.Pool
	OperationNotifier  opfeed.Notifier
}
