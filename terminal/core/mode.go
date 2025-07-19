package core

import (
	"maps"
	"slices"
)

// A struct that maintains the state of all settable modes
type Mode struct {
	Name  string
	Value int
	/// True if this is an ANSI mode
	Ansi    bool
	Default bool
}

func entryForMode(name string, value int, ansi bool, defaultMode bool) Mode {
	return Mode{
		Name:    name,
		Value:   value,
		Ansi:    ansi,
		Default: defaultMode,
	}
}

var (
	// ansi modes
	ModeDisableKeyboard = entryForMode("disable Keyboard", 2, true, false)  // KAM
	ModeInsert          = entryForMode("insert", 4, true, false)            // IRM
	ModeSendReceiveMode = entryForMode("send_receive_mode", 12, true, true) // SRM
	ModeLineFeed        = entryForMode("line feed", 20, true, false)        // LNM

	// DEC modes
	ModeWraparound = entryForMode("wraparound", 7, false, true) // DECCWM
	ModeOrigin     = entryForMode("origin", 6, false, false)    // DECOM

	// The full list of avialbe entries. For documentation on these modes, see
	// how they are used in the VT100 and ECMA-48 standards or google their values.
	entries = []Mode{
		ModeDisableKeyboard,
		ModeInsert,
		ModeSendReceiveMode,
		ModeLineFeed,
		ModeWraparound,
		ModeOrigin,
	}
)

// A Packed map of all settable modes. This shouldn't be used directly but
// rather through the ModeState struct
var ModePacked = func() map[Mode]bool {
	packed := make(map[Mode]bool, len(entries))
	for _, m := range entries {
		packed[m] = m.Default
	}
	return packed
}()

type ModeState struct {
	// The values of current modes
	values map[Mode]bool
	// The default values of modes
	defaults map[Mode]bool
}

func NewModeState(values map[Mode]bool, def map[Mode]bool) *ModeState {
	state := &ModeState{
		defaults: def,
		values:   values,
	}
	if values == nil {
		state.values = make(map[Mode]bool)
	}
	if def == nil {
		state.defaults = make(map[Mode]bool)
	}
	return state
}

func (s *ModeState) Set(m Mode, value bool) {
	s.values[m] = value
}

func (s *ModeState) Get(m Mode) bool {
	return s.values[m]
}

func (s *ModeState) Reset() {
	s.values = make(map[Mode]bool)
	maps.Copy(s.values, s.defaults)
}

func ModeFromInt(input int, ansi bool) *Mode {
	for entry := range slices.Values(entries) {
		if entry.Value == input && entry.Ansi == ansi {
			return &entry
		}
	}
	return nil
}

/* Helpful doc:
DECOM (originMode) doc: https://documentation.help/putty/config-decom.html
*/
