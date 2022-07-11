package cdp

import "github.com/chromedp/cdproto"

type Watcher struct {
	// TODO: Some other extensible way of indexing subscribers?
	subs map[string]map[string]chan<- Event
}

func (w *Watcher) SubscribeToEvent(sessionID, frameID string, evt Event) (<-chan Event, error) {
}

func (w *Watcher) SubscribeToMessage(msgID int64) (<-chan *cdproto.Message, error) {
}
