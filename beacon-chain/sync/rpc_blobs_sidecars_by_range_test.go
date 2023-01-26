package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	chainMock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db/iface"
	db "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	leakybucket "github.com/prysmaticlabs/prysm/v3/container/leaky-bucket"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

func TestRPCBBlobsSidecarsByRange_RPCHandlerRateLimit_NoOverflow(t *testing.T) {
	d := db.SetupDB(t)

	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")

	capacity := int64(params.BeaconNetworkConfig().MaxRequestBlobsSidecars)
	r := &Service{cfg: &config{p2p: p1, beaconDB: d, chain: &chainMock.ChainService{}}, rateLimiter: newRateLimiter(p1)}

	pcl := protocol.ID(p2p.RPCBlocksByRangeTopicV1)
	topic := string(pcl)
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(0.000001, capacity, time.Second, false)
	req := &ethpb.BlobsSidecarsByRangeRequest{
		Count: uint64(capacity),
	}

	saveBlobs(d, req, 1)
	assert.NoError(t, sendBlobsRequest(t, p1, p2, r, req, true, true, 1))

	remainingCapacity := r.rateLimiter.limiterMap[topic].Remaining(p2.PeerID().String())
	expectedCapacity := int64(0) // Whole capacity is used, but no overflow.
	assert.Equal(t, expectedCapacity, remainingCapacity, "Unexpected rate limiting capacity")
}

func TestRPCBBlobsSidecarsByRange_RPCHandlerRateLimit_ThrottleWithoutOverflowAndMultipleBlobsPerSlot(t *testing.T) {
	d := db.SetupDB(t)
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")

	capacity := int64(params.BeaconNetworkConfig().MaxRequestBlobsSidecars)
	r := &Service{cfg: &config{p2p: p1, beaconDB: d, chain: &chainMock.ChainService{}}, rateLimiter: newRateLimiter(p1)}

	pcl := protocol.ID(p2p.RPCBlocksByRangeTopicV1)
	topic := string(pcl)
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(0.000001, capacity, time.Second, false)
	req := &ethpb.BlobsSidecarsByRangeRequest{
		Count: uint64(capacity) / 2,
	}

	saveBlobs(d, req, 2)
	assert.NoError(t, sendBlobsRequest(t, p1, p2, r, req, true, true, 2))

	remainingCapacity := r.rateLimiter.limiterMap[topic].Remaining(p2.PeerID().String())
	expectedCapacity := int64(0) // Whole capacity is used, but no overflow.
	assert.Equal(t, expectedCapacity, remainingCapacity, "Unexpected rate limiting capacity")
}

func TestRPCBBlobsSidecarsByRange_RPCHandlerRateLimit_ThrottleWithOverflow(t *testing.T) {
	d := db.SetupDB(t)
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")

	capacity := int64(params.BeaconNetworkConfig().MaxRequestBlobsSidecars) / 2
	r := &Service{cfg: &config{p2p: p1, beaconDB: d, chain: &chainMock.ChainService{}}, rateLimiter: newRateLimiter(p1)}

	pcl := protocol.ID(p2p.RPCBlocksByRangeTopicV1)
	topic := string(pcl)
	// 30 blobs per second streams blobs in ~4s
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(30, capacity, time.Second, false)
	req := &ethpb.BlobsSidecarsByRangeRequest{
		Count: uint64(capacity) * 2,
	}

	saveBlobs(d, req, 1)
	assert.NoError(t, sendBlobsRequest(t, p1, p2, r, req, true, true, 1))

	remainingCapacity := r.rateLimiter.limiterMap[topic].Remaining(p2.PeerID().String())
	expectedCapacity := int64(0)
	assert.NotEqual(t, expectedCapacity, remainingCapacity, "Unexpected rate limiting capacity")
}

func TestRPCBBlobsSidecarsByRange_RPCHandlerRateLimit_MultipleRequestsThrottle(t *testing.T) {
	d := db.SetupDB(t)
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")

	capacity := int64(params.BeaconNetworkConfig().MaxRequestBlobsSidecars) / 2
	r := &Service{cfg: &config{p2p: p1, beaconDB: d, chain: &chainMock.ChainService{}}, rateLimiter: newRateLimiter(p1)}

	pcl := protocol.ID(p2p.RPCBlocksByRangeTopicV1)
	topic := string(pcl)
	// 64 blobs per second - takes ~6 seconds to stream all blobs
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(64, capacity, time.Second, false)

	for i := 0; i < 2; i++ {
		req := &ethpb.BlobsSidecarsByRangeRequest{
			Count: uint64(capacity) * 2,
		}
		saveBlobs(d, req, 1)
		assert.NoError(t, sendBlobsRequest(t, p1, p2, r, req, true, true, 1))
	}
}

func saveBlobs(d iface.Database, req *ethpb.BlobsSidecarsByRangeRequest, blobsPerSlot uint64) {
	for i := req.StartSlot; i < req.StartSlot.Add(req.Count); i += types.Slot(1) {
		for j := uint64(0); j < blobsPerSlot; j++ {
			blob := &ethpb.BlobsSidecar{
				BeaconBlockSlot: i,
				BeaconBlockRoot: make([]byte, fieldparams.RootLength),
				AggregatedProof: make([]byte, 48),
			}
			// always non-empty to ensure that blobs are always sent
			blob.BeaconBlockRoot[0] = 0x1
			blob.BeaconBlockRoot[1] = byte(j)
			d.SaveBlobsSidecar(context.Background(), blob)
		}
	}
}

func sendBlobsRequest(t *testing.T, p1, p2 *p2ptest.TestP2P, r *Service,
	req *ethpb.BlobsSidecarsByRangeRequest, validateBlocks bool, success bool, blobsPerSlot uint64) error {
	var wg sync.WaitGroup
	wg.Add(1)
	pcl := protocol.ID(p2p.RPCBlocksByRangeTopicV1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		if !validateBlocks {
			return
		}
		for i := req.StartSlot; i < req.StartSlot.Add(req.Count); i += types.Slot(1) {
			if !success {
				continue
			}

			for j := uint64(0); j < blobsPerSlot; j++ {
				SetRPCStreamDeadlines(stream)

				expectSuccess(t, stream)
				res := new(ethpb.BlobsSidecar)
				assert.NoError(t, r.cfg.p2p.Encoding().DecodeWithMaxLength(stream, res))
				if slots.AbsoluteValueSlotDifference(res.BeaconBlockSlot, i) != 0 {
					t.Errorf("Received unexpected blob slot %d. expected %d", res.BeaconBlockSlot, i)
				}
			}
		}
	})
	stream, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
	require.NoError(t, err)
	if err := r.blobsSidecarsByRangeRPCHandler(context.Background(), req, stream); err != nil {
		return err
	}
	if util.WaitTimeout(&wg, 20*time.Second) {
		t.Fatal("Did not receive stream within 10 sec")
	}
	return nil
}
