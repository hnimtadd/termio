package termio

import (
	"bytes"
	"fmt"
	"runtime/debug"

	"github.com/hnimtadd/termio/logger"
	"github.com/hnimtadd/termio/terminal"
	"github.com/hnimtadd/termio/terminal/core"
	"github.com/hnimtadd/termio/terminal/point"
	"github.com/hnimtadd/termio/terminal/size"
	"github.com/hnimtadd/termio/terminal/stream"
)

type TerminalIO struct {
	// The terminal emulator internal state. This is the abstract "terminal"
	// that manages input, grid updating, etc. and is renderer-agnostic. It
	// just stores internal state about a grid.
	terminal *terminal.Terminal

	// The stream parser. This parses the stream of escape codes and so on
	// from the child process and calls callbacks in the stream handler.
	terminalStream *stream.Stream

	// Event manager for handling callbacks
	eventManager *EventManager

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
	term := terminal.NewTerminal(
		terminal.Options{
			Rows:   opts.Rows,
			Cols:   opts.Cols,
			Modes:  modes,
			Logger: opts.Logger,
		},
	)

	// Create our stream handler.
	handler := &StreamHandler{
		terminal:     term,
		logger:       opts.Logger,
		eventManager: NewEventManager(),
	}
	termio := &TerminalIO{
		terminal: term,
		terminalStream: stream.NewStream(
			handler,
			opts.Logger,
		),
		eventManager: handler.eventManager,
		logger:       opts.Logger,
	}
	return termio
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
	defer func() {
		if r := recover(); r != nil {
			t.logger.Error("Panic in Process: %v", r)
			fmt.Println(string(debug.Stack()))
			err = fmt.Errorf("panic in Process: %v", r)
		}
	}()
	t.terminalStream.Next(c)
	err = nil
	return
}

// ProcessForOutput processes PTY input and returns bytes that should be written to stdout
// This is the proper way to handle terminal emulation - process escape sequences 
// and return the current terminal state that should be displayed
func (t *TerminalIO) ProcessForOutput(buf []byte) ([]byte, error) {
	// Process the input through termio to update internal state
	err := t.ProcessOutput(buf)
	if err != nil {
		return nil, err
	}
	
	// For now, return the input as-is but let termio process it internally
	// This maintains the proper terminal state while allowing raw sequences through
	return buf, nil
}

func (t *TerminalIO) DumpString() string {
	return t.terminal.PlainString()
}

func (t *TerminalIO) DumpStringWithCursor() string {
	// Get the plain content
	content := t.terminal.PlainString()
	
	// Get cursor position (1-based for ANSI sequences)
	cursorX := int(t.terminal.Screen.Cursor.X) + 1
	cursorY := int(t.terminal.Screen.Cursor.Y) + 1
	
	// Add cursor positioning after content
	return content + fmt.Sprintf("\033[%d;%dH", cursorY, cursorX)
}

// DumpStringWithFormatting returns the terminal content with ANSI formatting preserved
func (t *TerminalIO) DumpStringWithFormatting() string {
	w := bytes.NewBuffer(nil)
	if err := t.terminal.Screen.DumpStringWithFormatting(w, point.TagViewPort); err != nil {
		return ""
	}
	return w.String()
}

func (t *TerminalIO) Write(p []byte) (n int, err error) {
	t.terminalStream.NextSlice(p)
	return len(p), nil
}

func (t *TerminalIO) Close() error {
	// Currently there are no resources that need explicit cleanup
	// The terminal and stream are managed by Go's garbage collector
	// This method is here for interface compatibility and future resource management
	if t.logger != nil {
		t.logger.Info("TerminalIO closed")
	}
	return nil
}

// RegisterCallback registers a callback for a specific event type
func (t *TerminalIO) RegisterCallback(eventType EventType, callback EventCallback) {
	t.eventManager.RegisterCallback(eventType, callback)
}

// UnregisterCallback removes a callback for a specific event type
func (t *TerminalIO) UnregisterCallback(eventType EventType, callback EventCallback) {
	t.eventManager.UnregisterCallback(eventType, callback)
}

// RegisterAllEvents registers the same callback for all event types
func (t *TerminalIO) RegisterAllEvents(callback EventCallback) {
	t.eventManager.RegisterAllEvents(callback)
}
