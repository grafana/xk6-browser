package cdp

import (
	"sync"

	"github.com/chromedp/cdproto"
)

type Watcher struct {
	// TODO: Some other extensible way of indexing subscribers?
	eventMu   sync.RWMutex
	eventSubs map[string]map[string]chan *Event
	msgMu     sync.RWMutex
	msgSubs   map[int64]chan *cdproto.Message
}

func NewWatcher() *Watcher {
	return &Watcher{
		eventSubs: make(map[string]map[string]chan *Event),
		msgSubs:   make(map[int64]chan *cdproto.Message),
	}
}

func (w *Watcher) Subscribe(sessionID, frameID string, evt *Event) <-chan *Event {
	w.eventMu.RLock()
	if ch, ok := w.eventSubs[sessionID][frameID]; ok {
		w.eventMu.RUnlock()
		return ch
	}
	w.eventMu.Lock()
	defer w.eventMu.Unlock()
	ch := make(chan *Event)
	w.eventSubs[sessionID][frameID] = ch
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

func (w *Watcher) onEventReceived(evt *Event) {
}

// func (w *Watcher) onMessageReceived(msg *cdproto.Message) {
// }
