package p2p

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/go-bitfield"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var attestationSubnetCount = params.BeaconNetworkConfig().AttestationSubnetCount

var attSubnetEnrKey = params.BeaconNetworkConfig().AttSubnetKey

// FindPeersWithSubnet performs a network search for peers
// subscribed to a particular subnet. Then we try to connect
// with those peers.
func (s *Service) FindPeersWithSubnet(ctx context.Context, index uint64) (bool, error) {
	if s.dv5Listener == nil {
		// return if discovery isn't set
		return false, nil
	}
	iterator := s.dv5Listener.RandomNodes()

	// Select appropriate size for search.
	maxSize := uint64(len(s.dv5Listener.AllNodes()))
	min := int(mathutil.Min(maxSize, lookupLimit))

	nodes := enode.ReadNodes(iterator, min)
	exists := false
	for _, node := range nodes {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if node.IP() == nil {
			continue
		}
		// do not look for nodes with no tcp port set
		if err := node.Record().Load(enr.WithEntry("tcp", new(enr.TCP))); err != nil {
			if !enr.IsNotFound(err) {
				log.WithError(err).Debug("Could not retrieve tcp port")
			}
			continue
		}
		subnets, err := retrieveAttSubnets(node.Record())
		if err != nil {
			log.Debugf("could not retrieve subnets: %v", err)
			continue
		}
		for _, comIdx := range subnets {
			if comIdx == index {
				info, multiAddr, err := convertToAddrInfo(node)
				if err != nil {
					return false, err
				}
				if s.peers.IsActive(info.ID) {
					exists = true
					continue
				}
				if s.host.Network().Connectedness(info.ID) == network.Connected {
					exists = true
					continue
				}
				s.peers.Add(node.Record(), info.ID, multiAddr, network.DirUnknown)
				if err := s.connectWithPeer(ctx, *info); err != nil {
					log.WithError(err).Tracef("Could not connect with peer %s", info.String())
					continue
				}
				exists = true
			}
		}
	}
	return exists, nil
}

func (s *Service) hasPeerWithSubnet(subnet uint64) bool {
	return len(s.Peers().SubscribedToSubnet(subnet)) > 0
}

// Updates the service's discv5 listener record's attestation subnet
// with a new value for a bitfield of subnets tracked. It also updates
// the node's metadata by increasing the sequence number and the
// subnets tracked by the node.
func (s *Service) updateSubnetRecordWithMetadata(bitV bitfield.Bitvector64) {
	entry := enr.WithEntry(attSubnetEnrKey, &bitV)
	s.dv5Listener.LocalNode().Set(entry)
	s.metaData = &pb.MetaData{
		SeqNumber: s.metaData.SeqNumber + 1,
		Attnets:   bitV,
	}
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
func retrieveAttSubnets(record *enr.Record) ([]uint64, error) {
	bitV, err := retrieveBitvector(record)
	if err != nil {
		return nil, err
	}
	committeeIdxs := []uint64{}
	for i := uint64(0); i < attestationSubnetCount; i++ {
		if bitV.BitAt(i) {
			committeeIdxs = append(committeeIdxs, i)
		}
	}
	return committeeIdxs, nil
}

// Parses the attestation subnets ENR entry in a node and extracts its value
// as a bitvector for further manipulation.
func retrieveBitvector(record *enr.Record) (bitfield.Bitvector64, error) {
	bitV := bitfield.NewBitvector64()
	entry := enr.WithEntry(attSubnetEnrKey, &bitV)
	err := record.Load(entry)
	if err != nil {
		return nil, err
	}
	return bitV, nil
}
