package stream

import (
	"slices"

	"github.com/hnimtadd/termio/logger"
	"github.com/hnimtadd/termio/terminal/ansi"
	"github.com/hnimtadd/termio/terminal/core"
	"github.com/hnimtadd/termio/terminal/handler"
	"github.com/hnimtadd/termio/terminal/parser"
	"github.com/hnimtadd/termio/terminal/sequences/csi"
	"github.com/hnimtadd/termio/terminal/sequences/dcs"
	"github.com/hnimtadd/termio/terminal/sequences/esc"
	"github.com/hnimtadd/termio/terminal/sequences/osc"
	"github.com/hnimtadd/termio/terminal/sgr"
	"github.com/hnimtadd/termio/terminal/utils"
)

// This is the maximum number of codepoints we can decode
// at one time for this function call. This is somewhat arbitrary
// so if someone can demonstrate a better number then we can switch.
const MaxCodePoints = 4096

// Flip this to true when you want verbose debug output for
// debugging terminal stream issues. In addition to louder
// output this will also disable the chunk optimizations in
// order to make it easier to see every byte.
const debug = true

// This type can be used to process a stream of tty control characters.
// This will call various callsback functions on type T. Type T only has to
// implement the callbacks it cares about; any unimplemented callbacks will
// logged at runtime
//
// To figure out what callback are available, we try to cast the type T
// into specific golang interface
type Stream struct {
	handler     any
	parser      *parser.Parser
	utf8Decoder *UTF8Decoder

	logger logger.Logger

	debug bool
}

func NewStream(handler any, logger logger.Logger) *Stream {
	return &Stream{
		handler:     handler,
		parser:      parser.NewParser(),
		utf8Decoder: NewUTF8Decoder(),
		logger:      logger,
	}
}

// Nextslice prcess a string of characters
func (s *Stream) NextSlice(input []uint8) {
	if debug {
		for c := range slices.Values(input) {
			s.Next(c)
		}
	}
	// This is the maximum number of codepoints we can decode
	// at one time for this function call. This is somewhat arbitrary
	// so if someone can demonstrate a better number then we can switch.
	cpBuf := make([]uint32, MaxCodePoints)
	// split the input into chunks that fit into cp_buf
	i := 0
	for {
		bufLen := min(len(cpBuf), len(input)-i)
		s.nextSliceCapped(input[i:i+bufLen], cpBuf)
		i += bufLen
		if i >= len(input) {
			break
		}
	}
}

func (s *Stream) nextSliceCapped(input []uint8, cpBuf []uint32) {
	utils.Assert(len(input) < len(cpBuf))
	offset := 0

	for s.utf8Decoder.state != 0 {
		if offset >= len(input) {
			break
		}
		s.nextUtf8(input[offset])
		offset += 1
	}
	if offset >= len(input) {
		return
	}

	// If we're not in the ground state then we process until we are. This
	// can happen if the last chunk of input put us in the middle of a control
	// sequence.
	offset += s.consumeUntilGround(input[offset:])
	if offset >= len(input) {
		return
	}
	offset += s.consumeAllEscapes(input[offset:])

	// If we're in the ground state then we can process the input
	// until we see an ESC (0x1B) since all other chracters up to that point
	// are just UTF-8 characters
	for (s.parser.State == parser.StateGround) && (offset < len(input)) {
		decoded, consumed := s.utf8Decoder.DecodeUntilControlSeq(input[offset:], cpBuf)
		for cp := range slices.Values(cpBuf[:decoded]) {
			// At this point we can assume that the input
			// is already in valid range of ground state
			// We only suuport code code before SI
			if cp <= uint32(ansi.C0.SI) { // cp <= 0x0F
				s.execute(uint8(cp))
			} else {
				s.print(cp)
			}
		}
		// Consume the bytes we just processed.
		offset += consumed
		if offset >= len(input) {
			return
		}

		// If our offset is NOT an escape then we must have a
		// partial UTF-8 sequence. In that case, we pass it off
		// to the scalar parser.
		if input[offset] != ansi.C0.ESC {
			rem := input[offset:]
			for c := range slices.Values(rem) {
				s.nextUtf8(c)
			}
		}

		// Process control sequence until we run out.
		offset += s.consumeAllEscapes(input[offset:])
	}
}

// Next prcess a single character, this is the most basic mode and should not
// be used if the input is a string of characters due to performance.
//
// Consider NextSlice for higher performance

// Like nextSlice but takes one byte and is necessarily a scalar
// operation that can't use SIMD. Prefer nextSlice if you can and
// try to get multiple bytes at once.
func (s *Stream) Next(c uint8) {
	// The scalar path can be responsible for decoding UTF-8.
	switch s.parser.State {
	case parser.StateGround:
		s.nextUtf8(c)
	default:
		s.nextNonUtf8(c)
	}
}

// nextUtf8 processes a single UTF-8 character and print as necessary.
//
// This assumes we're in the UTF-8 decoding state. If we may not
// be in the UTF-8 decoding state call nextSlice or next.
func (s *Stream) nextUtf8(c uint8) {
	utils.Assert(s.parser.State == parser.StateGround)
	s.logger.Debug("nextUtf8", "code", ansi.String(c))

	cp, generated, consumed := s.utf8Decoder.Next(c)
	if generated {
		s.handleCodepoint(cp)
	}

	if !consumed {
		cp, generated, consumed := s.utf8Decoder.Next(c)

		// It should be impossible for the utf8Decoder
		// to not consume the byte twice in a row.
		utils.Assert(consumed)
		if generated {
			s.handleCodepoint(cp)
		}
	}
}

// To be called whenever the utf-8 decoder produces a codepoint.
//
// This function is abstracted this way to handle the case where
// the decoder emits a 0x1B after rejecting an ill-formed sequence.
//
// The first 128 characters of UTF-8, which correspond one-to-one with ASCII (ansi-7bit),
// are encoded using a single byte with the same binary value as ASCII,
//
// so we could assumes if cp < 0xF then we should treat it like normal uint8 c
func (s *Stream) handleCodepoint(cp uint32) {
	// We only suuport code code before SI
	if cp <= uint32(ansi.C0.SI) { // cp <= 0x0F
		s.execute(uint8(cp))
		return
	}
	if cp == uint32(ansi.C0.ESC) {
		s.nextNonUtf8(uint8(cp))
		return
	}

	s.print(cp)
}

// Process the next character and call any callbacks if necessary.
//
// This assumes that we're not in the UTF-8 decoding state. If
// we may be in the UTF-8 decoding state call nextSlice or next.
func (s *Stream) nextNonUtf8(c uint8) {
	utils.Assert(s.parser.State != parser.StateGround || c == ansi.C0.ESC)
	s.logger.Debug("nextNonUtf8", "code", ansi.String(c))

	actions := s.parser.Next(c)
	for action := range slices.Values(actions[:]) {
		s.logger.Debug("action", action.String())
		if action == nil {
			continue
		}
		switch action.Type {
		case parser.ActionPrint:
			if handler, implemented := s.handler.(handler.PrintHandler); implemented {
				handler.Print(uint32(action.PrintData))
			}

		case parser.ActionExecute:
			s.execute(action.ExecuteData)

		case parser.ActionCSIDispatch:
			s.csiDispatch(action.CSIDispatchData)

		case parser.ActionESCDispatch:
			s.escDispatch(action.ESCDispatchData)

		case parser.ActionOSCEnd:
			switch {
			case action.OSCDispatchData != nil:
				s.oscDispatch(action.OSCDispatchData)
			default:
				s.logger.Warn("unimplemented OSC end")
				continue
			}

		case parser.ActionDCSHook:
			if action.DCSHookData != nil {
				s.logger.Warn("unimplemented hook")
				continue
			}
			if handler, implemented := s.handler.(dcs.HookHandler); implemented {
				handler.DCSHook(action.DCSHookData)
			}

		case parser.ActionDCSPut:
			if handler, implemented := s.handler.(dcs.PutHandler); implemented {
				handler.DCSPut(action.DCSPutData)
			}

		case parser.ActionDCSUnHook:
			if handler, implemented := s.handler.(dcs.UnhookHandler); implemented {
				handler.DCSUnhook()
			}
		}
	}
}

func (s *Stream) execute(c uint8) {
	if s.handler == nil {
		s.logger.Warn("handler is nil, ignoring")
		return
	}
	c0 := ansi.C0
	switch c {
	case c0.BS:
		if handler, implemented := s.handler.(handler.EditorHandler); implemented {
			handler.Backspace()
		} else {
			s.logger.Warn("unimplemented execute", "codepoint", c)
		}

	case c0.HT:
		if handler, implemented := s.handler.(handler.EditorHandler); implemented {
			handler.SetCursorTabRight(1)
		} else {
			s.logger.Warn("unimplemented execute", "codepoint", c)
		}

	case c0.LF, c0.VT, c0.FF:
		if handler, implemented := s.handler.(handler.EditorHandler); implemented {
			handler.LineFeed()
		} else {
			s.logger.Warn("unimplemented execute", "codepoint", c)
		}

	case c0.CR:
		if handler, implemented := s.handler.(handler.EditorHandler); implemented {
			handler.CarriageReturn()
		} else {
			s.logger.Warn("unimplemented execute", "codepoint", c)
		}

	// KAI do not support these characters as the moment, just put them here
	// as a TODO for later enhancement.
	case c0.NUL, c0.ENQ, c0.BEL, c0.SO, c0.SI:
		s.logger.Warn("unimplemented characters, ignoring", "codepoint", c)
		return

	default:
		s.logger.Warn("invalid c0 character, ignoring", "codepoint", c)
	}
}

func (s *Stream) print(c uint32) {
	if handler, implemented := s.handler.(handler.PrintHandler); implemented {
		handler.Print(c)
	} else {
		s.logger.Warn("unimplemented print", "codepoint", c)
	}
}

// csiDispatch implemented VT100 compatiable control sequences dispatch
//
// not all VT100 control sequences supported by KAI,
// escecially VT100 to Host control sequences
//
// It's gurantee that all Host to VT100 control sequences are supported
func (s *Stream) csiDispatch(c *csi.Command) {
	// by alphabets order of Final character
	switch c.Final {
	case 'A', 'k':
		// CUU - Cursor Up
		switch len(c.Intermediates) {
		case 0:
			handler, implemented := s.handler.(handler.EditorHandler)
			if !implemented {
				s.logger.Warn("unimplemented CUU command", "codepoint", c)
				return
			}

			var offset uint16
			switch len(c.Params) {
			case 0:
				offset = 1
			case 1:
				offset = c.Params[0]
			default:
				s.logger.Warn("invalid CUU command", "codepoint", c)
				return
			}

			handler.SetCursorUp(offset, false)
		default:
			s.logger.Warn("unimplemented CSI A with intermediates", "codepoint", c)
			return
		}

	case 'B':
		// CUD - Cursor Down
		switch len(c.Intermediates) {
		case 0:
			handler, implemented := s.handler.(handler.EditorHandler)
			if !implemented {
				s.logger.Warn("unimplemented CUD command", "codepoint", c)
				return
			}
			var offset uint16
			switch len(c.Params) {
			case 0:
				offset = 1
			case 1:
				offset = c.Params[0]
			default:
				s.logger.Warn("invalid CUD command", "codepoint", c)
				return
			}
			handler.SetCursorDown(offset, false)

		default:
			s.logger.Warn("unimplemented CSI B with intermediates", "codepoint", c)
		}

	case 'C':
		// CUF - Cursor Forward
		switch len(c.Intermediates) {
		case 0:
			handler, implemented := s.handler.(handler.EditorHandler)
			if !implemented {
				s.logger.Warn("unimplemented CUF command", "codepoint", c)
				return
			}

			var offset uint16
			switch len(c.Params) {
			case 0:
				offset = 1
			case 1:
				offset = c.Params[0]
			default:
				s.logger.Warn("invalid CUF command", "codepoint", c)
				return
			}
			handler.SetCursorLeft(offset)

		default:
			s.logger.Warn("unimplemented CSI C with intermediates", "codepoint", c)
		}

	case 'D', 'j':
		// CUB - Cursor Backward
		switch len(c.Intermediates) {
		case 0:
			handler, implemented := s.handler.(handler.EditorHandler)
			if !implemented {
				s.logger.Warn("unimplemented CUB command", "codepoint", c)
				return
			}

			var offset uint16
			switch len(c.Params) {
			case 0:
				offset = 1
			case 1:
				offset = c.Params[0]
			default:
				s.logger.Warn("invalid CUB command", "codepoint", c)
				return
			}
			handler.SetCursorRight(offset)

		default:
			s.logger.Warn("unimplemneted CSI D with Intermediates", "codepoint", c)
			return
		}

	case 'E':
		// CNL - Cursor Next Line
		switch len(c.Intermediates) {
		case 0:
			handler, implemented := s.handler.(handler.EditorHandler)
			if !implemented {
				s.logger.Warn("unimplemented CNL command", "codepoint", c)
				return
			}
			var offset uint16
			switch len(c.Params) {
			case 0:
				offset = 1
			case 1:
				offset = c.Params[0]
			default:
				s.logger.Warn("invalid CNL command", "codepoint", c)
				return
			}
			handler.SetCursorDown(offset, true)
		default:
			s.logger.Warn("unimplemented CSI E with intermediates", "codepoint", c)
			return
		}

	case 'F':
		// CPL - Cursor Preceding Line
		switch len(c.Intermediates) {
		case 0:
			handler, implemented := s.handler.(handler.EditorHandler)
			if !implemented {
				s.logger.Warn("unimplemented CPL command", "codepoint", c)
				return
			}
			var offset uint16
			switch len(c.Params) {
			case 0:
				offset = 1
			case 1:
				offset = c.Params[0]
			default:
				s.logger.Warn("invalid CPL command", "codepoint", c)
				return
			}
			handler.SetCursorUp(offset, true)
		default:
			s.logger.Warn("unimplemented CSI F with intermediates", "codepoint", c)
			return
		}
	case 'G', '`':
		// HPA - Cursor Horizontal Position Absolute
		switch len(c.Intermediates) {
		case 0:
			handler, implemented := s.handler.(handler.EditorHandler)
			if !implemented {
				s.logger.Warn("unimplemented HPA command", "codepoint", c)
				return
			}
			var col uint16
			switch len(c.Params) {
			case 0:
				col = 1
			case 1:
				col = c.Params[0]
			default:
				s.logger.Warn("invalid HPA command", "codepoint", c)
				return
			}
			handler.SetCursorCol(col)
		default:
			s.logger.Warn("unimplemented CSI G with intermediates", "codepoint", c)
			return
		}
	case 'H', 'f':
		// CUP - Cursor Position
		// HVP - Horizontal Vertical Position
		handler, implemented := s.handler.(handler.EditorHandler)
		if !implemented {
			s.logger.Warn("unimplemented CUP command", "codepoint", c)
			return
		}

		switch len(c.Params) {
		case 0:
			// row = 0
			// col = 0
			handler.SetCursorPosition(0, 0)
		case 1:
			// row != 0
			// col = 0
			handler.SetCursorPosition(c.Params[0], 0)
		case 2:
			// row != 0
			// col != 0
			handler.SetCursorPosition(c.Params[0], c.Params[1])
		default:
			s.logger.Warn("invalid CUP command", "codepoint", c)
			return
		}

	case 'I':
		// CHT - Cursor Horizontal Tabulation
		switch len(c.Intermediates) {
		case 0:
			handler, implemented := s.handler.(handler.EditorHandler)
			if !implemented {
				s.logger.Warn("unimplemented CHT command", "codepoint", c)
			}
			var numTab uint16
			switch len(c.Params) {
			case 0:
				numTab = 1
			case 1:
				numTab = c.Params[0]
			default:
				s.logger.Warn("invalid CHT command", "codepoint", c)
				return
			}
			handler.SetCursorTabRight(numTab)
		default:
			s.logger.Warn("unimplemented CSI I with intermediates", "codepoint", c)
			return
		}

	case 'J':
		// ED - Erase in Display
		switch len(c.Intermediates) {
		case 0, 1:
			// todo handle DECSED also
			handler, implemented := s.handler.(handler.EditorHandler)
			if !implemented {
				s.logger.Warn("unimplemented ED command", "codepoint", c)
				return
			}
			var mode csi.EDMode
			switch len(c.Params) {
			case 0:
				mode = csi.EDModeBelow
			case 1:
				mode = csi.EDMode(c.Params[0])
			default:
				s.logger.Warn("invalid ED command", "codepoint", c)
				return
			}
			handler.EraseInDisplay(mode)
		default:
			s.logger.Warn("unimplemented CSI J with intermediates", "codepoint", c)
			return
		}

	case 'K':
		// EL - Erase in Line
		switch len(c.Intermediates) {
		case 0, 1:
			// todo handle DECSEL also
			handler, implemented := s.handler.(handler.EditorHandler)
			if !implemented {
				s.logger.Warn("unimplemented EL command", "codepoint", c)
				return
			}
			var mode csi.ELMode
			switch len(c.Params) {
			case 0:
				mode = csi.ELModeRight
			case 1:
				mode = csi.ELMode(c.Params[0])
			default:
				s.logger.Warn("invalid EL command", "codepoint", c)
				return
			}
			handler.EraseInLine(mode)
		default:
			s.logger.Warn("unimplemented CSI K with intermediates", "codepoint", c)
			return
		}
	case 'L':
		// IL - Insert Lines
		switch len(c.Intermediates) {
		case 0:
			handler, implemented := s.handler.(handler.EditorHandler)
			if !implemented {
				s.logger.Warn("unimplemented IL command", "codepoint", c)
				return
			}
			var repeated uint16
			switch len(c.Params) {
			case 0:
				repeated = 1
			case 1:
				repeated = c.Params[0]
			default:
				s.logger.Warn("invalid IL ccommand", "codepoint", c)
				return
			}
			handler.InsertLines(repeated)
		default:
			s.logger.Warn("unimplemented CSI L with intermediates", "codepoint", c)
			return
		}
	case 'M':
		// DL - Delete Lines
		switch len(c.Intermediates) {
		case 0:
			handler, implemented := s.handler.(handler.EditorHandler)
			if !implemented {
				s.logger.Warn("unimplemented DL command", "codepoint", c)
				return
			}
			var repeated uint16
			switch len(c.Params) {
			case 0:
				repeated = 1
			case 1:
				repeated = c.Params[1]
			default:
				s.logger.Warn("invalid DL command", "codepoint", c)
			}
			handler.DeleteLines(repeated)
		default:
			s.logger.Warn("unimpletented CSI M with intermediates", "codepoint", c)
		}

	case 'P':
		// DCH - Delete Characters
		switch len(c.Intermediates) {
		case 0:
			handler, implemented := s.handler.(handler.EditorHandler)
			if !implemented {
				s.logger.Warn("unimplemented DCH command", "codepoint", c)
				return
			}
			var repeated uint16
			switch len(c.Params) {
			case 0:
				repeated = 1
			case 1:
				repeated = c.Params[1]
			default:
				s.logger.Warn("invalid DCH command", "codepoint", c)
			}
			handler.DeleteChars(repeated)
		default:
			s.logger.Warn("unimpletented CSI M with intermediates", "codepoint", c)
		}

	case 'S':
		// SD - Scroll Up

	case 'm':
		// SGR - Select Graphic Rendition
		switch len(c.Intermediates) {
		case 0:
			handler, implemented := s.handler.(handler.SGRHandler)
			if !implemented {
				s.logger.Warn("unimplemented SGR command", "codepoint", c)
				return
			}
			p := sgr.Parser{
				Params:    c.Params,
				ParamsSep: c.ParamsSet,
			}
			for attr := range p.Iter() {
				if attr != nil {
					handler.SetGraphicsRendition(attr)
				}
			}
		default:
			s.logger.Warn("unimplemented CSI m with intermediates", "codepoint", c)
			return
		}

	case 'h':
		// SM - Set Mode
		handler, implemented := s.handler.(handler.VT100Handler)
		if !implemented {
			s.logger.Warn("unimplemented SM command", "codepoint", c)
			return
		}
		var ansiMode bool
		switch {
		case len(c.Intermediates) == 0:
			ansiMode = true
		case len(c.Intermediates) == 1 && c.Intermediates[0] == '?':
			ansiMode = false
		default:
			s.logger.Warn("invalid set mode command", "codepoint", c)
		}
		for modeInt := range c.Params {
			if mode := core.ModeFromInt(modeInt, ansiMode); mode != nil {
				handler.SetMode(*mode, true)
			} else {
				s.logger.Warn("unimplemented mode", "mode", modeInt)
			}
		}

	case 'l':
		// RM - Reset Mode
		handler, implemented := s.handler.(handler.VT100Handler)
		if !implemented {
			s.logger.Warn("unimplemented RM command", "codepoint", c)
			return
		}
		var ansiMode bool
		switch {
		case len(c.Intermediates) == 0:
			ansiMode = true
		case len(c.Intermediates) == 1 && c.Intermediates[0] == '?':
			ansiMode = false
		default:
			s.logger.Warn("invalid reset mode command", "codepoint", c)
		}
		for modeInt := range c.Params {
			if mode := core.ModeFromInt(modeInt, ansiMode); mode != nil {
				handler.SetMode(*mode, false)
			} else {
				s.logger.Warn("unimplemented mode", "mode", modeInt)
			}
		}
	case '@':
		// ICH - Insert Blanks
		handler, implemented := s.handler.(handler.EditorHandler)
		if !implemented {
			s.logger.Warn("unimplemented ICH command", "codepoint", c)
			return
		}
		switch len(c.Params) {
		case 0:
			handler.InsertBlanks(1)
		case 1:
			handler.InsertBlanks(c.Params[0])
		default:
			s.logger.Warn("invalid ICH command", "codepoint", c)
			return
		}
	}
}

// escDispatch implemented VT100 compatiable control sequences dispatch
//
// not all VT100 control sequences supported by KAI,
// escecially VT100 to Host control sequences
func (s *Stream) escDispatch(c *esc.Command) {
	switch c.Final {
	case 'D':
		// IND - Index
		handler, implemented := s.handler.(handler.FormatEffectorHandler)
		if !implemented {
			s.logger.Warn("unimplemented IND command", "codepoint", c)
			return
		}
		switch len(c.Intermediates) {
		case 0:
			handler.Index()
		default:
			s.logger.Warn("invalid IND command", "codepoint", c)
			return
		}
	case 'E':
		// NEL - NextLine
		handler, implemented := s.handler.(handler.FormatEffectorHandler)
		if !implemented {
			s.logger.Warn("unimplemented NEL command", "codepoint", c)
			return
		}
		switch len(c.Intermediates) {
		case 0:
			handler.NextLine()
		default:
			s.logger.Warn("invalid NEL command", "codepoint", c)
			return
		}

	case 'H':
		// HTS- Tabset
		handler, implemented := s.handler.(handler.FormatEffectorHandler)
		if !implemented {
			s.logger.Warn("unimplemented HTS command", "codepoint", c)
			return
		}
		switch len(c.Intermediates) {
		case 0:
			handler.TabSet()
		default:
			s.logger.Warn("invalid HTS command", "codepoint", c)
			return
		}

	case 'M':
		// RI - Reverse Index
		handler, implemented := s.handler.(handler.FormatEffectorHandler)
		if !implemented {
			s.logger.Warn("unimplemented RI command", "codepoint", c)
			return
		}
		switch len(c.Intermediates) {
		case 0:
			handler.ReverseIndex()
		default:
			s.logger.Warn("invalid RI command", "codepoint", c)
			return
		}

	case 'c':
		// RIS - Full Reset
		handler, implemented := s.handler.(handler.FormatEffectorHandler)
		if !implemented {
			s.logger.Warn("unimplemented RIS command", "codepoint", c)
			return
		}
		switch len(c.Intermediates) {
		case 0:
			handler.FullReset()
		default:
			s.logger.Warn("invalid RIS command", "codepoint", c)
			return
		}
	case '\\':
		// ST - String terminator
		//  We don't have to do anything.
	}
}

// oscDispatch implemented VT100 compatiable osc
//
// not all VT100 control sequences supported by KAI,
// escecially VT100 to Host control sequences
func (s *Stream) oscDispatch(osc *osc.Command) {
	s.logger.Warn("unimplemented osc dispatch", "command", osc)
}

// consumeUntilGround read the stream until we got the ground state
// then return the number of bytes consumed
func (s *Stream) consumeUntilGround(input []uint8) int {
	offset := 0
	for s.parser.State != parser.StateGround {
		if offset >= len(input) {
			return len(input)
		}
		s.nextNonUtf8(input[offset])
		offset += 1
	}
	return offset
}

// Parse escape sequences back-to-back until none are left.
// Returns the number of bytes consumed from the provided input.
//
// Expects input to start with ansi ESC, use consumeUntilGround first
// if the stream is in the middle of escape sequence.
func (s *Stream) consumeAllEscapes(input []uint8) int {
	offset := 0
	for input[offset] == ansi.C0.ESC {
		s.parser.State = parser.StateEscape
		s.parser.Clear()
		offset += 1
		offset += s.consumeUntilGround(input[offset:])
		if offset >= len(input) {
			return len(input)
		}
	}
	return offset
}
