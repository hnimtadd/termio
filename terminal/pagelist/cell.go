package pagelist

import (
	"github.com/hnimtadd/termio/terminal/datastruct"
	"github.com/hnimtadd/termio/terminal/page"
	"github.com/hnimtadd/termio/terminal/size"
)

type Cell struct {
	Node   *datastruct.Node[*page.Page]
	Row    *page.Row
	Cell   *page.Cell
	RowIdx size.CellCountInt
	ColIdx size.CellCountInt
}

func (c *Cell) IsDirty() bool {
	return c.Node.Data.IsRowDirty(c.RowIdx)
}
