package kafka

import (
	"bytes"
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/iface"
	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

var _ = iface.Database(&Exporter{})
var log = logrus.WithField("prefix", "exporter")
var marshaler = &jsonpb.Marshaler{}

type Exporter struct {
	db iface.Database
	p  *kafka.Producer
}

func Wrap(db iface.Database) (iface.Database, error) {
	if  featureconfig.Get().KafkaBootstrapServers == "" {
		return db, nil
	}

	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": featureconfig.Get().KafkaBootstrapServers})
	if err != nil {
		return nil, err
	}

	return &Exporter{db: db, p: p}, nil
}

func (e Exporter) publish(ctx context.Context, topic string, msg proto.Message) error {
	ctx, span := trace.StartSpan(ctx, "kafka.publish")
	defer span.End()

	buf := bytes.NewBuffer(nil)
	if err := marshaler.Marshal(buf, msg); err != nil {
		traceutil.AnnotateError(span, err)
		return err
	}

	key, err := ssz.HashTreeRoot(msg)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return err
	}

	if err := e.p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic: &topic,
		},
		Value: buf.Bytes(),
		Key:   key[:],
	}, nil); err != nil {
		traceutil.AnnotateError(span, err)
		return err
	}
	return nil
}

func (e Exporter) Close() error {
	e.p.Close()
	return e.db.Close()
}

func (e Exporter) SaveAttestation(ctx context.Context, att *eth.Attestation) error {
	go func() {
		if err := e.publish(ctx, "attestation", att); err != nil {
			log.WithError(err).Error("Failed to publish attestation")
		}
	}()

	return e.db.SaveAttestation(ctx, att)
}

func (e Exporter) SaveAttestations(ctx context.Context, atts []*eth.Attestation) error {
	go func() {
		for _, att := range atts {
			if err := e.publish(ctx, "attestation", att); err != nil {
				log.WithError(err).Error("Failed to publish attestation")
			}
		}
	}()
	return e.db.SaveAttestations(ctx, atts)
}

func (e Exporter) SaveBlock(ctx context.Context, block *eth.BeaconBlock) error {
	go func() {
		if err := e.publish(ctx, "block", block); err != nil {
			log.WithError(err).Error("Failed to publish block")
		}
	}()

	return e.db.SaveBlock(ctx, block)
}

func (e Exporter) SaveBlocks(ctx context.Context, blocks []*eth.BeaconBlock) error {
	go func() {
		for _, block := range blocks {
		if err := e.publish(ctx, "block", block); err != nil {
			log.WithError(err).Error("Failed to publish block")
		}
		}
	}()

	return e.db.SaveBlocks(ctx, blocks)
}

func (e Exporter) SaveHeadBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	return e.db.SaveHeadBlockRoot(ctx, blockRoot)
}

func (e Exporter) SaveGenesisBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	return e.db.SaveGenesisBlockRoot(ctx, blockRoot)
}

func (e Exporter) SaveValidatorLatestVote(ctx context.Context, validatorIdx uint64, vote *ethereum_beacon_p2p_v1.ValidatorLatestVote) error {
	return e.db.SaveValidatorLatestVote(ctx, validatorIdx, vote)
}

func (e Exporter) SaveValidatorLatestVotes(ctx context.Context, validatorIndices []uint64, votes []*ethereum_beacon_p2p_v1.ValidatorLatestVote) error {
	return e.db.SaveValidatorLatestVotes(ctx, validatorIndices, votes)
}

func (e Exporter) SaveValidatorIndex(ctx context.Context, publicKey [48]byte, validatorIdx uint64) error {
	return e.db.SaveValidatorIndex(ctx, publicKey, validatorIdx)
}

func (e Exporter) SaveState(ctx context.Context, state *ethereum_beacon_p2p_v1.BeaconState, blockRoot [32]byte) error {
	return e.db.SaveState(ctx, state, blockRoot)
}

func (e Exporter) SaveProposerSlashing(ctx context.Context, slashing *eth.ProposerSlashing) error {
	return e.db.SaveProposerSlashing(ctx, slashing)
}

func (e Exporter) SaveAttesterSlashing(ctx context.Context, slashing *eth.AttesterSlashing) error {
	return e.db.SaveAttesterSlashing(ctx, slashing)
}

func (e Exporter) SaveVoluntaryExit(ctx context.Context, exit *eth.VoluntaryExit) error {
	return e.db.SaveVoluntaryExit(ctx, exit)
}

func (e Exporter) SaveJustifiedCheckpoint(ctx context.Context, checkpoint *eth.Checkpoint) error {
	return e.db.SaveJustifiedCheckpoint(ctx, checkpoint)
}

func (e Exporter) SaveFinalizedCheckpoint(ctx context.Context, checkpoint *eth.Checkpoint) error {
	return e.db.SaveFinalizedCheckpoint(ctx, checkpoint)
}

func (e Exporter) SaveArchivedActiveValidatorChanges(ctx context.Context, epoch uint64, changes *eth.ArchivedActiveSetChanges) error {
	return e.db.SaveArchivedActiveValidatorChanges(ctx, epoch, changes)
}

func (e Exporter) SaveArchivedCommitteeInfo(ctx context.Context, epoch uint64, info *eth.ArchivedCommitteeInfo) error {
	return e.db.SaveArchivedCommitteeInfo(ctx, epoch, info)
}

func (e Exporter) SaveArchivedBalances(ctx context.Context, epoch uint64, balances []uint64) error {
	return e.db.SaveArchivedBalances(ctx, epoch, balances)
}

func (e Exporter) SaveArchivedValidatorParticipation(ctx context.Context, epoch uint64, part *eth.ValidatorParticipation) error {
	return e.db.SaveArchivedValidatorParticipation(ctx, epoch, part)
}

func (e Exporter) SaveDepositContractAddress(ctx context.Context, addr common.Address) error {
	return e.db.SaveDepositContractAddress(ctx, addr)
}
