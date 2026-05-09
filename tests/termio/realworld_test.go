package termio_test

import (
	"strings"
	"testing"

	"github.com/hnimtadd/termio/terminal/core"
	"github.com/stretchr/testify/assert"
)

func TestTerminalIORealWorldShellSessionReplay(t *testing.T) {
	ti := newTestTermio(80, 24)

	transcript := strings.Join([]string{
		"\x1b]0;user@host: ~\x1b\\", // OSC title
		"\x1b[?2004h",               // bracketed paste on
		"\x1b[1;32muser@host\x1b[0m:\x1b[34m~\x1b[0m$ ls\r\n",
		"\x1b[38;5;33mfile.txt\x1b[0m  \x1b[38;5;40msrc\x1b[0m\r\n",
		"\x1b[1;32muser@host\x1b[0m:\x1b[34m~\x1b[0m$ ",
		"\x1b[?2004l", // bracketed paste off
	}, "")

	content := runTranscript(t, ti, transcript)

	assert.Contains(t, content, "user@host:~$ ls")
	assert.Contains(t, content, "file.txt  src")
	assert.Contains(t, content, "user@host:~$")
	assert.False(t, ti.GetMode(core.ModeBracketedPaste))
}

func TestTerminalIORealWorldProgressAndCarriageReturn(t *testing.T) {
	ti := newTestTermio(60, 8)

	transcript := strings.Join([]string{
		"Downloading 010%\r",
		"Downloading 040%\r",
		"Downloading 100%\r\n",
		"Done\r\n",
	}, "")

	content := runTranscript(t, ti, transcript)

	assert.Contains(t, content, "Downloading 100%")
	assert.Contains(t, content, "Done")
	assert.NotContains(t, content, "Downloading 040%")
}

func TestTerminalIORealWorldFullscreenRedrawLoop(t *testing.T) {
	ti := newTestTermio(40, 10)

	transcript := strings.Join([]string{
		"\x1b[2J\x1b[H", // clear + home
		"CPU 10%\r\nMEM 40%\r\nTASKS 120\r\n",
		"\x1b[H", // redraw from home
		"CPU 35%\r\nMEM 45%\r\nTASKS 122\r\n",
	}, "")

	content := runTranscript(t, ti, transcript)

	assert.Contains(t, content, "CPU 35%")
	assert.Contains(t, content, "MEM 45%")
	assert.Contains(t, content, "TASKS 122")
	assert.NotContains(t, content, "CPU 10%")
}

func TestTerminalIORealWorldMixedVTAndUTF8Replay(t *testing.T) {
	ti := newTestTermio(32, 8)

	transcript := strings.Join([]string{
		"\x1b7",       // save cursor
		"Build: [",    //
		"\x1b)0\x0eq", // designate g1 + shift out + draw horizontal line
		"\x0f",        // shift in
		"] 50%\r\n",
		"Status: chạy thử 😄\r\n", // mixed UTF-8
		"\x1b8",                  // restore cursor to line 0
		"Done ",                  // overwrite start
	}, "")

	content := runTranscript(t, ti, transcript)
	cursor := ti.GetCursor()

	assert.Contains(t, content, "Done ")
	assert.Contains(t, content, "─")
	assert.Contains(t, content, "Status: chạy thử 😄")
	assert.Equal(t, 5, cursor.X)
	assert.Equal(t, 0, cursor.Y)
}
