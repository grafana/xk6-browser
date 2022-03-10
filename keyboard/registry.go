package keyboard

import (
	"fmt"
	"sync"
)

//nolint:gochecknoglobals
var (
	layouts = make(map[string]Layout)
	mu      sync.RWMutex
)

// LayoutFor returns the keyboard layout registered with name.
func LayoutFor(name string) Layout {
	mu.RLock()
	defer mu.RUnlock()
	return layouts[name]
}

// Register the given keyboard layout.
// This function panics if a keyboard layout with the same name is already registered.
func register(lang string, validKeys map[Key]bool, keys map[Key]Definition) {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := layouts[lang]; ok {
		panic(fmt.Sprintf("keyboard layout already registered: %s", lang))
	}
	layouts[lang] = Layout{
		Name:      lang,
		ValidKeys: validKeys,
		Keys:      keys,
	}
}
