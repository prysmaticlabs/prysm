package sync

import (
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/encoder"
	"github.com/prysmaticlabs/prysm/v4/network/forks"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
)

func TestSubTopicHandler_CRUD(t *testing.T) {
	h := newSubTopicHandler()
	// Non-existent topic
	assert.Equal(t, false, h.topicExists("junk"))
	assert.Equal(t, false, h.digestExists([4]byte{}))

	digest, err := forks.CreateForkDigest(time.Now(), make([]byte, 32))
	assert.NoError(t, err)
	enc := encoder.SszNetworkEncoder{}

	// Valid topic added in.
	//topic := fmt.Sprintf(p2p.BlockSubnetTopicFormat, digest) + enc.ProtocolSuffix()
	topic := p2p.BlockSubnetTopicFormat.ConvertToStringWithForkDigestAndSuffix(digest, enc.ProtocolSuffix())
	h.addTopic(topic, new(pubsub.Subscription))
	assert.Equal(t, true, h.topicExists(topic))
	assert.Equal(t, true, h.digestExists(digest))
	assert.Equal(t, 1, len(h.allTopics()))

	h.removeTopic(topic)
	assert.Equal(t, false, h.topicExists(topic))
	assert.Equal(t, false, h.digestExists(digest))
	assert.Equal(t, 0, len(h.allTopics()))

	h = newSubTopicHandler()
	// Multiple Topics added in.
	//topic = fmt.Sprintf(p2p.BlockSubnetTopicFormat, digest) + enc.ProtocolSuffix()
	topic = p2p.BlockSubnetTopicFormat.ConvertToStringWithForkDigestAndSuffix(digest, enc.ProtocolSuffix())
	h.addTopic(topic, new(pubsub.Subscription))
	assert.Equal(t, true, h.topicExists(topic))

	//topic = fmt.Sprintf(p2p.ExitSubnetTopicFormat, digest) + enc.ProtocolSuffix()
	topic = p2p.ExitSubnetTopicFormat.ConvertToStringWithForkDigestAndSuffix(digest, enc.ProtocolSuffix())
	h.addTopic(topic, new(pubsub.Subscription))
	assert.Equal(t, true, h.topicExists(topic))

	//topic = fmt.Sprintf(p2p.ProposerSlashingSubnetTopicFormat, digest) + enc.ProtocolSuffix()
	topic = p2p.ProposerSlashingSubnetTopicFormat.ConvertToStringWithForkDigestAndSuffix(digest, enc.ProtocolSuffix())
	h.addTopic(topic, new(pubsub.Subscription))
	assert.Equal(t, true, h.topicExists(topic))

	//topic = fmt.Sprintf(p2p.AttesterSlashingSubnetTopicFormat, digest) + enc.ProtocolSuffix()
	topic = p2p.AttesterSlashingSubnetTopicFormat.ConvertToStringWithForkDigestAndSuffix(digest, enc.ProtocolSuffix())
	h.addTopic(topic, new(pubsub.Subscription))
	assert.Equal(t, true, h.topicExists(topic))

	//topic = fmt.Sprintf(p2p.AggregateAndProofSubnetTopicFormat, digest) + enc.ProtocolSuffix()
	topic = p2p.AggregateAndProofSubnetTopicFormat.ConvertToStringWithForkDigestAndSuffix(digest, enc.ProtocolSuffix())
	h.addTopic(topic, new(pubsub.Subscription))
	assert.Equal(t, true, h.topicExists(topic))

	//topic = fmt.Sprintf(p2p.SyncContributionAndProofSubnetTopicFormat, digest) + enc.ProtocolSuffix()
	topic = p2p.SyncContributionAndProofSubnetTopicFormat.ConvertToStringWithForkDigestAndSuffix(digest, enc.ProtocolSuffix())
	h.addTopic(topic, new(pubsub.Subscription))
	assert.Equal(t, true, h.topicExists(topic))

	//topic = fmt.Sprintf(p2p.BlsToExecutionChangeSubnetTopicFormat, digest) + enc.ProtocolSuffix()
	topic = p2p.BlsToExecutionChangeSubnetTopicFormat.ConvertToStringWithForkDigestAndSuffix(digest, enc.ProtocolSuffix())
	h.addTopic(topic, new(pubsub.Subscription))
	assert.Equal(t, true, h.topicExists(topic))

	assert.Equal(t, 7, len(h.allTopics()))

	// Remove multiple topics
	//topic = fmt.Sprintf(p2p.AttesterSlashingSubnetTopicFormat, digest) + enc.ProtocolSuffix()
	topic = p2p.AttesterSlashingSubnetTopicFormat.ConvertToStringWithForkDigestAndSuffix(digest, enc.ProtocolSuffix())
	h.removeTopic(topic)
	assert.Equal(t, false, h.topicExists(topic))

	//topic = fmt.Sprintf(p2p.ExitSubnetTopicFormat, digest) + enc.ProtocolSuffix()
	topic = p2p.ExitSubnetTopicFormat.ConvertToStringWithForkDigestAndSuffix(digest, enc.ProtocolSuffix())
	h.removeTopic(topic)
	assert.Equal(t, false, h.topicExists(topic))

	//topic = fmt.Sprintf(p2p.ProposerSlashingSubnetTopicFormat, digest) + enc.ProtocolSuffix()
	topic = p2p.ProposerSlashingSubnetTopicFormat.ConvertToStringWithForkDigestAndSuffix(digest, enc.ProtocolSuffix())
	h.removeTopic(topic)
	assert.Equal(t, false, h.topicExists(topic))

	assert.Equal(t, true, h.digestExists(digest))
	assert.Equal(t, 4, len(h.allTopics()))

	// Remove remaining topics.
	//topic = fmt.Sprintf(p2p.BlockSubnetTopicFormat, digest) + enc.ProtocolSuffix()
	topic = p2p.BlockSubnetTopicFormat.ConvertToStringWithForkDigestAndSuffix(digest, enc.ProtocolSuffix())
	h.removeTopic(topic)
	assert.Equal(t, false, h.topicExists(topic))

	//topic = fmt.Sprintf(p2p.AggregateAndProofSubnetTopicFormat, digest) + enc.ProtocolSuffix()
	topic = p2p.AggregateAndProofSubnetTopicFormat.ConvertToStringWithForkDigestAndSuffix(digest, enc.ProtocolSuffix())
	h.removeTopic(topic)
	assert.Equal(t, false, h.topicExists(topic))

	//topic = fmt.Sprintf(p2p.SyncContributionAndProofSubnetTopicFormat, digest) + enc.ProtocolSuffix()
	topic = p2p.SyncContributionAndProofSubnetTopicFormat.ConvertToStringWithForkDigestAndSuffix(digest, enc.ProtocolSuffix())
	h.removeTopic(topic)
	assert.Equal(t, false, h.topicExists(topic))

	//topic = fmt.Sprintf(p2p.BlsToExecutionChangeSubnetTopicFormat, digest) + enc.ProtocolSuffix()
	topic = p2p.BlsToExecutionChangeSubnetTopicFormat.ConvertToStringWithForkDigestAndSuffix(digest, enc.ProtocolSuffix())
	h.removeTopic(topic)
	assert.Equal(t, false, h.topicExists(topic))

	assert.Equal(t, false, h.digestExists(digest))
	assert.Equal(t, 0, len(h.allTopics()))
}
