package parser

import "math"

// This contains a compile-time generated state transition table for VT emulation.
//
// This is based on the vt100.net state machine: https://vt100.net/emu/dec_ansi_parser
type parserTable map[uint8]map[State]Transition

// Function to generate the full state transition table for the VT emulation
func newParserTable() parserTable {
	var t parserTable = make(map[uint8]map[State]Transition)

	// init table
	for ch := range math.MaxUint8 {
		t[uint8(ch)] = make(map[State]Transition)
	}

	// anywhere
	{
		anywhere := []State{
			StateGround,
			StateCSIEntry,
			StateCSIParam,
			StateCSIIntermediate,
			StateCsiIgnore,
			StateDCSEntry,
			StateDCSParam,
			StateDCSIntermediate,
			StateDCSPassthrough,
			StateDCSIgnore,
			StateOSCString,
			StateSosPmApcString,
		}
		for _, source := range anywhere {
			// => ground
			t.addSingle(0x18, source, StateGround, ActionExecute)
			t.addSingle(0x1A, source, StateGround, ActionExecute)
			t.addRange(0x80, 0x8F, source, StateGround, ActionExecute)
			t.addRange(0x91, 0x97, source, StateGround, ActionExecute)
			t.addSingle(0x99, source, StateGround, ActionExecute)
			t.addSingle(0x9A, source, StateGround, ActionExecute)
			t.addSingle(0x9C, source, StateGround, ActionNone)

			// => sosPmApcString
			t.addSingle(0x98, source, StateSosPmApcString, ActionNone)
			t.addSingle(0x9E, source, StateSosPmApcString, ActionNone)
			t.addSingle(0x9F, source, StateSosPmApcString, ActionNone)

			// => escape
			t.addSingle(0x1B, source, StateEscape, ActionNone)

			// => dcsEntry
			t.addSingle(0x90, source, StateDCSEntry, ActionNone)

			// => oscString
			t.addSingle(0x9D, source, StateOSCString, ActionNone)

			// => csiEntry
			t.addSingle(0x9B, source, StateCSIEntry, ActionNone)

		}
	}

	// ground
	{
		source := StateGround

		// internal events
		t.addRange(0x00, 0x17, source, source, ActionExecute)
		t.addSingle(0x19, source, source, ActionExecute)
		t.addRange(0x1C, 0x1F, source, source, ActionExecute)
		t.addRange(0x20, 0x7F, source, source, ActionPrint)
	}

	// escapeIntermediate
	{
		source := StateEscapeIntermediate
		// escapeIntermediate => ground
		t.addRange(0x30, 0x7E, source, StateGround, ActionESCDispatch)

		// internal events
		t.addRange(0x00, 0x17, source, source, ActionExecute)
		t.addSingle(0x19, source, source, ActionExecute)
		t.addRange(0x1C, 0x1F, source, source, ActionExecute)
		t.addRange(0x20, 0x2F, source, source, ActionCollect)
		t.addSingle(0x7F, source, source, ActionIgnore)
	}

	// sosPmApcString
	{
		source := StateSosPmApcString
		// => ground
		t.addSingle(0x9C, source, StateGround, ActionNone)

		// internal events
		t.addRange(0x00, 0x17, source, source, ActionIgnore)
		t.addSingle(0x19, source, source, ActionIgnore)
		t.addRange(0x1C, 0x1F, source, source, ActionIgnore)
		t.addRange(0x20, 0x7F, source, source, ActionIgnore)
	}

	// escape
	{
		source := StateEscape

		// => ground
		t.addRange(0x30, 0x4F, source, StateGround, ActionESCDispatch)
		t.addRange(0x51, 0x57, source, StateGround, ActionESCDispatch)
		t.addSingle(0x59, source, StateGround, ActionESCDispatch)
		t.addSingle(0x5A, source, StateGround, ActionESCDispatch)
		t.addSingle(0x5C, source, StateGround, ActionESCDispatch)
		t.addRange(0x60, 0x7E, source, StateGround, ActionESCDispatch)

		// => escapeIntermediate
		t.addRange(0x20, 0x2F, source, StateEscapeIntermediate, ActionCollect)

		// => sosPmApcString
		t.addSingle(0x58, source, StateSosPmApcString, ActionNone)
		t.addSingle(0x5E, source, StateSosPmApcString, ActionNone)
		t.addSingle(0x5F, source, StateSosPmApcString, ActionNone)

		// => dcsEntry
		t.addSingle(0x50, source, StateDCSEntry, ActionNone)

		// => oscString
		t.addSingle(0x5D, source, StateOSCString, ActionNone)

		// => csiEntry
		t.addSingle(0x5B, source, StateCSIEntry, ActionNone)

		// internal events
		t.addRange(0x00, 0x17, source, source, ActionExecute)
		t.addSingle(0x19, source, source, ActionExecute)
		t.addRange(0x1C, 0x1F, source, source, ActionExecute)
		t.addSingle(0x7F, source, source, ActionIgnore)
	}

	// dcsEntry
	{
		source := StateDCSEntry
		// => dcsIntermediate
		t.addRange(0x20, 0x2F, source, StateDCSIntermediate, ActionCollect)

		// => dcsIgnore
		t.addSingle(0x3A, source, StateDCSIgnore, ActionNone)

		// => dcsParam
		t.addRange(0x30, 0x39, source, StateDCSParam, ActionParam)
		t.addSingle(0x3B, source, StateDCSParam, ActionParam)
		t.addRange(0x3C, 0x3F, source, StateDCSParam, ActionCollect)

		// => dcsPassthrough
		t.addRange(0x40, 0x7E, source, StateDCSPassthrough, ActionNone)

		// internal events
		t.addRange(0x00, 0x17, source, source, ActionIgnore)
		t.addSingle(0x19, source, source, ActionIgnore)
		t.addRange(0x1C, 0x1F, source, source, ActionIgnore)
		t.addSingle(0x7F, source, source, ActionIgnore)
	}

	// dcsIntermedite
	{
		source := StateDCSIntermediate
		// => dcsIgnore
		t.addRange(0x30, 0x3F, source, StateDCSIgnore, ActionNone)

		// => dcsPassthrough
		t.addRange(0x40, 0x7E, source, StateDCSPassthrough, ActionNone)

		// internal events
		t.addRange(0x00, 0x17, source, source, ActionIgnore)
		t.addSingle(0x19, source, source, ActionIgnore)
		t.addRange(0x1C, 0x1F, source, source, ActionIgnore)
		t.addRange(0x20, 0x2F, source, source, ActionCollect)
		t.addSingle(0x7F, source, source, ActionIgnore)
	}

	// csiParam
	{
		source := StateCSIParam
		// => ground
		t.addRange(0x40, 0x7E, source, StateGround, ActionCSIDispatch)

		// => csiIgnore
		t.addSingle(0x3A, source, StateCsiIgnore, ActionNone)
		t.addRange(0x3C, 0x3F, source, StateCsiIgnore, ActionNone)

		// => csiIntermediate
		t.addRange(0x20, 0x2F, source, StateCSIIntermediate, ActionCollect)

		// internal events
		t.addRange(0x00, 0x17, source, source, ActionExecute)
		t.addSingle(0x19, source, source, ActionExecute)
		t.addRange(0x1C, 0x1F, source, source, ActionExecute)
		t.addRange(0x30, 0x39, source, source, ActionParam)
		t.addSingle(0x3B, source, source, ActionParam)
		t.addSingle(0x7F, source, source, ActionIgnore)
	}

	// dcsIgnore
	{
		source := StateDCSIgnore

		// ground
		t.addSingle(0x9C, source, StateGround, ActionNone)

		// internal events
		t.addRange(0x00, 0x17, source, source, ActionIgnore)
		t.addSingle(0x19, source, source, ActionIgnore)
		t.addRange(0x1C, 0x1F, source, source, ActionIgnore)
		t.addRange(0x20, 0x7F, source, source, ActionIgnore)
	}

	// csiIgnore
	{
		source := StateCsiIgnore

		// => ground
		t.addRange(0x40, 0x7E, source, StateGround, ActionNone)

		// internal events
		t.addRange(0x00, 0x17, source, source, ActionExecute)
		t.addSingle(0x19, source, source, ActionExecute)
		t.addRange(0x1C, 0x1F, source, source, ActionExecute)
		t.addRange(0x20, 0x3F, source, source, ActionIgnore)
		t.addSingle(0x7F, source, source, ActionIgnore)
	}

	// oscString
	{
		source := StateOSCString

		// ground
		t.addSingle(0x9C, source, StateGround, ActionNone)

		// internal events
		t.addRange(0x00, 0x17, source, source, ActionIgnore)
		t.addSingle(0x19, source, source, ActionIgnore)
		t.addRange(0x1C, 0x1F, source, source, ActionIgnore)
		t.addRange(0x20, 0x7F, source, source, ActionOSCPut)
	}

	// dcsParam
	{
		source := StateDCSParam

		// => dcsIntermediate
		t.addRange(0x20, 0x2F, source, StateDCSIntermediate, ActionCollect)

		// => dcsIgnore
		t.addSingle(0x3A, source, StateDCSIgnore, ActionNone)
		t.addRange(0x3C, 0x3F, source, StateDCSIgnore, ActionNone)

		// => dcsPassthrough
		t.addRange(0x40, 0x7E, source, StateDCSPassthrough, ActionNone)

		// internal events
		t.addRange(0x00, 0x17, source, source, ActionIgnore)
		t.addSingle(0x19, source, source, ActionIgnore)
		t.addRange(0x1C, 0x1F, source, source, ActionIgnore)
		t.addRange(0x30, 0x39, source, source, ActionParam)
		t.addSingle(0x3B, source, source, ActionParam)
		t.addSingle(0x7F, source, source, ActionIgnore)
	}

	// csiIntermediate
	{
		source := StateCSIIntermediate

		// => ground
		t.addRange(0x40, 0x7E, source, StateGround, ActionCSIDispatch)

		// => csiIgnore
		t.addRange(0x30, 0x3F, source, StateCsiIgnore, ActionNone)

		// internal events
		t.addRange(0x00, 0x17, source, source, ActionExecute)
		t.addSingle(0x19, source, source, ActionExecute)
		t.addRange(0x1C, 0x1F, source, source, ActionExecute)
		t.addRange(0x20, 0x2F, source, source, ActionCollect)
		t.addSingle(0x7F, source, source, ActionIgnore)
	}

	// csiEntry
	{
		source := StateCSIEntry
		// => ground
		t.addRange(0x40, 0x7E, source, StateGround, ActionCSIDispatch)

		// csiParam
		t.addRange(0x30, 0x39, source, StateCSIParam, ActionParam)
		t.addSingle(0x3B, source, StateCSIParam, ActionParam)
		t.addRange(0x3C, 0x3F, source, StateCSIParam, ActionCollect)

		// => csiIgnore
		t.addSingle(0x3A, source, StateCsiIgnore, ActionNone)

		// => csiIntermediate
		t.addRange(0x20, 0x2F, source, StateCSIIntermediate, ActionCollect)

		// internal events
		t.addRange(0x00, 0x17, source, source, ActionExecute)
		t.addSingle(0x19, source, source, ActionExecute)
		t.addRange(0x1C, 0x1F, source, source, ActionExecute)
		t.addSingle(0x7F, source, source, ActionIgnore)
	}

	// dcsPassthrough
	{
		source := StateDCSPassthrough

		// => ground
		t.addSingle(0x9C, source, StateGround, ActionNone)

		// internal events
		t.addRange(0x00, 0x17, source, source, ActionDCSPut)
		t.addSingle(0x19, source, source, ActionDCSPut)
		t.addRange(0x1C, 0x1F, source, source, ActionDCSPut)
		t.addRange(0x20, 0x7E, source, source, ActionDCSPut)
		t.addSingle(0x7F, source, source, ActionIgnore)
	}
	return t
}

func (t parserTable) addSingle(c uint8, s0 State, s1 State, a ActionType) {
	t[c][s0] = transition(s1, a)
}

func (t parserTable) addRange(from uint8, to uint8, s0 State, s1 State, a ActionType) {
	i := from
	for {
		if i <= to {
			t.addSingle(i, s0, s1, a)
		}
		// If to is 0xFF, increase i will overflow, Return early
		if i == to {
			break
		}
		i++
	}
}

type Transition struct {
	state  State
	action ActionType
}

func transition(state State, action ActionType) Transition {
	return Transition{state: state, action: action}
}
