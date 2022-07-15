package cdp

import (
	"context"
	"fmt"
	"sync"

	"github.com/chromedp/cdproto"
)

type LifecycleEvent string

const (
	LifecycleEventLoad             LifecycleEvent = "load"
	LifecycleEventDOMContentLoaded LifecycleEvent = "domcontentloaded"
	LifecycleEventNetworkIdle      LifecycleEvent = "networkidle"
)

type Event struct {
	Name cdproto.MethodType
	Data interface{}
}

type subKey struct {
	sessionID, frameID string
}

type eventWatcher struct {
	ctx    context.Context
	subsMu sync.RWMutex
	subs   map[cdproto.MethodType]map[subKey]chan *Event
}

func newEventWatcher(ctx context.Context) *eventWatcher {
	return &eventWatcher{
		ctx:  ctx,
		subs: make(map[cdproto.MethodType]map[subKey]chan *Event),
	}
}

// func (w *eventeventWatcher) subscribe(sessionID, frameID string, evt *event) <-chan *event {
// TODO: Handle event unsubscriptions
// func (w *eventWatcher) subscribe(sessionID, frameID string, events ...cdproto.MethodType) <-chan *Event {
func (w *eventWatcher) subscribe(
	sessionID, frameID string, events ...cdproto.MethodType,
) (<-chan *Event, func()) {
	w.subsMu.Lock()
	defer w.subsMu.Unlock()
	evtCh := make(chan *Event, 1)
	key := subKey{sessionID, frameID}
	for _, evtName := range events {
		if _, ok := w.subs[evtName]; !ok {
			w.subs[evtName] = make(map[subKey]chan *Event)
		}
		w.subs[evtName][key] = evtCh
		fmt.Printf(">>> subscribed to event %s\n", evtName)
	}

	unsub := func() {
		close(evtCh)
		w.subsMu.Lock()
		defer w.subsMu.Unlock()
		for _, evtName := range events {
			delete(w.subs[evtName], key)
		}
	}

	return evtCh, unsub
}

func (w *eventWatcher) notify(evt *Event) {
	w.subsMu.RLock()
	defer w.subsMu.RUnlock()
	subs, ok := w.subs[evt.Name]
	if !ok {
		return
	}

	for key, ch := range subs {
		fmt.Printf(">>> notifying subscriber %s of event %s\n", key, evt.Name)
		select {
		case ch <- evt:
		case <-w.ctx.Done():
			return
		default:
			// TODO: Log warning of skipped event
		}
	}
}
