package adapter

import (
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"go.opencensus.io/trace"
)

var TracingAdapter p2p.Adapter = func(next p2p.Handler) p2p.Handler {
	return func(msg p2p.Message) {
		var messageSpan *trace.Span
		msg.Ctx, messageSpan = trace.StartSpan(msg.Ctx, "handleP2pMessage")
		next(msg)
		messageSpan.End()
	}
}
