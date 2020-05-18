// Package logutil creates a Multi writer instance that
// write all logs that are written to stdout.
package logutil

import (
	"context"
	"io"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/sirupsen/logrus"
)

// ConfigurePersistentLogging adds a log-to-file writer. File content is identical to stdout.
func ConfigurePersistentLogging(logFileName string) error {
	logrus.WithField("logFileName", logFileName).Info("Logs will be made persistent")
	f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	mw := io.MultiWriter(os.Stdout, f)
	logrus.SetOutput(mw)

	logrus.Info("File logging initialized")
	return nil
}

// LogDebugRequestInfoUnaryInterceptor this method logs the gRPC backend as well as request duration when the log level is set to debug
// or higher.
func LogDebugRequestInfoUnaryInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// Shortcut when debug logging is not enabled.
	if logrus.GetLevel() < logrus.DebugLevel {
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	var header metadata.MD
	opts = append(
		opts,
		grpc.Header(&header),
	)
	start := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)
	logrus.WithField("backend", header["x-backend"]).
		WithField("method", method).WithField("duration", time.Now().Sub(start)).
		Debug("gRPC request finished.")
	return err
}
