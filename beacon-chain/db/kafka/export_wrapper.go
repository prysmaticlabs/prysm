// Package kafka defines an implementation of Database interface
// which exports streaming data using Kafka for data analysis.
package kafka

import (
	"context"

	fssz "github.com/ferranbt/fastssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/iface"
	"github.com/prysmaticlabs/prysm/proto/interfaces"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
	jsonpb "google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	_ "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka/librdkafka" // Required for c++ kafka library.
	"gopkg.in/errgo.v2/fmt/errors"
)

var _ iface.Database = (*Exporter)(nil)
var marshaler = jsonpb.MarshalOptions{}

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

func (e Exporter) publish(ctx context.Context, topic string, msg interfaces.SignedBeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "kafka.publish")
	defer span.End()

	var err error
	var buf []byte
	if buf, err = marshaler.Marshal(msg.Proto()); err != nil {
		traceutil.AnnotateError(span, err)
		return err
	}

	var key [32]byte
	if v, ok := msg.(fssz.HashRoot); ok {
		key, err = v.HashTreeRoot()
	} else {
		err = errors.New("object does not follow hash tree root interface")
	}
	if err != nil {
		traceutil.AnnotateError(span, err)
		return err
	}

	if err := e.p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic: &topic,
		},
		Value: buf,
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
func (e Exporter) SaveBlock(ctx context.Context, block interfaces.SignedBeaconBlock) error {
	go func() {
		if err := e.publish(ctx, "beacon_block", block); err != nil {
			log.WithError(err).Error("Failed to publish block")
		}
	}()

	return e.db.SaveBlock(ctx, block)
}

// SaveBlocks publishes to the kafka topic for beacon blocks.
func (e Exporter) SaveBlocks(ctx context.Context, blocks []interfaces.SignedBeaconBlock) error {
	go func() {
		for _, block := range blocks {
			if err := e.publish(ctx, "beacon_block", block); err != nil {
				log.WithError(err).Error("Failed to publish block")
			}
		}
	}()

	return e.db.SaveBlocks(ctx, blocks)
}
