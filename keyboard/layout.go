package keyboard

// ModifierKey is a key modifier like ALT, CTRL, or Shift.
type ModifierKey int64

const (
	// ModifierKeyAlt is the ALT key modifier.
	ModifierKeyAlt ModifierKey = 1 << iota
	// ModifierKeyControl is the CTRL key modifier.
	ModifierKeyControl
	// ModifierKeyMeta is the meta key modifier.
	ModifierKeyMeta
	// ModifierKeyShift is the Shift key modifier.
	ModifierKeyShift
)

// Key is a keyboard key name.
type Key string

// Definition represents information about a keyboard key.
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
// Like: US.
type Layout struct {
	Name      string
	Keys      map[Key]Definition
	ValidKeys map[Key]bool
}

// KeyDefinition returns true with the key definition of a given key input.
// It returns false and an empty key definition if it cannot find the key.
func (l Layout) KeyDefinition(key Key) (Definition, bool) {
	for _, d := range l.Keys {
		if d.Key == string(key) {
			return d, true
		}
	}
	return Definition{}, false
}

// ShiftKeyDefinition returns shift key definition of a given key input.
// It returns an empty key definition if it cannot find the key.
func (l Layout) ShiftKeyDefinition(key Key) Definition {
	for _, d := range l.Keys {
		if d.ShiftKey == string(key) {
			return d
		}
	}
	return Definition{}
}

// ModifiedKeyDefinition returns a key definition by applying a modifier key.
// TODO: Complex
//nolint
func (l *Layout) ModifiedKeyDefinition(key Key, m ModifierKey) Definition {
	shift := m & ModifierKeyShift

	// Find directly from the keyboard layout
	srcKeyDef, ok := l.Keys[key]
	// Try to find based on key value instead of code
	if !ok {
		srcKeyDef, ok = l.KeyDefinition(key)
	}
	// Try to find with the shift key value
	if !ok {
		srcKeyDef = l.ShiftKeyDefinition(key)
		shift = m | ModifierKeyShift
	}

	var keyDef Definition
	if srcKeyDef.Key != "" {
		keyDef.Key = srcKeyDef.Key
		keyDef.Text = srcKeyDef.Key
	}
	if shift != 0 && srcKeyDef.ShiftKeyCode != 0 {
		keyDef.KeyCode = srcKeyDef.ShiftKeyCode
	}
	if srcKeyDef.KeyCode != 0 {
		keyDef.KeyCode = srcKeyDef.KeyCode
	}
	if key != "" {
		keyDef.Code = string(key)
	}
	if srcKeyDef.Location != 0 {
		keyDef.Location = srcKeyDef.Location
	}
	if srcKeyDef.Text != "" {
		keyDef.Text = srcKeyDef.Text
	}
	if shift != 0 && srcKeyDef.ShiftKey != "" {
		keyDef.Key = srcKeyDef.ShiftKey
		keyDef.Text = srcKeyDef.ShiftKey
	}
	// If any modifiers besides shift are pressed, no text should be sent
	if m & ^ModifierKeyShift != 0 {
		keyDef.Text = ""
	}

	return keyDef
}

// ModifierBitFromKey returns the modifier key value from string.
func (l *Layout) ModifierBitFromKey(key string) ModifierKey {
	switch key {
	case "Alt":
		return ModifierKeyAlt
	case "Control":
		return ModifierKeyControl
	case "Meta":
		return ModifierKeyMeta
	case "Shift":
		return ModifierKeyShift
	}

	return 0
}

// IsValidKey returns true if the layout has the key.
func (l *Layout) IsValidKey(key Key) bool {
	_, ok := l.ValidKeys[key]
	return ok
}
