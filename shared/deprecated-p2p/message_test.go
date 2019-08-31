package p2p

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	testpb "github.com/prysmaticlabs/prysm/proto/testing"
)

func TestMessageType(t *testing.T) {
	tests := []struct {
		msg      proto.Message
		expected reflect.Type
	}{
		{
			msg:      &testpb.TestMessage{},
			expected: reflect.TypeOf(testpb.TestMessage{}),
		},
		{
			msg:      &testpb.Puzzle{},
			expected: reflect.TypeOf(testpb.Puzzle{}),
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.expected), func(t *testing.T) {
			got := messageType(tt.msg)
			if got != tt.expected {
				t.Errorf("Wanted %v but got %v", tt.expected, got)
			}
		})
	}
}
