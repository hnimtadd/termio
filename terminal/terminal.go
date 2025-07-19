package terminal

import (
	"bytes"

	"github.com/hnimtadd/termio/logger"
	"github.com/hnimtadd/termio/terminal/coordinate"
	"github.com/hnimtadd/termio/terminal/core"
	pagepkg "github.com/hnimtadd/termio/terminal/page"
	"github.com/hnimtadd/termio/terminal/point"
	"github.com/hnimtadd/termio/terminal/screen"
	"github.com/hnimtadd/termio/terminal/sequences/csi"
	"github.com/hnimtadd/termio/terminal/set"
	"github.com/hnimtadd/termio/terminal/sgr"
	"github.com/hnimtadd/termio/terminal/size"
	styleid "github.com/hnimtadd/termio/terminal/style/id"
	"github.com/hnimtadd/termio/terminal/tabstops"
	"github.com/hnimtadd/termio/terminal/utils"
	dw "github.com/mattn/go-runewidth"
)

type (
	Options struct {
		Cols int // The number of columns in the terminal
		Rows int // The number of rows in the terminal

		// The default mode state. When the terminal gets a reset, it wqill revert
		// back to this state
		Modes map[core.Mode]bool

		Logger logger.Logger
	}
	// Terminal mainly implemented for terminal that used to
	// execute 1 command only
	Terminal struct {
		// Screen-related fields
		Screen *screen.Screen

		// The size of the terminal
		rows, cols size.CellCountInt

		// The size of display in pixels
		width, height int
		Modes         *core.ModeState

		pwd string // Current working directory

		// The previous printed character, we need this one for the repeat
		// previous char CSI (ESC [ <n> b).
		previousChar *uint32

		// Where the tabstops are.
		tabstops *tabstops.Tabstops

		// The current sscrolling region.
		scrollingRegion *ScrollingRegion

		logger logger.Logger
	}

	// Scroll region is the are of the screen designated where scolling
	// occurs. When scrolling the screen, on this viewport is scroled.
	ScrollingRegion struct {
		// Top and bottom of the scroll region (0-indexed)
		// Precondition: top < bottom.
		top    size.CellCountInt
		bottom size.CellCountInt

		// Left/right scroll regions.
		// Precondition: right > left
		// Precondition: right <= cols - 1
		left  size.CellCountInt
		right size.CellCountInt
	}
)

func NewTerminal(opts Options) *Terminal {
	return &Terminal{
		Screen: screen.NewScreen(
			size.CellCountInt(opts.Cols),
			size.CellCountInt(opts.Rows),
		),
		rows:  size.CellCountInt(opts.Rows),
		cols:  size.CellCountInt(opts.Cols),
		Modes: core.NewModeState(opts.Modes, opts.Modes),
		tabstops: tabstops.NewTabstops(
			size.CellCountInt(opts.Cols),
			tabstops.TABSTOP_INTERVAL,
		),
		scrollingRegion: &ScrollingRegion{
			top:    0,
			bottom: size.CellCountInt(opts.Rows) - 1,
			left:   0,
			right:  size.CellCountInt(opts.Cols) - 1,
		},
		pwd:    "",
		logger: opts.Logger,
	}
}

// Backspace moves the cursor back a column (but not less than 0).
func (t *Terminal) Backspace() {
	t.SetCursorLeft(1)
}

// CarriageReturn moves cursor to first column of current line
func (t *Terminal) CarriageReturn() {
	// Always reset pending wrap state
	t.Screen.Cursor.PendingWrap = false

	var x size.CellCountInt
	// In origin mode, we always move to the left margin
	if t.Modes.Get(core.ModeOrigin) {
		x = t.scrollingRegion.left
	} else if t.Screen.Cursor.X >= t.scrollingRegion.left {
		x = t.scrollingRegion.left
	} else {
		x = 0
	}

	t.Screen.SetCursorHorizontalAbs(x)
}

func (t *Terminal) EraseInDisplay(mode csi.EDMode) {
	switch mode {
	case csi.EDModeComplete:
		// Delete all lines in the screen
		t.Screen.ClearRows(point.Point{
			Tag:        point.TagActive,
			Coordinate: coordinate.Point[size.CellCountInt]{},
		}, nil)
		t.Screen.Cursor.PendingWrap = false

	case csi.EDModeBelow:
		// All lines to the right (including the cursor)
		t.EraseInLine(csi.ELModeRight)

		// All lines below
		if t.Screen.Cursor.Y < t.rows-1 {
			t.Screen.ClearRows(
				point.Point{
					Tag: point.TagActive,
					Coordinate: coordinate.Point[size.CellCountInt]{
						Y: t.Screen.Cursor.Y + 1,
					},
				},
				nil,
			)
		}
		// Unset pending wrap state
		utils.Assert(!t.Screen.Cursor.PendingWrap)

	case csi.EDModeAbove:
		// All ines to the left (including the cursor)
		t.EraseInLine(csi.ELModeLeft)

		// Erase all line above
		if t.Screen.Cursor.Y > 0 {
			t.Screen.ClearRows(
				point.Point{
					Tag: point.TagActive,
					Coordinate: coordinate.Point[size.CellCountInt]{
						Y: 0,
					},
				},
				&point.Point{
					Tag: point.TagActive,
					Coordinate: coordinate.Point[size.CellCountInt]{
						Y: t.Screen.Cursor.Y - 1,
					},
				},
			)
		}
		// Unset pending wrap state

		utils.Assert(!t.Screen.Cursor.PendingWrap)
	case csi.EDModeScrollback:
		t.logger.Warn("scrollback not supported")
	default:
		t.logger.Warn("unimplemented erase display", "mode", mode)
	}
}

// EraseInLine implements terminalHandler.
func (t *Terminal) EraseInLine(mode csi.ELMode) {
	cursor := t.Screen.Cursor
	// Get our start/end positions depending on the mode.
	var start, end size.CellCountInt
	switch mode {
	case csi.ELModeRight:
		start = cursor.X

		// If our X is a wide spacer tail, then we need to erase the previous
		// cell too, as we don't want to split a multi-cell character.
		if start > 0 &&
			cursor.PageCell.Wide == pagepkg.WideSpacerTail {
			start--
		}
		end = t.cols
	case csi.ELModeLeft:
		start = 0

		// If our X is a wide char, then we need to erase the wide char tail
		// too, as we don't want to split a multi-cell character.
		if cursor.PageCell.Wide == pagepkg.WideWide {
			start++
		}
		end = cursor.X + 1

	case csi.ELModeAll:
		start = 0
		end = t.cols
	default:
		t.logger.Error("unimplemented erase line", "mode", mode)
		return
	}

	utils.Assert(end > start)

	// All modes will clear the pending wrap state and we know we have a valid
	// mode at this point
	cursor.PendingWrap = false

	// We always mark our row as dirty
	t.Screen.CursorMarkDirty()

	t.Screen.ClearCells(cursor.PagePin.Node.Data, cursor.PageRow, start, end)
}

// Full reset.
//
// This will attempt to free the existing screen memory
func (t *Terminal) FullReset() {
	t.Screen.Reset()
	t.Modes.Reset()
	t.previousChar = nil
	t.pwd = ""
}

// Linefeed moves the cursor to the next line.
func (t *Terminal) LineFeed() {
	t.Index()
	if t.Modes.Get(core.ModeLineFeed) {
		t.CarriageReturn()
	}
}

// Print implements terminalHandler.
func (t *Terminal) Print(c uint32) {
	// After doing any printing, wrapping, etc. we want to ensure that our
	// display remains in a consistent state.
	defer t.Screen.AssertIntegrity()
	var rightLimit size.CellCountInt

	// our right margin depends where our cursor is now
	if t.Screen.Cursor.X > t.scrollingRegion.right {
		rightLimit = t.cols
	} else {
		rightLimit = t.scrollingRegion.right + 1
	}

	// Determine the width of this character so we can handle
	// non-single-width characters properly. We have a fast-path for byte-sized
	// characters since they're so common. We can ignore control characters
	// because they are always filtered prior.
	var width size.CellCountInt
	if c <= 0xFF {
		width = 1
	} else {
		width = size.CellCountInt(dw.RuneWidth(rune(c)))
	}

	utils.Assert(width <= 2)

	// Attach zero-width characters to our cell
	if width == 0 {
		t.logger.Warn("zero-width character:, ignoring", c)
		// This one is for grapheme cluster or any non-mono-width character
		// currently, mark as don't support
		return
	}
	t.previousChar = &c

	// If we're soft-wrapping, then handle that first.
	if t.Screen.Cursor.PendingWrap && t.Modes.Get(core.ModeWraparound) {
		t.PrintWrap()
	}

	// If we have insert mode enabled, then we need to handle that.
	// We only do insert mode if we are not at the end of the line.
	if t.Modes.Get(core.ModeInsert) && t.Screen.Cursor.X+width < t.cols {
		t.InsertBlanks(uint16(width))
	}
	switch width {
	// Single cell, is very easy, just write in the cell
	case 1:
		t.Screen.CursorMarkDirty()
		t.printCell(c, pagepkg.WideNarrow)

	// Wide character requires a spacer. We print this by using two cells:
	// the first is flagged "wide" and has the wide char. The second is spacer
	// tail if we are not at the end of the line, or spacer head if we are at
	// the end of the line.
	case 2:
		if (rightLimit - t.scrollingRegion.left) > 1 {
			//  If we don't have space for the wide char, we need to insert
			//  spacers and wrap, then we just print the wide char as normal
			if t.Screen.Cursor.X == rightLimit-1 {
				// If we don't have wraparound enabled, then we don't print
				// this character at all and don't move the cursor.
				// This is how xterm behaves
				if t.Modes.Get(core.ModeWraparound) {
					return
				}

				if rightLimit == t.cols {
					t.printCell(c, pagepkg.WideSpacerHead)
				} else {
					t.printCell(c, pagepkg.WideNarrow)
				}
				t.PrintWrap()
			}

			// We are at new line now, and ready to write 1 widechar with space
			// tail
			t.Screen.CursorMarkDirty()
			t.printCell(c, pagepkg.WideWide)
			t.Screen.SetCursorRight(1)
			t.printCell(0, pagepkg.WideSpacerTail)
		} else {
			// This is pretty broken, terminals should nevel be only 1-wide.
			// We should pretty much never hit this case.
			t.Screen.CursorMarkDirty()
			t.printCell(c, pagepkg.WideNarrow)
		}
	}

	// If we are at the end of the line, we need to wrap the next time
	// In this case, we don't move the cursor
	if t.Screen.Cursor.X+width == rightLimit {
		t.Screen.Cursor.PendingWrap = true
		return
	}

	t.Screen.SetCursorRight(1)
}

func (t *Terminal) PrintWrap() {
	// We only mark that we soft-wrapped if we're at the edge of our
	// full screen. We don't mark the row as wrapped if we're in the
	// middle due to a right margin.
	markWrap := t.Screen.Cursor.X == t.cols-1
	if markWrap {
		t.Screen.Cursor.PageRow.Wrap = true
	}

	// Get the old semantic prompt so we can extend it to the next
	// line. We need to do this before we index() because we may
	// modify memory.
	oldPrompt := t.Screen.Cursor.PageRow.SemanticPrompt

	// Move to the next line
	t.Index()
	t.Screen.SetCursorHorizontalAbs(t.scrollingRegion.left)
	if markWrap {
		// New line must inherit semantic prompt of the old line
		t.Screen.Cursor.PageRow.SemanticPrompt = oldPrompt
		t.Screen.Cursor.PageRow.WrapContinuation = true
	}
	// Assure that our screen is consistent
	t.Screen.AssertIntegrity()
}

func (t *Terminal) printCell(c uint32, wide pagepkg.Wide) {
	cursor := t.Screen.Cursor
	defer t.Screen.AssertIntegrity()

	cell := t.Screen.Cursor.PageCell

	// If the wide property of this cell is the same, then we don't need to do
	// the special handling here because the structure will be the same. If it
	// is NOT the same, then we may need to clear some cells. Ignoring now, as
	// we don't support 0-width character yet. E.g: grapheme cluster
	if cell.Wide != wide {
		switch cell.Wide {
		case pagepkg.WideNarrow:
			break
		// Previous cell was wide, so we need to clear the tail and head.
		case pagepkg.WideWide:
			if cursor.X >= t.cols-1 {
				break
			}

			t.Screen.ClearCells(
				cursor.PagePin.Node.Data,
				cursor.PageRow,
				cursor.X+1, // spacer cell is at cursor.X+1
				cursor.X+2,
			)

			if cursor.Y > 0 && cursor.X <= 1 {
				headCell := t.Screen.CursorCellEndOfPrevious()
				headCell.Wide = pagepkg.WideNarrow
			}
		case pagepkg.WideSpacerTail:
			utils.Assert(cursor.X > 0)

			// So integrity check pass. We fix this up later so we don't need
			// to do this without safety checks.
			cell.Wide = pagepkg.WideNarrow

			t.Screen.ClearCells(
				cursor.PagePin.Node.Data,
				cursor.PageRow,
				cursor.X-1, // cursor.X-1 is the wide cell
				cursor.X,
			)
			if cursor.Y > 0 && cursor.X <= 1 {
				// We need to clear the head of the previous cell
				headCell := t.Screen.CursorCellEndOfPrevious()
				headCell.Wide = pagepkg.WideNarrow
			}
			// this case was not handled in the old terminal implementation
			// but it feels like we should do something. investigate other
			// terminals (xterm mainly) and see whats up
		case pagepkg.WideSpacerHead:
			break

		}
	}

	// We don't need to update the style refs unless the cell's new style
	// will be different after writing.
	styleChanged := cell.StyleID != cursor.StyleID
	if styleChanged {
		page := cursor.PagePin.Node.Data
		// Release the old style
		if cell.StyleID != styleid.DefaultID {
			utils.Assert(cursor.PageRow.Styled)
			page.Styles.Release(set.ID(cell.StyleID))
		}
	}

	{
		(*cell).ContentTag = pagepkg.ContentTagCP
		(*cell).ContentCP = c
		(*cell).StyleID = cursor.StyleID
		(*cell).Wide = wide
	}

	if styleChanged {
		page := cursor.PagePin.Node.Data

		// Use the new style
		if cell.StyleID != styleid.DefaultID {
			page.Styles.Use(set.ID(cell.StyleID))
			cursor.PageRow.Styled = true
		}
	}
}

// SetCursorRow implements terminalHandler.
func (t *Terminal) SetCursorRow(row uint16) {
	t.Screen.SetCursorHorizontalAbs(size.CellCountInt(row))
}

func (t *Terminal) SetCursorTabStop() {
	// Set the current cursor position as a tabstop
	t.tabstops.Set(t.Screen.Cursor.X)
}

// Move the cursor left amount collumns. If amount is greater than the maximum
// move distance then it is internally adjusted to the maximum move distance.
// If amount is 0, adjust it to 1.
func (t *Terminal) SetCursorLeft(offset uint16) {
	// Wrapping behavior depneds on various terminal modes
	switch {
	case t.Modes.Get(core.ModeWraparound):
		// If offset is 0, adjust it to 1
		count := size.CellCountInt(max(offset, 1))

		t.Screen.SetCursorLeft(min(count, t.Screen.Cursor.X))
		t.Screen.Cursor.PendingWrap = false

		// TODO: handle reverse wrap
	}
}

// Move the cursor down amount line. If amount is greater than the maximum
// move distance then it is internally adjusted to the maximum move distance.
// If amount is 0, adjust it to 1.
func (t *Terminal) SetCursorDown(offset uint16, carriage bool) {
	// Always reset pending wrap state
	t.Screen.Cursor.PendingWrap = false

	// The maximum amount the cursor can move up depends on scrolling regions
	var maxm size.CellCountInt
	if t.Screen.Cursor.Y <= t.scrollingRegion.bottom {
		// inside of scrolling region, margin is to the bottom of the scrolling
		// region
		maxm = t.scrollingRegion.bottom - t.Screen.Cursor.Y
	} else {
		// outside of scrolling region, margin is to the bottom of the screen
		maxm = (t.rows - 1) - t.Screen.Cursor.Y
	}
	adjustedCount := min(maxm, max(size.CellCountInt(offset), 1))

	t.Screen.SetCursorDown(adjustedCount)
}

// Move the cursor up amount line. If amount is greater than the maximum move
// distance then it is internally adjusted to the maximum move distance.
// If amount is 0, adjust it to 1.
func (t *Terminal) SetCursorUp(offset uint16, carriage bool) {
	// Always reset pending wrap state
	t.Screen.Cursor.PendingWrap = false

	// The maximum amount the cursor can move up depends on scrolling regions
	var maxm size.CellCountInt
	if t.Screen.Cursor.Y >= t.scrollingRegion.top {
		// inside scrolling region, margin is to the top of scrolling region
		maxm = t.Screen.Cursor.Y - t.scrollingRegion.top
	} else {
		// outside scrolling region, margin is to the top of the screen
		maxm = t.Screen.Cursor.Y
	}

	adjustedCount := min(maxm, max(size.CellCountInt(offset), 1))

	t.Screen.SetCursorUp(adjustedCount)
}

// Move the cursor right amount collumns. If amount is greater than the maximum
// move distance then it is internally adjusted to the maximum move distance.
// If amount is 0, adjust it to 1.
func (t *Terminal) SetCursorRight(offset uint16) {
	// Always reset pending wrap state
	t.Screen.Cursor.PendingWrap = false

	// The maximum amount the cursor can move to depends where the cursor is
	var maxm size.CellCountInt
	if t.Screen.Cursor.X <= t.scrollingRegion.right {
		// inside scrolling region, margin is to the right of scrolling region
		maxm = t.scrollingRegion.right - t.Screen.Cursor.X
	} else {
		// outside of scrolling region, margin is to the right of the screen
		maxm = t.cols - t.Screen.Cursor.X - 1
	}
	offset = min(uint16(maxm), max(offset, 1))
	t.Screen.SetCursorRight(size.CellCountInt(offset))
}

// SetCursorTabRight move the cursor to the next tabstop, clearing the screen
// toe the left of the tabstop.
func (t *Terminal) SetCursorTabRight(repeated uint16) {
	for range repeated {
		for t.Screen.Cursor.X < t.cols {
			// Move the cursor right
			t.Screen.SetCursorRight(1)

			// If the last cursor position was a tabstop, we return. We do
			// "last cursor position" becasue we want a space to be written
			// at the tab stop unless we're at the end.
			if t.tabstops.Get(t.Screen.Cursor.X) {
				return
			}
		}
	}
}

// SetCursorTabLeft similar to SetCursorTabRight, but move the cursor to the
// previous tabstop instead
func (t *Terminal) SetCursorTabLeft(repeated uint16) {
	var leftLimit size.CellCountInt
	// With origin mode enabled, our leftmost limit is the left margin
	if t.Modes.Get(core.ModeOrigin) {
		leftLimit = t.scrollingRegion.left
	} else {
		leftLimit = 0
	}
	for range repeated {
		for t.Screen.Cursor.X > leftLimit {
			// Move the cursor left
			t.Screen.SetCursorLeft(1)

			if t.tabstops.Get(t.Screen.Cursor.X) {
				return
			}
		}
	}
}

// SetGraphicsRendition implements terminalHandler.
func (t *Terminal) SetGraphicsRendition(sgr *sgr.Attribute) {
	t.Screen.SetGraphicsRendition(sgr)
}

// TabSet implements terminalHandler.
func (t *Terminal) TabSet() {
	t.tabstops.Set(t.Screen.Cursor.X)
}

// Moves the cursor to the next line.
//
// If the cursor is outside of the scrolling region: move the cursor one line
// down if it isn not on the bottom-most line of the screen.
//
// If the cursor is inside the scrolling region:
//   - If the cursor is on the bottom-most line of the scrolling region,
//     a scroll up is performed
//     with amount=1
//   - If the cursor is not on the bottom-most line of the scrolllng region,
//     move the cursor one line down
//
// This unset the pending wrap state without wraping.
func (t *Terminal) Index() {
	// Unset pending wrap state
	t.Screen.Cursor.PendingWrap = false

	// Outside of the scrolling region, we move the cursor one line down.
	if t.Screen.Cursor.Y < t.scrollingRegion.top ||
		t.Screen.Cursor.Y > t.scrollingRegion.bottom {
		// We only move down if we are not already at the bottom of the
		// screen
		if t.Screen.Cursor.Y < t.rows-1 {
			t.Screen.SetCursorDown(1)
		}
		return
	}
	// If the cursor is inside the scrolling region, and on the bottom-most
	// line, then we scroll up. If our scrolling region is the full screen
	// we create scrollback.
	if (t.Screen.Cursor.Y == t.scrollingRegion.bottom) &&
		t.Screen.Cursor.X >= t.scrollingRegion.left &&
		t.Screen.Cursor.X <= t.scrollingRegion.right {

		// If our scrlling region is at the top, we create scrollback.
		if t.scrollingRegion.top == 0 &&
			t.scrollingRegion.left == 0 &&
			t.scrollingRegion.right == t.cols-1 {
			t.Screen.SetCursorScrollUp()
			return

		}
		// Preserve old cursor just for assertions.
		oldCursor := t.Screen.Cursor

		t.Screen.Pages.EraseRowsBounded(point.Point{
			Tag: point.TagActive,
			Coordinate: coordinate.Point[size.CellCountInt]{
				Y: t.scrollingRegion.top,
			},
		}, t.scrollingRegion.bottom-t.scrollingRegion.top)

		// eraseRow will end up moving the cursor pin up by 1, so we need to move
		// it back down.
		utils.Assert(t.Screen.Cursor.X == oldCursor.X)
		utils.Assert(t.Screen.Cursor.Y == oldCursor.Y)
		t.Screen.Cursor.Y -= 1
		t.Screen.SetCursorDown(1)

		// The operations above can prune our cursor style, so we need to
		// update. This should never fail because the above can only FREE
		// memory
		t.Screen.ManualStyleUpdate()
		return
	}

	// Increase the cursor by 1, maximum to bottom of view region
	if t.Screen.Cursor.Y < t.scrollingRegion.bottom {
		t.SetCursorDown(1, false)
	}
}

// ReverseIndex moves the cursor to the previous line, possibly scrolling.
//
// If the cursor is outside of the scrolling region, move the cursor one line
// up if it is not on the top-most line of the screen
//
// If the cursor is inside the scrolling region:
//
// * If the cursor is on the top-most line: invoke scrolldown with amount=1
// * If the cursor is not on the top-most line: just move 1 line up
func (t *Terminal) ReverseIndex() {
	if t.Screen.Cursor.Y != t.scrollingRegion.top ||
		t.Screen.Cursor.X < t.scrollingRegion.left ||
		t.Screen.Cursor.X > t.scrollingRegion.right {
		t.SetCursorUp(1, false)
		return
	}
	t.cursorScrollDown(1)
}

// SetCursorPosition move cursor to the position indicated
// by row and col (1-indexed). If collumn = 0, it is adjusted to 1.
// If column > the right-most col, it is adjusted to the right-most col.
// If row = 0, it is adjusted to 1.
// If row > the bottom-most row, it is adjusted to the bottom-most row.
func (t *Terminal) SetCursorPosition(row uint16, col uint16) {
	// If cursor origin mode is set the cursor row will be moved relative to
	// the top margin row and adjusted to be above or at bottom-most row
	// in the current scroll region.
	//
	// If origin mode is set and left and right margin mode is set the cursor
	// will be moved relative to the left margin column and adjusted to be on
	// or left of the right margin column.
	type params struct {
		xOffset size.CellCountInt
		yOffset size.CellCountInt
		xMax    size.CellCountInt
		yMax    size.CellCountInt
	}
	var p params

	if t.Modes.Get(core.ModeOrigin) {
		p = params{
			xOffset: t.scrollingRegion.left,
			yOffset: t.scrollingRegion.top,
			xMax:    t.scrollingRegion.right + 1,  // 1-indexed
			yMax:    t.scrollingRegion.bottom + 1, // 1-indexed
		}
	} else {
		p = params{
			xMax: t.cols,
			yMax: t.rows,
		}
	}

	// Unset pending wrap state
	t.Screen.Cursor.PendingWrap = false

	// Calculate new x/y
	var irow, icol size.CellCountInt
	if row == 0 {
		irow = 1
	} else {
		irow = size.CellCountInt(row)
	}

	if col == 0 {
		icol = 1
	} else {
		icol = size.CellCountInt(col)
	}

	var y, x size.CellCountInt
	x = max(min(p.xMax, icol+p.xOffset)-1, 0)
	y = max(min(p.yMax, irow+p.yOffset)-1, 0)
	cursor := t.Screen.Cursor

	// If the y is unchanged then this is fast pointer math
	if y == cursor.Y {
		if x > cursor.X {
			t.Screen.SetCursorRight(x - cursor.X)
		} else {
			t.Screen.SetCursorLeft(cursor.X - x)
		}
		return
	}

	// If everything changed we do an absolute change which is slightly slower
	t.Screen.SetCursorAbs(x, y)
}

// Removes repeated lines from the top of the scroll region. The remaining
// lines to the bottom margin are shifted up and space from the bottom margin
// up is filled with empty lines.
//
// The new lines are created according to the current SGR state.
//
// Does not change the (absolute) cursor position.
func (t *Terminal) cursorScrollUp(repeated uint16) {
	// Preserve our x/y to restore
	oldX := t.Screen.Cursor.X
	oldY := t.Screen.Cursor.Y
	oldWrap := t.Screen.Cursor.PendingWrap
	defer func() {
		t.Screen.SetCursorAbs(oldX, oldY)
		t.Screen.Cursor.PendingWrap = oldWrap
	}()

	// Move the cursor to the top of the scroll region
	t.Screen.SetCursorAbs(t.scrollingRegion.left, t.scrollingRegion.top)
	t.DeleteLines(repeated)
}

// Scroll the text down by one row.
func (t *Terminal) cursorScrollDown(repeated uint16) {
	// Preserve our x/y to restore
	oldX, oldY, oldWrap := t.Screen.Cursor.X, t.Screen.Cursor.Y, t.Screen.Cursor.PendingWrap
	defer func() {
		t.Screen.SetCursorAbs(oldX, oldY)
		t.Screen.Cursor.PendingWrap = oldWrap
	}()

	// Move the cursor to the top of the screen
	t.Screen.SetCursorAbs(0, 0)
	t.InsertLines(repeated)
}

// Insert line repeated time at the current cursor row. The content of the
// line at the current cursor row and below (to the bottom-most line in the
// scrollingRegion) are shifted down by amount lines.
//
// This unsets the pending wrap state without wrapping. If the current cursor
// position is outside of the current scroll region it does nothing.
//
// If amount is greater than the remaining number of lines in the scrolling
// region it is adjusted down (still alowing for scrolling out every remaini
// line in the scrlling region)
//
// If left and right margin mode the margins are respected; lines are only
// scrolled in the scroll region.
//
// All cleared space is colored according to the current SGR state.
//
// Move the cursor to the left margin
func (t *Terminal) InsertLines(repeated uint16) {
	if repeated == 0 {
		return
	}

	// If the cursor is outside the scroll region, we do nothing
	if t.Screen.Cursor.Y < t.scrollingRegion.top ||
		t.Screen.Cursor.Y > t.scrollingRegion.bottom ||
		t.Screen.Cursor.X < t.scrollingRegion.left ||
		t.Screen.Cursor.X > t.scrollingRegion.right {
		return
	}

	// At the end, we need to return the cursor to the row it started on.
	startY := t.Screen.Cursor.Y
	defer func() {
		t.Screen.SetCursorAbs(startY, startY)
		// Always reset pending wrap state
		t.Screen.Cursor.PendingWrap = false
	}()
	/*
	* ------------------------------------------
	* |                                        |
	* |                                        |
	* |                                   |----| -| <- start row
	* |                                   |    |  |
	* |                                   |    |  | repeated
	* |                             rem   |    | -|
	* |                                   |    |
	* |                                   |    |
	* |                                   |    |
	* |                                   |----|
	* ------------------------------------------
	**/

	// We have a slower path if we have left or right scroll margin.
	leftRight := t.scrollingRegion.left > 0 ||
		t.scrollingRegion.right < t.cols-1

	// Remaining rows from our cursor to the bottom of the screen
	rem := t.rows - t.Screen.Cursor.Y + 1
	// t.screen.InsertNewLines(repeated)

	// We can only insert delete up to our remaining lines in the screen, so we
	// take wichever is smaller
	adjustedCount := min(size.CellCountInt(repeated), rem)

	// Create a new tracked pin which we will use to navigate the page list
	// so that if we need to adjust capacity, it will properly tracked.
	curP := t.Screen.Pages.TrackPin(*t.Screen.Cursor.PagePin.Down(rem - 1))
	defer t.Screen.Pages.UntrackPin(curP)

	// y is our current y position relative to the cursor.
	for y := rem; y > 0; y-- {
		curRAC := curP.RowAndCell()
		curRow := curRAC.Row

		// Mark the row as dirty.
		curP.MarkDirty()

		// If this is one of the lines we need to shift, do so
		if y > adjustedCount {
			offP := curP.Up(adjustedCount)
			offRAC := offP.RowAndCell()
			offRow := offRAC.Row
			t.rowWillBeShifted(curP.Node.Data, curRow)
			t.rowWillBeShifted(offP.Node.Data, offRow)

			// If our scrolling region is full width, then we unset wrap
			if !leftRight {
				t.logger.Info("handle unset wrap here")
			}
			srcP := offP
			srcRow := offRAC.Row
			dstP := curP
			dstRow := curRAC.Row

			// if our page doesn't match, then we need to do a copy from one
			// page to another. This is the slow path.
			if srcP.Node != dstP.Node {
				if err := dstP.Node.Data.ClonePartialRowFrom(
					srcP.Node.Data,
					dstRow,
					srcRow,
					t.scrollingRegion.left,
					t.scrollingRegion.right+1,
				); err != nil {
					// continue the loop to try handling this row again.
					continue
				}
			} else {
				if !leftRight {
					// Swap the src/dst cells. This ensures that our dst gets
					// the proper shifted rows and src gets non-garbage cell
					// data that we can clear.
					dst := *dstRow
					dstRow.Cells = srcRow.Cells
					dstRow.SemanticPrompt = srcRow.SemanticPrompt
					dstRow.WrapContinuation = srcRow.WrapContinuation
					dstRow.Wrap = srcRow.Wrap

					*srcRow = dst

					// Ensure that we didn't corrupt the page
					curP.Node.Data.AssertIntegrity()
				} else {
					// Left/right scroll margins we have to
					// copy cells, which is much slower...
					page := curP.Node.Data
					page.MoveCells(
						srcRow, t.scrollingRegion.left,
						dstRow, t.scrollingRegion.left,
						size.CellCountInt(t.scrollingRegion.right-t.scrollingRegion.left+1),
					)
				}
			}
		} else {
			// Clear the cells for this row, it's has been shifted
			page := curP.Node.Data
			t.Screen.ClearCells(page, curRow,
				t.scrollingRegion.left, t.scrollingRegion.right+1)
		}
	}
}

// Remove line repeated times from the cursor row doward. The remaining lines
// to the bottom margin are shifted up and space from the bottom margin up is
// filled with empty lines.
//
// If the cursor is outside of the scrolling region, this does nothing.
//
// If repeated is greater than the remaining number of lines in the scrolling
// region it is adjusted down
//
// In left and right margin mode, the margins are respected; lines are only
// scrolled in the scroll region.
//
// # If the cell movement split a multi-cell character, that character cleared,
// by replacing it by spaces, keepings its current attributes. All other
// cleared space is colored according to the current SGR state.
//
// Moves the cursor to the left margin.
func (t *Terminal) DeleteLines(repeated uint16) {
	if repeated == 0 {
		return
	}

	// If the cursor is outside the scrolling region, we do nothing.
	if t.Screen.Cursor.Y < t.scrollingRegion.top ||
		t.Screen.Cursor.Y > t.scrollingRegion.bottom ||
		t.Screen.Cursor.X < t.scrollingRegion.left ||
		t.Screen.Cursor.X > t.scrollingRegion.right {
		return
	}

	// At the end, we need to return the cursor to the row it started on.
	startY := t.Screen.Cursor.Y
	defer func() {
		t.Screen.SetCursorAbs(startY, startY)
		// Always reset pending wrap state
		t.Screen.Cursor.PendingWrap = false
	}()
	/*
	* ------------------------------------------
	* |                                        |
	* |                                        |
	* |                                   |----| -| <- start row
	* |                                   |    |  |
	* |                                   |    |  | repeated
	* |                             rem   |    | -|
	* |                                   |    |
	* |                                   |    |
	* |                                   |    |
	* |                                   |----|
	* ------------------------------------------
	 */

	// We have a slower path if we have left or right scroll margin.
	leftRight := t.scrollingRegion.left > 0 ||
		t.scrollingRegion.right < t.cols

	// Remaining rows from our cursor to the bottom of the screen
	rem := t.rows - t.Screen.Cursor.Y + 1

	// We can only insert delete up to our remaining lines in the screen, so we
	// take wichever is smaller
	adjustedCount := min(size.CellCountInt(repeated), rem)

	// Create a new tracked pin which we will use to navigate the curP list
	// so that if we need to adjust capacity, it will properly tracked.
	curP := t.Screen.Cursor.PagePin

	for y := size.CellCountInt(0); y < rem; {
		curRAC := curP.RowAndCell()
		curRow := curRAC.Row
		curP.MarkDirty()

		// If this is one of the lines we need to shift, do so
		if y < adjustedCount {
			offPage := curP.Down(adjustedCount)
			offRAC := offPage.RowAndCell()
			offRow := offRAC.Row

			t.rowWillBeShifted(curP.Node.Data, curRow)
			t.rowWillBeShifted(curP.Node.Data, offRow)

			if !leftRight {
				t.logger.Info("handle unset wrap here")
			}
			srcP := offPage
			srcRow := offRow
			dstP := curP
			dstRow := curRow

			// If our page doesn't match, then we need to do a copy from one
			// page to another. This is the slow path.
			if srcP.Node != dstP.Node {
				if err := dstP.Node.Data.ClonePartialRowFrom(
					srcP.Node.Data,
					dstRow,
					srcRow,
					t.scrollingRegion.left,
					t.scrollingRegion.right+1,
				); err != nil {
					continue
				}
			} else {
				if !leftRight {
					// swap the src/dst cells. This ensures that our dst gets
					// the proper shifted rows and src gets non-garbage cell
					// data that we can clear.
					dst := *dstRow

					dstRow.Cells = srcRow.Cells
					dstRow.SemanticPrompt = srcRow.SemanticPrompt
					dstRow.WrapContinuation = srcRow.WrapContinuation
					dstRow.Wrap = srcRow.Wrap

					*srcRow = dst

					// Ensure that we didn't corrupt the page
					curP.Node.Data.AssertIntegrity()
				} else {
					// Left/right scroll margins we have to
					// copy cells, which is much slower...
					page := curP.Node.Data
					page.MoveCells(
						srcRow, t.scrollingRegion.left,
						dstRow, t.scrollingRegion.left,
						size.CellCountInt(t.scrollingRegion.right-t.scrollingRegion.left+1),
					)
				}
			}
		} else {
			// Clear the cells for this row, it's from out of bound
			page := curP.Node.Data
			t.Screen.ClearCells(page, curRow,
				t.scrollingRegion.left, t.scrollingRegion.right+1)
		}
		// we have sucessfully process a line.
		y += 1
		curP = curP.Down(1)
	}
}

// Inserts spaces at current cursor position moving existing cell contents
// to the right. The contents of the count right-most columns in the scroll
// region are lost. The cursor position is not changed.
//
// This unset the pending wrap state without wraping.
//
// The inserted cells are colored according the the current SGR state.
func (t *Terminal) InsertBlanks(repeated uint16) {
	cursor := t.Screen.Cursor
	// Unset pending wrap state without wrapping. Not: this purposely happens
	// BEFORE the scroll region check below.
	cursor.PendingWrap = false

	// If our cursor is outside the margins then do nothing. We DO reset
	// wrap state still so this must remain below the above logic.
	if cursor.X < t.scrollingRegion.left ||
		cursor.X > t.scrollingRegion.right {
		return
	}

	// leftX is the cursor position
	leftX := t.Screen.Cursor.X
	page := cursor.PagePin.Node.Data

	// if our X is a wide spacer tail, then we need to erase the the previous
	// cell too, as we don't want to split a multi-cell character.
	if cursor.PageCell.Wide == pagepkg.WideSpacerTail {
		utils.Assert(cursor.X > 0)
		t.Screen.ClearCells(page, cursor.PageRow, leftX-1, leftX)
	}

	// Remaining cols from our cursor to the right margin
	rem := t.cols - cursor.X + 1

	// We can only insert blanks up to our remaining cols
	adjustedCount := min(size.CellCountInt(repeated), rem)

	// This is the amount of space at the right of the line that will not be
	// Blank, so we need to shift the correct cols right.
	// "amount" is the number of such cols.
	amount := rem - adjustedCount
	if amount > 0 {
		page.PauseIntegrityChecks(true)
		defer page.PauseIntegrityChecks(false)

		x := leftX + (amount - 1)
		// if our last cell we're shifting is a wide, then we need to clear
		// it to be empty, so we don't split a multi-cell char.
		end := cursor.PageRow.Cells[x]
		if end.Wide == pagepkg.WideWide {
			utils.Assert(cursor.PageRow.Cells[x+1].Wide == pagepkg.WideSpacerTail)
			t.Screen.ClearCells(page, cursor.PageRow, x, x+1)
		}

		// We work backwards, so we don't overwrite data.
		for ; x >= leftX; x-- {
			src := cursor.PageRow.Cells[x]
			dst := cursor.PageRow.Cells[x+adjustedCount]
			page.SwapCells(src, dst)
		}
	}

	// Insert blanks. The blanks preserve the background color.
	t.Screen.ClearCells(page, cursor.PageRow, leftX, leftX+adjustedCount)

	// Our row is alway dirty
	t.Screen.CursorMarkDirty()
}

// Remove characters repeated times from the current position to the right.
// The remaining characters are shifted to the left and space from the right
// is filled with spaces
//
// If repated is greater than the remaining number of characters in the
// scrolling region, it is adjusted down.
//
// Does not move the cursor.; i++
func (t *Terminal) DeleteChars(repeated uint16) {
	if repeated == 0 {
		return
	}

	cursor := t.Screen.Cursor

	// If our cursor is outside the margins then do nothing. We DO reset
	// wrap state still so this must remain below the above logic.
	if t.Screen.Cursor.X < t.scrollingRegion.left ||
		t.Screen.Cursor.X > t.scrollingRegion.right {
		return
	}

	// leftX is the cursor position
	leftX := t.Screen.Cursor.X
	page := cursor.PagePin.Node.Data

	// Remaining cols from our cursor to the right margin
	rem := t.cols - t.Screen.Cursor.X + 1

	// We can only insert blanks up to our remaining cols
	count := min(size.CellCountInt(repeated), rem)

	t.Screen.SplitCellBoundary(t.Screen.Cursor.X)
	t.Screen.SplitCellBoundary(t.Screen.Cursor.X + count)
	t.Screen.SplitCellBoundary(t.scrollingRegion.right + 1)

	// This is the amount of space at the right of the line that will not
	// be blank, so we need to shift the correct cols right.
	// "amount" is the number of such cols.
	amount := rem - count
	x := leftX
	if amount > 0 {
		page.PauseIntegrityChecks(true)
		defer page.PauseIntegrityChecks(false)

		rightX := leftX + (amount - 1)

		for ; x < rightX; x++ {
			src := cursor.PageRow.Cells[x+count]
			dst := cursor.PageRow.Cells[x]
			page.SwapCells(src, dst)
		}
	}

	// Insert blanks. The blanks preserve the background color.
	t.Screen.ClearCells(page, cursor.PageRow, leftX, leftX+amount)

	// Our row's soft-wrap is always reset

	// Our row is always dirty
	t.Screen.CursorMarkDirty()
}

// To be called before shifting a row (as in InsertLines and deleteLines).
//
// Take care of boundary conditions such as potentially split wide chars
// across scrolling region boundaries and orphaned spacer heads at line ends.
func (t *Terminal) rowWillBeShifted(page *pagepkg.Page, row *pagepkg.Row) {
	// TODO: perform check if the last cell in this row is part of wide
	// character or not.
}

// Mark the current semantic prompt information. Current escape sequences
// (OSC 133) only allow setting this for wherever the current active cursor is
// located
func (t *Terminal) MarkSemanticPrompt(prompt pagepkg.SemanticPromptType) {
	switch prompt {
	case pagepkg.SemanticPromptTypePrompt,
		pagepkg.SemanticPromptTypeOutput,
		pagepkg.SemanticPromptTypeInput,
		pagepkg.SemanticPromptTypeContinuation:
		t.Screen.Cursor.PageRow.SemanticPrompt = prompt
	}
}

// Returns true if the cursor is currently at a prompt. Another way to look
// at this is it returns false if the shell is currently outputing something.
// This requires shll integration (sematic prompt integration).
//
// If the shell integration is not enabled, this will always return false.
func (t *Terminal) CursorIsAtPrompt() bool {
	// Reverse through the screen
	startX, startY := t.Screen.Cursor.X, t.Screen.Cursor.Y
	defer t.Screen.SetCursorAbs(startX, startY)

	for i := range startY + 1 {
		if i > 0 {
			t.Screen.SetCursorUp(1)
		}
		switch t.Screen.Cursor.PageRow.SemanticPrompt {
		// IF we're at a prompt or input area, then we are at a prompt
		case pagepkg.SemanticPromptTypePrompt,
			pagepkg.SemanticPromptTypeContinuation,
			pagepkg.SemanticPromptTypeInput:
			return true
		// If we have command output, then we're most certainly not at a prompt
		case pagepkg.SemanticPromptTypeOutput:
			return false
		default:
			continue
		}
	}
	return false
}

// Return the current string value of the terminal. Newline are encoded as "\n"
// This omits any formatting such as fg/bg.
func (t *Terminal) PlainString() string {
	w := bytes.NewBuffer(nil)
	if err := t.Screen.DumpString(w, point.TagViewPort); err != nil {
		return ""
	}
	return w.String()
}

// Resize the underlying terminal
func (t *Terminal) Resize(cols, rows size.CellCountInt) {
	// If our cols/rows didn't change, then we don't need to do anything
	if t.cols == cols && t.rows == rows {
		return
	}

	// Resize the tabstops
	if t.cols != cols {
		t.tabstops = tabstops.NewTabstops(cols, tabstops.TABSTOP_INTERVAL)
	}
	if t.Modes.Get(core.ModeWraparound) {
		t.Screen.ResizeWithReflow(cols, rows)
	} else {
		// If we're making the screen smaller, re-flow the screen
		t.Screen.ResizeWithoutReflow(cols, rows)
	}

	t.cols = cols
	t.rows = rows

	t.scrollingRegion = &ScrollingRegion{
		top:    0,
		bottom: rows - 1,
		left:   0,
		right:  cols - 1,
	}
}

// Set a style attibute
func (t *Terminal) SetAttribute(attr sgr.Attribute) {
	t.Screen.SetAttribute(attr)
}

// Set the pwd for the terminal
func (t *Terminal) SetPwd(pwd string) {
	t.pwd = pwd
}

// function to get the current pwd
func (t *Terminal) GetPwd() string {
	return t.pwd
}

// Returns true if the point is dirty, used for testing.
func (t *Terminal) isDirty(pt point.Point) bool {
	return t.Screen.Pages.GetCell(pt).IsDirty()
}

// Clear all dirty bits. Testing only.
func (t *Terminal) clearDirty() {
	t.Screen.Pages.ClearDirty()
}
