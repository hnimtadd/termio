package screen

// TODO: @dat.nguyen continue with pagelist implementation in screen

import (
	"fmt"

	"github.com/hnimtadd/termio/terminal/color"
	pagepkg "github.com/hnimtadd/termio/terminal/page"
	"github.com/hnimtadd/termio/terminal/pagelist"
	"github.com/hnimtadd/termio/terminal/point"
	"github.com/hnimtadd/termio/terminal/set"
	"github.com/hnimtadd/termio/terminal/sgr"
	"github.com/hnimtadd/termio/terminal/size"
	"github.com/hnimtadd/termio/terminal/style"
	styleid "github.com/hnimtadd/termio/terminal/style/id"
	"github.com/hnimtadd/termio/terminal/utils"
	dw "github.com/mattn/go-runewidth"
	"golang.org/x/text/encoding/unicode"
	"io"
)

//go:generate mockery --outpkg=screenmock --name=ScreenInt --filename=screen_mock.go --structname=MockScreen
type ScreenInt interface {
	// AssertIntegrity asserts that the display is in a consistent state
	// This is a no-op in production, but can be used in tests to ensure
	// that the display is in a valid state.
	AssertIntegrity()
	// CursorCellRight returns the cell at the right of the cursor
	CursorCellRight(n size.CellCountInt) *pagepkg.Cell
	// CursorCellLeft returns the cell at the left of the cursor
	CursorCellLeft(n size.CellCountInt) *pagepkg.Cell
	// SetCursorRight moves the cursor to the right by n cells
	// This is a specialized function that is very fast if the caller can
	// guarantee that we have space to move right (no wrapping)
	SetCursorRight(n size.CellCountInt)
	// SetCursorLeft moves the cursor to the left by n cells
	// This is a specialized function that is very fast if the caller can
	// guarantee that we have space to move left (no wrapping)
	SetCursorLeft(n size.CellCountInt)
	// SetCursorUp moves the cursor up by n cells
	// This is a specialized function that is very fast if the caller can
	// guarantee that we have space to move up (no wrapping)
	// Precondition: The cursor is not at the top of the screen
	SetCursorUp(n size.CellCountInt)
	// SetCursorDown moves the cursor down by n cells
	// This is a specialized function that is very fast if the caller can
	// guarantee that we have space to move down (no wrapping)
	// Precondition: The cursor is not at the bottom of the screen
	SetCursorDown(n size.CellCountInt)
	// SetCursorAbs moves the cursor to an absolute position
	SetCursorAbs(x size.CellCountInt, y size.CellCountInt)
	// CursorMarkDirty marks the cursor as dirty
	CursorMarkDirty()
	// SetCursorHorizontalAbs moves the cursor to an absolute horizontal
	// position
	SetCursorHorizontalAbs(x size.CellCountInt)
	// SetCursorVerticalAbs moves the cursor to an absolute vertical position
	SetCursorVerticalAbs(y size.CellCountInt)
	// GetCursor returns the current cursor position
	GetCursor() *Cursor
	// CursorCellEndOfPrevious returns the cell at the end of the previous
	// line
	CursorCellEndOfPrevious() *pagepkg.Cell
	// GetSize returns the current size of the display in rows and columns
	GetSize() (rows, cols size.CellCountInt)
	// SetGraphicsRendition sets the graphics rendition for the display
	SetGraphicsRendition(sgr *sgr.Attribute)
	// Resize the screen without any reflow. In this mode, columns/rows will
	// be truncated as they are shrunk. If they are grown, the new space is filled
	// with zeros.
	ResizeWithoutReflow(cols, rows size.CellCountInt)
	// ResizeWithReflow the screen with reflow
	ResizeWithReflow(cols, rows size.CellCountInt)
	// SetCursorScrollUp sets the cursor to scroll up
	SetCursorScrollUp()
	// SplitCellBoundary splits the cell boundary
	SplitCellBoundary(x size.CellCountInt)
	// ClearCells clears the cells
	ClearCells(page *pagepkg.Page, row *pagepkg.Row, fromX, toX size.CellCountInt)
	// ClearRows clears a range of rows
	ClearRows(tl point.Point, bl *point.Point)
	// Reset resets the display
	Reset()
	// DumpString dumps the screen to a string
	DumpString(writer io.Writer, tl point.Tag) error
	// // Row returns the index row
	// Row(y size.CellCountInt) *page.Row
}

var _ ScreenInt = &Screen{}

// Screen is a Screen for the terminal
type Screen struct {
	Cursor *Cursor
	// cells  []*page.Cell

	Pages *pagelist.PageList

	rows, cols size.CellCountInt

	// Special-case where we want no scrollback whatsever. We have to flag,
	//  this because MaxSize 0 in PageLists gets rounded up to two pages so we
	//  can alwasy have an active screen..
	NoScrollback bool
}

// Initialize a new display
func NewScreen(cols, rows size.CellCountInt) *Screen {
	// initialize out backing pages
	pages := pagelist.NewPageList(cols, rows)

	// Create our tracked pin for the cursor.
	pagePin := pages.TrackPin(pagelist.Pin{Node: pages.Pages.First})
	pageRAC := pagePin.RowAndCell()

	return &Screen{
		Cursor: &Cursor{
			X:        0,
			Y:        0,
			PageRow:  pageRAC.Row,
			PageCell: pageRAC.Cell,
			PagePin:  pagePin,
		},
		Pages: pages,
		rows:  rows,
		cols:  cols,
	}
}

// Asert that the screen is in a consistent state. This doesn't check all pages
// in the pages list because that is SO SLOW event just for tests. This only
// asserts the screen spcific data so callers should ensure they're also
// calling page integrity checks if neccessary
func (s *Screen) AssertIntegrity() {
	// TODO: add feature flag to disable this
	utils.Assert(s.Cursor != nil)
	utils.Assert(s.Cursor.X < s.cols && s.Cursor.Y < s.rows)
}

// Move the cursor to the right by n cells. This is specialized function
// that is very fast if the caller can guarantee that we have space to move
// right (no wrapping)
//
// NOTE this is no wrapping move
func (s *Screen) SetCursorRight(n size.CellCountInt) {
	utils.Assert(s.Cursor.X+n < s.Pages.Cols)
	defer s.AssertIntegrity()

	s.Cursor.PageCell = s.Cursor.PageRow.Cells[s.Cursor.X+n]
	s.Cursor.PagePin.X += n
	s.Cursor.X += n
}

// Move the cursor to the left by n cells. This is specialized function
// that is very fast if the caller can guarantee that we have space to move
// left (no wrapping)
//
// NOTE this is no wrapping move
func (s *Screen) SetCursorLeft(n size.CellCountInt) {
	utils.Assert(s.Cursor.X >= n)
	defer s.AssertIntegrity()

	s.Cursor.PageCell = s.Cursor.PageRow.Cells[s.Cursor.X-n]
	s.Cursor.PagePin.X -= n
	s.Cursor.X -= n
}

// Move the cursor up
//
// Precondition: The cursor is not at the top of the screen
func (s *Screen) SetCursorUp(n size.CellCountInt) {
	utils.Assert(s.Cursor.Y >= n)
	defer s.AssertIntegrity()

	s.Cursor.Y -= n

	pagePin := s.Cursor.PagePin.Up(n)
	s.CursorChangePin(pagePin)
	pageRAC := s.Cursor.PagePin.RowAndCell()
	s.Cursor.PageRow = pageRAC.Row
	s.Cursor.PageCell = pageRAC.Cell
}

// Move the cursor down
//
// Precondition: The cursor is not at the bottom of the screen
func (s *Screen) SetCursorDown(n size.CellCountInt) {
	utils.Assert(s.Cursor.Y+n < s.rows)
	defer s.AssertIntegrity()

	s.Cursor.Y += n // must be set before CursorChangePin

	// We move the offset into our page list to the next row and then
	// get the pointers to the row/cell and set all the cursor state up.
	pagePin := s.Cursor.PagePin.Down(n)
	s.CursorChangePin(pagePin)
	pageRAC := s.Cursor.PagePin.RowAndCell()
	s.Cursor.PageRow = pageRAC.Row
	s.Cursor.PageCell = pageRAC.Cell
}

func (s *Screen) SetCursorAbs(x size.CellCountInt, y size.CellCountInt) {
	utils.Assert(x < s.cols && y < s.rows)
	defer s.AssertIntegrity()
	var pagePin *pagelist.Pin
	if y < s.Cursor.Y {
		// Move up.
		pagePin = s.Cursor.PagePin.Up(s.Cursor.Y - y)
	} else if y > s.Cursor.Y {
		// Move down.
		pagePin = s.Cursor.PagePin.Down(y - s.Cursor.Y)
	} else {
		//  keep the same row
		pagePin = s.Cursor.PagePin
	}
	pagePin.X = x
	s.Cursor.X = x // Must be set before CursorChangePin
	s.Cursor.Y = y
	s.CursorChangePin(pagePin)
	pageRAC := s.Cursor.PagePin.RowAndCell()
	s.Cursor.PageRow = pageRAC.Row
	s.Cursor.PageCell = pageRAC.Cell
}

// Move the cursor to some absolute horizontal position
func (s *Screen) SetCursorHorizontalAbs(x size.CellCountInt) {
	utils.Assert(x < s.cols)
	defer s.AssertIntegrity()

	s.Cursor.PagePin.X = x
	pageRAC := s.Cursor.PagePin.RowAndCell()
	s.Cursor.PageCell = pageRAC.Cell
	s.Cursor.X = x
}

func (s *Screen) SetCursorVerticalAbs(y size.CellCountInt) {
	utils.Assert(y < s.rows)
	defer s.AssertIntegrity()

	// since we have to move the pin to different rows, so using SetCursorAbs
	// is fine here. We only want to optimize the case if we are on the same
	// row.
	s.SetCursorAbs(s.Cursor.X, y)
}

// This scrolls the active area at and above the cursor.
// The lines below the cursor are not scrolled.
func (s *Screen) SetCursorScrollUp() {
	// We unconditionally mark the cursor as dirty here because
	// the cursor always changes page rows inside this function, and when
	// that happens, it can mean the text in the old row need to be re-shaped
	// because the cursor splits runs to break ligatures.
	s.Cursor.PagePin.MarkDirty()

	// If the cursor is on the bottom of the screen, its faster to use our
	// specialized function for that case.
	if s.Cursor.Y == s.Pages.Rows-1 {
		s.SetCursorDownScroll()
		return
	}

	defer s.AssertIntegrity()

	// Logic below assumes we always have at least one row that isn't moving.
	utils.Assert(s.Cursor.Y < s.Pages.Rows-1)

	// Explanation:
	//  We don't actually move every that's at or above the cursor
	//  since this would require us to shift up our ENTIRE scrollback, which
	//  would be ridiculously expensive. Instead, we insert a new row at the
	//  end of the pagelist (use grow()) and move everything BELOW the cursor
	//  DOWN by one row. This has the same practical results but is' a whole
	//  lot cheaper in 99% of cases. As number of rows below the cursor are
	//  > 90% case less than the number of rows above the cursor.
	oldPin := s.Cursor.PagePin
	if s.Pages.Grow() != nil {
		s.SetCursorScrollAboveRotate()
	} else {
		// In this case, it means grow() didn't allocate a new page.

		if s.Cursor.PagePin.Node == s.Pages.Pages.Last {
			// If we're on the last page, we can do a very fast path because
			// all the rows we need to move around are within a single page.

			// We don't need to change cursor pin here, because the pin page
			// is the same so there is no accounting to do for styles or any of
			// that.
			utils.Assert(oldPin.Node == s.Cursor.PagePin.Node)
			s.Cursor.PagePin = s.Cursor.PagePin.Down(1)

			pin := s.Cursor.PagePin
			page := s.Cursor.PagePin.Node.Data

			// Rotate the rows so that newly created empty row is at the
			// beginning.
			// [ 0 1 2 3 ] => [ 3 0 1 2 ]
			utils.RotateOnceR(page.Rows[pin.Y:page.Size.Rows])

			// Mark all our rotated row as dirty.
			dirty := page.DirtyBitSet()
			dirty.SetRange(int(pin.Y), int(page.Size.Rows))

			// Setup our cursor caches after the rotation so it points to
			// the correct data.
			pageRAC := s.Cursor.PagePin.RowAndCell()
			s.Cursor.PageRow = pageRAC.Row
			s.Cursor.PageCell = pageRAC.Cell
		} else {
			// We didn't grow pages but our cursor isn't on the last page.
			// In this case we need to do more work because we need to copy
			// elements between pages.
			//
			// An example scerario of this is:
			//
			//     : +------------+ : = PAGE 0
			// ... : :            : :
			//     +----------------+ ACTIVE AREA
			// 151 | |1A0000000000| | 0
			// 152 | |2B0000000000| | 1
			//     : :^           : : = CURSOR PIN
			// 153 | |3C0000000000| | 2
			//     : +------------+ :
			//     : +------------+ : = PAGE 1
			//   0 | |4D0000000000| | 3
			//   1 | |5E0000000000| | 4
			//     : +------------+ :
			//     +----------------+
			s.SetCursorScrollAboveRotate()
		}
	}

	if s.Cursor.StyleID != styleid.DefaultID {
		// The newly created line needs to be styled according to the
		// the bg color if it is set.
		if cell := s.Cursor.Style.BGCell(); cell != nil {
			cells := s.Cursor.PageRow.Cells
			// Replace every cells from the begin to the cursor with bgcell
			for i := range s.Cursor.X {
				cells[i] = cell
			}
		}
	}
}

// Scroll the screen through pages below the the cursor pin.
//
// This is specialized for the case wehere we have a cursor pin that is not in
// the last pages, and we need to shift all the rows down by one.
//
// See SetCursorScrollUp for more detail on its usage.
func (s *Screen) SetCursorScrollAboveRotate() {
	s.CursorChangePin(s.Cursor.PagePin.Down(1))

	// Go through each of the pages folllowing our pin, shiftall rows down
	// by one, and copy the last row of the previous page.
	curr := s.Pages.Pages.Last

	for ; curr != nil && curr != s.Cursor.PagePin.Node; curr = curr.Prev {
		prev := curr.Prev
		prevPage := prev.Data
		currPage := curr.Data
		prevRows := prevPage.Rows
		currRows := currPage.Rows

		// Rotatethe pages down: [ 0 1 2 3 ] => [ 3 0 1 2 ]
		utils.RotateOnceR(currRows[0:currPage.Size.Rows])

		// Copy the last row of the previous page to the top of current page.
		currPage.CloneRowFrom(prevPage, currRows[0], prevRows[prevPage.Size.Rows-1])

		// All rows we rotated are dirty.
		dirty := currPage.DirtyBitSet()
		dirty.SetRange(0, int(currPage.Size.Rows))
	}

	// Our current is our cursor page, we need to rotate down from the cursor
	// to the end of the page.
	utils.Assert(curr == s.Cursor.PagePin.Node)
	currPage := curr.Data
	currRows := currPage.Rows

	utils.RotateOnceR(currRows[s.Cursor.PagePin.Y:currPage.Size.Rows])
	s.ClearCells(currPage, currRows[s.Cursor.PagePin.Y], 0, currPage.Size.Cols)

	// Set all the rows we rotated as dirty.
	dirty := currPage.DirtyBitSet()
	dirty.SetRange(int(s.Cursor.PagePin.Y), int(currPage.Size.Rows))

	// Reset the cursor cache data.
	pageRAC := s.Cursor.PagePin.RowAndCell()
	s.Cursor.PageRow = pageRAC.Row
	s.Cursor.PageCell = pageRAC.Cell
}

// Scroll the active area and keep the cursor at the bottom of the screen.
// This is a very specialized function but it keeps it fast.
func (s *Screen) SetCursorDownScroll() {
	utils.Assert(s.Cursor.Y == s.Pages.Rows-1)
	defer s.AssertIntegrity()

	// If we have no scrollback, then we shift all our rows instead.
	if s.NoScrollback {
		// If we have a single-row screen, we have no rows to shift so
		// our cursor is in the correct place, we just have to clear the cells
		if s.Pages.Rows == 1 {
			page := s.Cursor.PagePin.Node.Data
			s.ClearCells(page, s.Cursor.PageRow, 0, s.Pages.Cols)

			dirty := page.DirtyBitSet()
			dirty.Set(0)
		} else {
			// EraseRow will shift everything below it up.
			s.Pages.EraseRow(point.Point{Tag: point.TagActive})

			// NOTE, we don't need to mark anything dirty in this branch, as
			// EraseRow already does that for us.

			// Update the cursor cache.
			pagePin := s.Cursor.PagePin.Down(1)
			s.CursorChangePin(pagePin)
			pageRAC := s.Cursor.PagePin.RowAndCell()
			s.Cursor.PageRow = pageRAC.Row
			s.Cursor.PageCell = pageRAC.Cell

			// The above may clear our cursor so we need to update that again.
			s.ManualStyleUpdate()
		}
	} else {
		pin := s.Cursor.PagePin

		// Grow our pages by one row. The PageList will handle if we need
		// to allocate, prune scrollback, etc.
		s.Pages.Grow()

		// If the page pin doesn't change, it means we are still on the same
		// page with before, so we can just move the pin down.
		if pin.Node == s.Cursor.PagePin.Node {
			pin = pin.Down(1)
		} else {
			// If our page pin change, it means the page the pin was on was
			// pruned. In this case, grow() moves the pin to the top-left of
			// the new page. This effectively moves it by one already, we have
			// to fix the x value.
			pin = s.Cursor.PagePin
			pin.X = s.Cursor.X
		}

		s.CursorChangePin(pin)
		pageRAC := pin.RowAndCell()
		s.Cursor.PageRow = pageRAC.Row
		s.Cursor.PageCell = pageRAC.Cell

		// Our new row is always dirty.
		s.CursorMarkDirty()

		// lear the new row so it gets our bg color. We only do this if we
		// have a bg color at all.
		if s.Cursor.Style.BackgroundColor.Type != style.ColorTypeNone {
			page := pin.Node.Data
			s.ClearCells(page, s.Cursor.PageRow, 0, page.Size.Cols)
		}
	}
	if s.Cursor.StyleID != styleid.DefaultID {
		// The newly created line need to be styled according to the bg color
		// if it is set
		if cell := s.Cursor.Style.BGCell(); cell != nil {
			cells := s.Cursor.PageRow.Cells

			// Replace every cells from the begin to the cursor with bgcell
			for i := range s.Pages.Cols {
				cells[i] = cell
			}
		}
	}
}

// Clean up boundary conditions where a cell will become discontiguous with
// a neighboring cell because either one of them will be moved and/or cleard.
//
// For performance reasons this is specialized to operate on the cursor row.
//
// So, for example, if the cursor is at [a, b] (inclusive), call this function
// with `x=a` and `x=b+1`. It is okay if `x` is out of bounds by 1, this
// will be interpreted as correctly.
//
// DOES NOT MODIFY ROW WRAP STATE! See `CursorResetWrap` for that.
//
// The following boundary conditions are handled:
// - `x-1` is a wide character and `x` is a spacer tail:
//   - Both cells will be cleared
func (s *Screen) SplitCellBoundary(x size.CellCountInt) {
	page := s.Cursor.PagePin.Node.Data
	page.PauseIntegrityChecks(true)
	defer page.PauseIntegrityChecks(false)

	// `x` maybe up to an INCLUDING `cols`, since that signifiles spliting
	// the boundary to the right of the final cell in the rows`
	utils.Assert(x <= s.cols)

	// [ A B C D F F|]
	//              ^ Boundary between final cell and row end.
	if x == s.cols {
		if !s.Cursor.PageRow.Wrap {
			return
		}
		// Ignore spacer_head for now
	}
	// If x is 0 then we're done.
	if x == 0 {
		return
	}

	// [ ... X|Y ... ]
	//        ^ Boundary between two cells in the middle of the row.
	{
		utils.Assert(x > 0 && x < s.cols)
		cells := s.Cursor.PageRow.Cells

		left := cells[x-1]
		switch left.Wide {
		// A wide char would be split, so must be cleared
		case pagepkg.WideWide:
			s.ClearCells(
				page,
				s.Cursor.PageRow,
				x-1, x+1)
		}
	}
}

// Clear the cells with the blank cell. This takes care to handle cleaning
// up styles.
func (s *Screen) ClearCells(
	page *pagepkg.Page,
	row *pagepkg.Row,
	fromX, toX size.CellCountInt,
) {
	// This whole operation deos unsafe things, so we just want to assert the
	// end state.
	page.PauseIntegrityChecks(true)
	defer func() {
		page.PauseIntegrityChecks(false)
		page.AssertIntegrity()
		s.AssertIntegrity()
	}()
	if row.Styled {
		for i := fromX; i < toX; i++ {
			cell := row.Cells[i]
			if cell.StyleID == styleid.DefaultID {
				continue
			}
			page.Styles.Release(set.ID(cell.StyleID))
		}

		// If we have no left/right scroll region we can be sure that the row
		// is no longer styled.
		if toX-fromX+1 == s.Pages.Cols {
			row.Styled = false
		}
	}
	for i := fromX; i < toX; i++ {
		row.Cells[i] = s.blankCell()
	}
}

// Return the blank cell to use when doing terminal operations that require
// preserving the bg color.
func (s *Screen) blankCell() *pagepkg.Cell {
	if s.Cursor.StyleID == styleid.DefaultID {
		// If we have no style, then we can just return a blank cell
		return &pagepkg.Cell{}
	}
	return s.Cursor.Style.BGCell()
}

// Reset the screen according to the logic of DEC RIS sequence.
//
// - Clear the screen and attempt to reclaim memory
// - Moves the cursor to the top left corner
func (s *Screen) Reset() {
	// Reset our pages.
	s.Pages.Reset()

	// The above reset preserves tracked pins so we can still use our cursor
	// pin, which should be at the top-left already.
	cursorPin := s.Cursor.PagePin
	utils.Assert(cursorPin.Node == s.Pages.Pages.First)
	utils.Assert(cursorPin.X == 0 && cursorPin.Y == 0)
	cursorRAC := cursorPin.RowAndCell()
	s.Cursor = &Cursor{
		PageCell: cursorRAC.Cell,
		PageRow:  cursorRAC.Row,
		PagePin:  cursorPin,
	}
}

// Dump the screen to a string. The writer given should be buffered;
// this function does not attempt to efficiently write and generally writes
// one byte at a time.
func (s *Screen) dumpString(
	w io.Writer,
	opts pagelist.EncodeUtf8Options,
) error {
	_, err := s.Pages.EncodeUtf8(w, opts)
	if err != nil {
		return err
	}
	return nil
}

// Dump the screen to a string. The writer given should be buffered;
// this function does not attempt to efficiently write and generally writes
// one byte at a time.
func (s *Screen) DumpString(
	w io.Writer,
	tl point.Tag,
) error {
	tlPin := s.Pages.GetTopLeft(tl)
	brPin := s.Pages.GetBottomRight(tl)
	if tlPin == nil {
		return fmt.Errorf("invalid top-left point %v for tag %v", tl, tl)
	}
	return s.dumpString(w,
		pagelist.EncodeUtf8Options{
			Unwrap:      false,
			TopLeft:     *tlPin,
			BottomRight: brPin,
		},
	)
}

// Set a style attribute for the current cursor.
func (s *Screen) SetGraphicsRendition(attr *sgr.Attribute) {
	switch attr.Type {
	case sgr.AttributeTypeUnset:
		s.Cursor.Style.Reset()

	case sgr.AttributeTypeBold:
		s.Cursor.Style.Bold = true

	case sgr.AttributeTypeResetBold:
		// Bold and faint share the same SGR code for this
		s.Cursor.Style.Bold = false
		s.Cursor.Style.Faint = false

	case sgr.AttributeTypeItalic:
		s.Cursor.Style.Italic = true

	case sgr.AttributeTypeResetItalic:
		s.Cursor.Style.Italic = false

	case sgr.AttributeTypeFaint:
		s.Cursor.Style.Faint = true

	case sgr.AttributeTypeUnderline:
		s.Cursor.Style.Underline = attr.Underline

	case sgr.AttributeTypeResetUnderline:
		s.Cursor.Style.Underline = sgr.UnderlineTypeNone

	case sgr.AttributeTypeUnderlineColor:
		s.Cursor.Style.UnderlineColor = style.Color{
			Type: style.ColorTypeRGB,
			RGB: color.RGB{
				R: attr.UnderlineColor.R,
				G: attr.UnderlineColor.G,
				B: attr.UnderlineColor.B,
			},
		}
	case sgr.AttributeTypeResetUnderlineColor:
		s.Cursor.Style.UnderlineColor = style.Color{
			Type: style.ColorTypeNone,
		}
	case sgr.AttributeTypeOverline:
		s.Cursor.Style.Overline = true

	case sgr.AttributeTypeResetOverline:
		s.Cursor.Style.Overline = false

	case sgr.AttributeTypeBlink:
		s.Cursor.Style.Blink = true

	case sgr.AttributeTypeResetBlink:
		s.Cursor.Style.Blink = false

	case sgr.AttributeTypeInverse:
		s.Cursor.Style.Inverse = true

	case sgr.AttributeTypeResetInverse:
		s.Cursor.Style.Inverse = false

	case sgr.AttributeTypeInvisible:
		s.Cursor.Style.Invisible = true

	case sgr.AttributeTypeResetInvisible:
		s.Cursor.Style.Invisible = false

	case sgr.AttributeTypeStrikethrough:
		s.Cursor.Style.Strikethrough = true

	case sgr.AttributeTypeResetStrikethrough:
		s.Cursor.Style.Strikethrough = false

	case sgr.AttributeTypeDirectColorFg:
		s.Cursor.Style.ForegroundColor = style.Color{
			Type: style.ColorTypeRGB,
			RGB: color.RGB{
				R: attr.DirectColorFg.R,
				G: attr.DirectColorFg.G,
				B: attr.DirectColorFg.B,
			},
		}

	case sgr.AttributeTypeResetFg:
		s.Cursor.Style.ForegroundColor = style.Color{
			Type: style.ColorTypeNone,
		}

	case sgr.AttributeTypeDirectColorBg:
		s.Cursor.Style.BackgroundColor = style.Color{
			Type: style.ColorTypeRGB,
			RGB: color.RGB{
				R: attr.DirectColorBg.R,
				G: attr.DirectColorBg.G,
				B: attr.DirectColorBg.B,
			},
		}

	case sgr.AttributeTypeResetBg:
		s.Cursor.Style.BackgroundColor = style.Color{
			Type: style.ColorTypeNone,
		}

	// We don't handle unknown attributes in the screen, so we just ignore
	// them
	case sgr.AttributeTypeUnknown:

	default:
		utils.Assert(false, fmt.Sprintf("unknown sgr attribute type %v", attr.Type))
	}
	s.ManualStyleUpdate()
}

// Set a style attribute for the current cursor.
//
// This can cause a page split if the current page cannot fit this style.
// This is only scenario an error return is possible.
func (s *Screen) SetAttribute(attr sgr.Attribute) {
	switch attr.Type {
	case sgr.AttributeTypeUnset:
		s.Cursor.Style = style.Style{}

	case sgr.AttributeTypeBold:
		s.Cursor.Style.Bold = true

	case sgr.AttributeTypeResetBold:
		// Bold and faint share the same SGR code for this.
		s.Cursor.Style.Bold = false
		s.Cursor.Style.Faint = false

	case sgr.AttributeTypeItalic:
		s.Cursor.Style.Italic = true

	case sgr.AttributeTypeResetItalic:
		s.Cursor.Style.Italic = false

	case sgr.AttributeTypeFaint:
		s.Cursor.Style.Faint = true

	case sgr.AttributeTypeUnderline:
		s.Cursor.Style.Underline = attr.Underline

	case sgr.AttributeTypeResetUnderline:
		s.Cursor.Style.Underline = sgr.UnderlineTypeNone

	case sgr.AttributeTypeUnderlineColor:
		rgb := attr.UnderlineColor
		s.Cursor.Style.UnderlineColor = style.Color{
			Type: style.ColorTypeRGB,
			RGB:  color.RGB{R: rgb.R, G: rgb.G, B: rgb.B},
		}

	case sgr.AttributeTypeResetUnderlineColor:
		s.Cursor.Style.UnderlineColor = style.Color{
			Type: style.ColorTypeNone,
		}

	case sgr.AttributeTypeOverline:
		s.Cursor.Style.Overline = true

	case sgr.AttributeTypeResetOverline:
		s.Cursor.Style.Overline = false

	case sgr.AttributeTypeBlink:
		s.Cursor.Style.Blink = true

	case sgr.AttributeTypeResetBlink:
		s.Cursor.Style.Blink = false

	case sgr.AttributeTypeInverse:
		s.Cursor.Style.Inverse = true

	case sgr.AttributeTypeResetInverse:
		s.Cursor.Style.Inverse = false

	case sgr.AttributeTypeInvisible:
		s.Cursor.Style.Invisible = true

	case sgr.AttributeTypeResetInvisible:
		s.Cursor.Style.Invisible = false

	case sgr.AttributeTypeStrikethrough:
		s.Cursor.Style.Strikethrough = true

	case sgr.AttributeTypeResetStrikethrough:
		s.Cursor.Style.Strikethrough = false

	case sgr.AttributeTypeDirectColorFg:
		rgb := attr.DirectColorFg
		s.Cursor.Style.ForegroundColor = style.Color{
			Type: style.ColorTypeRGB,
			RGB:  color.RGB{R: rgb.R, G: rgb.G, B: rgb.B},
		}

	case sgr.AttributeTypeDirectColorBg:
		rgb := attr.DirectColorBg
		s.Cursor.Style.BackgroundColor = style.Color{
			Type: style.ColorTypeRGB,
			RGB:  color.RGB{R: rgb.R, G: rgb.G, B: rgb.B},
		}

	case sgr.AttributeTypeResetFg:
		s.Cursor.Style.ForegroundColor = style.Color{
			Type: style.ColorTypeNone,
		}

	case sgr.AttributeTypeResetBg:
		s.Cursor.Style.BackgroundColor = style.Color{
			Type: style.ColorTypeNone,
		}

	case sgr.AttributeTypeUnknown:
		return
	}
	s.ManualStyleUpdate()
}

// Call this whenever we manually change the cursor style.
func (s *Screen) ManualStyleUpdate() {
	page := s.Cursor.PagePin.Node.Data
	// Release our previous style if it was not default.
	if s.Cursor.StyleID != styleid.DefaultID {
		page.Styles.Release(set.ID(s.Cursor.StyleID))
	}

	// If our new style is the default, just reset to that
	if s.Cursor.Style.IsDefault() {
		s.Cursor.StyleID = styleid.DefaultID
		return
	}

	// Clear the cursor style ID to prevent weird things from happening
	// if the page capacity has to be adjusted which would end up calling
	// manualStyleUpdate again.
	s.Cursor.StyleID = styleid.DefaultID

	// After setting the style, we need to update our style map.
	// Note that we COULD lazily do this in print. We should look into
	// if that makes a meaningful difference. Our priority is to keep print
	// fast because setting a ton of styles that do nothing is uncommon
	// and weird.
	id := page.Styles.Add(s.Cursor.Style)
	s.Cursor.StyleID = styleid.ID(id)
	s.AssertIntegrity()
}

// GetCursor returns the current cursor position
func (s *Screen) GetCursor() *Cursor {
	utils.Assert(s.Cursor != nil)
	return s.Cursor
}

// GetSize returns the current size of the display in rows and columns
func (s *Screen) GetSize() (rows, cols size.CellCountInt) {
	return s.rows, s.cols
}

func (s *Screen) CursorMarkDirty() {
	s.Cursor.PagePin.MarkDirty()
}

func (s *Screen) ResizeWithReflow(cols, rows size.CellCountInt) {
	// defer d.AssertIntegrity()
	//
	// // On reflow, the main thing that cause refloew is column changes. If only
	// // rows change, refleow is impossible. So we change our behavior based
	// // on the change of collumns
	// switch cmp.Compare(cols, d.cols) {
	//
	// // Equal
	// case 0:
	// 	d.ResizeWithoutReflow(cols, rows)
	//
	// // cols > d.cols
	// case 1:
	// 	// We grow rows after cols so that we can do our unwrapping/reflow
	// 	// before we do a no-reflow grow.
	// 	d.resizeCols(cols, d.Cursor)
	// 	d.ResizeWithoutReflow(cols, rows)
	//
	// // cols < d.cols
	// case -1:
	// 	// We first change our row count so that we have the proper amount
	// 	// we can use when shriking our cols
	// 	d.ResizeWithoutReflow(d.cols, rows)
	// 	d.resizeCols(cols, d.Cursor)
	// }
}

func (s *Screen) ResizeWithoutReflow(cols, rows size.CellCountInt) {
	// d.resizeInternal(cols, rows, false)
}

// Clear the region specified by tl and bl, inclusive. Cleared cells are colored with
// the current style background color. This will clear all cells in the rows.
// bl is optional and if not provided, the region will be cleared to the end.
func (s *Screen) ClearRows(tl point.Point, bl *point.Point) {
}

// This is basically a really jank version of Terminal.printString. We
// have to reimplement it here because we want a way to print to the screen
// to test it but don't want all the features of Terminal.
func (s *Screen) testWriteString(text []uint8) error {
	dec := unicode.UTF8.NewDecoder()
	decoded, err := dec.Bytes(text)
	if err != nil {
		return err
	}
	for _, c := range decoded {
		// Explicit newline forces a new row
		if c == '\n' {
			s.SetCursorDownOrScroll()
			s.SetCursorHorizontalAbs(0)
			s.Cursor.PageRow.Wrap = false
			continue
		}
		width := dw.RuneWidth(rune(c))
		if width == 0 {
			// do not support grapheme clusters
			continue
		}
		if s.Cursor.PendingWrap {
			utils.Assert(s.Cursor.X == s.cols-1)
			s.Cursor.PendingWrap = false
			s.Cursor.PageRow.Wrap = true
			s.SetCursorDownOrScroll()
			s.SetCursorHorizontalAbs(0)
			s.Cursor.PageRow.WrapContinuation = true
		}
		utils.Assert(width == 1 || width == 2)
		switch width {
		case 1:
			s.Cursor.PageCell.ContentTag = pagepkg.ContentTagCP
			s.Cursor.PageCell.ContentCP = uint32(c)
			s.Cursor.PageCell.StyleID = s.Cursor.StyleID
			// s.Cursor.PageCell.Protected = s.Cursor.Protected

			// if we have a ref-counted style, increase.
			if s.Cursor.StyleID != styleid.DefaultID {
				page := s.Cursor.PagePin.Node.Data
				page.Styles.Use(set.ID(s.Cursor.StyleID))
				s.Cursor.PageRow.Styled = true
			}
		case 2:
			// Need a wide spacer head
			if s.Cursor.X == s.cols-1 {
				s.Cursor.PageCell.ContentTag = pagepkg.ContentTagCP
				s.Cursor.PageCell.ContentCP = 0 // wide spacer head
				s.Cursor.PageCell.Wide = pagepkg.WideSpacerHead

				s.Cursor.PageRow.Wrap = true
				s.SetCursorDown(1)
				s.SetCursorHorizontalAbs(0)
				s.Cursor.PageRow.WrapContinuation = true
			}

			// Write our wide char
			s.Cursor.PageCell.ContentTag = pagepkg.ContentTagCP
			s.Cursor.PageCell.ContentCP = uint32(c)
			s.Cursor.PageCell.StyleID = s.Cursor.StyleID
			s.Cursor.PageCell.Wide = pagepkg.WideWide

			// Write our tail
			s.SetCursorRight(1)
			s.Cursor.PageCell.ContentTag = pagepkg.ContentTagCP
			s.Cursor.PageCell.ContentCP = 0 // wide spacer tail
			s.Cursor.PageCell.Wide = pagepkg.WideSpacerTail

			// If we have a ref-counted style, increase twice.
			if s.Cursor.StyleID != styleid.DefaultID {
				page := s.Cursor.PagePin.Node.Data
				page.Styles.Use(set.ID(s.Cursor.StyleID))
				page.Styles.Use(set.ID(s.Cursor.StyleID))
				s.Cursor.PageRow.Styled = true
			}
		}

		// if we don't stand at the end of the row, we can move right
		if s.Cursor.X+1 < s.cols {
			s.SetCursorRight(1)
		} else {
			s.Cursor.PendingWrap = true
		}
	}
	return nil
}

// Move the cursor down if we're not at the bottom of scren. Otherwise, do
// a scroll. Currently only used for testing.
func (s *Screen) SetCursorDownOrScroll() {
	if s.Cursor.Y < s.rows-1 {
		s.SetCursorDown(1)
	} else {
		s.SetCursorDownScroll()
	}
}

// Always use this to write to cusor.PagePin.*
//
// This specifically handles the case when the new pin is on different page
// than the old AND we have a style set. In that case, we must release
// our old one and insert the new one, since styles are per-page specific.
func (s *Screen) CursorChangePin(newPin *pagelist.Pin) {
	// Moving the cursor affects text run splitting (ligatures) so we must
	// mark the old and new page dirty. We do this as long as the pins are
	// not equal
	if !s.Cursor.PagePin.Equal(newPin) {
		s.Cursor.PagePin.MarkDirty()
		newPin.MarkDirty()
	}

	// If our pin is on the same page, then we can just update the pin.
	// We don't need to migrate any state.
	if s.Cursor.PagePin.Node == newPin.Node {
		s.Cursor.PagePin = newPin
		return
	}
	var oldStyle *style.Style = nil
	if s.Cursor.StyleID != styleid.DefaultID {
		oldStyle = &s.Cursor.Style
	}

	if oldStyle != nil {
		// Update the style
		s.Cursor.Style = style.Style{}
		s.ManualStyleUpdate()
	}

	s.Cursor.PagePin = newPin

	if oldStyle != nil {
		s.Cursor.Style = *oldStyle
		s.ManualStyleUpdate()
	}
}

// CursorCellRight implements Screen.
func (s *Screen) CursorCellRight(n size.CellCountInt) *pagepkg.Cell {
	utils.Assert(s.Cursor.X+n < s.cols)
	return s.Cursor.PageRow.Cells[s.Cursor.X+n]
}

func (s *Screen) CursorCellLeft(n size.CellCountInt) *pagepkg.Cell {
	utils.Assert(s.Cursor.X >= n)
	return s.Cursor.PageRow.Cells[s.Cursor.X-n]
}

// CursorCellEndOfPrevious returns the cell at the end of the previous line.
// If the previous line is not available, it returns nil.
func (s *Screen) CursorCellEndOfPrevious() *pagepkg.Cell {
	utils.Assert(s.Cursor.X > 0)
	pagePin := s.Cursor.PagePin.Up(1)
	if pagePin == nil {
		return nil
	}
	pagePin.X = s.Pages.Cols - 1
	pageRAC := pagePin.RowAndCell()
	return pageRAC.Cell
}
