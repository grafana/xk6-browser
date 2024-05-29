package k6test

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"

	"github.com/grafana/xk6-browser/env"
	"github.com/grafana/xk6-browser/k6ext"

	"go.k6.io/k6/event"
	k6event "go.k6.io/k6/event"
	k6common "go.k6.io/k6/js/common"
	k6eventloop "go.k6.io/k6/js/eventloop"
	k6modulestest "go.k6.io/k6/js/modulestest"
	"go.k6.io/k6/lib"
	k6lib "go.k6.io/k6/lib"
	k6executor "go.k6.io/k6/lib/executor"
	k6testutils "go.k6.io/k6/lib/testutils"
	k6trace "go.k6.io/k6/lib/trace"
	k6metrics "go.k6.io/k6/metrics"
)

// VU is a k6 VU instance.
// TODO: Do we still need this VU wrapper?
// ToGojaValue can be a helper function that takes a goja.Runtime (although it's
// not much of a helper from calling ToValue(i) directly...), and we can access
// EventLoop from modulestest.Runtime.EventLoop.
type VU struct {
	*k6modulestest.VU
	Loop      *k6eventloop.EventLoop
	toBeState *k6lib.State
	samples   chan k6metrics.SampleContainer
	TestRT    *k6modulestest.Runtime
}

// ToGojaValue is a convenience method for converting any value to a goja value.
func (v *VU) ToGojaValue(i any) goja.Value { return v.Runtime().ToValue(i) }

// ActivateVU mimicks activation of the VU as in k6.
// It transitions the VU from the init stage to the execution stage by
// setting the VU's state to the state that was passed to NewVU.
func (v *VU) ActivateVU() {
	v.VU.StateField = v.toBeState
	v.VU.InitEnvField = nil
}

// AssertSamples asserts each sample VU received since AssertSamples
// is last called, then it returns the number of received samples.
func (v *VU) AssertSamples(assertSample func(s k6metrics.Sample)) int {
	var n int
	for _, bs := range k6metrics.GetBufferedSamples(v.samples) {
		for _, s := range bs.GetSamples() {
			assertSample(s)
			n++
		}
	}
	return n
}

// WithScenarioName is used to set the scenario name in the IterData
// for the 'IterStart' event.
type WithScenarioName = string

// WithVUID is used to set the VU id in the IterData for the 'IterStart'
// event.
type WithVUID = uint64

// WithIteration is used to set the iteration in the IterData for the
// 'IterStart' event.
type WithIteration = int64

// StartIteration generates a new IterStart event through the VU event system.
//
// opts can be used to parameterize the iteration data such as:
//   - WithScenarioName: sets the scenario name (default is 'default').
//   - WithVUID: sets the VUID (default 1).
//   - WithIteration: sets the iteration (default 0).
func (v *VU) StartIteration(tb testing.TB, opts ...any) {
	tb.Helper()
	v.iterEvent(tb, k6event.IterStart, "IterStart", opts...)
}

// EndIteration generates a new IterEnd event through the VU event system.
//
// opts can be used to parameterize the iteration data such as:
//   - WithScenarioName: sets the scenario name (default is 'default').
//   - WithVUID: sets the VUID (default 1).
//   - WithIteration: sets the iteration (default 0).
func (v *VU) EndIteration(tb testing.TB, opts ...any) {
	tb.Helper()
	v.iterEvent(tb, k6event.IterEnd, "IterEnd", opts...)
}

// iterEvent generates an iteration event for the VU.
func (v *VU) iterEvent(tb testing.TB, eventType event.Type, eventName string, opts ...any) {
	tb.Helper()

	data := k6event.IterData{
		Iteration:    0,
		VUID:         1,
		ScenarioName: "default",
	}

	for _, opt := range opts {
		switch opt := opt.(type) {
		case WithScenarioName:
			data.ScenarioName = opt
		case WithVUID:
			data.VUID = opt
		case WithIteration:
			data.Iteration = opt
		}
	}

	events, ok := v.EventsField.Local.(*k6event.System)
	require.True(tb, ok, "want *k6event.System; got %T", events)
	waitDone := events.Emit(&k6event.Event{
		Type: eventType,
		Data: data,
	})
	require.NoError(tb, waitDone(context.Background()), "error waiting on %s done", eventName)
}

// WithSamples is used to indicate we want to use a bidirectional channel
// so that the test can read the metrics being emitted to the channel.
type WithSamples chan k6metrics.SampleContainer

// WithTracerProvider allows to set the VU TracerProvider.
type WithTracerProvider k6lib.TracerProvider

// NewVU returns a mock k6 VU.
//
// opts can be one of the following:
//   - WithSamples: a bidirectional channel that will be used to emit metrics.
//   - env.LookupFunc: a lookup function that will be used to lookup environment variables.
//   - WithTracerProvider: a TracerProvider that will be set as the VU TracerProvider.
func NewVU(tb testing.TB, opts ...any) *VU {
	tb.Helper()

	var (
		samples                             = make(chan k6metrics.SampleContainer, 1000)
		lookupFunc                          = env.EmptyLookup
		tracerProvider k6lib.TracerProvider = k6trace.NewNoopTracerProvider()
	)
	for _, opt := range opts {
		switch opt := opt.(type) {
		case WithSamples:
			samples = opt
		case env.LookupFunc:
			lookupFunc = opt
		case WithTracerProvider:
			tracerProvider = opt
		}
	}

	logger := k6testutils.NewLogger(tb)

	testRT := k6modulestest.NewRuntime(tb)
	testRT.VU.InitEnvField.LookupEnv = lookupFunc
	testRT.VU.EventsField = k6common.Events{
		Global: k6event.NewEventSystem(100, logger),
		Local:  k6event.NewEventSystem(100, logger),
	}

	state := &k6lib.State{
		Options: k6lib.Options{
			MaxRedirects: null.IntFrom(10),
			UserAgent:    null.StringFrom("TestUserAgent"),
			Throw:        null.BoolFrom(true),
			SystemTags:   &k6metrics.DefaultSystemTagSet,
			Batch:        null.IntFrom(20),
			BatchPerHost: null.IntFrom(20),
			// HTTPDebug:    null.StringFrom("full"),
			Scenarios: k6lib.ScenarioConfigs{
				"default": &TestExecutor{
					BaseConfig: k6executor.BaseConfig{
						Options: &k6lib.ScenarioOptions{
							Browser: map[string]any{
								"type": "chromium",
							},
						},
					},
				},
			},
		},
		Logger:     logger,
		BufferPool: k6lib.NewBufferPool(),
		Samples:    samples,
		Tags: k6lib.NewVUStateTags(
			testRT.VU.InitEnvField.Registry.RootTagSet().With("group", lib.RootGroupPath),
		),
		BuiltinMetrics: k6metrics.RegisterBuiltinMetrics(k6metrics.NewRegistry()),
		TracerProvider: tracerProvider,
	}

	ctx := k6ext.WithVU(testRT.VU.CtxField, testRT.VU)
	ctx = k6lib.WithScenarioState(ctx, &k6lib.ScenarioState{Name: "default"})
	testRT.VU.CtxField = ctx

	return &VU{VU: testRT.VU, Loop: testRT.EventLoop, toBeState: state, samples: samples, TestRT: testRT}
}
