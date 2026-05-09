package termio_test

import (
	"testing"

	"github.com/hnimtadd/termio/terminal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerminalIOComplexCursorSaveRestoreAndCharset(t *testing.T) {
	ti := newTestTermio(20, 6)

	seq := []byte(
		"hello" + // cursor at (5, 0)
			"\x1b7" + // DECSC
			"\x1b[3;4H" + // CUP -> row3 col4
			"\x1b)0" + // designate DEC special graphics to G1
			"\x0e" + // SO (shift out to G1)
			"q" + // renders box drawing char
			"\x0f" + // SI (shift in to G0)
			"\x1b8" + // DECRC
			"!", // print at restored cursor
	)

	require.NoError(t, ti.ProcessOutput(seq))

	cursor := ti.GetCursor()
	assert.Equal(t, 6, cursor.X)
	assert.Equal(t, 0, cursor.Y)

	content := ti.DumpString()
	assert.Contains(t, content, "hello!")
	assert.Contains(t, content, "─")
}

func TestTerminalIOComplexScrollingRegionBehavior(t *testing.T) {
	ti := newTestTermio(5, 5)

	seq := []byte(
		"\x1b[1;1H11111" +
			"\x1b[2;1H22222" +
			"\x1b[3;1H33333" +
			"\x1b[4;1H44444" +
			"\x1b[5;1H55555" +
			"\x1b[2;4r" + // DECSTBM top=2 bottom=4
			"\x1b[4;1H" + // move to bottom of region
			"\n" + // linefeed should scroll only region
			"\x1b[4;1HZZZZZ",
	)

	require.NoError(t, ti.ProcessOutput(seq))

	content := ti.DumpString()
	assert.Contains(t, content, "11111")
	assert.Contains(t, content, "33333")
	assert.Contains(t, content, "44444")
	assert.Contains(t, content, "ZZZZZ")
	assert.NotContains(t, content, "22222")
}

func TestTerminalIOComplexOSCAndDCSDoNotBreakStream(t *testing.T) {
	ti := newTestTermio(40, 4)

	seq := []byte(
		"start" +
			"\x1b]0;demo-title\x1b\\" + // OSC (terminated by ST: ESC \)
			"mid" +
			"\x1bP1;2|abc\x1b\\" + // DCS passthrough terminated by ST
			"end",
	)

	require.NoError(t, ti.ProcessOutput(seq))
	assert.Contains(t, ti.DumpString(), "startmidend")
}

func TestTerminalIOComplexModeToggles(t *testing.T) {
	ti := newTestTermio(20, 4)

	require.NoError(t, ti.ProcessOutput([]byte("\x1b[4h"))) // IRM set
	assert.True(t, ti.GetMode(core.ModeInsert))

	require.NoError(t, ti.ProcessOutput([]byte("\x1b[4l"))) // IRM reset
	assert.False(t, ti.GetMode(core.ModeInsert))

	require.NoError(t, ti.ProcessOutput([]byte("\x1b[?2004h"))) // bracketed paste set
	assert.True(t, ti.GetMode(core.ModeBracketedPaste))

	require.NoError(t, ti.ProcessOutput([]byte("\x1b[?2004l"))) // bracketed paste reset
	assert.False(t, ti.GetMode(core.ModeBracketedPaste))
}
