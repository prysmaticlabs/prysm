package builder

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/api/client/builder"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/network"
	v1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	log "github.com/sirupsen/logrus"
)

type BlockBuilder interface {
	SubmitBlindedBlock(ctx context.Context, block *ethpb.SignedBlindedBeaconBlockBellatrix) (*v1.ExecutionPayload, error)
	GetHeader(ctx context.Context, slot types.Slot, parentHash [32]byte, pubKey [48]byte) (*ethpb.SignedBuilderBid, error)
	Status() error
	RegisterValidator(ctx context.Context, reg *ethpb.SignedValidatorRegistrationV1) error
}

// config defines a config struct for dependencies into the service.
type config struct {
	builderEndpoint network.Endpoint
}

type Service struct {
	cfg *config
	c   *builder.Client
}

func NewService(ctx context.Context, opts ...Option) (*Service, error) {
	s := &Service{
		cfg: &config{},
	}
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err

		}
	}
	c, err := builder.NewClient(s.cfg.builderEndpoint.Url)
	if err != nil {
		return nil, err
	}
	sk, err := bls.RandKey()
	if err != nil {
		return nil, err
	}

	reg := &ethpb.ValidatorRegistrationV1{
		FeeRecipient: params.BeaconConfig().DefaultFeeRecipient.Bytes(),
		GasLimit:     100000000,
		Timestamp:    uint64(time.Now().Unix()),
		Pubkey:       sk.PublicKey().Marshal(),
	}
	sig := sk.Sign(reg.Pubkey)

	if err := c.RegisterValidator(ctx, &ethpb.SignedValidatorRegistrationV1{
		Message:   reg,
		Signature: sig.Marshal(),
	}); err != nil {
		return nil, err
	}

	h := "a0513a503d5bd6e89a144c3268e5b7e9da9dbf63df125a360e3950a7d0d67131"
	data, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	b, err := c.GetHeader(ctx, 1, bytesutil.ToBytes32(data), [48]byte{})
	if err != nil {
		return nil, err
	}
	msg := b.Message
	header := msg.Header
	log.WithFields(log.Fields{
		"bidValue":     bytesutil.BytesToUint64BigEndian(msg.Value),
		"feeRecipient": fmt.Sprintf("%#x", header.FeeRecipient),
		"parentHash":   fmt.Sprintf("%#x", header.ParentHash),
		"txRoot":       fmt.Sprintf("%#x", header.TransactionsRoot),
		"gasLimit":     header.GasLimit,
		"gasUsed":      header.GasUsed,
	}).Info("Received builder bid")

	sb := HydrateSignedBlindedBeaconBlockBellatrix(&ethpb.SignedBlindedBeaconBlockBellatrix{})
	sb.Block.Body.ExecutionPayloadHeader = header
	sb.Block.Body.SyncAggregate.SyncCommitteeBits = bitfield.NewBitvector512()
	if _, err := c.SubmitBlindedBlock(ctx, sb); err != nil {
		return nil, err
	}

	log.Fatal("End of test")

	return s, nil
}

func (s *Service) Start() {}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) SubmitBlindedBlock(context.Context, *ethpb.SignedBlindedBeaconBlockBellatrix) (*v1.ExecutionPayload, error) {
	panic("implement me")
}

func (s *Service) GetHeader(context.Context, types.Slot, [32]byte, [48]byte) (*ethpb.SignedBuilderBid, error) {
	panic("implement me")
}

func (s *Service) Status() error {
	panic("implement me")
}

func (s *Service) RegisterValidator(context.Context, *ethpb.SignedValidatorRegistrationV1) error {
	panic("implement me")
}

func HydrateSignedBlindedBeaconBlockBellatrix(b *ethpb.SignedBlindedBeaconBlockBellatrix) *ethpb.SignedBlindedBeaconBlockBellatrix {
	if b.Signature == nil {
		b.Signature = make([]byte, fieldparams.BLSSignatureLength)
	}
	b.Block = HydrateBlindedBeaconBlockBellatrix(b.Block)
	return b
}

// HydrateBlindedBeaconBlockBellatrix hydrates a blinded beacon block with correct field length sizes
// to comply with fssz marshalling and unmarshalling rules.
func HydrateBlindedBeaconBlockBellatrix(b *ethpb.BlindedBeaconBlockBellatrix) *ethpb.BlindedBeaconBlockBellatrix {
	if b == nil {
		b = &ethpb.BlindedBeaconBlockBellatrix{}
	}
	if b.ParentRoot == nil {
		b.ParentRoot = make([]byte, fieldparams.RootLength)
	}
	if b.StateRoot == nil {
		b.StateRoot = make([]byte, fieldparams.RootLength)
	}
	b.Body = HydrateBlindedBeaconBlockBodyBellatrix(b.Body)
	return b
}

// HydrateBlindedBeaconBlockBodyBellatrix hydrates a blinded beacon block body with correct field length sizes
// to comply with fssz marshalling and unmarshalling rules.
func HydrateBlindedBeaconBlockBodyBellatrix(b *ethpb.BlindedBeaconBlockBodyBellatrix) *ethpb.BlindedBeaconBlockBodyBellatrix {
	if b == nil {
		b = &ethpb.BlindedBeaconBlockBodyBellatrix{}
	}
	if b.RandaoReveal == nil {
		b.RandaoReveal = make([]byte, fieldparams.BLSSignatureLength)
	}
	if b.Graffiti == nil {
		b.Graffiti = make([]byte, 32)
	}
	if b.Eth1Data == nil {
		b.Eth1Data = &ethpb.Eth1Data{
			DepositRoot: make([]byte, fieldparams.RootLength),
			BlockHash:   make([]byte, 32),
		}
	}
	if b.SyncAggregate == nil {
		b.SyncAggregate = &ethpb.SyncAggregate{
			SyncCommitteeBits:      make([]byte, 64),
			SyncCommitteeSignature: make([]byte, fieldparams.BLSSignatureLength),
		}
	}
	if b.ExecutionPayloadHeader == nil {
		b.ExecutionPayloadHeader = &ethpb.ExecutionPayloadHeader{
			ParentHash:       make([]byte, 32),
			FeeRecipient:     make([]byte, 20),
			StateRoot:        make([]byte, fieldparams.RootLength),
			ReceiptsRoot:     make([]byte, fieldparams.RootLength),
			LogsBloom:        make([]byte, 256),
			PrevRandao:       make([]byte, 32),
			BaseFeePerGas:    make([]byte, 32),
			BlockHash:        make([]byte, 32),
			TransactionsRoot: make([]byte, fieldparams.RootLength),
		}
	}
	return b
}
