package termio_test

import (
	"testing"

	"github.com/hnimtadd/termio/terminal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerminalIODiagnosticsCursorAndSize(t *testing.T) {
	ti := newTestTermio(12, 6)

	require.NoError(t, ti.ProcessOutput([]byte("abc")))

	cursor := ti.GetCursor()
	assert.Equal(t, 3, cursor.X)
	assert.Equal(t, 0, cursor.Y)

	sz := ti.GetSize()
	assert.Equal(t, 12, sz.Cols)
	assert.Equal(t, 6, sz.Rows)
}

func TestTerminalIODiagnosticsModeQueries(t *testing.T) {
	ti := newTestTermio(10, 4)

	require.NoError(t, ti.ProcessOutput([]byte("\x1b[4h")))
	assert.True(t, ti.GetMode(core.ModeInsert))

	enabled, found := ti.GetModeByValue(4, true)
	assert.True(t, found)
	assert.True(t, enabled)

	_, found = ti.GetModeByValue(9999, true)
	assert.False(t, found)
}

func TestTerminalIOSnapshot(t *testing.T) {
	ti := newTestTermio(20, 5)

	require.NoError(t, ti.ProcessOutput([]byte("\x1b[?2004hhello")))

	snap := ti.Snapshot()
	assert.Contains(t, snap.Content, "hello")
	assert.NotEmpty(t, snap.FormattedContent)
	assert.Equal(t, 5, snap.Cursor.X)
	assert.Equal(t, 0, snap.Cursor.Y)
	assert.Equal(t, 20, snap.Size.Cols)
	assert.Equal(t, 5, snap.Size.Rows)
	assert.True(t, snap.Modes[core.ModeBracketedPaste.Name])
}
