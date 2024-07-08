package sync

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	GoKZG "github.com/crate-crypto/go-kzg-4844"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	swarmt "github.com/libp2p/go-libp2p/p2p/net/swarm/testing"
	mock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/peerdas"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/peers"
	p2ptest "github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/testing"
	p2pTypes "github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/types"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
	"github.com/sirupsen/logrus"
)

func TestRandomizeColumns(t *testing.T) {
	const count uint64 = 128

	// Generate columns.
	columns := make(map[uint64]bool, count)
	for i := uint64(0); i < count; i++ {
		columns[i] = true
	}

	// Randomize columns.
	randomizedColumns := randomizeColumns(columns)

	// Convert back to a map.
	randomizedColumnsMap := make(map[uint64]bool, count)
	for _, column := range randomizedColumns {
		randomizedColumnsMap[column] = true
	}

	// Check duplicates and missing columns.
	require.Equal(t, len(columns), len(randomizedColumnsMap))

	// Check the values.
	for column := range randomizedColumnsMap {
		require.Equal(t, true, column < count)
	}
}

// createAndConnectPeer creates a peer with a private key `offset` fixed.
// The peer is added and connected to `p2pService`
func createAndConnectPeer(
	t *testing.T,
	p2pService *p2ptest.TestP2P,
	chainService *mock.ChainService,
	dataColumnSidecars []*ethpb.DataColumnSidecar,
	custodyCount uint64,
	columnsNotToRespond map[uint64]bool,
	offset int,
) *p2ptest.TestP2P {
	emptyRoot := [fieldparams.RootLength]byte{}
	emptySignature := [fieldparams.BLSSignatureLength]byte{}
	emptyKzgCommitmentInclusionProof := [4][]byte{
		emptyRoot[:], emptyRoot[:], emptyRoot[:], emptyRoot[:],
	}

	// Create the private key, depending on the offset.
	privateKeyBytes := make([]byte, 32)
	for i := 0; i < 32; i++ {
		privateKeyBytes[i] = byte(offset + i)
	}

	privateKey, err := crypto.UnmarshalSecp256k1PrivateKey(privateKeyBytes)
	require.NoError(t, err)

	// Create the peer.
	peer := p2ptest.NewTestP2P(t, swarmt.OptPeerPrivateKey(privateKey))

	// TODO: Do not hardcode the topic.
	peer.SetStreamHandler("/eth2/beacon_chain/req/data_column_sidecars_by_root/1/ssz_snappy", func(stream network.Stream) {
		// Decode the request.
		req := new(p2pTypes.DataColumnSidecarsByRootReq)
		err := peer.Encoding().DecodeWithMaxLength(stream, req)
		require.NoError(t, err)

		for _, identifier := range *req {
			// Filter out the columns not to respond.
			if columnsNotToRespond[identifier.ColumnIndex] {
				continue
			}

			// Create the response.
			resp := dataColumnSidecars[identifier.ColumnIndex]

			// Send the response.
			err := WriteDataColumnSidecarChunk(stream, chainService, p2pService.Encoding(), resp)
			require.NoError(t, err)
		}

		// Close the stream.
		closeStream(stream, log)
	})

	// Create the record and set the custody count.
	enr := &enr.Record{}
	enr.Set(peerdas.Csc(custodyCount))

	// Add the peer and connect it.
	p2pService.Peers().Add(enr, peer.PeerID(), nil, network.DirOutbound)
	p2pService.Peers().SetConnectionState(peer.PeerID(), peers.PeerConnected)
	p2pService.Connect(peer)

	return peer
}

func deterministicRandomness(seed int64) [32]byte {
	// Converts an int64 to a byte slice
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, seed)
	if err != nil {
		logrus.WithError(err).Error("Failed to write int64 to bytes buffer")
		return [32]byte{}
	}
	bytes := buf.Bytes()

	return sha256.Sum256(bytes)
}

// Returns a serialized random field element in big-endian
func getRandFieldElement(seed int64) [32]byte {
	bytes := deterministicRandomness(seed)
	var r fr.Element
	r.SetBytes(bytes[:])

	return GoKZG.SerializeScalar(r)
}

// Returns a random blob using the passed seed as entropy
func getRandBlob(seed int64) kzg.Blob {
	var blob kzg.Blob
	for i := 0; i < len(blob); i += 32 {
		fieldElementBytes := getRandFieldElement(seed + int64(i))
		copy(blob[i:i+32], fieldElementBytes[:])
	}
	return blob
}

func generateCommitmentAndProof(blob *kzg.Blob) (*kzg.Commitment, *kzg.Proof, error) {
	commitment, err := kzg.BlobToKZGCommitment(blob)
	if err != nil {
		return nil, nil, err
	}
	proof, err := kzg.ComputeBlobKZGProof(blob, commitment)
	if err != nil {
		return nil, nil, err
	}
	return &commitment, &proof, err
}

type dataSamplerTest struct {
	ctx        context.Context
	root       [fieldparams.RootLength]byte
	header     *ethpb.BeaconBlockHeader
	headerRoot [fieldparams.RootLength]byte
	p2pSvc     *p2ptest.TestP2P
	peers      []*p2ptest.TestP2P
	ctxMap     map[[4]byte]int
	chainSvc   *mock.ChainService
}

func setupDefaultDataColumnSamplerTest(t *testing.T) (*dataSamplerTest, *dataColumnSampler1D) {
	test, sampler := setupDataColumnSamplerTest(t)
	// Custody columns: [6, 38, 70, 102]
	p1 := createAndConnectPeer(t, test.p2pSvc, test.chainSvc, test.header, 1, map[uint64]bool{}, 1)
	// Custody columns: [3, 35, 67, 99]
	p2 := createAndConnectPeer(t, test.p2pSvc, test.chainSvc, test.header, 1, map[uint64]bool{}, 2)
	// Custody columns: [12, 44, 76, 108]
	p3 := createAndConnectPeer(t, test.p2pSvc, test.chainSvc, test.header, 1, map[uint64]bool{}, 3)
	test.peers = []*p2ptest.TestP2P{p1, p2, p3}

	return test, sampler
}

func setupDataColumnSamplerTest(t *testing.T) (*dataSamplerTest, *dataColumnSampler1D) {
	const custodyRequirement uint64 = 1

	for i := int64(0); i < blobCount; i++ {
		blob := getRandBlob(int64(i))

		kzgCommitment, kzgProof, err := generateCommitmentAndProof(&blob)
		require.NoError(t, err)

		blobs[i] = blob
		kzgCommitments[i] = kzgCommitment[:]
		kzgProofs[i] = kzgProof[:]
	}

	emptyHeaderRoot, err := emptyHeader.HashTreeRoot()
	require.NoError(t, err)

	p2pSvc := p2ptest.NewTestP2P(t)
	chainSvc, clock := defaultMockChain(t)

	test := &dataSamplerTest{
		ctx:        context.Background(),
		root:       emptyRoot,
		header:     emptyHeader,
		headerRoot: emptyHeaderRoot,
		p2pSvc:     p2pSvc,
		peers:      []*p2ptest.TestP2P{},
		ctxMap:     map[[4]byte]int{{245, 165, 253, 66}: version.Deneb},
		chainSvc:   chainSvc,
	}
	sampler := newDataColumnSampler1D(p2pSvc, clock, test.ctxMap, nil)

	return test, sampler
}

func TestDataColumnSampler1D_PeerManagement(t *testing.T) {
	test, sampler := setupDefaultDataColumnSamplerTest(t)
	p1, p2, p3 := test.peers[0], test.peers[1], test.peers[2]

	sampler.refreshPeerInfo()
	require.Equal(t, params.BeaconConfig().NumberOfColumns, uint64(len(sampler.peerFromColumn)))
	require.Equal(t, 3, len(sampler.columnFromPeer))
	require.Equal(t, true, sampler.peerFromColumn[6][p1.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[38][p1.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[70][p1.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[102][p1.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[3][p2.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[35][p2.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[67][p2.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[99][p2.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[12][p3.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[44][p3.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[76][p3.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[108][p3.PeerID()])

	err := test.p2pSvc.Disconnect(p1.PeerID())
	test.p2pSvc.Peers().SetConnectionState(p1.PeerID(), peers.PeerDisconnected)
	require.NoError(t, err)

	// test peer pruning.
	sampler.refreshPeerInfo()
	require.Equal(t, params.BeaconConfig().NumberOfColumns, uint64(len(sampler.peerFromColumn)))
	require.Equal(t, 2, len(sampler.columnFromPeer))
	require.Equal(t, 0, len(sampler.columnFromPeer[p1.PeerID()]))
	require.Equal(t, false, sampler.peerFromColumn[6][p1.PeerID()])
	require.Equal(t, false, sampler.peerFromColumn[38][p1.PeerID()])
	require.Equal(t, false, sampler.peerFromColumn[70][p1.PeerID()])
	require.Equal(t, false, sampler.peerFromColumn[102][p1.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[3][p2.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[35][p2.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[67][p2.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[99][p2.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[12][p3.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[44][p3.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[76][p3.PeerID()])
	require.Equal(t, true, sampler.peerFromColumn[108][p3.PeerID()])
}

func TestDataColumnSampler1D_SampleDistribution(t *testing.T) {
	test, sampler := setupDefaultDataColumnSamplerTest(t)
	p1, p2, p3 := test.peers[0], test.peers[1], test.peers[2]

	sampler.refreshPeerInfo()
	columns := []uint64{6, 3, 12}
	dist := sampler.distributeSamplesToPeer(columns)
	require.Equal(t, 3, len(dist))
	require.Equal(t, true, dist[p1.PeerID()][6])
	require.Equal(t, true, dist[p2.PeerID()][3])
	require.Equal(t, true, dist[p3.PeerID()][12])

	columns = []uint64{6, 3, 12, 38, 35, 44}
	dist = sampler.distributeSamplesToPeer(columns)
	require.Equal(t, 3, len(dist))
	require.Equal(t, true, dist[p1.PeerID()][6])
	require.Equal(t, true, dist[p2.PeerID()][3])
	require.Equal(t, true, dist[p3.PeerID()][12])
	require.Equal(t, true, dist[p1.PeerID()][38])
	require.Equal(t, true, dist[p2.PeerID()][35])
	require.Equal(t, true, dist[p3.PeerID()][44])

	columns = []uint64{6, 38, 70}
	dist = sampler.distributeSamplesToPeer(columns)
	require.Equal(t, 1, len(dist))
	require.Equal(t, true, dist[p1.PeerID()][6])
	require.Equal(t, true, dist[p1.PeerID()][38])
	require.Equal(t, true, dist[p1.PeerID()][70])

	// missing peer for column
	columns = []uint64{11}
	dist = sampler.distributeSamplesToPeer(columns)
	require.Equal(t, 0, len(dist))
}

func TestDataColumnSampler1D_SampleDataColumns(t *testing.T) {
	test, sampler := setupDefaultDataColumnSamplerTest(t)
	sampler.refreshPeerInfo()

	// Sample all columns.
	sampleColumns := []uint64{6, 3, 12, 38, 35, 44, 70, 67, 76, 102, 99, 108}
	retrieved := sampler.sampleDataColumns(test.ctx, test.headerRoot, sampleColumns)
	require.Equal(t, 12, len(retrieved))
	for _, column := range sampleColumns {
		require.Equal(t, true, retrieved[column])
	}

	// Sample a subset of columns.
	sampleColumns = []uint64{6, 3, 12, 38, 35, 44}
	retrieved = sampler.sampleDataColumns(test.ctx, test.headerRoot, sampleColumns)
	require.Equal(t, 6, len(retrieved))
	for _, column := range sampleColumns {
		require.Equal(t, true, retrieved[column])
	}

	// Sample a subset of columns with missing columns.
	sampleColumns = []uint64{6, 3, 12, 127}
	retrieved = sampler.sampleDataColumns(test.ctx, test.headerRoot, sampleColumns)
	require.Equal(t, 3, len(retrieved))
	require.DeepEqual(t, map[uint64]bool{6: true, 3: true, 12: true}, retrieved)
}

func TestDataColumnSampler1D_IncrementalDAS(t *testing.T) {
	testCases := []struct {
		name                     string
		samplesCount             uint64
		possibleColumnsToRequest []uint64
		columnsNotToRespond      map[uint64]bool
		expectedSuccess          bool
		expectedRoundSummaries   []roundSummary
	}{
		{
			name:                     "All columns are correctly sampled in a single round",
			samplesCount:             5,
			possibleColumnsToRequest: []uint64{70, 35, 99, 6, 38, 3, 67, 102, 12, 44, 76, 108},
			columnsNotToRespond:      map[uint64]bool{},
			expectedSuccess:          true,
			expectedRoundSummaries: []roundSummary{
				{
					RequestedColumns: []uint64{70, 35, 99, 6, 38},
					MissingColumns:   map[uint64]bool{},
				},
			},
		},
		{
			name:                     "Two missing columns in the first round, ok in the second round",
			samplesCount:             5,
			possibleColumnsToRequest: []uint64{70, 35, 99, 6, 38, 3, 67, 102, 12, 44, 76, 108},
			columnsNotToRespond:      map[uint64]bool{6: true, 70: true},
			expectedSuccess:          true,
			expectedRoundSummaries: []roundSummary{
				{
					RequestedColumns: []uint64{70, 35, 99, 6, 38},
					MissingColumns:   map[uint64]bool{70: true, 6: true},
				},
				{
					RequestedColumns: []uint64{3, 67, 102, 12, 44, 76},
					MissingColumns:   map[uint64]bool{},
				},
			},
		},
		{
			name:                     "Two missing columns in the first round, one missing in the second round. Fail to sample.",
			samplesCount:             5,
			possibleColumnsToRequest: []uint64{70, 35, 99, 6, 38, 3, 67, 102, 12, 44, 76, 108},
			columnsNotToRespond:      map[uint64]bool{6: true, 70: true, 3: true},
			expectedSuccess:          false,
			expectedRoundSummaries: []roundSummary{
				{
					RequestedColumns: []uint64{70, 35, 99, 6, 38},
					MissingColumns:   map[uint64]bool{70: true, 6: true},
				},
				{
					RequestedColumns: []uint64{3, 67, 102, 12, 44, 76},
					MissingColumns:   map[uint64]bool{3: true},
				},
			},
		},
	}

	for _, tc := range testCases {
		test, sampler := setupDataColumnSamplerTest(t)
		p1 := createAndConnectPeer(t, test.p2pSvc, test.chainSvc, test.header, 1, tc.columnsNotToRespond, 1)
		p2 := createAndConnectPeer(t, test.p2pSvc, test.chainSvc, test.header, 1, tc.columnsNotToRespond, 2)
		p3 := createAndConnectPeer(t, test.p2pSvc, test.chainSvc, test.header, 1, tc.columnsNotToRespond, 3)
		test.peers = []*p2ptest.TestP2P{p1, p2, p3}

		sampler.refreshPeerInfo()

		success, summaries, err := sampler.incrementalDAS(test.ctx, test.headerRoot, tc.possibleColumnsToRequest, tc.samplesCount)
		require.NoError(t, err)
		require.Equal(t, tc.expectedSuccess, success)
		require.DeepEqual(t, tc.expectedRoundSummaries, summaries)
	}
}
