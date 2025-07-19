package point

import (
	"github.com/hnimtadd/termio/terminal/coordinate"
	"github.com/hnimtadd/termio/terminal/size"
)

// The possible reference locations for a point. "(42, 80)" in the context
// of a terminal, that could mean multiple things:
// - It is the current visiable viewport
// - It is the current active area of the screen where the cursor is?
// - It is the entire scollback history
// - Etc.
// This tag is used to differenate those case
type Tag int

const (
	// Top-left is the visible viewport. This means that if the user has
	// scrolled in any direction, top-left changes. The bottom-right is the
	// last written row from the top-left.
	TagViewPort Tag = iota

	// Top-left is part of the active area where a running program can jump
	// the cursor and make changes. The active area is the "editable" part of
	// the screen.
	//
	// The bottom-right of active tag differs from all other tags because it
	// includes the full height (rows) of thescreen, including rows that may
	// not be written yet. This is required because the active area is fully
	// "addressable" by the running program, whereas the other tags are used
	// primarily for reading/modifying past-written data so they can't address
	// unwritten rows.
	TagActive

	// Top-left is the furthest back in the scrollback history supported by the
	// screen and the bottom-right is the bottom-right of the last written row.
	// Note this last point is important: the bottom right is NOT necessarily
	// the same as "active" because "active" always allows referencing the full
	// rows tall of the screen whereas "screen" only contains written rows.
	TagScreen

	// The top-left is the same as "screen" but the bottom-right is the line
	// just before the top of "active". This contains only the scrollback
	// history.
	TagHistory
)

func (t Tag) String() string {
	switch t {
	case TagViewPort:
		return "viewport"
	case TagActive:
		return "active"
	case TagScreen:
		return "screen"
	case TagHistory:
		return "history"
	default:
		return "unknown"
	}
}

// An x/y point in the terminal for some definition of location (tag).
type Point struct {
	Tag        Tag
	Coordinate coordinate.Point[size.CellCountInt]
}
