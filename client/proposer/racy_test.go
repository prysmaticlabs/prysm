package proposer

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/client/internal"
	"github.com/prysmaticlabs/prysm/client/mainchain"
	"github.com/prysmaticlabs/prysm/client/params"
	"github.com/prysmaticlabs/prysm/client/syncer"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/database"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func settingUpProposer(t *testing.T) (*Proposer, *internal.MockClient) {
	backend, smc := internal.SetupMockClient(t)
	node := &internal.MockClient{SMC: smc, T: t, Backend: backend}

	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Failed to start server %v", err)
	}
	server.Start()

	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}

	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("Failed create shardDB %v", err)
	}

	fakeSyncer, err := syncer.NewSyncer(params.DefaultConfig(), &mainchain.SMCClient{}, server, db, 1)
	if err != nil {
		t.Fatalf("Failed to start syncer %v", err)
	}

	fakeProposer, err := NewProposer(params.DefaultConfig(), node, server, db, 1, fakeSyncer)
	if err != nil {
		t.Fatalf("Failed to create proposer %v", err)
	}
	fakeProposer.config.CollationSizeLimit = int64(math.Pow(float64(2), float64(10)))

	return fakeProposer, node
}

// waitForLogMsg keeps checking for log messages until "want" is found or a
// timeout expires, returning the index of the message if found.
// The motivation is to be more resilient to connection failure errors that can
// appear in the log:
// Failed to connect to peer: dial attempt failed: <peer.ID 16Uiu2> --> <peer.ID 16Uiu2> dial attempt failed: connection refused
func waitForLogMsg(hook *logTest.Hook, want string) (int, error) {
	timeout := time.After(60 * time.Second)
	tick := time.Tick(500 * time.Millisecond)
	for {
		select {
		case <-timeout:
			return 0, fmt.Errorf("didn't get %q log entry on time", want)
		case <-tick:
			for index, msg := range hook.AllEntries() {
				if msg.Message == want {
					return index, nil
				}
			}
		}
	}
}

func TestProposerRoundTrip(t *testing.T) {
	hook := logTest.NewGlobal()
	fakeProposer, node := settingUpProposer(t)
	defer func() {
		fakeProposer.dbService.Close()
		fakeProposer.p2p.Stop()
	}()

	input := make([]byte, 0, 2000)
	for len(input) < int(fakeProposer.config.CollationSizeLimit/4) {
		input = append(input, []byte{'t', 'e', 's', 't', 'i', 'n', 'g'}...)
	}
	tx := pb.Transaction{Input: input}

	for i := 0; i < 5; i++ {
		node.CommitWithBlock()
	}
	fakeProposer.Start()
	defer fakeProposer.Stop()

	for i := 0; i < 4; i++ {
		fakeProposer.p2p.Broadcast(&tx)
	}

	if _, err := waitForLogMsg(hook, "Collation created"); err != nil {
		t.Fatal(err.Error())
	}
}

func TestIncompleteCollation(t *testing.T) {
	hook := logTest.NewGlobal()
	fakeProposer, node := settingUpProposer(t)
	defer func() {
		fakeProposer.dbService.Close()
		fakeProposer.p2p.Stop()
	}()

	input := make([]byte, 0, 2000)
	for int64(len(input)) < (fakeProposer.config.CollationSizeLimit)/4 {
		input = append(input, []byte{'t', 'e', 's', 't', 'i', 'n', 'g'}...)
	}
	tx := pb.Transaction{Input: input}

	for i := 0; i < 5; i++ {
		node.CommitWithBlock()
	}
	fakeProposer.Start()

	for i := 0; i < 3; i++ {
		fakeProposer.p2p.Broadcast(&tx)
	}
	fakeProposer.Stop()

	index, err := waitForLogMsg(hook, "Stopping proposer service")
	if err != nil {
		t.Fatal(err.Error())
	}
	for i := index; i >= 0; i-- {
		if hook.AllEntries()[i].Message == "Collation created" {
			t.Fatal("Collation created before stopping proposer")
		}
	}
}

func TestCollationWitInDiffPeriod(t *testing.T) {
	hook := logTest.NewGlobal()
	fakeProposer, node := settingUpProposer(t)
	defer func() {
		fakeProposer.dbService.Close()
		fakeProposer.p2p.Stop()
	}()

	input := make([]byte, 0, 2000)
	for int64(len(input)) < (fakeProposer.config.CollationSizeLimit)/4 {
		input = append(input, []byte{'t', 'e', 's', 't', 'i', 'n', 'g'}...)
	}
	tx := pb.Transaction{Input: input}

	for i := 0; i < 5; i++ {
		node.CommitWithBlock()
	}
	fakeProposer.Start()
	defer fakeProposer.Stop()

	for i := 0; i < 5; i++ {
		node.CommitWithBlock()
	}

	fakeProposer.p2p.Broadcast(&tx)

	if _, err := waitForLogMsg(hook, "Collation created"); err != nil {
		t.Fatal(err.Error())
	}
}
