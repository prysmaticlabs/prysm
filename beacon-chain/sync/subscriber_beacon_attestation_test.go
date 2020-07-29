package sync

import (
	"context"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestService_committeeIndexBeaconAttestationSubscriber_ValidMessage(t *testing.T) {

	p := p2ptest.NewTestP2P(t)
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{DisableDynamicCommitteeSubnets: true})
	defer resetCfg()

	ctx := context.Background()
	db, _ := dbtest.SetupDB(t)
	s, sKeys := testutil.DeterministicGenesisState(t, 64 /*validators*/)
	require.NoError(t, s.SetGenesisTime(uint64(time.Now().Unix())))
	blk, err := testutil.GenerateFullBlock(s, sKeys, nil, 1)
	require.NoError(t, err)
	root, err := stateutil.BlockRoot(blk.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, blk))

	savedState := testutil.NewBeaconState()
	require.NoError(t, db.SaveState(context.Background(), savedState, root))

	c, err := lru.New(10)
	require.NoError(t, err)
	r := &Service{
		attPool: attestations.NewPool(),
		chain: &mock.ChainService{
			State:            s,
			Genesis:          time.Now(),
			ValidAttestation: true,
			ValidatorsRoot:   [32]byte{'A'},
		},
		chainStarted:         true,
		p2p:                  p,
		db:                   db,
		ctx:                  ctx,
		stateNotifier:        (&mock.ChainService{}).StateNotifier(),
		attestationNotifier:  (&mock.ChainService{}).OperationNotifier(),
		initialSync:          &mockSync.Sync{IsSyncing: false},
		seenAttestationCache: c,
		stateSummaryCache:    cache.NewStateSummaryCache(),
	}
	err = r.initCaches()
	require.NoError(t, err)
	p.Digest, err = r.forkDigest()
	require.NoError(t, err)
	r.registerSubscribers()
	r.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.Initialized,
		Data: &statefeed.InitializedData{
			StartTime: time.Now(),
		},
	})

	att := &eth.Attestation{
		Data: &eth.AttestationData{
			Slot:            0,
			BeaconBlockRoot: root[:],
			Target:          &eth.Checkpoint{},
		},
		AggregationBits: bitfield.Bitlist{0b0101},
	}
	domain, err := helpers.Domain(s.Fork(), att.Data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester, s.GenesisValidatorRoot())
	require.NoError(t, err)
	attRoot, err := helpers.ComputeSigningRoot(att.Data, domain)
	require.NoError(t, err)
	att.Signature = sKeys[16].Sign(attRoot[:]).Marshal()

	p.ReceivePubSub("/eth2/%x/beacon_attestation_0", att)

	time.Sleep(time.Second * 1)

	ua := r.attPool.UnaggregatedAttestations()
	if len(ua) == 0 {
		t.Error("No attestations put into pool")
	}
}
