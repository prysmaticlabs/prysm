package rpc

import (
	"context"
	"io"
	"testing"

	"github.com/gorilla/mux"
	pb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var _ pb.AuthServer = (*Server)(nil)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(io.Discard)
}

func TestLifecycleWithMinimumConfig(t *testing.T) {
	hook := logTest.NewGlobal()

	server := NewServer(context.Background(), &Config{Router: mux.NewRouter()})
	server.Start()

	require.LogsContain(t, hook, "validator gRPC server listening")
	assert.NoError(t, server.Stop())
	require.LogsContain(t, hook, "Completed graceful stop of validator gRPC server")
}
