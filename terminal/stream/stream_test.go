package stream

import (
	"testing"

	"github.com/hnimtadd/termio/logger"
	"github.com/hnimtadd/termio/terminal/core"
	"github.com/hnimtadd/termio/terminal/sequences/csi"
	"github.com/hnimtadd/termio/terminal/sgr"
	"github.com/stretchr/testify/assert"
)

type modeCall struct {
	mode    core.Mode
	enabled bool
}

type marginCall struct {
	top    uint16
	bottom uint16
}

type mockStreamHandler struct {
	deleteLinesCalls []uint16
	deleteCharsCalls []uint16
	modeCalls        []modeCall
	marginCalls      []marginCall

	saveCursorCalls       int
	restoreCursorCalls    int
	reverseLineFeedCalls  int
	shiftInCalls          int
	shiftOutCalls         int
	charsetDesignateCalls []struct {
		isG1    bool
		charset uint8
	}
}

func (m *mockStreamHandler) NextLine()                             {}
func (m *mockStreamHandler) Index()                                {}
func (m *mockStreamHandler) ReverseIndex()                         {}
func (m *mockStreamHandler) TabSet()                               {}
func (m *mockStreamHandler) FullReset()                            {}
func (m *mockStreamHandler) SetGraphicsRendition(_ *sgr.Attribute) {}
func (m *mockStreamHandler) DeleteChars(repeated uint16) {
	m.deleteCharsCalls = append(m.deleteCharsCalls, repeated)
}
func (m *mockStreamHandler) DeleteLines(repeated uint16) {
	m.deleteLinesCalls = append(m.deleteLinesCalls, repeated)
}
func (m *mockStreamHandler) InsertLines(_ uint16)           {}
func (m *mockStreamHandler) InsertBlanks(_ uint16)          {}
func (m *mockStreamHandler) EraseInLine(_ csi.ELMode)       {}
func (m *mockStreamHandler) EraseInDisplay(_ csi.EDMode)    {}
func (m *mockStreamHandler) LineFeed()                      {}
func (m *mockStreamHandler) Backspace()                     {}
func (m *mockStreamHandler) SetCursorRow(_ uint16)          {}
func (m *mockStreamHandler) SetCursorCol(_ uint16)          {}
func (m *mockStreamHandler) SetCursorPosition(_, _ uint16)  {}
func (m *mockStreamHandler) SetCursorUp(_ uint16, _ bool)   {}
func (m *mockStreamHandler) SetCursorDown(_ uint16, _ bool) {}
func (m *mockStreamHandler) SetCursorLeft(_ uint16)         {}
func (m *mockStreamHandler) SetCursorRight(_ uint16)        {}
func (m *mockStreamHandler) SetCursorTabRight(_ uint16)     {}
func (m *mockStreamHandler) SetCursorTabLeft(_ uint16)      {}
func (m *mockStreamHandler) CarriageReturn()                {}
func (m *mockStreamHandler) SetTopBottomMargins(top, bottom uint16) {
	m.marginCalls = append(m.marginCalls, marginCall{top: top, bottom: bottom})
}
func (m *mockStreamHandler) SetMode(mode core.Mode, enabled bool) {
	m.modeCalls = append(m.modeCalls, modeCall{mode: mode, enabled: enabled})
}
func (m *mockStreamHandler) SaveCursor() {
	m.saveCursorCalls++
}
func (m *mockStreamHandler) RestoreCursor() {
	m.restoreCursorCalls++
}
func (m *mockStreamHandler) ReverseLineFeed() {
	m.reverseLineFeedCalls++
}
func (m *mockStreamHandler) DesignateCharset(isG1 bool, charset uint8) {
	m.charsetDesignateCalls = append(m.charsetDesignateCalls, struct {
		isG1    bool
		charset uint8
	}{isG1: isG1, charset: charset})
}
func (m *mockStreamHandler) ShiftIn()       { m.shiftInCalls++ }
func (m *mockStreamHandler) ShiftOut()      { m.shiftOutCalls++ }
func (m *mockStreamHandler) Print(_ uint32) {}

func TestStreamModeSetResetUsesParamValues(t *testing.T) {
	handler := &mockStreamHandler{}
	s := NewStream(handler, logger.DefaultLogger)

	s.NextSlice([]byte("\x1b[4h\x1b[4l"))

	if assert.Len(t, handler.modeCalls, 2) {
		assert.Equal(t, core.ModeInsert.Value, handler.modeCalls[0].mode.Value)
		assert.True(t, handler.modeCalls[0].enabled)
		assert.Equal(t, core.ModeInsert.Value, handler.modeCalls[1].mode.Value)
		assert.False(t, handler.modeCalls[1].enabled)
	}
}

func TestStreamDLAndDCHUseFirstParameter(t *testing.T) {
	handler := &mockStreamHandler{}
	s := NewStream(handler, logger.DefaultLogger)

	s.NextSlice([]byte("\x1b[3M\x1b[2P"))

	assert.Equal(t, []uint16{3}, handler.deleteLinesCalls)
	assert.Equal(t, []uint16{2}, handler.deleteCharsCalls)
}

func TestStreamDispatchesESCAndC0VT102Controls(t *testing.T) {
	handler := &mockStreamHandler{}
	s := NewStream(handler, logger.DefaultLogger)

	s.NextSlice([]byte("\x1b7\x1b8\x1bI\x1b)0\x1b(B\x0e\x0f"))

	assert.Equal(t, 1, handler.saveCursorCalls)
	assert.Equal(t, 1, handler.restoreCursorCalls)
	assert.Equal(t, 1, handler.reverseLineFeedCalls)
	assert.Equal(t, 1, handler.shiftOutCalls)
	assert.Equal(t, 1, handler.shiftInCalls)

	if assert.Len(t, handler.charsetDesignateCalls, 2) {
		assert.True(t, handler.charsetDesignateCalls[0].isG1)
		assert.Equal(t, uint8('0'), handler.charsetDesignateCalls[0].charset)
		assert.False(t, handler.charsetDesignateCalls[1].isG1)
		assert.Equal(t, uint8('B'), handler.charsetDesignateCalls[1].charset)
	}
}

func TestStreamDispatchesDECSTBM(t *testing.T) {
	handler := &mockStreamHandler{}
	s := NewStream(handler, logger.DefaultLogger)

	s.NextSlice([]byte("\x1b[2;5r"))

	if assert.Len(t, handler.marginCalls, 1) {
		assert.Equal(t, uint16(2), handler.marginCalls[0].top)
		assert.Equal(t, uint16(5), handler.marginCalls[0].bottom)
	}
}
