package screen

import (
	"github.com/hnimtadd/termio/terminal/page"
	"github.com/hnimtadd/termio/terminal/pagelist"
	"github.com/hnimtadd/termio/terminal/size"
	"github.com/hnimtadd/termio/terminal/style"
	styleid "github.com/hnimtadd/termio/terminal/style/id"
)

// The cursor position and style.
type Cursor struct {
	X           size.CellCountInt
	Y           size.CellCountInt
	PendingWrap bool          // Whether the cursor is pending to wrap
	PageCell    *page.Cell    // Cell at the cursor position
	PageRow     *page.Row     // Row of the page
	PagePin     *pagelist.Pin // Pin of the page

	// The current active style. This is the concerte style value that
	// should be kept up to date. The style ID to use for cell writing
	// is below
	Style style.Style

	// The current active style ID. The style is page-specific so when
	// we change pages, we need to ensurethat update that page with our style
	// when used.
	StyleID styleid.ID
}
