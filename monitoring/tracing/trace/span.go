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
		return nil
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

func (e EmptySpan) IsRecordingEvents() bool {
	return false
}

func (e EmptySpan) End() {
}

func (e EmptySpan) SpanContext() trace.SpanContext {
	return trace.SpanContext{}
}

func (e EmptySpan) SetName(name string) {

}

func (e EmptySpan) SetStatus(status trace.Status) {

}

func (e EmptySpan) AddAttributes(attributes ...trace.Attribute) {
}

func (e EmptySpan) Annotate(attributes []trace.Attribute, str string) {

}

func (e EmptySpan) Annotatef(attributes []trace.Attribute, format string, a ...interface{}) {
}

func (e EmptySpan) AddMessageSendEvent(messageID, uncompressedByteSize, compressedByteSize int64) {
}

func (e EmptySpan) AddMessageReceiveEvent(messageID, uncompressedByteSize, compressedByteSize int64) {
}

func (e EmptySpan) AddLink(l trace.Link) {
}

func (e EmptySpan) String() string {
	return ""
}
