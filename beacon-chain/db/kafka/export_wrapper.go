// Package kafka defines an implementation of Database interface
// which exports streaming data using Kafka for data analysis.
package kafka

import (
	"bytes"
	"context"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/iface"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	_ "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka/librdkafka" // Required for c++ kafka library.
)

var _ = iface.Database(&Exporter{})
var log = logrus.WithField("prefix", "exporter")
var marshaler = &jsonpb.Marshaler{}

// Exporter wraps a database interface and exports certain objects to kafka topics.
type Exporter struct {
	db iface.Database
	p  *kafka.Producer
}

// Wrap the db with kafka exporter. If the feature flag is not enabled, this service does not wrap
// the database, but returns the underlying database pointer itself.
func Wrap(db iface.Database) (iface.Database, error) {
	if featureconfig.Get().KafkaBootstrapServers == "" {
		log.Debug("Empty Kafka bootstrap servers list, database was not wrapped with Kafka exporter")
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

// Close closes kafka producer and underlying db.
func (e Exporter) Close() error {
	e.p.Close()
	return e.db.Close()
}

// SaveBlock publishes to the kafka topic for beacon blocks.
func (e Exporter) SaveBlock(ctx context.Context, block *eth.SignedBeaconBlock) error {
	go func() {
		if err := e.publish(ctx, "beacon_block", block); err != nil {
			log.WithError(err).Error("Failed to publish block")
		}
	}()

	return e.db.SaveBlock(ctx, block)
}

// SaveBlocks publishes to the kafka topic for beacon blocks.
func (e Exporter) SaveBlocks(ctx context.Context, blocks []*eth.SignedBeaconBlock) error {
	go func() {
		for _, block := range blocks {
			if err := e.publish(ctx, "beacon_block", block); err != nil {
				log.WithError(err).Error("Failed to publish block")
			}
		}
	}()

	return e.db.SaveBlocks(ctx, blocks)
}
