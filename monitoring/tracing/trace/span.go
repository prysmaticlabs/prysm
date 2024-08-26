package trace

import (
	"context"

	"go.opencensus.io/trace"
)

// TracingEnabled tracks whether tracing is enabled in prysm.
var TracingEnabled = false

// StartSpan is a wrapper over the opencensus package method. This is to allow us to skip
// calling that particular method if tracing has been disabled.
func StartSpan(ctx context.Context, name string, o ...trace.StartOption) (context.Context, *trace.Span) {
	if !TracingEnabled {
		// Return an empty span if tracing has been disabled.
		return ctx, trace.NewSpan(EmptySpan{})
	}
	return trace.StartSpan(ctx, name, o...)
}

// NewContext is a wrapper which returns back the parent context
// if tracing is disabled.
func NewContext(parent context.Context, s *trace.Span) context.Context {
	if !TracingEnabled {
		return parent
	}
	return trace.NewContext(parent, s)
}

// FromContext is a wrapper which returns a nil span
// if tracing is disabled.
func FromContext(ctx context.Context) *trace.Span {
	if !TracingEnabled {
		return trace.NewSpan(EmptySpan{})
	}
	return trace.FromContext(ctx)
}

// Int64Attribute --
func Int64Attribute(key string, value int64) trace.Attribute {
	return trace.Int64Attribute(key, value)
}

// StringAttribute --
func StringAttribute(key, value string) trace.Attribute {
	return trace.StringAttribute(key, value)
}

// BoolAttribute --
func BoolAttribute(key string, value bool) trace.Attribute {
	return trace.BoolAttribute(key, value)
}

type EmptySpan struct{}

func (EmptySpan) IsRecordingEvents() bool {
	return false
}

func (EmptySpan) End() {
}

func (EmptySpan) SpanContext() trace.SpanContext {
	return trace.SpanContext{}
}

func (EmptySpan) SetName(string) {

}

func (EmptySpan) SetStatus(trace.Status) {

}

func (EmptySpan) AddAttributes(...trace.Attribute) {
}

func (EmptySpan) Annotate([]trace.Attribute, string) {

}

func (EmptySpan) Annotatef([]trace.Attribute, string, ...interface{}) {
}

func (EmptySpan) AddMessageSendEvent(_, _, _ int64) {
}

func (EmptySpan) AddMessageReceiveEvent(_, _, _ int64) {
}

func (EmptySpan) AddLink(trace.Link) {
}

func (EmptySpan) String() string {
	return ""
}
