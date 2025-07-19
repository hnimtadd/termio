package screen

import (
	"bytes"
	"testing"

	"github.com/hnimtadd/termio/terminal/point"
	"github.com/hnimtadd/termio/terminal/sgr"
	styleid "github.com/hnimtadd/termio/terminal/style/id"
	"github.com/stretchr/testify/assert"
)

type testWriter struct {
	bytes.Buffer
}

func TestScreen_ReadAndWrite(t *testing.T) {
	s := NewScreen(80, 24)
	assert.NotNil(t, s)
	assert.Equal(t, styleid.ID(0), s.Cursor.StyleID)

	// Write some data to the screen
	err := s.testWriteString([]byte("Hello, World!"))
	assert.NoError(t, err)

	writer := &testWriter{bytes.Buffer{}}
	err = s.DumpString(writer, point.TagScreen)
	assert.NoError(t, err)
	// Read back the data
	assert.Equal(t, "Hello, World!", writer.String())
}

func TestScreen_ReadAndWriteNewLine(t *testing.T) {
	s := NewScreen(80, 24)
	assert.Equal(t, styleid.ID(0), s.Cursor.StyleID)

	// Write some data to the screen with a newline
	err := s.testWriteString([]byte(`hello\nworld`))
	assert.NoError(t, err)
	writer := &testWriter{bytes.Buffer{}}
	err = s.DumpString(writer, point.TagScreen)
	assert.NoError(t, err)
	assert.Equal(t, `hello\nworld`, writer.String())
}

func TestScreen_ReadAndWriteScrollback(t *testing.T) {
	s := NewScreen(80, 2)

	err := s.testWriteString([]byte("Line 1\nLine 2\nLine 3"))
	assert.NoError(t, err)

	writer := &testWriter{bytes.Buffer{}}
	err = s.DumpString(writer, point.TagScreen)
	assert.NoError(t, err)
	assert.Equal(t, "Line 1\nLine 2\nLine 3", writer.String())

	writer = &testWriter{bytes.Buffer{}}
	err = s.DumpString(writer, point.TagActive)
	assert.NoError(t, err)
	assert.Equal(t, "Line 2\nLine 3", writer.String())
}

func TestScreen_StyleBasics(t *testing.T) {
	s := NewScreen(80, 24)
	page := s.Cursor.PagePin.Node.Data
	assert.Equal(t, 0, page.Styles.Count())

	// Set a new style
	s.SetAttribute(sgr.Attribute{Type: sgr.AttributeTypeBold})
	assert.NotEqual(t, styleid.ID(0), s.Cursor.StyleID)
	assert.Equal(t, 1, page.Styles.Count())
	assert.True(t, s.Cursor.Style.Bold)

	// Set another style, we should still only have one since it was unused
	s.SetAttribute(sgr.Attribute{Type: sgr.AttributeTypeItalic})
	assert.NotEqual(t, styleid.ID(0), s.Cursor.StyleID)
	assert.Equal(t, 1, page.Styles.Count())
	assert.True(t, s.Cursor.Style.Italic)
}

func TestScreen_StyleReset(t *testing.T) {
	s := NewScreen(80, 24)
	page := s.Cursor.PagePin.Node.Data
	assert.Equal(t, 0, page.Styles.Count())

	// Set some styles
	s.SetAttribute(sgr.Attribute{Type: sgr.AttributeTypeBold})
	assert.NotEqual(t, styleid.ID(0), s.Cursor.StyleID)
	assert.Equal(t, 1, page.Styles.Count())
	assert.True(t, s.Cursor.Style.Bold)

	// Reset the style to default
	s.SetAttribute(sgr.Attribute{Type: sgr.AttributeTypeResetBold})
	assert.Equal(t, styleid.ID(0), s.Cursor.StyleID)
	assert.Equal(t, 0, page.Styles.Count())
}

func TestScreen_ResetWithUnset(t *testing.T) {
	s := NewScreen(80, 24)
	page := s.Cursor.PagePin.Node.Data
	assert.Equal(t, 0, page.Styles.Count())

	// Set a new style
	s.SetAttribute(sgr.Attribute{Type: sgr.AttributeTypeBold})
	assert.NotEqual(t, styleid.ID(0), s.Cursor.StyleID)
	assert.Equal(t, 1, page.Styles.Count())

	// Reset to defaul.
	s.SetAttribute(sgr.Attribute{Type: sgr.AttributeTypeUnset})
	assert.Equal(t, styleid.ID(0), s.Cursor.StyleID)
	assert.Equal(t, 0, page.Styles.Count())
}
