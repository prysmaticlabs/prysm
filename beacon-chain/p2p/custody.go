package p2p

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/peerdas"
	"github.com/prysmaticlabs/prysm/v5/config/params"
)

// GetValidCustodyPeers returns a list of peers that custody a super set of the local node's custody columns.
func (s *Service) GetValidCustodyPeers(peers []peer.ID) ([]peer.ID, error) {
	// Get the total number of columns.
	numberOfColumns := params.BeaconConfig().NumberOfColumns

	localCustodySubnetCount := peerdas.CustodySubnetCount()
	localCustodyColumns, err := peerdas.CustodyColumns(s.NodeID(), localCustodySubnetCount)
	if err != nil {
		return nil, errors.Wrap(err, "custody columns for local node")
	}

	localCustodyColumnsCount := uint64(len(localCustodyColumns))

	// Find the valid peers.
	validPeers := make([]peer.ID, 0, len(peers))

loop:
	for _, pid := range peers {
		// Get the custody subnets count of the remote peer.
		remoteCustodySubnetCount := s.CustodyCountFromRemotePeer(pid)

		// Get the remote node ID from the peer ID.
		remoteNodeID, err := ConvertPeerIDToNodeID(pid)
		if err != nil {
			return nil, errors.Wrap(err, "convert peer ID to node ID")
		}

		// Get the custody columns of the remote peer.
		remoteCustodyColumns, err := peerdas.CustodyColumns(remoteNodeID, remoteCustodySubnetCount)
		if err != nil {
			return nil, errors.Wrap(err, "custody columns")
		}

		remoteCustodyColumnsCount := uint64(len(remoteCustodyColumns))

		// If the remote peer custodies less columns than the local node, skip it.
		if remoteCustodyColumnsCount < localCustodyColumnsCount {
			continue
		}

		// If the remote peers custodies all the possible columns, add it to the list.
		if remoteCustodyColumnsCount == numberOfColumns {
			copiedId := pid
			validPeers = append(validPeers, copiedId)
			continue
		}

		// Filter out invalid peers.
		for c := range localCustodyColumns {
			if !remoteCustodyColumns[c] {
				continue loop
			}
		}

		copiedId := pid

		// Add valid peer to list
		validPeers = append(validPeers, copiedId)
	}

	return validPeers, nil
}

func (s *Service) custodyCountFromRemotePeerEnr(pid peer.ID) uint64 {
	// By default, we assume the peer custodies the minimum number of subnets.
	custodyRequirement := params.BeaconConfig().CustodyRequirement

	// Retrieve the ENR of the peer.
	record, err := s.peers.ENR(pid)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"peerID":       pid,
			"defaultValue": custodyRequirement,
		}).Debug("Failed to retrieve ENR for peer, defaulting to the default value")

		return custodyRequirement
	}

	// Retrieve the custody subnets count from the ENR.
	custodyCount, err := peerdas.CustodyCountFromRecord(record)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"peerID":       pid,
			"defaultValue": custodyRequirement,
		}).Debug("Failed to retrieve custody count from ENR for peer, defaulting to the default value")

		return custodyRequirement
	}

	return custodyCount
}

// CustodyCountFromRemotePeer retrieves the custody count from a remote peer.
func (s *Service) CustodyCountFromRemotePeer(pid peer.ID) uint64 {
	// Try to get the custody count from the peer's metadata.
	metadata, err := s.peers.Metadata(pid)
	if err != nil {
		log.WithError(err).WithField("peerID", pid).Debug("Failed to retrieve metadata for peer, defaulting to the ENR value")
		return s.custodyCountFromRemotePeerEnr(pid)
	}

	// If the metadata is nil, default to the ENR value.
	if metadata == nil {
		log.WithField("peerID", pid).Debug("Metadata is nil, defaulting to the ENR value")
		return s.custodyCountFromRemotePeerEnr(pid)
	}

	// Get the custody subnets count from the metadata.
	custodyCount := metadata.CustodySubnetCount()

	// If the custody count is null, default to the ENR value.
	if custodyCount == 0 {
		log.WithField("peerID", pid).Debug("The custody count extracted from the metadata equals to 0, defaulting to the ENR value")
		return s.custodyCountFromRemotePeerEnr(pid)
	}

	return custodyCount
}
