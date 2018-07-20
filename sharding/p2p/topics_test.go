package p2p

import (
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
)

type testStruct struct{}

func TestReverseMapping(t *testing.T) {
	tests := []struct {
		input map[pb.Topic]reflect.Type
		want  map[reflect.Type]pb.Topic
	}{
		{
			input: map[pb.Topic]reflect.Type{
				pb.Topic_UNKNOWN: reflect.TypeOf(testStruct{}),
			},
			want: map[reflect.Type]pb.Topic{
				reflect.TypeOf(testStruct{}): pb.Topic_UNKNOWN,
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
		want  pb.Topic
	}{
		{
			input: pb.CollationBodyRequest{},
			want:  pb.Topic_COLLATION_BODY_REQUEST,
		},
		{
			input: &pb.CollationBodyRequest{},
			want:  pb.Topic_COLLATION_BODY_REQUEST,
		},
		{
			input: CustomStruct{},
			want:  pb.Topic_UNKNOWN,
		},
	}

	for _, tt := range tests {
		got := topic(tt.input)
		if got != tt.want {
			t.Errorf("topic(%T) = %v. wanted %v", tt.input, got, tt.want)
		}
	}
}
