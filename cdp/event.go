package cdp

import (
	"context"
	"fmt"
	"sync"

	"github.com/chromedp/cdproto"
	cdpext "github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/target"
)

type LifecycleEvent string

const (
	LifecycleEventLoad             LifecycleEvent = "load"
	LifecycleEventDOMContentLoaded LifecycleEvent = "domcontentloaded"
	LifecycleEventNetworkIdle      LifecycleEvent = "networkidle"
)

type Event struct {
	Name      cdproto.MethodType
	Data      interface{}
	sessionID target.SessionID
	frameID   cdpext.FrameID
}

type EventHandler func(context.Context, *Event)

type subKey struct {
	sessionID, targetID string
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

	key := subKey{string(evt.sessionID), string(evt.frameID)}
	if ch, ok := subs[key]; ok {
		fmt.Printf(">>> notifying subscriber %s of event %s with data: %#+v\n", key, evt.Name, evt.Data)
		select {
		case ch <- evt:
		case <-w.ctx.Done():
			return
		default:
			// TODO: Log warning of skipped event
		}
	}
}
