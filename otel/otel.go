// Package otel provides higher level APIs around Open Telemetry instrumentation.
package otel

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/grafana/xk6-browser/log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	serviceName = "k6-browser"
	tracerName  = "browser"
)

// ErrUnsupportedProto indicates that the defined exporter protocol is not supported.
var ErrUnsupportedProto = errors.New("unsupported protocol")

// TraceProvider provides methods for tracers initialization and shutdown of the
// processing pipeline.
type TraceProvider interface {
	Tracer(name string, options ...trace.TracerOption) trace.Tracer
	Shutdown(ctx context.Context) error
}

type (
	traceProvShutdownFunc func(ctx context.Context) error
)

type traceProvider struct {
	trace.TracerProvider

	noop bool

	shutdown traceProvShutdownFunc
}

// NewTraceProvider creates a new trace provider.
func NewTraceProvider(
	ctx context.Context, proto, endpoint string, insecure bool,
) (TraceProvider, error) {
	client, err := newClient(proto, endpoint, insecure)
	if err != nil {
		return nil, fmt.Errorf("creating exporter client: %w", err)
	}

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("creating exporter: %w", err)
	}

	prov := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(newResource()),
	)

	otel.SetTracerProvider(prov)

	return &traceProvider{
		TracerProvider: prov,
		shutdown:       prov.Shutdown,
	}, nil
}

func newResource() *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	)
}

func newClient(proto, endpoint string, insecure bool) (otlptrace.Client, error) {
	// TODO: Support gRPC
	switch strings.ToLower(proto) {
	case "http":
		return newHTTPClient(endpoint, insecure), nil
	default:
		return nil, ErrUnsupportedProto
	}
}

func newHTTPClient(endpoint string, insecure bool) otlptrace.Client {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
	}
	if insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	return otlptracehttp.NewClient(opts...)
}

// NewNoopTraceProvider creates a new noop trace provider.
func NewNoopTraceProvider() TraceProvider {
	prov := trace.NewNoopTracerProvider()

	otel.SetTracerProvider(prov)

	return &traceProvider{
		TracerProvider: prov,
		noop:           true,
	}
}

// Shutdown shuts down TracerProvider releasing any held computational resources.
// After Shutdown is called, all methods are no-ops.
func (tp *traceProvider) Shutdown(ctx context.Context) error {
	if tp.noop {
		return nil
	}

	return tp.shutdown(ctx)
}

// Trace generates a trace span and a context containing the generated span.
// If the input context already contains a span, the generated spain will be a child of that span
// otherwise it will be a root span. This behavior can be overridden by providing `WithNewRoot()`
// as a SpanOption, causing the newly-created Span to be a root span even if `ctx` contains a Span.
// When creating a Span it is recommended to provide all known span attributes using the `WithAttributes()`
// SpanOption as samplers will only have access to the attributes provided when a Span is created.
// Any Span that is created MUST also be ended. This is the responsibility of the user. Implementations of
// this API may leak memory or other resources if Spans are not ended.
func Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, spanName, opts...)
}

type liveSpan struct {
	ctx  context.Context
	span trace.Span
}

// TODO: Need to reset `liveSpans` on a new iteration.
// TODO: Move out of package scope.
var liveSpansMu = &sync.RWMutex{}
var liveSpans = map[string]*liveSpan{}

// TracePageNavigation is to only be used when a frame has been navigated.
// This span should record when a navigation starts until either:
//  1. A new navigation occurs.
//  2. The test iteration ends.
//  3. The whole test run ends.
//
// If we do not correctly end this span, then child spans will end up being
// linked to the root span. For one main frame, there is only ever
// one inflight PageNavigation span.
func TracePageNavigation(ctx context.Context, targetID string, url string, opts ...trace.SpanStartOption) trace.Span {
	liveSpansMu.Lock()
	defer liveSpansMu.Unlock()

	// TODO: Maybe we should keep track of all spans even ones that are closed to
	// ensure we associate web vitals to the spans in the current iteration.
	ls := liveSpans[targetID]
	if ls == nil {
		ls = &liveSpan{}
	}

	ls.ctx, ls.span = Trace(ctx, url, opts...)
	liveSpans[targetID] = ls

	return ls.span
}

// AddEventToTrace will add the given event to the current PageNavigation
// span. It's to be used for async events such as web vitals.
//
// If the targetID passed in doesn't match the live PageNavigation's
// targetID then the event will be ignored (but an error will be logged).
//
// If the spanID passed in doesn't match the live PageNavigation's
// spanID then the event will be ignored (but an error will be logged).
func AddEventToTrace(logger *log.Logger, targetID string, eventName string, spanID string, options ...trace.EventOption) {
	liveSpansMu.Lock()
	defer liveSpansMu.Unlock()

	ls := liveSpans[targetID]
	if ls == nil {
		// TODO: Should we try to add this event to the iteration trace instead when this occurs?
		logger.Errorf("AddEventToTrace", "missing targetID %q, skipping event for %q with %v, have %q", targetID, eventName, spanID, options)
		return
	}

	sid := ls.span.SpanContext().SpanID().String()
	if sid != spanID {
		// TODO: Should we try to add this event to the iteration trace instead when this occurs?
		logger.Errorf("AddEventToTrace", "skipping %q event for %q with %v, have %q", eventName, spanID, options, sid)
		return
	}

	ls.span.AddEvent(eventName, options...)
}

// TraceAPICall will attach a new span to the associated current
// PageNavigation span.
//
// The context will be used if a live PageNavigation span is not
// found with the given targetID.
//
// TODO: Could retrieve the ctx given the targetID in the mapping
// layer allowing API calls to just work with the Trace method above.
func TraceAPICall(ctx context.Context, targetID string, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	liveSpansMu.Lock()
	defer liveSpansMu.Unlock()

	ls := liveSpans[targetID]
	if ls == nil {
		return otel.Tracer(tracerName).Start(ctx, spanName, opts...)
	}

	return otel.Tracer(tracerName).Start(ls.ctx, spanName, opts...)
}
