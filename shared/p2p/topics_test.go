package p2p

import (
	"reflect"
	"testing"

	shardpb "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
)

type testStruct struct{}

func TestReverseMapping(t *testing.T) {
	tests := []struct {
		input map[shardpb.Topic]reflect.Type
		want  map[reflect.Type]shardpb.Topic
	}{
		{
			input: map[shardpb.Topic]reflect.Type{
				shardpb.Topic_UNKNOWN: reflect.TypeOf(testStruct{}),
			},
			want: map[reflect.Type]shardpb.Topic{
				reflect.TypeOf(testStruct{}): shardpb.Topic_UNKNOWN,
			},
		},
	}

	for _, tt := range tests {
		got := reverseMapping(tt.input)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("reverseMapping(%+v) = %+v. Wanted %+v", tt.input, got, tt.want)
		}
	}
}

func TestTopic(t *testing.T) {
	type CustomStruct struct{}

	tests := []struct {
		input interface{}
		want  shardpb.Topic
	}{
		{
			input: shardpb.CollationBodyRequest{},
			want:  shardpb.Topic_COLLATION_BODY_REQUEST,
		},
		{
			input: &shardpb.CollationBodyRequest{},
			want:  shardpb.Topic_COLLATION_BODY_REQUEST,
		},
		{
			input: CustomStruct{},
			want:  shardpb.Topic_UNKNOWN,
		},
	}

	for _, tt := range tests {
		got := topic(tt.input)
		if got != tt.want {
			t.Errorf("topic(%T) = %v. wanted %v", tt.input, got, tt.want)
		}
	}
}
