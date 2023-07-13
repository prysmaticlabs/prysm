//go:build go1.18

package p2p_test

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/encoder"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
)

func FuzzMsgID(f *testing.F) {
	//validTopic := fmt.Sprintf(p2p.BlockSubnetTopicFormat, []byte{0xb5, 0x30, 0x3f, 0x2a}) + "/" + encoder.ProtocolSuffixSSZSnappy
	validTopic := p2p.BlockSubnetTopicFormat.ConvertToString([4]byte{0xb5, 0x30, 0x3f, 0x2a}, "/"+encoder.ProtocolSuffixSSZSnappy, 0)
	f.Add(validTopic)

	f.Fuzz(func(t *testing.T, topic string) {
		_, err := p2p.ExtractGossipDigest(topic)
		if err != nil {
			return
		}
	})
}
