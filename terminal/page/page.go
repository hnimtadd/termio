package page

import (
	"fmt"

	"github.com/hnimtadd/termio/io"
	"github.com/hnimtadd/termio/terminal/set"
	"github.com/hnimtadd/termio/terminal/size"
	styleid "github.com/hnimtadd/termio/terminal/style/id"
	"github.com/hnimtadd/termio/terminal/utils"
	internalutils "github.com/hnimtadd/termio/utils"
)

var ErrOutOfMemory = fmt.Errorf("page: out of memory")

// A page represents a specific section of terminal screen. The primary
// idea of a page is that it is a fully self-contained unit that can be
// serialized, copied, etc. as a convenient way to represent a section
// of the screen.
//
// This property is useful for renderers which want to copy just the pages
// for the visible portion of the screen, or for infinite scrollback where
// we may want to serialize and store pages that are sufficiently far
// away from the current viewport.
//
// Pages are always backed by a single contiguous block of memory that is
// aligned on a page boundary. This makes it easy and fast to copy pages
// around. Within the contiguous block of memory, the contents of a page are
// thoughtfully laid out to optimize primarily for terminal IO (VT streams)
// and to minimize memory usage.
type Page struct {
	// The backing Memory for the page. A page is alwasy made up of a single
	// contiguous block of Memory that is aligned on a page boundary, and is
	// always a multiple of the page size.
	Memory []uint8

	// The array of Rows in the page. The Rows are always in row order (i.e.
	// index 0 is the top row, index 1 is the second row, etc.)
	Rows []*Row

	// The arrays of Cells in the page. The Cells are NOT in row order, but
	// they are in column order. To determine the mapping of cellls, to row,
	// we hae to use the `rows` field. From the pointer to the first column,
	// all Cells in that row are laid out in column order.
	Cells []*Cell

	// The availabes set of styles in use on this page.
	Styles *set.RefCountedSet

	// Dirty bits in the page.
	// Each bit represents a row in the page, and if the bit is set,
	// then the row is dirty and requires a redraw. Dirty status is only ever
	// meant to convey that a cell has changed visually. A cell changes in a
	// way that doesn't affect the visual representation may not be marked as
	// dirty.
	//
	// Dirty tracking may have false positibes, but should never have false
	// negatives.
	// A false negative would result in a visual artifact on the screen.
	Dirty *utils.StaticBitSet

	Size     Size
	Capacity Capacity

	// If this is true then verify integrity will do nothing..
	pauseIntegrityCheck int
}

func InitPage(cap Capacity) *Page {
	// init the cells.
	cells := make([]*Cell, cap.Cols*cap.Rows)
	for i := range cells {
		cells[i] = &Cell{}
	}

	rows := make([]*Row, cap.Rows)
	// we need to go through and initialize all the rows so that they point
	// to a valid cells, since the rows zero-initialized aren't valid.
	for i := range rows {
		rows[i] = &Row{
			Cells: cells[i*int(cap.Cols) : (i+1)*int(cap.Cols)],
		}
	}

	return &Page{
		Memory: make([]uint8, cap.Cols*cap.Rows),
		Rows:   rows,
		Cells:  cells,
		Styles: set.NewRefCountedSet(set.Options{
			Cap: internalutils.PointerTo(uint64(cap.Styles)),
		}),
		Size:     Size{Cols: cap.Cols, Rows: cap.Rows},
		Capacity: cap,
		Dirty:    utils.NewStaticBitSet(int(cap.Rows)),
	}
}

func (p *Page) MoveCells(
	srcRow *Row,
	left size.CellCountInt,
	dstRow *Row,
	param4 size.CellCountInt,
	int size.CellCountInt,
) {
	panic("unimplemented")
}

// A helper that can be used to assert the integrity of the page. This is no-op
// if we disable debugging.
func (p *Page) AssertIntegrity() {
	utils.Assert(p.Size.Rows != 0, "page integrity violation zero row count")
	utils.Assert(p.Size.Cols != 0, "page integrity violation zero col count")
}

func (p *Page) ClonePartialRowFrom(
	data *Page,
	dstRow *Row,
	srcRow *Row,
	left size.CellCountInt,
	int size.CellCountInt,
) error {
	panic("unimplemented")
}

func (p *Page) SwapCells(src *Cell, dst *Cell) {
	panic("unimplemented")
}

// Temporarily pause integrity checks. This is useful when we are
// doing a lot of operations that would trigger integrity check
// violations but we know the page will end up in a consistent state.
func (p *Page) PauseIntegrityChecks(v bool) {
	if v {
		p.pauseIntegrityCheck += 1
	} else {
		p.pauseIntegrityCheck += 1
	}
	// panic("unimplemented")
}

// The size of this page
type Size struct {
	Cols size.CellCountInt
	Rows size.CellCountInt
}

// Capacity of this page.
type Capacity struct {
	// Number of Cols and rows we can know about:
	Cols size.CellCountInt
	Rows size.CellCountInt

	// Number of unique Styles that can be used on this page.
	Styles uint
}

func (c Capacity) Size() uint64 {
	panic("unimplemented")
}

type Adjustment struct {
	Cols size.CellCountInt
}

// Adjust the capacity parameters while retaining the same total size.
// Adjustments alwasy happen by limiting the row in pages. Everying else
// can grow. If it is impossible to achieve the desired capacity, OutOfMemory
// is returned.
func (c *Capacity) Adjust(req Adjustment) error {
	if req.Cols > 0 && req.Cols != c.Cols {
		totalCells := c.Cols * c.Rows
		new_rows := int(totalCells / req.Cols)
		// If our rows to to zero then we can't fit any row metadata for the
		// desired number of columns.
		if new_rows == 0 {
			return ErrOutOfMemory
		}
		c.Rows = size.CellCountInt(new_rows)
		c.Cols = req.Cols
	}
	return nil
}

// The standard capacity for a page that doesn't have special
// requirements. This is enough to support a very large number of cells.
// The standard capacity is chosen as the fast-path for allocation since
// pages of standard capacity use a pooled allocator instead of single-use
// mmaps.
var StandardCapacity = Capacity{
	Cols:   215,
	Rows:   215,
	Styles: 128,
}

type EncodeUtf8Options struct {
	// The range of rows to encode. If EndY is null, then it will
	// encode to the end of the page.
	StartY size.CellCountInt
	EndY   *size.CellCountInt
	Unwrap bool

	// Preceding state from encoding the previous page.
	// Use to preserve blanks properly across multiple pages.
	Preceding TrailingUtf8State
}

type TrailingUtf8State struct {
	Rows  uint
	Cells uint
}

// Encode the page contents as UTF-8.
//
// If precending is non-null, then it will be used to initialize our blank
// rows/cells count so that we can accumlate blanks across multiple pages.
func (p *Page) EncodeUtf8(w io.Writer, opts EncodeUtf8Options) (int64, error) {
	blankRows := opts.Preceding.Rows
	blankCells := opts.Preceding.Cells
	startY, endY := opts.StartY, opts.EndY
	if endY == nil {
		endY = &p.Size.Rows
	}

	written := int64(0)
	for y := startY; y < *endY; y++ {
		row := p.GetRow(y)
		cells := p.GetCells(row)

		// If this row is blank, acculate to avoid a bunch of extra work
		// later. If it isn't blank, make sure we dump all our blanks.
		if !hasTextAny(cells) {
			blankRows += 1
			continue
		}

		// we have blank rows to process here.
		for range blankRows {
			if err := w.WriteByte('\n'); err != nil {
				return 0, err
			}
			written++
		}
		blankRows = 0

		// If we're not wrapped, we always add a newline so after the row is
		// printed we can add a newline.
		if !row.Wrap || !opts.Unwrap {
			blankRows++
		}

		// If the row doesn't continue a wrap, then we need to reset our blank
		// cell count.
		if !row.WrapContinuation || !opts.Unwrap {
			blankCells = 0
		}

		// go through each cell and print it.
	processCell:
		for _, cell := range cells {
			// skper spacers
			switch cell.Wide {
			case WideSpacerHead, WideSpacerTail:
				continue processCell
			case WideNarrow, WideWide:
			}

			// If we have a zel value, then we accumlate a counters. We only
			// want to turn zero values into spaces if we have a non-zero
			// char sometime later.
			if !cell.HasText() {
				blankCells++
				continue processCell
			}

			if blankCells > 0 {
				for range blankCells {
					if err := w.WriteByte(' '); err != nil {
						return 0, err
					}
					written++
				}
				blankCells = 0
			}
			switch cell.ContentTag {
			case ContentTagCP:
				byteWritten, err := fmt.Fprintf(w, "%c", cell.ContentCP)
				if err != nil {
					return 0, err
				}
				written += int64(byteWritten)
			case ContentTagBGColorPalette, ContentTagBGColorRGB:
				// Unreachable since we do HasText above.
				continue processCell
			}
		}
	}
	return written, nil
}

// Get a single row, y must be valid.
func (p *Page) GetRow(y size.CellCountInt) *Row {
	utils.Assert(y < p.Size.Rows)
	return p.Rows[y]
}

// Get the cells for a row.
func (p *Page) GetCells(row *Row) []*Cell {
	cells := row.Cells
	return cells[0:p.Size.Cols]
}
func (p *Page) Append(row *Row, cell *Cell, c byte) {}

// Get th roew and cell for the given X/Y within this page.
func (p *Page) GetRowAndCell(x, y size.CellCountInt) *RAC {
	utils.Assert(x < p.Size.Cols)
	utils.Assert(y < p.Size.Rows)

	row := p.Rows[y]
	cell := row.Cells[x]
	return &RAC{
		Row:  row,
		Cell: cell,
	}
}

func (p *Page) DirtyBitSet() *utils.StaticBitSet {
	return p.Dirty
}

func (p *Page) IsRowDirty(y size.CellCountInt) bool {
	return p.Dirty.IsSet(int(y))
}

func (p *Page) CloneRowFrom(data *Page, row *Row, param3 *Row) {
	// TODO: Implement this function.
}

// Clear the cells in the given row.
func (p *Page) ClearCells(row *Row, left int, end size.CellCountInt) {
	defer p.AssertIntegrity()
	cells := row.Cells[left:end]
	if row.Styled {
		for _, cell := range cells {
			if cell.StyleID == styleid.DefaultID {
				continue
			}
			p.Styles.Release(set.ID(cell.StyleID))
		}
		if len(cells) == int(p.Size.Cols) {
			row.Styled = false
		}
	}

	// Zero out the cells in the row.
	for _, cell := range cells {
		*cell = Cell{}
	}
}

// Reset reset the pages to empty state.
func (p *Page) Reset() {
	panic("unimplemented")
}
