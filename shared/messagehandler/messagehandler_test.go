package messagehandler

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"

	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSafelyHandleMessage(t *testing.T) {
	hook := logTest.NewGlobal()

	SafelyHandleMessage(func(_ proto.Message) {
		panic("bad!")
	}, &pb.BeaconBlock{}, nil)

	testutil.AssertLogsContain(t, hook, "Panicked when handling p2p message!")
}

func TestSafelyHandleMessage_NoData(t *testing.T) {
	hook := logTest.NewGlobal()

	SafelyHandleMessage(func(_ proto.Message) {
		panic("bad!")
	}, nil, nil)

	entry := hook.LastEntry()
	if entry.Data["msg"] != "message contains no data" {
		t.Errorf("Message logged was not what was expected: %s", entry.Data["msg"])
	}
}
