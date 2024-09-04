// Package tracing includes useful functions for opentracing annotations.
package tracing

import (
	prysmTrace "github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// AnnotateError on span. This should be used any time a particular span experiences an error.
func AnnotateError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.SetAttributes(prysmTrace.BoolAttribute("error", true))
	span.SetStatus(codes.Error, err.Error())
}
