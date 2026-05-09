package termio_test

import (
	"testing"

	termio "github.com/hnimtadd/termio"
	"github.com/hnimtadd/termio/logger"
	"github.com/stretchr/testify/require"
)

func newTestTermio(cols, rows int) *termio.TerminalIO {
	return termio.NewTerminalIO(termio.Options{
		Rows:   rows,
		Cols:   cols,
		Logger: logger.DefaultLogger,
	})
}

func runTranscript(t *testing.T, ti *termio.TerminalIO, transcript string) string {
	t.Helper()
	require.NoError(t, ti.ProcessOutput([]byte(transcript)))
	return ti.DumpString()
}
