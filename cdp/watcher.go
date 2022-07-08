package cdp

type EventWatcher struct {
	// TODO: Some other extensible way of indexing subscribers?
	subs map[string]map[string]chan<- Event
}

func (ew *EventWatcher) Subscribe(sessionID, frameID string, evt Event) (<-chan Event, error) {
}
