package kafka

import (
	"bytes"
	"context"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/iface"
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
	if featureconfig.Get().KafkaBootstrapServers == "" {
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
