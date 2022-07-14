package event

import (
	"context"
	"sync"
)

type Watcher struct {
	ctx    context.Context
	subsMu sync.RWMutex
	subs   map[CDPName][]chan *Event
}

func NewWatcher(ctx context.Context) *Watcher {
	return &Watcher{
		ctx:  ctx,
		subs: make(map[CDPName][]chan *Event),
	}
}

// func (w *eventWatcher) subscribe(sessionID, frameID string, evt *Event) <-chan *Event {
// TODO: Handle event unsubscriptions
func (w *Watcher) Subscribe(evt CDPName) <-chan *Event {
	w.subsMu.Lock()
	defer w.subsMu.Unlock()
	ch := make(chan *Event, 1)
	w.subs[evt] = append(w.subs[evt], ch)
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

func (w *Watcher) OnEventReceived(evt *Event) {
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
