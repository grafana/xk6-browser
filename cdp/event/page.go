package event

type CDPName string

// TODO: Just use cdproto event names directly instead?
const (
	PageNavigated CDPName = "Page.navigated"
)
