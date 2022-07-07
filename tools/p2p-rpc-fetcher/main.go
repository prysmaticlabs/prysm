package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	emptypb "github.com/golang/protobuf/ptypes/empty"
	"github.com/libp2p/go-libp2p"
	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	corenet "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	p2ptypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	ecdsaprysm "github.com/prysmaticlabs/prysm/crypto/ecdsa"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/monitoring/tracing"
	"github.com/prysmaticlabs/prysm/network"
	"github.com/prysmaticlabs/prysm/network/forks"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/metadata"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/time/slots"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
)

var (
	peerMultiaddr     = "/ip4/10.0.0.74/tcp/13000/p2p/16Uiu2HAmB8PyNBA3bdScbCPiHbjep6w64T2DuTffcdRxPLqpY43r"
	beaconAPIEndpoint = "localhost:4000"
)

type service struct {
	h            host.Host
	meta         metadata.Metadata
	beaconClient pb.BeaconChainClient
	nodeClient   pb.NodeClient
}

func (s *service) Encoding() encoder.NetworkEncoding {
	return &encoder.SszNetworkEncoder{}
}

func (s *service) MetadataSeq() uint64 {
	return s.meta.SequenceNumber()
}

func main() {
	ipAdd := ipAddr()
	priv, err := privKey()
	if err != nil {
		panic(err)
	}
	meta, err := readMetadata()
	if err != nil {
		panic(err)
	}
	listen, err := multiAddressBuilder(ipAdd.String(), 13001)
	if err != nil {
		panic(err)
	}
	options := []libp2p.Option{
		privKeyOption(priv),
		libp2p.ListenAddrs(listen),
		libp2p.UserAgent(version.BuildData()),
		libp2p.Transport(tcp.NewTCPTransport),
	}
	options = append(options, libp2p.Security(noise.ID, noise.New))
	options = append(options, libp2p.Ping(false))
	host, err := libp2p.New(options...)
	if err != nil {
		panic(err)
	}

	host.RemoveStreamHandler(identify.IDDelta)
	conn, err := grpc.Dial(beaconAPIEndpoint, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	beaconClient := pb.NewBeaconChainClient(conn)
	nodeClient := pb.NewNodeClient(conn)
	srv := &service{h: host, beaconClient: beaconClient, nodeClient: nodeClient, meta: meta}

	srv.registerRPC(p2p.RPCPingTopicV1, srv.pingHandler)
	srv.registerRPC(p2p.RPCStatusTopicV1, srv.statusRPCHandler)

	srv.registerRPC(p2p.RPCBlocksByRangeTopicV1, func(ctx context.Context, i interface{}, stream libp2pcore.Stream) error {
		fmt.Println("Handling blocks by range")
		return nil
	})

	peers, err := peersFromStringAddrs([]string{peerMultiaddr})
	if err != nil {
		panic(err)
	}

	// Connect with all peers.
	addrInfos, err := peer.AddrInfosFromP2pAddrs(peers...)
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*20)
	defer cancel()
	for _, info := range addrInfos {
		if info.ID == host.ID() {
			continue
		}
		if err := host.Connect(ctx, info); err != nil {
			panic(err)
		}
	}
	fmt.Println("CONNECTED: ", host.Peerstore().Peers())
	genesisResp, err := nodeClient.GetGenesis(ctx, &emptypb.Empty{})
	if err != nil {
		panic(err)
	}
	currEpoch := slots.ToEpoch(slots.SinceGenesis(genesisResp.GenesisTime.AsTime()))
	currFork, err := forks.Fork(currEpoch)
	if err != nil {
		panic(err)
	}
	chain := &mock.ChainService{
		Genesis:        genesisResp.GenesisTime.AsTime(),
		Fork:           currFork,
		ValidatorsRoot: bytesutil.ToBytes32(genesisResp.GenesisValidatorsRoot),
	}

	for _, pr := range host.Peerstore().Peers() {
		req := &pb.BeaconBlocksByRangeRequest{
			StartSlot: 0,
			Count:     10,
			Step:      1,
		}
		blocks, err := SendBeaconBlocksByRangeRequest(ctx, chain, srv, pr, req)
		if err != nil {
			fmt.Println("GOT ERR IN SEND REQ", err)
		}
		fmt.Println(blocks)
	}
	time.Sleep(time.Minute * 10)
	if err := host.Close(); err != nil {
		panic(err)
	}
}

// Determines a private key for p2p networking from the p2p service's
// configuration struct. If no key is found, it generates a new one.
func privKey() (*ecdsa.PrivateKey, error) {
	priv, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		return nil, err
	}
	return ecdsaprysm.ConvertFromInterfacePrivKey(priv)
}

func readMetadata() (metadata.Metadata, error) {
	metaData := &pb.MetaDataV1{
		SeqNumber: 0,
		Attnets:   bitfield.NewBitvector64(),
	}
	return wrapper.WrappedMetadataV1(metaData), nil
}

// Retrieves an external ipv4 address and converts into a libp2p formatted value.
func ipAddr() net.IP {
	ip, err := network.ExternalIP()
	if err != nil {
		panic(err)
	}
	return net.ParseIP(ip)
}

func multiAddressBuilder(ipAddr string, port uint) (ma.Multiaddr, error) {
	parsedIP := net.ParseIP(ipAddr)
	if parsedIP.To4() == nil && parsedIP.To16() == nil {
		return nil, errors.Errorf("invalid ip address provided: %s", ipAddr)
	}
	if parsedIP.To4() != nil {
		return ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr, port))
	}
	return ma.NewMultiaddr(fmt.Sprintf("/ip6/%s/tcp/%d", ipAddr, port))
}

func multiAddressBuilderWithID(ipAddr, protocol string, port uint, id peer.ID) (ma.Multiaddr, error) {
	parsedIP := net.ParseIP(ipAddr)
	if parsedIP.To4() == nil && parsedIP.To16() == nil {
		return nil, errors.Errorf("invalid ip address provided: %s", ipAddr)
	}
	if id.String() == "" {
		return nil, errors.New("empty peer id given")
	}
	if parsedIP.To4() != nil {
		return ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/%s/%d/p2p/%s", ipAddr, protocol, port, id.String()))
	}
	return ma.NewMultiaddr(fmt.Sprintf("/ip6/%s/%s/%d/p2p/%s", ipAddr, protocol, port, id.String()))
}

// Adds a private key to the libp2p option if the option was provided.
// If the private key file is missing or cannot be read, or if the
// private key contents cannot be marshaled, an exception is thrown.
func privKeyOption(privkey *ecdsa.PrivateKey) libp2p.Option {
	return func(cfg *libp2p.Config) error {
		ifaceKey, err := ecdsaprysm.ConvertToInterfacePrivkey(privkey)
		if err != nil {
			return err
		}
		log.Debug("ECDSA private key generated")
		return cfg.Apply(libp2p.Identity(ifaceKey))
	}
}

func peersFromStringAddrs(addrs []string) ([]ma.Multiaddr, error) {
	var allAddrs []ma.Multiaddr
	enodeString, multiAddrString := parseGenericAddrs(addrs)
	for _, stringAddr := range multiAddrString {
		addr, err := multiAddrFromString(stringAddr)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not get multiaddr from string")
		}
		allAddrs = append(allAddrs, addr)
	}
	for _, stringAddr := range enodeString {
		enodeAddr, err := enode.Parse(enode.ValidSchemes, stringAddr)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not get enode from string")
		}
		addr, err := convertToSingleMultiAddr(enodeAddr)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not get multiaddr")
		}
		allAddrs = append(allAddrs, addr)
	}
	return allAddrs, nil
}

func parseGenericAddrs(addrs []string) (enodeString, multiAddrString []string) {
	for _, addr := range addrs {
		if addr == "" {
			// Ignore empty entries
			continue
		}
		_, err := enode.Parse(enode.ValidSchemes, addr)
		if err == nil {
			enodeString = append(enodeString, addr)
			continue
		}
		_, err = multiAddrFromString(addr)
		if err == nil {
			multiAddrString = append(multiAddrString, addr)
			continue
		}
		log.Errorf("Invalid address of %s provided: %v", addr, err)
	}
	return enodeString, multiAddrString
}

func convertToSingleMultiAddr(node *enode.Node) (ma.Multiaddr, error) {
	pubkey := node.Pubkey()
	assertedKey, err := ecdsaprysm.ConvertToInterfacePubkey(pubkey)
	if err != nil {
		return nil, errors.Wrap(err, "could not get pubkey")
	}
	id, err := peer.IDFromPublicKey(assertedKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not get peer id")
	}
	return multiAddressBuilderWithID(node.IP().String(), "tcp", uint(node.TCP()), id)
}

func multiAddrFromString(address string) (ma.Multiaddr, error) {
	return ma.NewMultiaddr(address)
}

// SendBeaconBlocksByRangeRequest sends BeaconBlocksByRange and returns fetched blocks, if any.
func SendBeaconBlocksByRangeRequest(
	ctx context.Context, chain blockchain.ChainInfoFetcher, p2pProvider *service, pid peer.ID,
	req *pb.BeaconBlocksByRangeRequest,
) ([]interfaces.SignedBeaconBlock, error) {
	sinceGenesis := slots.SinceGenesis(chain.GenesisTime())
	topic, err := p2p.TopicFromMessage(p2p.BeaconBlocksByRangeMessageName, slots.ToEpoch(sinceGenesis))
	if err != nil {
		return nil, errors.Wrap(err, "topic cannot find")
	}
	stream, err := p2pProvider.Send(ctx, req, topic, pid)
	if err != nil {
		return nil, errors.Wrap(err, "cannot send")
	}
	defer closeStream(stream)

	// Augment block processing function, if non-nil block processor is provided.
	blocks := make([]interfaces.SignedBeaconBlock, 0, req.Count)
	process := func(blk interfaces.SignedBeaconBlock) error {
		blocks = append(blocks, blk)
		return nil
	}
	var prevSlot types.Slot
	for i := uint64(0); ; i++ {
		isFirstChunk := i == 0
		blk, err := ReadChunkedBlock(stream, chain, p2pProvider, isFirstChunk)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		// The response MUST contain no more than `count` blocks, and no more than
		// MAX_REQUEST_BLOCKS blocks.
		if i >= req.Count || i >= params.BeaconNetworkConfig().MaxRequestBlocks {
			return nil, errors.New("invalid data")
		}
		// Returned blocks MUST be in the slot range [start_slot, start_slot + count * step).
		if blk.Block().Slot() < req.StartSlot || blk.Block().Slot() >= req.StartSlot.Add(req.Count*req.Step) {
			return nil, errors.New("invalid data")
		}
		// Returned blocks, where they exist, MUST be sent in a consecutive order.
		// Consecutive blocks MUST have values in `step` increments (slots may be skipped in between).
		isSlotOutOfOrder := false
		if prevSlot >= blk.Block().Slot() {
			isSlotOutOfOrder = true
		} else if req.Step != 0 && blk.Block().Slot().SubSlot(prevSlot).Mod(req.Step) != 0 {
			isSlotOutOfOrder = true
		}
		if !isFirstChunk && isSlotOutOfOrder {
			return nil, errors.New("invalid data")
		}
		prevSlot = blk.Block().Slot()
		if err := process(blk); err != nil {
			return nil, err
		}
	}
	return blocks, nil
}

func closeStream(stream corenet.Stream) {
	if err := stream.Close(); err != nil {
		log.Println(err)
	}
}

// Send a message to a specific peer. The returned stream may be used for reading, but has been
// closed for writing.
//
// When done, the caller must Close or Reset on the stream.
func (s *service) Send(ctx context.Context, message interface{}, baseTopic string, pid peer.ID) (corenet.Stream, error) {
	ctx, span := trace.StartSpan(ctx, "p2p.Send")
	defer span.End()
	topic := baseTopic + s.Encoding().ProtocolSuffix()
	span.AddAttributes(trace.StringAttribute("topic", topic))

	// Apply max dial timeout when opening a new stream.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	stream, err := s.h.NewStream(ctx, pid, protocol.ID(topic))
	if err != nil {
		tracing.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not open new stream")
	}
	// do not encode anything if we are sending a metadata request
	if baseTopic != p2p.RPCMetaDataTopicV1 && baseTopic != p2p.RPCMetaDataTopicV2 {
		castedMsg, ok := message.(ssz.Marshaler)
		if !ok {
			return nil, errors.Errorf("%T does not support the ssz marshaller interface", message)
		}
		if _, err := s.Encoding().EncodeWithMaxLength(stream, castedMsg); err != nil {
			tracing.AnnotateError(span, err)
			_err := stream.Reset()
			_ = _err
			return nil, err
		}
	}
	// Close stream for writing.
	if err := stream.CloseWrite(); err != nil {
		tracing.AnnotateError(span, err)
		_err := stream.Reset()
		_ = _err
		return nil, errors.Wrap(err, "could not close write")
	}

	return stream, nil
}

// ReadChunkedBlock handles each response chunk that is sent by the
// peer and converts it into a beacon block.
func ReadChunkedBlock(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, p2p *service, isFirstChunk bool) (interfaces.SignedBeaconBlock, error) {
	// Handle deadlines differently for first chunk
	if isFirstChunk {
		return readFirstChunkedBlock(stream, chain, p2p)
	}

	return readResponseChunk(stream, chain, p2p)
}

// readFirstChunkedBlock reads the first chunked block and applies the appropriate deadlines to
// it.
func readFirstChunkedBlock(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, p2p *service) (interfaces.SignedBeaconBlock, error) {
	code, errMsg, err := sync.ReadStatusCode(stream, p2p.Encoding())
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, errors.New(errMsg)
	}
	rpcCtx, err := readContextFromStream(stream, chain)
	if err != nil {
		return nil, err
	}
	blk, err := extractBlockDataType(rpcCtx, chain)
	if err != nil {
		return nil, err
	}
	err = p2p.Encoding().DecodeWithMaxLength(stream, blk)
	return blk, err
}

// readResponseChunk reads the response from the stream and decodes it into the
// provided message type.
func readResponseChunk(stream libp2pcore.Stream, chain blockchain.ChainInfoFetcher, p2p *service) (interfaces.SignedBeaconBlock, error) {
	code, errMsg, err := readStatusCodeNoDeadline(stream, p2p.Encoding())
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, errors.New(errMsg)
	}
	// No-op for now with the rpc context.
	rpcCtx, err := readContextFromStream(stream, chain)
	if err != nil {
		return nil, err
	}
	blk, err := extractBlockDataType(rpcCtx, chain)
	if err != nil {
		return nil, err
	}
	err = p2p.Encoding().DecodeWithMaxLength(stream, blk)
	return blk, err
}

func extractBlockDataType(digest []byte, chain blockchain.ChainInfoFetcher) (interfaces.SignedBeaconBlock, error) {
	if len(digest) == 0 {
		bFunc, ok := p2ptypes.BlockMap[bytesutil.ToBytes4(params.BeaconConfig().GenesisForkVersion)]
		if !ok {
			return nil, errors.New("no block type exists for the genesis fork version.")
		}
		return bFunc()
	}
	if len(digest) != 4 {
		return nil, errors.Errorf("invalid digest returned, wanted a length of %d but received %d", 4, len(digest))
	}
	vRoot := chain.GenesisValidatorsRoot()
	for k, blkFunc := range p2ptypes.BlockMap {
		rDigest, err := signing.ComputeForkDigest(k[:], vRoot[:])
		if err != nil {
			return nil, err
		}
		if rDigest == bytesutil.ToBytes4(digest) {
			return blkFunc()
		}
	}
	return nil, errors.New("no valid digest matched")
}

// reads any attached context-bytes to the payload.
func readContextFromStream(stream corenet.Stream, chain blockchain.ChainInfoFetcher) ([]byte, error) {
	rpcCtx, err := rpcContext(stream, chain)
	if err != nil {
		return nil, err
	}
	if len(rpcCtx) == 0 {
		return []byte{}, nil
	}
	// Read context (fork-digest) from stream
	b := make([]byte, 4)
	if _, err := stream.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

// retrieve expected context depending on rpc topic schema version.
func rpcContext(stream corenet.Stream, chain blockchain.ChainInfoFetcher) ([]byte, error) {
	_, _, version, err := p2p.TopicDeconstructor(string(stream.Protocol()))
	if err != nil {
		return nil, err
	}
	switch version {
	case p2p.SchemaVersionV1:
		// Return empty context for a v1 method.
		return []byte{}, nil
	case p2p.SchemaVersionV2:
		currFork := chain.CurrentFork()
		genRoot := chain.GenesisValidatorsRoot()
		digest, err := signing.ComputeForkDigest(currFork.CurrentVersion, genRoot[:])
		if err != nil {
			return nil, err
		}
		return digest[:], nil
	default:
		return nil, errors.New("invalid version of %s registered for topic: %s")
	}
}

var responseCodeSuccess = byte(0x00)

// reads data from the stream without applying any timeouts.
func readStatusCodeNoDeadline(stream corenet.Stream, encoding encoder.NetworkEncoding) (uint8, string, error) {
	b := make([]byte, 1)
	_, err := stream.Read(b)
	if err != nil {
		return 0, "", err
	}

	if b[0] == responseCodeSuccess {
		return 0, "", nil
	}

	msg := &p2ptypes.ErrorMessage{}
	if err := encoding.DecodeWithMaxLength(stream, msg); err != nil {
		return 0, "", err
	}

	return b[0], string(*msg), nil
}

type rpcHandler func(context.Context, interface{}, libp2pcore.Stream) error

// registerRPC for a given topic with an expected protobuf message type.
func (s *service) registerRPC(baseTopic string, handle rpcHandler) {
	topic := baseTopic + s.Encoding().ProtocolSuffix()
	s.h.SetStreamHandler(protocol.ID(topic), func(stream corenet.Stream) {
		defer func() {
			if r := recover(); r != nil {
				log.WithField("error", r).Error("Panic occurred")
				log.Errorf("%s", debug.Stack())
			}
		}()
		// Resetting after closing is a no-op so defer a reset in case something goes wrong.
		// It's up to the handler to Close the stream (send an EOF) if
		// it successfully writes a response. We don't blindly call
		// Close here because we may have only written a partial
		// response.
		defer func() {
			_err := stream.Reset()
			_ = _err
		}()

		log.WithField("peer", stream.Conn().RemotePeer().Pretty()).WithField("topic", string(stream.Protocol()))

		base, ok := p2p.RPCTopicMappings[baseTopic]
		if !ok {
			log.Errorf("Could not retrieve base message for topic %s", baseTopic)
			return
		}
		t := reflect.TypeOf(base)
		// Copy Base
		base = reflect.New(t)

		// since metadata requests do not have any data in the payload, we
		// do not decode anything.
		if baseTopic == p2p.RPCMetaDataTopicV1 || baseTopic == p2p.RPCMetaDataTopicV2 {
			if err := handle(context.Background(), base, stream); err != nil {
				if err != p2ptypes.ErrWrongForkDigestVersion {
					log.WithError(err).Debug("Could not handle p2p RPC")
				}
			}
			return
		}

		// Given we have an input argument that can be pointer or the actual object, this gives us
		// a way to check for its reflect.Kind and based on the result, we can decode
		// accordingly.
		if t.Kind() == reflect.Ptr {
			msg, ok := reflect.New(t.Elem()).Interface().(ssz.Unmarshaler)
			if !ok {
				log.Errorf("message of %T does not support marshaller interface", msg)
				return
			}
			if err := s.Encoding().DecodeWithMaxLength(stream, msg); err != nil {
				// Debug logs for goodbye/status errors
				if strings.Contains(topic, p2p.RPCGoodByeTopicV1) || strings.Contains(topic, p2p.RPCStatusTopicV1) {
					log.WithError(err).Debug("Could not decode goodbye stream message")
					return
				}
				log.WithError(err).Debug("Could not decode stream message")
				return
			}
			if err := handle(context.Background(), msg, stream); err != nil {
				if err != p2ptypes.ErrWrongForkDigestVersion {
					log.WithError(err).Debug("Could not handle p2p RPC")
				}
			}
		} else {
			nTyp := reflect.New(t)
			msg, ok := nTyp.Interface().(ssz.Unmarshaler)
			if !ok {
				log.Errorf("message of %T does not support marshaller interface", msg)
				return
			}
			if err := s.Encoding().DecodeWithMaxLength(stream, msg); err != nil {
				log.WithError(err).Debug("Could not decode stream message")
				return
			}
			if err := handle(context.Background(), nTyp.Elem().Interface(), stream); err != nil {
				if err != p2ptypes.ErrWrongForkDigestVersion {
					log.WithError(err).Debug("Could not handle p2p RPC")
				}
			}
		}
	})
}

// pingHandler reads the incoming ping rpc message from the peer.
func (s *service) pingHandler(_ context.Context, msg interface{}, stream libp2pcore.Stream) error {
	fmt.Println("RESPONDING WITH PING ITEM")
	//m, ok := msg.(*types.SSZUint64)
	//if !ok {
	//	return fmt.Errorf("wrong message type for ping, got %T, wanted *uint64", msg)
	//}
	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		return err
	}
	sq := types.SSZUint64(s.MetadataSeq())
	if _, err := s.Encoding().EncodeWithMaxLength(stream, &sq); err != nil {
		return err
	}
	closeStream(stream)
	return nil
}

// statusRPCHandler reads the incoming Status RPC from the peer and responds with our version of a status message.
// This handler will disconnect any peer that does not match our fork version.
func (s *service) statusRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	fmt.Println("RESPONDING WITH STATUS ITEM")
	//m, ok := msg.(*pb.Status)
	//if !ok {
	//	return errors.New("message is not type *pb.Status")
	//}
	if err := s.respondWithStatus(ctx, stream); err != nil {
		return err
	}
	closeStream(stream)
	return nil
}

func (s *service) respondWithStatus(ctx context.Context, stream corenet.Stream) error {
	chainHead, err := s.beaconClient.GetChainHead(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	resp, err := s.nodeClient.GetGenesis(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	digest, err := forks.CreateForkDigest(resp.GenesisTime.AsTime(), resp.GenesisValidatorsRoot)
	if err != nil {
		return err
	}
	status := &pb.Status{
		ForkDigest:     digest[:],
		FinalizedRoot:  chainHead.FinalizedBlockRoot,
		FinalizedEpoch: chainHead.FinalizedEpoch,
		HeadRoot:       chainHead.HeadBlockRoot,
		HeadSlot:       chainHead.HeadSlot,
	}

	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		log.WithError(err).Debug("Could not write to stream")
	}
	_, err = s.Encoding().EncodeWithMaxLength(stream, status)
	return err
}
