package pagelist

import (
	"github.com/hnimtadd/termio/coordinate"
	"github.com/hnimtadd/termio/io"
	"github.com/hnimtadd/termio/terminal/datastruct"
	"github.com/hnimtadd/termio/terminal/page"
	"github.com/hnimtadd/termio/terminal/point"
	"github.com/hnimtadd/termio/terminal/size"
	"github.com/hnimtadd/termio/terminal/utils"
)

type (
	List struct {
		datastruct.IntrusiveLinkedList[*page.Page]
	}
	PageList struct {
		Pages *List
		// Number the total amount of allocated pages. Note this does
		// not include the total allocated amount in the pool which may be more
		// than this due to preheating.
		PageSize uint64

		// The current desired screen dimensions. I say "desired" because individual
		// pages may still be a different size and not yet reflowed since we lazily
		// reflow text.
		Cols, Rows size.CellCountInt

		// The top-left of certain parts of the screen that are frequently
		// accessed so we don't have to traverse the linked list to find them.
		//
		// For other tags, don't need this:
		//   - screen: pages.first
		//   - history: active row minus one
		ViewPort ViewportTag

		// The pin used for when the viewport scrolls. This is always pre-allocated
		// so that scrolling doesn't have a failable memory allocation. This should
		// never be access directly; use `viewport`.
		ViewPortPin *Pin

		// The list of tracked pins. These are pins that are automatically
		// updated as the page list is modified.
		TrackedPins *datastruct.IntrusiveLinkedList[*Pin]

		// Maximum number of the page allocation. This only includes
		// pages that are used ONLY for scrollback. If the active area is
		// still partially in a page that also includes scrollback, then that
		// page is not included.
		MaxPagesSize uint64
	}
	// The viewport location.
	ViewportTag int
)

// Fast-path function to erase exactly 1 row. Erasing means that the row is
// completely REMOVED, not just cleared. All rows folloing the removed row
// will be shifted
// up by 1 to fill the empty space.
//
// Unlik EraseRows, EraseRow does not change the size of any pages. The caller
// is responsbile for adjusted the row count of the final page if that
// behavior is required.
func (p *PageList) EraseRow(pt point.Point) {
	pin := p.Pin(pt)
	node := pin.Node
	rows := node.Data.Rows

	// In order to move the folloing rows up, we rotate the rows array by 1.
	// [ 0 1 2 3 ] => [ 1 2 3 0]
	utils.RotateOnce(rows[pin.Y:node.Data.Size.Rows])

	// We adjusted the tracked pins in this page, moving up any that we below
	// the removed row.
	{
		for p := range p.TrackedPins.All() {
			if p.Node == node && p.Y >= pin.Y {
				p.Y -= 1
			}
		}
	}
	// We set all rotated rows as dirty.
	{
		dirty := node.Data.DirtyBitSet()
		dirty.SetRange(int(pin.Y), int(node.Data.Size.Rows))
	}

	// We iterate through all of the following pages in order to move their
	// rows up by 1 as well
	for next := node.Next; next != nil; {
		nextRows := node.Next.Data.Rows

		node.Data.CloneRowFrom(
			next.Data,
			rows[node.Data.Size.Rows-1],
			nextRows[0],
		)
		node = next
		rows = nextRows

		utils.RotateOnce(rows[0:node.Data.Size.Rows])

		// Set all the rows as dirty.
		dirty := node.Data.DirtyBitSet()
		dirty.SetRange(0, int(node.Data.Size.Rows))

		// Update tracked pins.
		for p := range p.TrackedPins.All() {
			if p.Node != node {
				continue
			}

			// If the pin is in row 0, that means the corresponding row was
			// moved from previous page, so we move it to the previous page.
			if p.Y == 0 {
				p.Node = node.Prev
				p.Y = node.Prev.Data.Size.Rows - 1
				continue
			}
			// Otherwise, move it up by 1.
			p.Y -= 1 // Move up by one row.
		}
	}

	// Clear the last row since it was rotated down from the top of some page.
	node.Data.ClearCells(rows[node.Data.Size.Rows-1], 0, node.Data.Size.Cols)
}

// Grow the active area by exactly one row.
//
// This might allocate, but also may not if our current page has more
// capacity we can use. This will prune scrollback if necessary to
// adhere to max_size.
func (p *PageList) Grow() *datastruct.Node[*page.Page] {
	last := p.Pages.Last
	if last != nil && last.Data.Size.Rows < last.Data.Capacity.Rows {
		// Fast path, if the last page has space, just grow it.
		last.Data.Size.Rows++
		last.Data.AssertIntegrity()
		return nil
	}

	// slower path: we have no space, we need to allocate a new page.
	//
	// If allocation would exceed the max size, we prune the first page.
	// We don't need to reallocate because we can simple reuse that first
	// page.
	//
	// We only take this path if we have more than one page, since pruning
	// resuses the popped page. It is possible to have a single page and exceed
	// the max size if the page was adjusted to be larger after initial
	// allocation.
	if p.Pages.First != nil &&
		p.Pages.First != p.Pages.Last &&
		p.PageSize+1 > p.MaxPagesSize {
		// Prune the first page.

		// If we can to add more memory to ensure our active area is
		// satisfied then we do not prune.
		{
			rows := 0
			page := p.Pages.Last
			for ; page != nil; page = page.Prev {
				rows += int(page.Data.Size.Rows)
				if rows > int(p.Rows) {
					goto prune
				}
			}
			goto skipPrune
		}
	prune:
		layout := page.StandardCapacity
		layout.Adjust(page.Adjustment{Cols: p.Cols})
		// Get our first page, and reset to prepare for reuse.
		first := p.Pages.First
		utils.Assert(first != last)

		// Initialize our new page and reinsert it as the last.
		first.Data = page.InitPage(layout)
		first.Data.Size.Rows = 1 // We always grow by one row.
		p.Pages.InsertAfter(last, first)

		// Update any tracked pins that point to this page to point to the new
		// first page to the top-left.
		for pin := range p.TrackedPins.All() {
			if pin.Node == first {
				pin.Node = p.Pages.First
				pin.Y = 0
				pin.X = 0
			}
		}
		// In this case, we do not need to update PageSize because we're
		// reusing an existing page so nothing has changed.
		first.Data.AssertIntegrity()
		return first
	}
skipPrune:
	// We need to allocate a new memory buffer
	layout := page.StandardCapacity
	layout.Adjust(page.Adjustment{Cols: p.Cols})
	nextNode := p.CreatePage(layout)
	p.Pages.Append(nextNode)
	nextNode.Data.Size.Rows = 1 // We always grow by one row.

	// We should never be more than our max size here beause we've verified the
	// case above
	nextNode.Data.AssertIntegrity()
	return nextNode
}

// Create a new page node. This does not add it to the list and this does
// not do any memory size accounting with MaxPagesSize, PageSize.
func (p *PageList) CreatePage(cap page.Capacity) *datastruct.Node[*page.Page] {
	data := page.InitPage(cap)
	data.Size.Rows = 0
	page := &datastruct.Node[*page.Page]{
		Data: data,
	}
	return page
}

// A variant of eraseRow that shiftss only a bounded numberf of following rows
// up, filling the space they leave behind with blanks rows.
//
// `limit` is exclusive of erased row. A limit of 1 will erase the target row
// and shift the row below in to its position, leaving a blank row below.
func (p *PageList) EraseRowsBounded(pt point.Point, limit size.CellCountInt) {
	pin := p.Pin(pt)

	node := pin.Node
	rows := node.Data.Rows

	// If the row is less than the remain rows before the end of the page,
	// then we clear the row, rotate it to the end of the boundary and update
	// our pin.
	if node.Data.Size.Rows-pin.Y > limit {
		node.Data.ClearCells(rows[pin.Y], 0, node.Data.Size.Cols)
		utils.RotateOnce(rows[pin.Y : limit+1])

		// Set all the rows as dirty
		dirty := node.Data.DirtyBitSet()
		dirty.SetRange(int(pin.Y), int(pin.Y+limit))

		// Update pins in the shifted region.
		// We need to traver through our trackedPins and update here
		for p := range p.TrackedPins.All() {
			if p.Node == node &&
				p.Y >= pin.Y &&
				p.Y < pin.Y+limit {
				if p.Y == 0 {
					p.X = 0
				} else {
					p.Y -= 1
				}
			}
		}
		return
	}

	utils.RotateOnce(rows[pin.Y:node.Data.Size.Rows])
	// All the rows in the page are dirty below the erased rows.
	{
		dirty := node.Data.DirtyBitSet()
		dirty.SetRange(int(pin.Y), int(node.Data.Size.Rows))
	}

	// We need to keep track of how many rows we have shifted up in current
	// page.
	// So that we can determin at what point twe need to do a partial shift
	// on subsequent pages.
	shifted := node.Data.Size.Rows - pin.Y

	// Update tracked pins on current page.
	{
		for p := range p.TrackedPins.All() {
			if p.Node == node && p.Y >= pin.Y {
				if p.Y == 0 {
					p.X = 0
				} else {
					p.Y -= 1
				}
			}
		}
	}

	for next := node.Next; next != nil; {
		nextRows := next.Data.Rows
		node.Data.CloneRowFrom(
			next.Data,
			rows[node.Data.Size.Rows-1],
			nextRows[0],
		)
		node = node.Next
		rows = nextRows

		// We check to see if this page contains enough rows to sastify the
		// specified limit, accounting for rows we've already shifted in previous
		// pages.
		//
		// After this, the logic is similar to the one before the loop.

		limit := limit - shifted

		if node.Data.Size.Rows-pin.Y > limit {
			node.Data.ClearCells(rows[pin.Y], 0, node.Data.Size.Cols)
			utils.RotateOnce(rows[pin.Y : limit+1])

			// Set all the rows as dirty
			dirty := node.Data.DirtyBitSet()
			dirty.SetRange(int(pin.Y), int(pin.Y+limit))

			// Update pins in the shifted region.
			for p := range p.TrackedPins.All() {
				if p.Node != node || p.Y > limit {
					continue
				}
				if p.Y == 0 {
					p.Node = node.Prev
					p.Y = node.Prev.Data.Size.Rows - 1
					continue
				}
				p.Y -= 1
			}
			return
		}

		utils.RotateOnce(rows[pin.Y:node.Data.Size.Rows])

		// Set all the rows as dirty.
		dirty := node.Data.DirtyBitSet()
		dirty.SetRange(int(pin.Y), int(node.Data.Size.Rows))
		shifted = node.Data.Size.Rows

		// Update tracked pins on current page.

		for p := range p.TrackedPins.All() {
			if p.Node == node && p.Y >= pin.Y {
				if p.Y == 0 {
					p.X = 0
				} else {
					p.Y -= 1
				}
			}
		}
	}

	// We reached the end of pagelist before the limit, so we clear the final
	// row since it was rotated down from the top of this page.
	node.Data.ClearCells(rows[node.Data.Size.Rows-1], 0, node.Data.Size.Cols)
}

// Clear all dirty bits on all pages. This is not efficient since it traverses
// the entire list of pages. This is used for testing/debugging.
func (p *PageList) ClearDirty() {
	node := p.Pages.First
	for ; node != nil; node = node.Next {
		set := node.Data.DirtyBitSet()
		set.Clear()
	}
}

func (p *PageList) Reset() {
	panic("unimplemented")
}

func NewPageList(cols size.CellCountInt, rows size.CellCountInt) *PageList {
	p := &PageList{
		Cols:        cols,
		Rows:        rows,
		PageSize:    0,
		ViewPort:    ViewportTagActive,
		ViewPortPin: &Pin{},
		TrackedPins: datastruct.NewIntrusiveLinkedList[*Pin](),
	}

	p.Pages = p.InitPages(cols, rows)
	p.ViewPortPin.Node = p.Pages.First

	return p
}

const (
	// The viewport is pinned to the active area. By using a specific marker
	// for this instead of tracking the row offset, we eliminate a number of
	// memory writes making scrolling faster.
	ViewportTagActive ViewportTag = iota

	// The viewport is pinned to the top of the screen, or the farthest
	// back in the scrollback history.
	ViewportTagTop

	// The viewport is pinned to a tracked pin. The tracked pin is ALWAYS
	// s.viewport_pin hence this has no value. We force that value to prevent
	// allocations.
	ViewportTagPin
)

func (p *PageList) InitPages(cols size.CellCountInt, rows size.CellCountInt) *List {
	pageList := &List{}

	cap := page.StandardCapacity
	if err := cap.Adjust(page.Adjustment{Cols: cols}); err != nil {
		return nil
	}

	// Add pages as needed to create our initial viewport.
	rem := rows
	for rem > 0 {
		node := &datastruct.Node[*page.Page]{
			Data: page.InitPage(cap),
		}
		node.Data.Size.Rows = min(rem, node.Data.Capacity.Rows)
		rem -= node.Data.Size.Rows

		// Add the page to the list.
		pageList.Append(node)
	}

	utils.Assert(pageList.First != nil, "PageList must have at least one page")
	return pageList
}

type EncodeUtf8Options struct {
	// If true, this will unwrap soft-wrapped lines. If false, this will
	// dump the screen as it is visually seen in a rendered window.
	Unwrap bool

	// The start and end points of the dump, both inclusive. The x will be
	// ignored, and the full row will always be dumped.
	TopLeft     Pin
	BottomRight *Pin // nil means no limit
}

// Encode the pagelist to utf8 to the given writer.
//
// The writer should be buffered; this function does not attempt to
// efficiently write and often writes one byte at a time.
//
// Note: this is tested using Screen.DumpString. This is a function that
// predates this and is a thin wrapper around it so the tests all live there.
func (p *PageList) EncodeUtf8(w io.Writer, opts EncodeUtf8Options) (int64, error) {
	pageOpts := page.EncodeUtf8Options{Unwrap: opts.Unwrap}
	iter := opts.TopLeft.PageIterator(directionRightDown, opts.BottomRight)

	var writen int64 = 0
	for chunk := range iter.Next() {
		page := chunk.Node.Data
		pageOpts.StartY = chunk.StartY
		pageOpts.EndY = &chunk.EndY

		dataWritten, err := page.EncodeUtf8(w, pageOpts)
		if err != nil {
			return 0, err
		}
		writen += dataWritten
	}
	return writen, nil
}

// Get the top-left of the screen for the given tag.
func (p *PageList) GetTopLeft(tag point.Tag) *Pin {
	switch tag {
	// The full screen or history is alwasy just the first page.
	case point.TagScreen, point.TagHistory:
		return &Pin{Node: p.Pages.First}
	case point.TagViewPort:
		switch p.ViewPort {
		case ViewportTagActive:
			return p.GetTopLeft(point.TagActive)
		case ViewportTagTop:
			return p.GetTopLeft(point.TagScreen)
		case ViewportTagPin:
			return p.ViewPortPin
		}
	// The active area is calculated backwards from the last page.
	// This makes getting the active top left slower but makes scrolling much
	// faster because we don't need to update the top-left.Under heavy
	// load, this makes a measureable difference.
	case point.TagActive:
		rem := p.Rows
		it := p.Pages.First
		for ; it != nil; it = it.Prev {
			if rem <= it.Data.Size.Rows {
				return &Pin{
					Node: it,
					Y:    it.Data.Size.Rows - rem,
				}
			}
			rem -= it.Data.Size.Rows
		}
	}
	return nil
}

// Get the bottom-right of the screen for the given tag.
func (p *PageList) GetBottomRight(tag point.Tag) *Pin {
	switch tag {
	case point.TagScreen, point.TagActive:
		node := p.Pages.Last
		return &Pin{
			Node: node,
			Y:    node.Data.Size.Rows - 1,
			X:    node.Data.Size.Cols - 1,
		}
	case point.TagViewPort:
		tl := p.GetTopLeft(point.TagViewPort)
		return tl.Down(p.Rows - 1)
	case point.TagHistory:
		tl := p.GetTopLeft(point.TagActive)
		// go up 1 row to get the last row of the history
		node := tl.Node.Prev
		return &Pin{
			Node: node,
			Y:    node.Data.Size.Rows - 1,
			X:    node.Data.Size.Cols - 1,
		}
	}
	return nil
}

// The total rows in the screen. This is the actual row count currently
// and not a capacity or maximum.
//
// This is very slow, it traverses the full list of pages to count the
// rows, so it is not pub. This is only used for testing/debugging.
func (p *PageList) totalRows() uint {
	var rows uint = 0
	node := p.Pages.First
	for node != nil {
		rows += uint(node.Data.Size.Rows)
		node = node.Next
	}
	return rows
}

// Convert a pin to a point in the given context. If the pin can't fit
// within the given tag (i.e. its in the history but you requested active),
// then this will return null.
//
// Note that this can be a very expensive operation depending on the tag and
// the location of the pin. This works by traversing the linked list of pages
// in the tagged region.
//
// Therefore, this is recommended only very rarely.
func (p *PageList) pointFromPin(tag point.Tag, pin Pin) *point.Point {
	tl := p.GetTopLeft(tag)
	// Count our first page which is special because it may be partial.
	coord := coordinate.Point[size.CellCountInt]{
		X: pin.X,
	}

	if pin.Node == tl.Node {
		// If our top-left is after our y then we're outside the range.
		if tl.Y > pin.Y {
			return nil
		}
		coord.Y = pin.Y - tl.Y
	} else {
		coord.Y += tl.Node.Data.Size.Rows - tl.Y
		node := tl.Node.Next
		for node != nil {
			if node == pin.Node {
				coord.Y += pin.Y
				break
			}

			coord.Y += node.Data.Size.Rows
			node = node.Next
		}
		if node == nil {
			// we never saw our node, meaning we're outside the range.
			return nil
		}
	}
	return &point.Point{
		Tag:        tag,
		Coordinate: coord,
	}
}

// Grow the number of rows available in the page list by repeat.
// This is only used for testing so it isn't optimized.
func (s *PageList) growRows(repeat uint) error {
	page := s.Pages.Last
	rem := repeat
	if page.Data.Size.Rows < page.Data.Capacity.Rows {
		add := min(size.CellCountInt(rem), page.Data.Capacity.Rows-page.Data.Size.Rows)
		page.Data.Size.Rows += add
		rem -= uint(add)
	}
	for rem > 0 {
		page, err := s.grow()
		if err != nil {
			return err
		}
		add := min(size.CellCountInt(rem), page.Data.Capacity.Rows)
		page.Data.Size.Rows = add
		rem -= uint(add)
	}
	return nil
}

// Grow the active area by exactly one row.
//
// This may allocate, but also may not if our current page has more
// capacity we can use. This will prune scrollback if necessary to
// adhere to max_size.
//
// This return nrewly allocated page if it was created.
func (s *PageList) grow() (*datastruct.Node[*page.Page], error) {
	last := s.Pages.Last
	if last.Data.Size.Rows < last.Data.Capacity.Rows {
		// Fast path, if the last page has space, just grow it.
		last.Data.Size.Rows++
		return nil, nil
	}
	// Slower path: we have no space, we need to allocate a new page.
	//
	// We need to allocate a new memory buffer.
	cap := page.StandardCapacity
	if err := cap.Adjust(page.Adjustment{Cols: s.Cols}); err != nil {
		return nil, err
	}
	nextNode := &datastruct.Node[*page.Page]{
		Data: page.InitPage(cap),
	}
	s.Pages.Append(nextNode)
	nextNode.Data.Size.Rows = 1 // We always grow by one row.

	return nextNode, nil
}

// Convert the given pin to a tracked pin. A tracked pin will always be
// automatically updated as the pagelist is modified. If the point the pin
// points to is removed completely, the tacked pin will be updated to the
// top-left of the scree.
func (p *PageList) TrackPin(pin Pin) *Pin {
	// Create our tracked pin.
	tracked := &Pin{
		Node: pin.Node,
		X:    pin.X,
		Y:    pin.Y,
	}

	// Add it to the tracked list.
	p.TrackedPins.Append(&datastruct.Node[*Pin]{
		Data: tracked,
	})
	return tracked
}

func (p *PageList) UntrackPin(pin *Pin) {
	utils.Assert(pin != p.ViewPortPin, "Cannot untrack the viewport pin")
	pinNode := p.TrackedPins.Search(pin)
	if pinNode == nil {
		// failed to find the pin, so we can't untrack it.
		return
	}
	p.TrackedPins.Remove(pinNode)
}

// Return the pin for the given point. The pin is not tracked so it is only
// valid as long as the pagelist is not modified.
//
// This will return nil if the point is not within the pagelist.
// The caller should clamp the point to the bounds of the coodinate space if
// needed.
func (p *PageList) Pin(pt point.Point) *Pin {
	x := pt.Coordinate.X
	if x >= p.Cols {
		return nil
	}
	// Grab the top left and move to the point
	pin := p.GetTopLeft(pt.Tag).Down(pt.Coordinate.Y)
	if pin == nil {
		return nil
	}

	pin.X = x
	return pin
}

func (p *PageList) GetCell(pt point.Point) *Cell {
	ptPin := p.Pin(pt)
	rac := ptPin.Node.Data.GetRowAndCell(ptPin.X, ptPin.Y)
	return &Cell{
		ColIdx: ptPin.X,
		RowIdx: ptPin.Y,
		Row:    rac.Row,
		Cell:   rac.Cell,
		Node:   ptPin.Node,
	}
}
