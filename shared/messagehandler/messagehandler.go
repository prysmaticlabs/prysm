package messagehandler

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/gogo/protobuf/proto"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

const noMsgData = "message contains no data"

var log = logrus.WithField("prefix", "message-handler")

// SafelyHandleMessage will recover and log any panic that occurs from the
// function argument.
func SafelyHandleMessage(ctx context.Context, fn func(ctx context.Context, message proto.Message) error, msg proto.Message) {
	defer func() {
		if r := recover(); r != nil {
			printedMsg := noMsgData
			if msg != nil {
				printedMsg = proto.MarshalTextString(msg)
			}
			log.WithFields(logrus.Fields{
				"r":   r,
				"msg": printedMsg,
			}).Error("Panicked when handling p2p message! Recovering...")

			debug.PrintStack()

			if ctx == nil {
				return
			}
			if span := trace.FromContext(ctx); span != nil {
				span.SetStatus(trace.Status{
					Code:    trace.StatusCodeInternal,
					Message: fmt.Sprintf("Panic: %v", r),
				})
			}
		}
	}()

	// Fingers crossed that it doesn't panic...
	if err := fn(ctx, msg); err != nil {
		// Report any error on the span, if one exists.
		if span := trace.FromContext(ctx); span != nil {
			span.SetStatus(trace.Status{
				Code:    trace.StatusCodeInternal,
				Message: err.Error(),
			})
		}
	}
}
