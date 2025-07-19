package termio

import (
	"github.com/hnimtadd/termio/logger"
	"github.com/hnimtadd/termio/terminal"
	"github.com/hnimtadd/termio/terminal/color"
	"github.com/hnimtadd/termio/terminal/core"
	"github.com/hnimtadd/termio/terminal/handler"
	"github.com/hnimtadd/termio/terminal/sequences/csi"
	"github.com/hnimtadd/termio/terminal/sequences/dcs"
	"github.com/hnimtadd/termio/terminal/sgr"
)

// This is used as the handler for the terminal.Stream type. This is stateful
// and is expected to live for the entire lifetime of the terminal. It is not
// valid to stop a stream handler, create a new one, and use that unless all
// of the member fields are copied.
type StreamHandler struct {
	terminal *terminal.Terminal
	rows     uint16
	cols     uint16

	// The default forground and background color are those set by the user's
	// config file.
	defaultForegroundColor color.RGB
	defaultBackgroundColor color.RGB

	// The foreground and background color as set by an OSC 10 or OSC 11
	// sequence. If unset the respective color is the default value.
	foregroundColor color.RGB
	backgroundColor color.RGB

	// -----------------------------------------------------------------------
	// Internal state

	// The DCS handler maintains DCS state. DCS is like CSI or OSC, but
	// requires more stateful parsing. This is used by functionality such
	// as XGETTCAP.
	dcs dcs.Handler

	logger logger.Logger
}

func (s *StreamHandler) DcsHook(dcs *dcs.DCS) {
	cmd := s.dcs.DCSHook(dcs)
	s.DCSCommand(cmd)
}

func (s *StreamHandler) DcsUnhook() {
	cmd := s.dcs.DCSUnhook()
	s.DCSCommand(cmd)
}

func (s *StreamHandler) DCSPut(c uint8) {
	cmd := s.dcs.DCSPut(c)
	s.DCSCommand(cmd)
}

func (s *StreamHandler) DCSCommand(cmd *dcs.Command) {}

// Backspace implements streamHandler.
func (s *StreamHandler) Backspace() {
	s.terminal.Backspace()
}

// CarriageReturn implements streamHandler.
func (s *StreamHandler) CarriageReturn() {
	s.terminal.CarriageReturn()
}

// DeleteChars implements streamHandler.
func (s *StreamHandler) DeleteChars(reepeated uint16) {
	s.terminal.DeleteChars(reepeated)
}

// DeleteLines implements streamHandler.
func (s *StreamHandler) DeleteLines(repeated uint16) {
	s.terminal.DeleteLines(repeated)
}

// EraseInDisplay implements streamHandler.
func (s *StreamHandler) EraseInDisplay(erase csi.EDMode) {
	s.terminal.EraseInDisplay(erase)
}

// EraseInLine implements streamHandler.
func (s *StreamHandler) EraseInLine(mode csi.ELMode) {
	s.terminal.EraseInLine(mode)
}

// FullReset implements streamHandler.
func (s *StreamHandler) FullReset() {
	s.terminal.FullReset()
}

// Index implements streamHandler.
func (s *StreamHandler) Index() {
	s.terminal.Index()
}

// InsertBlanks implements streamHandler.
func (s *StreamHandler) InsertBlanks(repeated uint16) {
	s.terminal.InsertBlanks(repeated)
}

// InsertLines implements streamHandler.
func (s *StreamHandler) InsertLines(repeated uint16) {
	s.terminal.InsertLines(repeated)
}

// LineFeed implements streamHandler.
func (s *StreamHandler) LineFeed() {
	s.terminal.LineFeed()
}

// NextLine implements streamHandler.
func (s *StreamHandler) NextLine() {
	s.terminal.Index()
	s.terminal.CarriageReturn()
}

// Print implements streamHandler.
func (s *StreamHandler) Print(c uint32) {
	s.terminal.Print(c)
}

// ReverseIndex implements streamHandler.
func (s *StreamHandler) ReverseIndex() {
	s.terminal.ReverseIndex()
}

// SetCursorCol implements streamHandler.
func (s *StreamHandler) SetCursorCol(col uint16) {
	// plus one because the cursor is 0-indexed and the display is 1-indexed
	s.terminal.SetCursorPosition(uint16(s.terminal.Screen.Cursor.Y+1), col)
}

// SetCursorDown implements streamHandler.
func (s *StreamHandler) SetCursorDown(offset uint16, carriage bool) {
	s.terminal.SetCursorDown(offset, carriage)
}

// SetCursorLeft implements streamHandler.
func (s *StreamHandler) SetCursorLeft(offset uint16) {
	s.terminal.SetCursorLeft(offset)
}

// SetCursorPosition implements streamHandler.
func (s *StreamHandler) SetCursorPosition(row uint16, col uint16) {
	s.terminal.SetCursorPosition(row, col)
}

// SetCursorRight implements streamHandler.
func (s *StreamHandler) SetCursorRight(offset uint16) {
	s.terminal.SetCursorRight(offset)
}

// SetCursorRow implements streamHandler.
func (s *StreamHandler) SetCursorRow(row uint16) {
	// plus one because the cursor is 0-indexed and the display is 1-indexed
	s.terminal.SetCursorPosition(row, uint16(s.terminal.Screen.Cursor.X+1))
}

// SetCursorTabLeft implements streamHandler.
func (s *StreamHandler) SetCursorTabLeft(repeated uint16) {
	s.terminal.SetCursorTabLeft(repeated)
}

// SetCursorTabRight implements streamHandler.
func (s *StreamHandler) SetCursorTabRight(repeated uint16) {
	s.terminal.SetCursorTabRight(repeated)
}

// SetCursorUp implements streamHandler.
func (s *StreamHandler) SetCursorUp(offset uint16, carriage bool) {
	s.terminal.SetCursorUp(offset, carriage)
}

// SetGraphicsRendition implements streamHandler.
func (s *StreamHandler) SetGraphicsRendition(attr *sgr.Attribute) {
	switch attr.Type {
	case sgr.AttributeTypeUnknown:
		s.logger.Warn("Unknown SGR attribute", "attribute", attr)
	default:
		s.terminal.SetGraphicsRendition(attr)
	}
}

// TabSet implements streamHandler.
func (s *StreamHandler) TabSet() {
	s.terminal.TabSet()
}

// SetMode implements streamHandler.
func (s *StreamHandler) SetMode(mode core.Mode, enabled bool) {
	s.terminal.Modes.Set(mode, enabled)
	panic("unimplemented")
}

// ---------------- IGNORE THIS ----------------
var _ streamHandler = (*StreamHandler)(nil)

// This handler marks handlers supported by KAI terminal
type streamHandler interface {
	handler.EditorHandler
	handler.FormatEffectorHandler
	handler.PrintHandler
	handler.SGRHandler
	handler.VT100Handler
}

// ---------------- IGNORE THIS ----------------
