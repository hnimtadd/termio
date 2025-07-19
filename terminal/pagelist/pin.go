package pagelist

import (
	"iter"

	"github.com/hnimtadd/termio/terminal/datastruct"
	"github.com/hnimtadd/termio/terminal/page"
	"github.com/hnimtadd/termio/terminal/size"
	"github.com/hnimtadd/termio/terminal/utils"
)

type Pin struct {
	// A single node within the PageList linked list.
	Node *datastruct.Node[*page.Page]

	X, Y size.CellCountInt
}

func (p *Pin) MarkDirty() {
	set := p.Node.Data.DirtyBitSet()
	set.Set(int(p.Y))
}

func (p *Pin) Equal(other *Pin) bool {
	return p.Node == other.Node &&
		p.Y == other.Y &&
		p.X == other.X
}

// Return true if p is before other. This is very expecisive since we have
// to traverse the linked list of pages. This should not be called in
// performance critical paths.
func (p *Pin) Before(other *Pin) bool {
	if p.Node == other.Node {
		if p.Y < other.Y {
			return true
		}
		if p.Y > other.Y {
			return false
		}
		return p.X < other.X
	}
	node := p.Node.Next
	for ; node != nil; node = node.Next {
		if node == other.Node {
			return true
		}
	}
	return false
}

// direction that iterators can move.
type direction int

const (
	directionLeftUp    direction = iota // Move left and up.
	directionRightDown                  // Move right and down.
)

func (p *Pin) RowAndCell() *page.RAC {
	rac := p.Node.Data.GetRowAndCell(p.X, p.Y)
	return &page.RAC{
		Row:  rac.Row,
		Cell: rac.Cell,
	}
}

// Move the pin up a certain number of rows, or return nil if the pin goes
// beyond the start of the screen.
func (p *Pin) Up(n size.CellCountInt) *Pin {
	// Index fits within this page.
	if n <= p.Y {
		return &Pin{
			Node: p.Node,
			Y:    p.Y - n,
			X:    p.X,
		}
	}

	// Need to traverse page links to find the page.
	node := p.Node.Prev
	rem := n - p.Y
	for rem > node.Data.Size.Rows {
		rem -= node.Data.Size.Rows
		node = node.Prev
		if node == nil {
			return nil // No more pages to traverse.
		}
	}

	// Here means we have a valid page and a valid row.
	return &Pin{
		Node: node,
		Y:    node.Data.Size.Rows - rem,
		X:    p.X,
	}
}

// Move the offset down n rows. If the offset goes beyond the end of screen,
// return nil
func (p *Pin) Down(n size.CellCountInt) *Pin {
	// Index fits within this page.

	availRows := p.Node.Data.Size.Rows - (p.Y + 1)
	if n <= availRows {
		return &Pin{
			Node: p.Node,
			Y:    p.Y + n,
			X:    p.X,
		}
	}

	// Need to traverse page links to find the page.
	node := p.Node.Next
	rem := n - availRows
	for rem > node.Data.Size.Rows {
		rem -= node.Data.Size.Rows
		node = node.Next
		if node == nil {
			return nil // No more pages to traverse.
		}
	}
	// Here means we have a valid page and a valid row.
	return &Pin{
		Node: node,
		Y:    rem - 1,
		X:    p.X,
	}
}

type LimitType int

const (
	LimitTypeNone  LimitType = iota // No limit.
	LimitTypeCount                  // Limit by number of rows.
	LimitTypeRow                    // Limit by a specific row.
)

type Limit struct {
	Type       LimitType
	LimitCount size.CellCountInt // Limit by number of rows.
	LimitRow   Pin               // Limit by a specific row pin.
}

type PageIterator struct {
	Row       *Pin
	Limit     Limit
	Direction direction
}

type Chunk struct {
	Node   *datastruct.Node[*page.Page]
	StartY size.CellCountInt
	EndY   size.CellCountInt
}

func (p *PageIterator) Next() iter.Seq[*Chunk] {
	return func(yield func(*Chunk) bool) {
		for {
			var next *Chunk
			switch p.Direction {
			case directionLeftUp:
				next = p.nextUp()
			case directionRightDown:
				next = p.nextDown()
			}
			if next == nil {
				return
			}
			if !yield(next) {
				return
			}
		}
	}
}

func (p *PageIterator) nextUp() *Chunk {
	// Get the current row location
	row := p.Row
	if row == nil {
		return nil
	}
	switch p.Limit.Type {
	case LimitTypeNone:
		// If we have no limit, then we can consume this entire page. Our next
		// row is the previouls page.
		prevPage := row.Node.Prev
		p.Row = &Pin{
			Node: prevPage,
			Y:    prevPage.Data.Size.Rows - 1,
		}
		return &Chunk{
			Node:   row.Node,
			StartY: 0,
			EndY:   row.Y + 1,
		}
	case LimitTypeCount:
		utils.Assert(p.Limit.LimitCount > 0)
		rem := min(row.Y, p.Limit.LimitCount)
		if rem < p.Limit.LimitCount {
			p.Row = row.Up(rem)
			p.Limit.LimitCount -= rem
		} else {
			p.Row = nil // We have consumed the entire limit.
		}
		return &Chunk{
			Node:   row.Node,
			StartY: row.Y - rem,
			EndY:   row.Y - 1,
		}
	case LimitTypeRow:
		// If this is not the same page as our limit, then we can consume the
		// entire page.
		if p.Limit.LimitRow.Node != row.Node {
			prevPage := row.Node.Prev
			p.Row = &Pin{
				Node: prevPage,
				Y:    prevPage.Data.Size.Rows - 1,
			}
			return &Chunk{
				Node:   row.Node,
				StartY: 0,
				EndY:   row.Y + 1,
			}
		}
		// If this is in the same page, then we can only consume up to the limit
		// row.
		p.Row = nil
		// This is invalid as we are going up.
		if row.Y < p.Limit.LimitRow.Y {
			return nil
		}
		return &Chunk{
			Node:   row.Node,
			StartY: p.Limit.LimitRow.Y,
			EndY:   row.Y + 1,
		}
	default:
		return nil
	}
}

func (p *PageIterator) nextDown() *Chunk {
	// Get our current row location
	row := p.Row
	if row == nil {
		return nil
	}
	switch p.Limit.Type {
	case LimitTypeNone:
		// If we have no limit, then we can consume this entire page. Our next
		// row is the next page.
		p.Row = &Pin{
			Node: row.Node.Next,
		}
		return &Chunk{
			Node:   row.Node,
			StartY: row.Y,
			EndY:   row.Node.Data.Size.Rows,
		}
	case LimitTypeCount:
		utils.Assert(p.Limit.LimitCount > 0)
		rem := min(row.Node.Data.Size.Rows-row.Y, p.Limit.LimitCount)
		if rem < p.Limit.LimitCount {
			p.Row = row.Down(rem)
			p.Limit.LimitCount -= rem
		} else {
			p.Row = nil // We have consumed the entire limit.
		}
		return &Chunk{
			Node:   row.Node,
			StartY: row.Y,
			EndY:   row.Y + rem,
		}
	case LimitTypeRow:
		// If this is not the same page as our limit, then we can consume the
		// entire page.
		if p.Limit.LimitRow.Node != row.Node {
			p.Row = &Pin{
				Node: row.Node.Next,
			}
			return &Chunk{
				Node:   row.Node,
				StartY: row.Y,
				EndY:   row.Node.Data.Size.Rows,
			}
		}
		// If this is in the same page, then we can only consume up to the limit
		// row.
		p.Row = nil
		// This is invalid as we are going down.
		if row.Y > p.Limit.LimitRow.Y {
			return nil
		}
		return &Chunk{
			Node:   row.Node,
			StartY: row.Y,
			EndY:   p.Limit.LimitRow.Y + 1,
		}
	default:
		return nil
	}
}

func (p *Pin) PageIterator(direction direction, limit *Pin) *PageIterator {
	if limit != nil {
		switch direction {
		case directionLeftUp:
			utils.Assert(p.Equal(limit) || limit.Before(p))
		case directionRightDown:
			utils.Assert(p.Equal(limit) || p.Before(limit))
		}
	}

	itLimit := Limit{
		Type: LimitTypeNone,
	}
	if limit != nil {
		itLimit.Type = LimitTypeRow
		itLimit.LimitRow = *limit
	}
	return &PageIterator{
		Row:       p,
		Limit:     itLimit,
		Direction: direction,
	}
}
