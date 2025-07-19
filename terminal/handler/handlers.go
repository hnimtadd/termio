package handler

import (
	"github.com/hnimtadd/termio/terminal/core"
	"github.com/hnimtadd/termio/terminal/sequences/csi"
	"github.com/hnimtadd/termio/terminal/sgr"
)

type (
	FormatEffectorHandler interface {
		// NextLine move cursor to the first position of next line, if the
		// cursor is at the bottom of the screen, a scroll up is performed.
		NextLine()
		// Index moves cursor downward one line without changing the
		// column position. If the active position is at the bottom of the
		// screen, a scroll up is performed.
		Index()
		// ReverseIndex moves cursor upward one line without changing the
		// column position. If the active position is at the top of the screen,
		// a scroll down is performed.
		ReverseIndex()
		// TabSet sets one horizontal stop at the active position.
		TabSet()
		// FullReset resets all attributes to their defaults.
		FullReset()
	}

	SGRHandler interface {
		SetGraphicsRendition(sgr *sgr.Attribute)
	}
	VT100Handler interface {
		// SetMode sets the mode to the given value, if the mode is not
		// settable, it skips.
		SetMode(mode core.Mode, value bool)
	}
	// EditorHandler interface includes all cursor movement and content
	// related methods
	EditorHandler interface {
		// DeleteChars deletes char repeated time start at the current cursor
		// position rightward
		DeleteChars(reepeated uint16)
		// DeleteLines deletes line repeated time start at the current cursor
		// position downward
		DeleteLines(repeated uint16)
		// InsertLines inserts line repeated time start at the current cursor
		// position downward
		InsertLines(repeated uint16)
		// InsertBlanks inserts blanks repeated time start at the current
		// cursor position rightward
		InsertBlanks(repeated uint16)
		// EraseInLine erases chars in line with behavior depends on mode
		EraseInLine(mode csi.ELMode)
		// EraseInDisplay erases chars in display with behavior depends on mode
		EraseInDisplay(erase csi.EDMode)
		// LineFeed moves cursor to the first position of next line,
		LineFeed()
		// Backspace moves cursor to the left one character position,
		// unless it is at the left margin, in which case no action occurs.
		Backspace()
		// SetCursorRow moves cursor to rows
		SetCursorRow(row uint16)
		// SetCursorCol moves cursor to cols
		SetCursorCol(col uint16)
		// SetCursorPosition moves cursor to row and col
		SetCursorPosition(row, col uint16)
		// SetCursorUp moves cursor up by offset, carriage controls whether
		// the cursor stays in same col position or movess to col 0
		SetCursorUp(offset uint16, carriage bool)
		// SetCursorDown moves cursor down by offset, carriage controls whether
		// the cursor stays in same col position or movess to col 0
		SetCursorDown(offset uint16, carriage bool)
		// SetCursorLeft moves cursor left by offset, unless it is at
		// the left margin, in which case no actions occurs
		SetCursorLeft(offset uint16)
		// SetCursorRight moves cursor right by offset, unless it is at
		// the right margin, in which case no actions occurs
		SetCursorRight(offset uint16)
		// SetCursorTabRight move cursor to the repeated next tab stop,
		// or to the right margin if no further tab stops are present
		// on the line.
		SetCursorTabRight(repeated uint16)
		// SetCursorTabLeft move cursor to the repeated previous tab stop,
		// or to the left margin if no further tab stops are present
		// on the line.
		SetCursorTabLeft(repeated uint16)
		// CarriageReturn moves cursor to left margin of the current line.
		CarriageReturn()
	}
)
