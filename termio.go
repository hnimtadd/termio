package termio

import (
	"fmt"
	"runtime/debug"

	"github.com/hnimtadd/termio/logger"
	terminalPkg "github.com/hnimtadd/termio/terminal"
	"github.com/hnimtadd/termio/terminal/core"
	"github.com/hnimtadd/termio/terminal/size"
	"github.com/hnimtadd/termio/terminal/stream"
)

type TerminalIO struct {
	// The terminal emulator internal state. This is the abstract "terminal"
	// that manages input, grid updating, etc. and is renderer-agnostic. It
	// just stores internal state about a grid.
	terminal *terminalPkg.Terminal

	// The stream parser. This parses the stream of escape codes and so on
	// from the child process and calls callbacks in the stream handler.
	terminalStream *stream.Stream

	logger logger.Logger
}

type Options struct {
	Rows, Cols int
	Logger     logger.Logger
}

// Initialize the termio state.
//
// This will also start the child process if the termio is configured
// to run a child process.
func NewTerminalIO(opts Options) *TerminalIO {
	// default terminal Mode
	modes := core.ModePacked

	// Create a new terminal instance
	term := terminalPkg.NewTerminal(
		terminalPkg.Options{
			Rows:   opts.Rows,
			Cols:   opts.Cols,
			Modes:  modes,
			Logger: opts.Logger,
		},
	)

	// Create our stream handler.
	handler := &StreamHandler{
		terminal: term,
		logger:   opts.Logger,
	}
	return &TerminalIO{
		terminal: term,
		terminalStream: stream.NewStream(
			handler,
			opts.Logger,
		),
	}
}

// resize the terminal
func (t *TerminalIO) Resize(cols, rows int) {
	t.terminal.Resize(size.CellCountInt(cols), size.CellCountInt(rows))
}

// proces output from the pty. This is the manual API that users can call
// with pty data
func (t *TerminalIO) ProcessOutput(buf []byte) (err error) {
	// Process the output from the pty
	defer func() {
		if r := recover(); r != nil {
			t.logger.Error("Panic in ProcessOutput: %v", r)
			fmt.Println(string(debug.Stack()))
			err = fmt.Errorf("panic in ProcessOutput: %v", r)
		}
	}()
	t.terminalStream.NextSlice(buf)
	err = nil
	return
}

// Process output from pty by byte. This is the manual API that users can call
// with pty data
//
// NOTE, this implementation is helpful for debugging as you can see the
// process of each byte, but it is not as efficient as the slice version.
//
// consider ProcessOutput for better performance
func (t *TerminalIO) Process(c byte) (err error) {
	// Process the output from the pty
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		logging.Error("Panic in Process: %v", r)
	// 		fmt.Println(string(debug.Stack()))
	// 		err = fmt.Errorf("panic in Process: %v", r)
	// 	}
	// }()
	t.terminalStream.Next(c)
	err = nil
	return
}

func (t *TerminalIO) DumpString() string {
	return t.terminal.PlainString()
}

func (t *TerminalIO) Write(p []byte) (n int, err error) {
	t.terminalStream.NextSlice(p)
	return len(p), nil
}

func (t *TerminalIO) Close() error {
	panic("unimplemented")
}
