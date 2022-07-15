package cdp

import (
	"context"
	"fmt"
	"sync"

	"github.com/chromedp/cdproto"
)

type Event struct {
	Name cdproto.MethodType
	Data interface{}
}

type eventWatcher struct {
	ctx    context.Context
	subsMu sync.RWMutex
	subs   map[cdproto.MethodType][]chan *Event
}

func newEventWatcher(ctx context.Context) *eventWatcher {
	return &eventWatcher{
		ctx:  ctx,
		subs: make(map[cdproto.MethodType][]chan *Event),
	}
}

// func (w *eventeventWatcher) subscribe(sessionID, frameID string, evt *event) <-chan *event {
// TODO: Handle event unsubscriptions
func (w *eventWatcher) subscribe(events ...cdproto.MethodType) <-chan *Event {
	w.subsMu.Lock()
	defer w.subsMu.Unlock()
	ch := make(chan *Event, 1)
	for _, evt := range events {
		w.subs[evt] = append(w.subs[evt], ch)
		fmt.Printf(">>> subscribed to event %s\n", evt)
	}
	return ch
}

// func (w *eventWatcher) subscribeToMessage(msgID int64) <-chan *cdproto.Message {
// 	w.msgMu.RLock()
// 	if ch, ok := w.msgSubs[msgID]; ok {
// 		w.msgMu.RUnlock()
// 		return ch
// 	}
// 	w.msgMu.Lock()
// 	defer w.msgMu.Unlock()
// 	ch := make(chan *cdproto.Message)
// 	w.msgSubs[msgID] = ch
// 	return ch
// }

func (w *eventWatcher) onEventReceived(evt *Event) {
	w.subsMu.RLock()
	defer w.subsMu.RUnlock()
	subs, ok := w.subs[evt.Name]
	if !ok {
		return
	}

	for i, ch := range subs {
		fmt.Printf(">>> notifying subscriber %d of event %s\n", i, evt.Name)
		select {
		case ch <- evt:
		case <-w.ctx.Done():
			return
		default:
			// TODO: Log warning of skipped event
		}
	}
}

// func (w *eventWatcher) onMessageReceived(msg *cdproto.Message) {
// }
