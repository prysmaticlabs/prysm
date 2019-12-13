package service

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/urfave/cli"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func TestLifecycle_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	context := cli.NewContext(app, set, nil)
	rpcService, err := NewRPCService(&Config{
		Port: 7348,
	}, context)
	if err != nil {
		t.Error("gRPC Service fail to initialize:", err)
	}
	waitForStarted(rpcService, t)
	rpcService.Close()
	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, "Listening on port")
	testutil.AssertLogsContain(t, hook, "Stopping service")

}

func TestRPC_BadEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	context := cli.NewContext(app, set, nil)
	rpcService, err := NewRPCService(&Config{
		Port: 99999999,
	}, context)
	if err != nil {
		t.Error("gRPC Service fail to initialize:", err)
	}
	testutil.AssertLogsDoNotContain(t, hook, "Could not listen to port in Start()")
	testutil.AssertLogsDoNotContain(t, hook, "Could not load TLS keys")
	testutil.AssertLogsDoNotContain(t, hook, "Could not serve gRPC")

	waitForStarted(rpcService, t)

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, "Could not listen to port in Start()")

	rpcService.Close()
}

func TestStatus_CredentialError(t *testing.T) {
	credentialErr := errors.New("credentialError")
	s := &Service{credentialError: credentialErr}

	if _, err := s.Status(); err != s.credentialError {
		t.Errorf("Wanted: %v, got: %v", s.credentialError, err)
	}
}

func TestRPC_InsecureEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	context := cli.NewContext(app, set, nil)
	rpcService, err := NewRPCService(&Config{
		Port: 5555,
	}, context)
	if err != nil {
		t.Error("gRPC Service fail to initialize:", err)
	}
	waitForStarted(rpcService, t)

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprint("Listening on port"))
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")

	rpcService.Close()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func waitForStarted(rpcService *Service, t *testing.T) {
	go rpcService.Start()
	tick := time.Tick(100 * time.Millisecond)
	for {
		<-tick
		s, err := rpcService.Status()
		if err != nil {
			t.Fatal(err)
			break
		}
		if s {
			break
		}
	}
}
