package node

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	statetrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	mocksync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func TestNodeServer_GetNodeInfo(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()
	addr := common.Address{1, 2, 3}
	if err := db.SaveDepositContractAddress(ctx, addr); err != nil {
		t.Fatal(err)
	}
	p2p := mockp2p.NewTestP2P(t)

	finalizedBlock := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1, ParentRoot: []byte{'A'}}}
	db.SaveBlock(context.Background(), finalizedBlock)
	fRoot, _ := ssz.HashTreeRoot(finalizedBlock.Block)
	justifiedBlock := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2, ParentRoot: []byte{'B'}}}
	db.SaveBlock(context.Background(), justifiedBlock)
	jRoot, _ := ssz.HashTreeRoot(justifiedBlock.Block)
	prevJustifiedBlock := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 3, ParentRoot: []byte{'C'}}}
	db.SaveBlock(context.Background(), prevJustifiedBlock)
	pjRoot, _ := ssz.HashTreeRoot(prevJustifiedBlock.Block)

	st, err := statetrie.InitializeFromProto(&pbp2p.BeaconState{
		Slot:                        1,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Epoch: 3, Root: pjRoot[:]},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Epoch: 2, Root: jRoot[:]},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Epoch: 1, Root: fRoot[:]},
	})
	if err != nil {
		t.Fatal(err)
	}

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: st.PreviousJustifiedCheckpoint().Epoch*params.BeaconConfig().SlotsPerEpoch + 1}}
	chain := &mock.ChainService{
		Genesis:                     time.Unix(0, 0),
		Block:                       b,
		State:                       st,
		FinalizedCheckPoint:         st.FinalizedCheckpoint(),
		CurrentJustifiedCheckPoint:  st.CurrentJustifiedCheckpoint(),
		PreviousJustifiedCheckPoint: st.PreviousJustifiedCheckpoint(),
	}
	peersProvider := &mockp2p.MockPeersProvider{}
	ns := &Server{
		BeaconDB:            db,
		GenesisTimeFetcher:  chain,
		SyncChecker:         &mocksync.Sync{IsSyncing: false},
		PeersFetcher:        peersProvider,
		HostFetcher:         p2p,
		HeadFetcher:         chain,
		FinalizationFetcher: chain,
	}
	res, err := ns.GetNodeInfo(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}

	if res.NodeId != p2p.Host().ID().String() {
		t.Errorf("GetNodeInfo() wanted node ID %v, received %v", p2p.Host().ID().String(), res.NodeId)
	}

	if res.Version != version.GetVersion() {
		t.Errorf("GetNodeInfo() wanted version %v, received %v", version.GetVersion(), res.SyncState)
	}

	if len(res.Addresses) != 1 {
		t.Errorf("GetNodeInfo() expected %d addresses, received %d", 1, len(res.Addresses))
	}
	expectedAddress := fmt.Sprintf("%v/p2p/%s", p2p.Host().Addrs()[0], res.NodeId)
	if res.Addresses[0] != expectedAddress {
		t.Errorf("GetNodeInfo() wanted addres *%s* received *%s*", expectedAddress, res.Addresses[0])
	}

	if len(res.Peers) != 2 {
		t.Errorf("GetNodeInfo() expected %d peers, received %d", 2, len(res.Peers))
	}

	if res.SyncState != ethpb.SyncState_SYNC_INACTIVE {
		t.Errorf("GetNodeInfo() wanted sync state %v, received %v", ethpb.SyncState_SYNC_INACTIVE, res.SyncState)
	}

	if res.CurrentEpoch != chain.Block.Block.Slot/params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("GetNodeInfo() wanted current epoch %v, received %v", chain.Block.Block.Slot/params.BeaconConfig().SlotsPerEpoch, res.CurrentEpoch)
	}
	if res.CurrentSlot != chain.Block.Block.Slot {
		t.Errorf("GetNodeInfo() wanted current slot %v, received %v", chain.Block.Block.Slot, res.CurrentSlot)
	}
	root, _ := ssz.HashTreeRoot(chain.Block.Block)
	if !bytes.Equal(res.CurrentBlockRoot, root[:]) {
		t.Errorf("GetNodeInfo() wanted current root %x, received %x", root, res.CurrentBlockRoot)
	}

	if res.FinalizedEpoch != finalizedBlock.Block.Slot/params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("GetNodeInfo() wanted current epoch %v, received %v", finalizedBlock.Block.Slot/params.BeaconConfig().SlotsPerEpoch, res.FinalizedEpoch)
	}
	if res.FinalizedSlot != finalizedBlock.Block.Slot {
		t.Errorf("GetNodeInfo() wanted current slot %v, received %v", finalizedBlock.Block.Slot, res.FinalizedSlot)
	}
	if !bytes.Equal(res.FinalizedBlockRoot, fRoot[:]) {
		t.Errorf("GetNodeInfo() wanted current root %x, received %x", fRoot, res.FinalizedBlockRoot)
	}

	if res.JustifiedEpoch != justifiedBlock.Block.Slot/params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("GetNodeInfo() wanted current epoch %v, received %v", justifiedBlock.Block.Slot/params.BeaconConfig().SlotsPerEpoch, res.JustifiedEpoch)
	}
	if res.JustifiedSlot != justifiedBlock.Block.Slot {
		t.Errorf("GetNodeInfo() wanted current slot %v, received %v", justifiedBlock.Block.Slot, res.JustifiedSlot)
	}
	if !bytes.Equal(res.JustifiedBlockRoot, jRoot[:]) {
		t.Errorf("GetNodeInfo() wanted current root %x, received %x", jRoot, res.JustifiedBlockRoot)
	}

	if res.PreviousJustifiedEpoch != prevJustifiedBlock.Block.Slot/params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("GetNodeInfo() wanted current epoch %v, received %v", prevJustifiedBlock.Block.Slot/params.BeaconConfig().SlotsPerEpoch, res.PreviousJustifiedEpoch)
	}
	if res.PreviousJustifiedSlot != prevJustifiedBlock.Block.Slot {
		t.Errorf("GetNodeInfo() wanted current slot %v, received %v", prevJustifiedBlock.Block.Slot, res.PreviousJustifiedSlot)
	}
	if !bytes.Equal(res.PreviousJustifiedBlockRoot, pjRoot[:]) {
		t.Errorf("GetNodeInfo() wanted current root %x, received %x", pjRoot, res.PreviousJustifiedBlockRoot)
	}
}

func TestNodeServer_GetImplementedServices(t *testing.T) {
	server := grpc.NewServer()
	ns := &Server{
		Server: server,
	}
	ethpb.RegisterNodeServer(server, ns)
	reflection.Register(server)

	res, err := ns.ListImplementedServices(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	// We verify the services include the node service + the registered reflection service.
	if len(res.Services) != 2 {
		t.Errorf("Expected 2 services, received %d: %v", len(res.Services), res.Services)
	}
}
