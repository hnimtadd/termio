package tabstops

import (
	"github.com/hnimtadd/termio/terminal/size"
	"github.com/hnimtadd/termio/terminal/utils"
)

// Unit is the type we use per tabstop unit .
type Unit = uint8

const (
	unitBits         size.CellCountInt = 8 // bits in Unit (uint8)
	preallocCols                       = 512
	preallocCount                      = int(preallocCols / unitBits)
	TABSTOP_INTERVAL                   = 8 // Default tabstop interval
)

// Tabstops efficiently tracks tabstop locations.
type Tabstops struct {
	cols     size.CellCountInt
	prealloc [preallocCount]Unit
	dynamic  []Unit
}

// Helper: bit mask for each bit in a Unit
var masks = func() [unitBits]Unit {
	var m [unitBits]Unit
	for i := range unitBits {
		m[i] = 1 << i
	}
	return m
}()

func entry(col size.CellCountInt) int { return int(col / unitBits) }
func index(col size.CellCountInt) int { return int(col % unitBits) }

// NewTabstops creates a new Tabstops for the given number of columns and interval.
func NewTabstops(cols size.CellCountInt, interval uint8) *Tabstops {
	prealloc := [preallocCount]Unit{}
	for i := range prealloc {
		prealloc[i] = 0
	}

	dynamic := []Unit{}

	t := &Tabstops{
		prealloc: prealloc,
		dynamic:  dynamic,
		cols:     cols,
	}
	t.Resize(cols)
	t.Reset(interval)
	return t
}

// Set sets the tabstop at a certain column (0-indexed).
func (t *Tabstops) Set(col size.CellCountInt) {
	i, idx := entry(col), index(col)
	if i < preallocCount {
		t.prealloc[i] |= masks[idx]
		return
	}
	dynI := i - preallocCount
	if dynI < len(t.dynamic) {
		t.dynamic[dynI] |= masks[idx]
	}
}

// Unset unsets the tabstop at a certain column (0-indexed).
func (t *Tabstops) Unset(col size.CellCountInt) {
	i, idx := entry(col), index(col)
	if i < preallocCount {
		t.prealloc[i] &^= masks[idx]
		return
	}
	dynI := i - preallocCount
	if dynI < len(t.dynamic) {
		t.dynamic[dynI] &^= masks[idx]
	}
}

// Get returns true if a tabstop is set at the given column.
func (t *Tabstops) Get(col size.CellCountInt) bool {
	i, idx := entry(col), index(col)
	mask := masks[idx]
	var unit Unit
	if i < preallocCount {
		unit = t.prealloc[i]
	} else {
		dynI := i - preallocCount
		utils.Assert(dynI < len(t.dynamic))
		if dynI < len(t.dynamic) {
			unit = t.dynamic[dynI]
		}
	}
	return unit&mask == mask
}

// Resize ensures the Tabstops can support up to cols columns.
func (t *Tabstops) Resize(cols size.CellCountInt) {
	// Set our new values.
	t.cols = cols

	// do nothing if it fits.
	if cols <= preallocCols {
		return
	}

	// What we need in the dynamic size
	needed := (cols - preallocCols)
	if int(needed) < len(t.dynamic) {
		return
	}
	new := make([]Unit, needed)
	if len(t.dynamic) > 0 {
		copy(new, t.dynamic)
	}
	t.dynamic = new
}

// Capacity returns the maximum number of columns this can support currently.
func (t *Tabstops) Capacity() int {
	return (preallocCount + len(t.dynamic)) * int(unitBits)
}

// Reset unsets all tabstops and then sets initial tabstops at the given interval.
func (t *Tabstops) Reset(interval uint8) {
	for i := range t.prealloc {
		t.prealloc[i] = 0
	}
	for i := range t.dynamic {
		t.dynamic[i] = 0
	}
	if interval > 0 {
		for i := size.CellCountInt(interval); i < t.cols-1; i += size.CellCountInt(interval) {
			t.Set(i)
		}
	}
}
