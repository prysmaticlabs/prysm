package client

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/runtime"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"google.golang.org/grpc/metadata"
)

var _ runtime.Service = (*ValidatorService)(nil)
var _ GenesisFetcher = (*ValidatorService)(nil)
var _ SyncChecker = (*ValidatorService)(nil)

func TestStop_CancelsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	vs := &ValidatorService{
		ctx:    ctx,
		cancel: cancel,
	}

	assert.NoError(t, vs.Stop())

	select {
	case <-time.After(1 * time.Second):
		t.Error("Context not canceled within 1s")
	case <-vs.ctx.Done():
	}
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	// Use canceled context so that the run function exits immediately..
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	validatorService := &ValidatorService{
		ctx:      ctx,
		cancel:   cancel,
		endpoint: "merkle tries",
		withCert: "alice.crt",
	}
	validatorService.Start()
	require.NoError(t, validatorService.Stop(), "Could not stop service")
	require.LogsContain(t, hook, "Stopping service")
}

func TestLifecycle_Insecure(t *testing.T) {
	hook := logTest.NewGlobal()
	// Use canceled context so that the run function exits immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	validatorService := &ValidatorService{
		ctx:      ctx,
		cancel:   cancel,
		endpoint: "merkle tries",
	}
	validatorService.Start()
	require.LogsContain(t, hook, "You are using an insecure gRPC connection")
	require.NoError(t, validatorService.Stop(), "Could not stop service")
	require.LogsContain(t, hook, "Stopping service")
}

func TestStatus_NoConnectionError(t *testing.T) {
	validatorService := &ValidatorService{}
	assert.ErrorContains(t, "no connection", validatorService.Status())
}

func TestStart_GrpcHeaders(t *testing.T) {
	hook := logTest.NewGlobal()
	// Use canceled context so that the run function exits immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for input, output := range map[string][]string{
		"should-break": {},
		"key=value":    {"key", "value"},
		"":             {},
		",":            {},
		"key=value,Authorization=Q=": {
			"key", "value", "Authorization", "Q=",
		},
		"Authorization=this is a valid value": {
			"Authorization", "this is a valid value",
		},
	} {
		validatorService := &ValidatorService{
			ctx:         ctx,
			cancel:      cancel,
			endpoint:    "merkle tries",
			grpcHeaders: strings.Split(input, ","),
		}
		validatorService.Start()
		md, _ := metadata.FromOutgoingContext(validatorService.ctx)
		if input == "should-break" {
			require.LogsContain(t, hook, "Incorrect gRPC header flag format. Skipping should-break")
		} else if len(output) == 0 {
			require.DeepEqual(t, md, metadata.MD(nil))
		} else {
			require.DeepEqual(t, md, metadata.Pairs(output...))
		}
	}
}
