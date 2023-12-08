// Package keyboard provides keyboard key interpretation and layout validation.
package keyboard

import (
	"fmt"
	"sync"
)

// Input represents a keyboard input.
type Input string

// Definition represents a keyboard key definition.
type Definition struct {
	Code                   string
	Key                    string
	KeyCode                int64
	KeyCodeWithoutLocation int64
	ShiftKey               string
	ShiftKeyCode           int64
	Text                   string
	Location               int64
}

// Layout represents a keyboard layout.
type Layout struct {
	ValidKeys map[Input]bool
	Keys      map[Input]Definition
}

// KeyDefinition returns true with the key definition of a given key input.
// It returns false and an empty key definition if it cannot find the key.
func (kl Layout) KeyDefinition(key Input) (Definition, bool) {
	for _, d := range kl.Keys {
		if d.Key == string(key) {
			return d, true
		}
	}
	return Definition{}, false
}

// ShiftKeyDefinition returns shift key definition of a given key input.
// It returns an empty key definition if it cannot find the key.
func (kl Layout) ShiftKeyDefinition(key Input) Definition {
	for _, d := range kl.Keys {
		if d.ShiftKey == string(key) {
			return d
		}
	}
	return Definition{}
}

//nolint:gochecknoglobals
var (
	kbdLayouts = make(map[string]Layout)
	mx         sync.RWMutex
)

// GetLayout returns the keyboard layout registered with name.
func GetLayout(name string) Layout {
	mx.RLock()
	defer mx.RUnlock()
	return kbdLayouts[name]
}

func init() {
	initUS()
}

// Register the given keyboard layout.
// This function panics if a keyboard layout with the same name is already registered.
func register(lang string, validKeys map[Input]bool, keys map[Input]Definition) {
	mx.Lock()
	defer mx.Unlock()

	if _, ok := kbdLayouts[lang]; ok {
		panic(fmt.Sprintf("keyboard layout already registered: %s", lang))
	}
	kbdLayouts[lang] = Layout{ValidKeys: validKeys, Keys: keys}
}
