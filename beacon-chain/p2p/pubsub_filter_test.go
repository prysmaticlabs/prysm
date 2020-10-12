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

func Test_scanfcheck(t *testing.T) {
	type args struct {
		input  string
		format string
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "no formatting, exact match",
			args: args{
				input:  "/foo/bar/zzzzzzzzzzzz/1234567",
				format: "/foo/bar/zzzzzzzzzzzz/1234567",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "no formatting, mismatch",
			args: args{
				input:  "/foo/bar/zzzzzzzzzzzz/1234567",
				format: "/bar/foo/yyyyyy/7654321",
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "formatting, match",
			args: args{
				input:  "/foo/bar/abcdef/topic_11",
				format: "/foo/bar/%x/topic_%d",
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "formatting, incompatible bytes",
			args: args{
				input:  "/foo/bar/zzzzzz/topic_11",
				format: "/foo/bar/%x/topic_%d",
			},
			want:    0,
			wantErr: true,
		},
		{ // Note: This method only supports integer compatible formatting values.
			name: "formatting, string match",
			args: args{
				input:  "/foo/bar/zzzzzz/topic_11",
				format: "/foo/bar/%s/topic_%d",
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := scanfcheck(tt.args.input, tt.args.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("scanfcheck() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("scanfcheck() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGossipTopicMapping_scanfcheck_GossipTopicFormattingSanityCheck(t *testing.T) {
	// scanfcheck only supports integer based substitutions at the moment. Any others will
	// inaccurately fail validation.
	for topic, _ := range GossipTopicMappings {
		t.Run(topic, func(t *testing.T) {
			for i, c := range topic {
				if string(c) == "%" {
					next := string(topic[i+1])
					if next != "d" && next != "x" {
						t.Errorf("Topic %s has formatting incompatiable with scanfcheck. Only %%d and %%x are supported", topic)
					}
				}
			}
		})
	}
}
