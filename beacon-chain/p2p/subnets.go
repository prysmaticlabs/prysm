package p2p

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/interfaces"
	"go.opencensus.io/trace"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var attestationSubnetCount = params.BeaconNetworkConfig().AttestationSubnetCount

var attSubnetEnrKey = params.BeaconNetworkConfig().AttSubnetKey

// FindPeersWithSubnet performs a network search for peers
// subscribed to a particular subnet. Then we try to connect
// with those peers. This method will block until the required amount of
// peers are found, the method only exits in the event of context timeouts.
func (s *Service) FindPeersWithSubnet(ctx context.Context, topic string,
	index, threshold uint64) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "p2p.FindPeersWithSubnet")
	defer span.End()

	span.AddAttributes(trace.Int64Attribute("index", int64(index)))

	if s.dv5Listener == nil {
		// return if discovery isn't set
		return false, nil
	}

	topic += s.Encoding().ProtocolSuffix()
	iterator := s.dv5Listener.RandomNodes()
	iterator = filterNodes(ctx, iterator, s.filterPeerForSubnet(index))

	currNum := uint64(len(s.pubsub.ListPeers(topic)))
	wg := new(sync.WaitGroup)
	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if currNum >= threshold {
			break
		}
		nodes := enode.ReadNodes(iterator, int(params.BeaconNetworkConfig().MinimumPeersInSubnetSearch))
		for _, node := range nodes {
			info, _, err := convertToAddrInfo(node)
			if err != nil {
				continue
			}
			wg.Add(1)
			go func() {
				if err := s.connectWithPeer(ctx, *info); err != nil {
					log.WithError(err).Tracef("Could not connect with peer %s", info.String())
				}
				wg.Done()
			}()
		}
		// Wait for all dials to be completed.
		wg.Wait()
		currNum = uint64(len(s.pubsub.ListPeers(topic)))
	}
	return true, nil
}

// returns a method with filters peers specifically for a particular attestation subnet.
func (s *Service) filterPeerForSubnet(index uint64) func(node *enode.Node) bool {
	return func(node *enode.Node) bool {
		if !s.filterPeer(node) {
			return false
		}
		subnets, err := attSubnets(node.Record())
		if err != nil {
			return false
		}
		indExists := false
		for _, comIdx := range subnets {
			if comIdx == index {
				indExists = true
				break
			}
		}
		return indExists
	}
}

// lower threshold to broadcast object compared to searching
// for a subnet. So that even in the event of poor peer
// connectivity, we can still broadcast an attestation.
func (s *Service) hasPeerWithSubnet(topic string) bool {
	return len(s.pubsub.ListPeers(topic+s.Encoding().ProtocolSuffix())) >= 1
}

// Updates the service's discv5 listener record's attestation subnet
// with a new value for a bitfield of subnets tracked. It also updates
// the node's metadata by increasing the sequence number and the
// subnets tracked by the node.
func (s *Service) updateSubnetRecordWithMetadata(bitV bitfield.Bitvector64) {
	entry := enr.WithEntry(attSubnetEnrKey, &bitV)
	s.dv5Listener.LocalNode().Set(entry)
	s.metaData = interfaces.WrappedMetadataV0(&pb.MetaDataV0{
		SeqNumber: s.metaData.SequenceNumber() + 1,
		Attnets:   bitV,
	})
}

// Initializes a bitvector of attestation subnets beacon nodes is subscribed to
// and creates a new ENR entry with its default value.
func intializeAttSubnets(node *enode.LocalNode) *enode.LocalNode {
	bitV := bitfield.NewBitvector64()
	entry := enr.WithEntry(attSubnetEnrKey, bitV.Bytes())
	node.Set(entry)
	return node
}

// Reads the attestation subnets entry from a node's ENR and determines
// the committee indices of the attestation subnets the node is subscribed to.
func attSubnets(record *enr.Record) ([]uint64, error) {
	bitV, err := bitvector(record)
	if err != nil {
		return nil, err
	}
	var committeeIdxs []uint64
	for i := uint64(0); i < attestationSubnetCount; i++ {
		if bitV.BitAt(i) {
			committeeIdxs = append(committeeIdxs, i)
		}
	}
	return committeeIdxs, nil
}

// Parses the attestation subnets ENR entry in a node and extracts its value
// as a bitvector for further manipulation.
func bitvector(record *enr.Record) (bitfield.Bitvector64, error) {
	bitV := bitfield.NewBitvector64()
	entry := enr.WithEntry(attSubnetEnrKey, &bitV)
	err := record.Load(entry)
	if err != nil {
		return nil, err
	}
	return bitV, nil
}

func (s *Service) subnetLocker(i uint64) *sync.RWMutex {
	s.subnetsLockLock.Lock()
	defer s.subnetsLockLock.Unlock()
	l, ok := s.subnetsLock[i]
	if !ok {
		l = &sync.RWMutex{}
		s.subnetsLock[i] = l
	}
	return l
}
