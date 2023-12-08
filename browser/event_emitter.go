package browser

import (
	"context"
	"sync"
)

const (
	// EventBrowserDisconnected is emitted when the browser is disconnected.
	EventBrowserDisconnected string = "disconnected"

	// EventContextPage is emitted when a new page in context is created.
	EventContextPage string = "page"

	// EventConnectionClose is emitted when the connection to the browser is closed.
	EventConnectionClose string = "close"

	// EventFrameNavigation is emitted when a frame is navigated.
	EventFrameNavigation string = "navigation"

	// EventFrameAddLifecycle is emitted when a new lifecycle event is added.
	EventFrameAddLifecycle string = "addlifecycle"

	// EventPageClose is emitted when a page is closed.
	EventPageClose string = "close"

	// EventPageConsole is emitted when a console message is added.
	EventPageConsole string = "console"

	// EventPageCrash is emitted when a page crashes.
	EventPageCrash string = "crash"

	// EventPageDialog is emitted when a dialog is opened.
	EventPageDialog string = "dialog"

	// EventPageDownload is emitted when a download is started.
	EventPageDownload string = "download"

	// EventPageFilechooser is emitted when a file chooser is opened.
	EventPageFilechooser string = "filechooser"

	// EventPageFrameAttached is emitted when a frame is attached.
	EventPageFrameAttached string = "frameattached"

	// EventPageFrameDetached is emitted when a frame is detached.
	EventPageFrameDetached string = "framedetached"

	// EventPageFrameNavigated is emitted when a frame is navigated.
	EventPageFrameNavigated string = "framenavigated"

	// EventPageError is emitted when a page error occurs.
	EventPageError string = "pageerror"

	// EventPagePopup is emitted when a popup is opened.
	EventPagePopup string = "popup"

	// EventPageRequest is emitted when a request is made.
	EventPageRequest string = "request"

	// EventPageRequestFailed is emitted when a request fails.
	EventPageRequestFailed string = "requestfailed"

	// EventPageRequestFinished is emitted when a request is finished.
	EventPageRequestFinished string = "requestfinished"

	// EventPageResponse is emitted when a response is received.
	EventPageResponse string = "response"

	// EventPageWebSocket is emitted when a websocket is created.
	EventPageWebSocket string = "websocket"

	// EventPageWorker is emitted when a worker is created.
	EventPageWorker string = "worker"

	// EventSessionClosed is emitted when a session is closed.
	EventSessionClosed string = "close"

	// EventWorkerClose is emitted when a worker is closed.
	EventWorkerClose string = "close"
)

// Event as emitted by an EventEmiter.
type Event struct {
	typ  string
	data any
}

// NavigationEvent is emitted when we receive a Page.frameNavigated or
// Page.navigatedWithinDocument CDP event.
// See:
// - https://chromedevtools.github.io/devtools-protocol/tot/Page/#event-frameNavigated
// - https://chromedevtools.github.io/devtools-protocol/tot/Page/#event-navigatedWithinDocument
type NavigationEvent struct {
	newDocument *DocumentInfo
	url         string
	name        string
	err         error
}

type queue struct {
	writeMutex sync.Mutex
	write      []Event
	readMutex  sync.Mutex
	read       []Event
}

type eventHandler struct {
	ctx   context.Context
	ch    chan Event
	queue *queue
}

// EventEmitter that all event emitters need to implement.
type EventEmitter interface {
	emit(event string, data any)
	on(ctx context.Context, events []string, ch chan Event)
	onAll(ctx context.Context, ch chan Event)
}

// syncFunc functions are passed through the syncCh for synchronously handling
// eventHandler requests.
type syncFunc func() (done chan struct{})

// BaseEventEmitter emits events to registered handlers.
type BaseEventEmitter struct {
	handlers    map[string][]*eventHandler
	handlersAll []*eventHandler

	queues map[chan Event]*queue

	syncCh chan syncFunc
	ctx    context.Context
}

// NewBaseEventEmitter creates a new instance of a base event emitter.
func NewBaseEventEmitter(ctx context.Context) BaseEventEmitter {
	bem := BaseEventEmitter{
		handlers: make(map[string][]*eventHandler),
		syncCh:   make(chan syncFunc),
		ctx:      ctx,
		queues:   make(map[chan Event]*queue),
	}
	go bem.syncAll(ctx)
	return bem
}

// syncAll receives work requests from BaseEventEmitter methods
// and processes them one at a time for synchronization.
//
// It returns when the BaseEventEmitter context is done.
func (e *BaseEventEmitter) syncAll(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case fn := <-e.syncCh:
			// run the function and signal when it's done
			done := fn()
			done <- struct{}{}
		}
	}
}

// sync is a helper for sychronized access to the BaseEventEmitter.
func (e *BaseEventEmitter) sync(fn func()) {
	done := make(chan struct{})
	select {
	case <-e.ctx.Done():
		return
	case e.syncCh <- func() chan struct{} {
		fn()
		return done
	}:
	}
	// wait for the function to return
	<-done
}

func (e *BaseEventEmitter) emit(event string, data any) {
	emitEvent := func(eh *eventHandler) {
		eh.queue.readMutex.Lock()
		defer eh.queue.readMutex.Unlock()

		// We try to read from the read queue (queue.read).
		// If there isn't anything on the read queue, then there must
		// be something being populated by the synched emitTo
		// func below.
		// Swap around the read queue with the write queue.
		// Queue is now being populated again by emitTo, and all
		// emitEvent goroutines can continue to consume from
		// the read queue until that is again depleted.
		if len(eh.queue.read) == 0 {
			eh.queue.writeMutex.Lock()
			eh.queue.read, eh.queue.write = eh.queue.write, eh.queue.read
			eh.queue.writeMutex.Unlock()
		}

		select {
		case eh.ch <- eh.queue.read[0]:
			eh.queue.read = eh.queue.read[1:]
		case <-eh.ctx.Done():
			// TODO: handle the error
		}
	}
	emitTo := func(handlers []*eventHandler) (updated []*eventHandler) {
		for i := 0; i < len(handlers); {
			handler := handlers[i]
			select {
			case <-handler.ctx.Done():
				handlers = append(handlers[:i], handlers[i+1:]...)
				continue
			default:
				handler.queue.writeMutex.Lock()
				handler.queue.write = append(handler.queue.write, Event{typ: event, data: data})
				handler.queue.writeMutex.Unlock()

				go emitEvent(handler)
				i++
			}
		}
		return handlers
	}
	e.sync(func() {
		e.handlers[event] = emitTo(e.handlers[event])
		e.handlersAll = emitTo(e.handlersAll)
	})
}

// On registers a handler for a specific event.
func (e *BaseEventEmitter) on(ctx context.Context, events []string, ch chan Event) {
	e.sync(func() {
		q, ok := e.queues[ch]
		if !ok {
			q = &queue{}
			e.queues[ch] = q
		}

		for _, event := range events {
			e.handlers[event] = append(e.handlers[event], &eventHandler{ctx: ctx, ch: ch, queue: q})
		}
	})
}

// OnAll registers a handler for all events.
func (e *BaseEventEmitter) onAll(ctx context.Context, ch chan Event) {
	e.sync(func() {
		q, ok := e.queues[ch]
		if !ok {
			q = &queue{}
			e.queues[ch] = q
		}

		e.handlersAll = append(e.handlersAll, &eventHandler{ctx: ctx, ch: ch, queue: q})
	})
}
