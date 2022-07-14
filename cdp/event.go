package cdp

import (
	"context"
	"sync"
)

// TODO: Move this into subpackages? cdp/event/page, cdp/event/network, etc.
// Or expose global structs for each CDP domain?
// import "cdp/event"
// event.Page.Navigated, event.Target.AttachedToTarget, event.Network.LoadingFinished
type Event struct {
	Name string
	Data interface{}
}

type eventWatcher struct {
	ctx    context.Context
	subsMu sync.RWMutex
	subs   map[string][]chan *Event
}

func newEventWatcher(ctx context.Context) *eventWatcher {
	return &eventWatcher{
		ctx:  ctx,
		subs: make(map[string][]chan *Event),
	}
}

// func (w *eventWatcher) subscribe(sessionID, frameID string, evt *Event) <-chan *Event {
func (w *eventWatcher) subscribe(evt *Event) <-chan *Event {
	w.subsMu.Lock()
	defer w.subsMu.Unlock()
	ch := make(chan *Event, 1)
	w.subs[evt.Name] = append(w.subs[evt.Name], ch)
	return ch
}

// func (w *Watcher) subscribeToMessage(msgID int64) <-chan *cdproto.Message {
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
	subs, ok := w.subs[evt.Name]
	if !ok {
		return
	}

	for _, ch := range subs {
		select {
		case ch <- evt:
		case <-w.ctx.Done():
			return
		default:
		}
	}
}

// func (w *Watcher) onMessageReceived(msg *cdproto.Message) {
// }
