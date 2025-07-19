package terminal

import (
	"slices"
	"testing"

	"github.com/hnimtadd/termio/logger"
	"github.com/hnimtadd/termio/terminal/coordinate"
	"github.com/hnimtadd/termio/terminal/core"
	"github.com/hnimtadd/termio/terminal/point"
	"github.com/hnimtadd/termio/terminal/size"
	"github.com/stretchr/testify/assert"
)

func TestTerminal_InputWithNoControlCharacters(t *testing.T) {
	const rows = 40
	const cols = 40
	term := NewTerminal(Options{
		Cols:   cols,
		Rows:   rows,
		Modes:  core.ModePacked,
		Logger: logger.DefaultLogger,
	})

	// Basic grid writing
	input := "hello"
	for c := range slices.Values([]byte(input)) {
		term.Print(uint32(c))
	}
	// Check cursor position
	assert.Equal(t, size.CellCountInt(0), term.Screen.Cursor.Y)
	assert.Equal(t, size.CellCountInt(5), term.Screen.Cursor.X)

	// Check screen content
	content := term.PlainString()
	assert.Equal(t, input, content)
	// Written row should be dirty
	assert.True(t, term.isDirty(point.Point{
		Tag:        point.TagScreen,
		Coordinate: coordinate.Point[size.CellCountInt]{X: 4, Y: 0},
	}))
	assert.False(t, term.isDirty(point.Point{
		Tag:        point.TagScreen,
		Coordinate: coordinate.Point[size.CellCountInt]{X: 5, Y: 1},
	}))
}

func TestTerminal_InputWithWraparound(t *testing.T) {
	const rows = 40
	const cols = 5

	term := NewTerminal(Options{
		Cols:   cols,
		Rows:   rows,
		Modes:  core.ModePacked,
		Logger: logger.DefaultLogger,
	})

	// Basic grid writing
	input := "helloworldabc12"
	for _, c := range input {
		// check print wrap
		term.Print(uint32(c))
	}

	// Verify cursor position and wrap state
	assert.Equal(t, size.CellCountInt(2), term.Screen.Cursor.Y,
		"cursor Y should be 2")
	assert.Equal(t, size.CellCountInt(4), term.Screen.Cursor.X,
		"cursor X should be 4")
	assert.True(t, term.Screen.Cursor.PendingWrap,
		"cursor should be pending wrap")

	// Mock DumpString to return the expected content
	expectedContent := "hello\nworld\nabc12"

	// Check screen content
	content := term.PlainString()
	assert.Equal(
		t,
		expectedContent,
		content,
		"screen content should match expected",
	)
}

func TestTerminal_InputWithBasicWraparoundDirty(t *testing.T) {
	const rows = 40
	const cols = 5
	term := NewTerminal(Options{
		Cols:   cols,
		Rows:   rows,
		Modes:  core.ModePacked,
		Logger: logger.DefaultLogger,
	})
	// Basic grid writing
	for _, c := range "hello" {
		// check print wrap
		term.Print(uint32(c))
	}

	assert.True(t, term.isDirty(point.Point{
		Tag:        point.TagScreen,
		Coordinate: coordinate.Point[size.CellCountInt]{X: 4, Y: 0},
	}))
	term.clearDirty()
	term.Print('w')

	// Old row is dirty as we moved from there
	assert.True(t, term.isDirty(point.Point{
		Tag:        point.TagScreen,
		Coordinate: coordinate.Point[size.CellCountInt]{X: 4, Y: 0},
	}))
	assert.True(t, term.isDirty(point.Point{
		Tag:        point.TagScreen,
		Coordinate: coordinate.Point[size.CellCountInt]{X: 0, Y: 1},
	}))
}

func TestTerminal_InputThatForcesScroll(t *testing.T) {
	rows := 5
	cols := 1

	term := NewTerminal(Options{
		Cols:   cols,
		Rows:   rows,
		Modes:  core.ModePacked,
		Logger: logger.DefaultLogger,
	})

	// Basic grid writing
	input := "abcdef"
	for _, c := range input {
		term.Print(uint32(c))
	}

	assert.Equal(t, size.CellCountInt(4), term.Screen.Cursor.Y,
		"cursor Y should be 5")
	assert.Equal(t, size.CellCountInt(0), term.Screen.Cursor.X,
		"cursor X should be 0")
	{
		str := term.PlainString()
		assert.Equal(t, "b\nc\nd\ne\nf", str,
			"screen content should match expected")
	}
}

// Takes a look at this
// func TestTerminal_InputUniqueStylePerCell(t *testing.T) {
// 	cols := 30
// 	rows := 30
// 	term := NewTerminal(Options{
// 		Cols:   cols,
// 		Rows:   rows,
// 		Modes:  core.ModePacked,
// 		Logger: logger.DefaultLogger,
// 	})
//
// 	for y := range term.rows {
// 		for x := range term.cols {
// 			term.SetCursorPosition(uint16(y), uint16(x))
// 			term.SetAttribute(sgr.Attribute{
// 				Type: sgr.AttributeTypeDirectColorBg,
// 				DirectColorBg: color.RGB{
// 					R: uint8(x),
// 					G: uint8(y),
// 					B: 0,
// 				},
// 			})
// 			term.Print('x')
// 		}
// 	}
// }

func TestTerminal_ZeroWidthCharacterAtStart(t *testing.T) {
	cols := 30
	rows := 30
	term := NewTerminal(Options{
		Cols:   cols,
		Rows:   rows,
		Modes:  core.ModePacked,
		Logger: logger.DefaultLogger,
	})

	// Write a zero-width character at the start, we will ignore this character
	// right now.
	term.Print(uint32('\u200b')) // Zero-width space

	assert.Equal(t, size.CellCountInt(0), term.Screen.Cursor.X,
		"cursor X should be 0")
	assert.Equal(t, size.CellCountInt(0), term.Screen.Cursor.Y,
		"cursor Y should be 0")

	// Should not be dirty since we changed nothing.
	assert.False(t, term.isDirty(point.Point{
		Tag:        point.TagScreen,
		Coordinate: coordinate.Point[size.CellCountInt]{X: 0, Y: 0},
	}))
}

func TestTerminal_PrintSingleVeryLongLine(t *testing.T) {
	cols := 5
	rows := 5
	term := NewTerminal(Options{
		Cols:   cols,
		Rows:   rows,
		Modes:  core.ModePacked,
		Logger: logger.DefaultLogger,
	})

	// We assert the terminal will not crash here.
	for range 10000 {
		term.Print('x')
	}
}
