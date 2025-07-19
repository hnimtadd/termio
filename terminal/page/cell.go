package page

import (
	"github.com/hnimtadd/termio/terminal/color"
	styleid "github.com/hnimtadd/termio/terminal/style/id"
)

type ContentTag int

const (
	// A single codepoint, could be zero to be empty cell.
	ContentTagCP ContentTag = 1
	// The cell has no text but only a background color. This is an
	// optimization so that cells with only backgrounds don't take up style
	// map space and also don't require a style map lookup.
	ContentTagBGColorPalette ContentTag = 2
	ContentTagBGColorRGB     ContentTag = 3
)

type Cell struct {
	ContentTag          ContentTag
	ContentCP           uint32
	ContentColorPalette uint8
	ContentColorRGB     color.RGB
	// The wide property of this cell, for wide characters. Characters in a
	// terminal grid can only be 1 or 2 cells wide. A wide character is alwasy
	// next to a spacer. This is used to determine both width and spacer
	// properties of a cell.
	Wide    Wide
	IsDirty bool

	// The style ID to use for this cell within the style map. Zero
	// is always the default style so no lookup is required.
	StyleID styleid.ID
}

func (c *Cell) Codepoint() uint32 {
	return c.ContentCP
}

// The width in grid cells that this cell takes up.
func (c *Cell) Width() uint8 {
	switch c.Wide {
	case WideNarrow, WideSpacerHead, WideSpacerTail:
		return 1
	case WideWide:
		return 2
	default:
		panic("unknown cell wide")
	}
}

func (c *Cell) IsEmpty() bool {
	return c.ContentTag == ContentTagCP && c.ContentCP == 0
}

// Returns true if this cell represents a cell with text to render.
//
// Cases this returns false:
//   - Cell text is blank
//   - Cell is styled but only with a background color and no text
//   - Cell has a unicode placeholder for Kitty graphics protocol
func (c *Cell) HasText() bool {
	switch c.ContentTag {
	case ContentTagCP:
		return c.ContentCP != 0
	case ContentTagBGColorPalette, ContentTagBGColorRGB:
		return false
	default:
		// If we don't know the content tag, we assume it doesn't text.
		return false
	}
}

// Returns true if the set of cells has text in it.
func hasTextAny(cells []*Cell) bool {
	for _, cell := range cells {
		if cell.HasText() {
			return true
		}
	}
	return false
}
