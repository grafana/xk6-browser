// Package trace provides tracing instrumentation tailored for k6 browser needs.
package trace

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	k6lib "go.k6.io/k6/lib"
)

const tracerName = "k6.browser"

// liveSpan represents an active span associated with a page navigation.
//
// Because a frame navigation and WebVitals event parsing happen async
// (see frame_session/onFrameNavigated and frame_session/parseAndEmitWebVitalMetric)
// there is not a reference to the last span generated from these methods, therefore
// we have to keep a reference to the active span for each frame from the tracer
// and make this accessible for these methods.
type liveSpan struct {
	ctx  context.Context
	span trace.Span
}

// Tracer represents a traces generator tailored to k6 browser needs.
// Specifically implements methods to generate spans for navigations, API calls and page events,
// accepting input parameters that allow correlating async operations, such as Web Vitals events,
// with the page to which they belong to.
type Tracer struct {
	logger logrus.FieldLogger

	trace.Tracer

	metadata []attribute.KeyValue

	liveSpansMu sync.RWMutex
	liveSpans   map[string]*liveSpan
}

// NewTracer creates a new Tracer from the given TracerProvider.
func NewTracer(logger logrus.FieldLogger, tp k6lib.TracerProvider, metadata map[string]string, options ...trace.TracerOption) *Tracer {
	return &Tracer{
		logger:    logger,
		Tracer:    tp.Tracer(tracerName, options...),
		metadata:  buildMetadataAttributes(metadata),
		liveSpans: make(map[string]*liveSpan),
	}
}

// Start overrides the underlying OTEL tracer method to include the tracer metadata.
func (t *Tracer) Start(
	ctx context.Context, spanName string, opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	opts = append(opts, trace.WithAttributes(t.metadata...))
	return t.Tracer.Start(ctx, spanName, opts...)
}

func GetTraceID(spanCtx trace.SpanContext) string {
	if spanCtx.HasTraceID() {
		traceID := spanCtx.TraceID()
		return traceID.String()
	}
	return ""
}

// TraceAPICall adds a new span to the current liveSpan for the given targetID and returns it. It
// is the caller's responsibility to close the generated span.
// If there is not a liveSpan for the given targetID, the new span is created based on the given
// context, which means that it might be a root span or not depending if the context already wraps
// a span.
func (t *Tracer) TraceAPICall(
	ctx context.Context, targetID string, spanName string, opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	t.liveSpansMu.Lock()
	defer t.liveSpansMu.Unlock()

	opts = append(opts, trace.WithAttributes(t.metadata...))

	ls := t.liveSpans[targetID]
	if ls == nil {
		t.logger.Infof("TraceAPICall: no live span spanName: %q targetID: %q", spanName, targetID)
		sCtx, span := t.Start(ctx, spanName, opts...)

		return sCtx, &SpanLogger{Span: span, logger: t.logger, spanName: spanName}
	}

	spanCtx := trace.SpanContextFromContext(ls.ctx)
	traceID := GetTraceID(spanCtx)
	t.logger.Infof("TraceAPICall: with live span spanName: %q traceID: %q targetID: %q", spanName, traceID, targetID)
	sCtx, span := t.Start(ls.ctx, spanName, opts...)

	return sCtx, &SpanLogger{Span: span, logger: t.logger, spanName: spanName}
}

// TraceNavigation is only to be used when a frame has navigated.
// It records a new liveSpan for the given targetID which identifies the frame that
// has navigated. If there was already a liveSpan for the given targetID, it is closed
// before creating the new one, otherwise it's the caller's responsibility to close the
// generated span.
// Posterior calls to TraceEvent or TraceAPICall given the same targetID will try to
// associate new spans for these actions to the liveSpan created in this call.
func (t *Tracer) TraceNavigation(
	ctx context.Context, targetID string, opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	t.liveSpansMu.Lock()
	defer t.liveSpansMu.Unlock()

	// TODO: Should we keep track of all spans, even ones that are closed, to
	// ensure we associate web vitals to the spans in the current iteration?

	ls := t.liveSpans[targetID]
	if ls != nil {
		// If there is a previous live span
		// for the targetID, end it
		ls.span.End()
	} else {
		ls = &liveSpan{}
	}

	opts = append(opts, trace.WithAttributes(t.metadata...))

	spanName := "navigation"
	ls.ctx, ls.span = t.Start(ctx, spanName, opts...)
	t.liveSpans[targetID] = ls

	spanCtx := trace.SpanContextFromContext(ls.ctx)
	traceID := GetTraceID(spanCtx)
	t.logger.Infof("TraceNavigation: spanName: %q traceID: %q targetID: %q", spanName, traceID, targetID)

	return ls.ctx, &SpanLogger{Span: ls.span, logger: t.logger, spanName: spanName}
}

// TraceEvent creates a new span representing the specified event and associates it with the current
// liveSpan for the given targetID only if the spanID matches with the liveSpan.
// It is the caller's responsibility to close the generated span.
//
// If no liveSpan is found for the given targetID, the action is ignored and a noopSpan is returned.
// If the given spanID does not match the one for the current liveSpan associated with the targetID,
// it means the specified target has navigated, generating a new span for that navigation, therefore
// the event is not associated with that span, and instead a noopSpan is returned.
func (t *Tracer) TraceEvent(
	ctx context.Context, targetID string, eventName string, spanID string, opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	t.liveSpansMu.Lock()
	defer t.liveSpansMu.Unlock()

	ls := t.liveSpans[targetID]
	if ls == nil {
		t.logger.Infof("TraceEvent: no live span spanName: %q targetID: %q", eventName, targetID)

		// If there is not a liveSpan for the given targetID,
		// avoid associating the event with the root span possibly
		// wrapped in ctx, and instead return a noopSpan
		return ctx, NoopSpan{}
	}

	sid := ls.span.SpanContext().SpanID().String()
	if sid != spanID {
		t.logger.Infof("TraceEvent: no sid spanName: %q targetID: %q", eventName, targetID)

		// If the given spanID does not match the current liveSpan for
		// targetID, it means the target has navigated to a different
		// page than the one the event should be associated with.
		// Therefore avoid associating the event with the current span,
		// and return a noopSpan instead
		return ctx, NoopSpan{}
	}

	opts = append(opts, trace.WithAttributes(t.metadata...))

	spanCtx := trace.SpanContextFromContext(ls.ctx)
	traceID := GetTraceID(spanCtx)
	t.logger.Infof("TraceEvent: spanName: %q traceID: %q targetID: %q", eventName, traceID, targetID)
	sCtx, span := t.Start(ls.ctx, eventName, opts...)

	return sCtx, &SpanLogger{Span: span, logger: t.logger, spanName: eventName}
}

func buildMetadataAttributes(metadata map[string]string) []attribute.KeyValue {
	meta := make([]attribute.KeyValue, 0, len(metadata))
	for mk, mv := range metadata {
		meta = append(meta, attribute.String(mk, mv))
	}

	return meta
}

// NoopSpan represents a noop span.
type NoopSpan struct {
	trace.Span
}

// SpanContext returns a void span context.
func (NoopSpan) SpanContext() trace.SpanContext { return trace.SpanContext{} }

// IsRecording returns false.
func (NoopSpan) IsRecording() bool { return false }

// SetStatus is noop.
func (NoopSpan) SetStatus(codes.Code, string) {}

// SetError is noop.
func (NoopSpan) SetError(bool) {}

// SetAttributes is noop.
func (NoopSpan) SetAttributes(...attribute.KeyValue) {}

// End is noop.
func (NoopSpan) End(...trace.SpanEndOption) {}

// RecordError is noop.
func (NoopSpan) RecordError(error, ...trace.EventOption) {}

// AddEvent is noop.
func (NoopSpan) AddEvent(string, ...trace.EventOption) {}

// SetName is noop.
func (NoopSpan) SetName(string) {}

// TracerProvider returns a noop tracer provider.
func (NoopSpan) TracerProvider() trace.TracerProvider { return trace.NewNoopTracerProvider() }

// SpanLogger is a Span that will log the method calls.
type SpanLogger struct {
	trace.Span
	logger   logrus.FieldLogger
	spanName string
}

// SetStatus will log some info before calling the underlying SetStatus.
func (i *SpanLogger) SetStatus(code codes.Code, description string) {
	traceID := GetTraceID(i.SpanContext())
	i.logger.Infof("SetStatus: spanName: %q traceID: %q code: %q description: %q", i.spanName, traceID, code, description)

	i.Span.SetStatus(code, description)
}

// End will log some info before calling the underlying End.
func (i *SpanLogger) End(options ...trace.SpanEndOption) {
	traceID := GetTraceID(i.SpanContext())
	i.logger.Infof("End: spanName: %q traceID: %q", i.spanName, traceID)

	i.Span.End(options...)
}

// RecordError will log some info before calling the underlying RecordError.
func (i *SpanLogger) RecordError(err error, options ...trace.EventOption) {
	traceID := GetTraceID(i.SpanContext())
	i.logger.Infof("SetStatus: spanName: %q traceID: %q err: %q", i.spanName, traceID, err)

	i.Span.RecordError(err, options...)
}
