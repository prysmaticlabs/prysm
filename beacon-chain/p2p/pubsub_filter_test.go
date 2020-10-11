package p2p

import (
	"fmt"
	"testing"
)

func Test_subscriptionFilter_CanSubscribe(t *testing.T) {
	currentFork := [4]byte{0x01, 0x02, 0x03, 0x04}
	previousFork := [4]byte{0x11, 0x12, 0x13, 0x14}
	type test struct {
		name  string
		topic string
		want  bool
	}

	tests := []test{
		// TODO: Add test cases.
	}

	// Ensure all gossip topic mappings pass validation.
	for topic, _ := range GossipTopicMappings {
		formatting := []interface{}{currentFork}

		// Special case for attestation subnets which have a second formatting placeholder.
		if topic == AttestationSubnetTopicFormat {
			formatting = append(formatting, 0 /* some subnet ID */)
		}

		tt := test{
			name:  topic,
			topic: fmt.Sprintf(topic, formatting...),
			want:  true,
		}
		tests = append(tests, tt)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sf := &subscriptionFilter{
				currentForkDigest:  fmt.Sprintf("%x", currentFork),
				previousForkDigest: fmt.Sprintf("%x", previousFork),
			}
			if got := sf.CanSubscribe(tt.topic); got != tt.want {
				t.Errorf("CanSubscribe(%s) = %v, want %v", tt.topic, got, tt.want)
			}
		})
	}
}
