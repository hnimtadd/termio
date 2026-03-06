# TermIO - Go Terminal Emulator Engine

A high-performance terminal emulator engine written in Go that provides VT100/xterm compatibility with modern ANSI formatting support. Perfect for building terminal applications, web-based terminals, and terminal automation tools.

## Overview

TermIO is a terminal emulation library that processes escape sequences, manages screen state, and provides an abstraction layer for terminal operations. It's designed to be embedded in terminal applications, emulators, or any software that needs to process and render terminal output.

## Key Features

### Core Terminal Emulation
- **VT100/xterm compatibility** - Full support for standard terminal escape sequences
- **ANSI Formatting** - Rich text with colors, bold, italic, underline, and more
- **Complete terminal state management** - Cursor position, scrolling regions, modes
- **Screen buffer management** - Efficient rendering with dirty tracking
- **Multi-cell character support** - Full Unicode and wide character (CJK) support

### Advanced Features
- **Scrollback buffer** - Historical terminal output preservation
- **Semantic prompts** - Shell integration support (OSC 133 sequences)
- **Terminal modes** - Wrap-around, insert, origin, bracketed paste, and other VT modes
- **High Performance** - Efficient stream processing with minimal memory overhead
- **PTY Integration** - Seamless pseudo-terminal support for shell interaction

### Stream Processing
- **Escape sequence parsing** - Handles CSI, ESC, DCS, OSC sequences
- **UTF-8 stream processing** - Proper Unicode handling in byte streams
- **Error recovery** - Robust parsing with panic recovery

## Installation & Local Setup

### Prerequisites

- Go 1.19 or later
- Unix-like system (Linux, macOS, WSL)
- A terminal that supports ANSI escape codes

### Quick Setup

```bash
# Clone the repository
git clone https://github.com/hnimtadd/termio.git
cd termio

# Install dependencies
go mod download

# Build the project
go build .

# Run the interactive terminal example
cd examples/terminal
go run main.go
```

### Package Installation

```bash
go get github.com/hnimtadd/termio
```

### Interactive Demo

For a guided demonstration of TermIO's capabilities:

```bash
./demo.sh
```

This will build the project and launch an interactive terminal where you can experience all the features firsthand.

## Live Demo

### Interactive Terminal

```bash
cd examples/terminal
go run main.go
```

**What you'll experience:**
- Full zsh shell with syntax highlighting and colors
- Proper cursor positioning and line editing
- Command completion (use `Ctrl+D` for completion lists)
- ANSI formatting preservation

**Example output:**
```
[34mtermio/examples/terminal[0m [90mmain[0m
[35m❯[0m ls [K[65Cat [1;33m14:30:45[0m
[1;38;2;234;154;151mlogs[0m     [1;38;2;246;193;119mmain.go[0m
[38;2;224;222;244mdebug.log[0m [1;38;2;156;207;216mtest_tab.sh[0m
```

### 6. **Event-Driven Terminal Analytics with Callbacks**

Use TermIO's new callback system to register for specific terminal events and build precise analytics:

```go
// Register callbacks for different event types
termio := termio.NewTerminalIO(termio.Options{Rows: 24, Cols: 80})

// Character input tracking
termio.RegisterCallback(termio.EventTypeCharacter, func(event *termio.Event) {
    if charEvent, ok := event.Data.(termio.CharacterEvent); ok {
        fmt.Printf("Character '%c' at position (%d,%d)\n",
            charEvent.Char, charEvent.Position.X, charEvent.Position.Y)
    }
})

// Command detection using cursor and line feed events
termio.RegisterCallback(termio.EventTypeCarriageReturn, func(event *termio.Event) {
    // Detect command execution patterns
    analyzeCommandExecution()
})

// Color/formatting detection
termio.RegisterCallback(termio.EventTypeSGR, func(event *termio.Event) {
    if sgrEvent, ok := event.Data.(termio.SGREvent); ok {
        // Track ANSI color usage, bold/italic formatting, etc.
        trackFormattingUsage(sgrEvent.Attribute)
    }
})

// Cursor movement tracking
termio.RegisterCallback(termio.EventTypeCursorMove, func(event *termio.Event) {
    if moveEvent, ok := event.Data.(termio.CursorMoveEvent); ok {
        // Analyze cursor patterns, screen navigation efficiency
        analyzeCursorMovement(moveEvent.FromX, moveEvent.FromY,
                              moveEvent.ToX, moveEvent.ToY)
    }
})

// Built-in command detector
detector := termio.NewCommandDetector(
    func(cmd string) { fmt.Printf("Command started: %s\n", cmd) },
    func(cmd string, dur time.Duration) {
        fmt.Printf("Command '%s' took %v\n", cmd, dur)
    },
)
detector.RegisterWithTermIO(termio)
```

**Available Event Types:**
- `EventTypeCharacter` - Individual character input with position
- `EventTypeCarriageReturn` / `EventTypeLineFeed` - Line control events
- `EventTypeCursorMove` - Cursor positioning and movement
- `EventTypeSGR` - ANSI color/style formatting events
- `EventTypeMode` - Terminal mode changes
- `EventTypeCSI`, `EventTypeESC`, `EventTypeDCS`, `EventTypeOSC` - Raw escape sequences

**Benefits:**
- **Precise Event Detection** - No more pattern matching or output parsing
- **Real-time Analytics** - Get events as they happen, not after processing
- **Granular Control** - Register only for events you care about
- **Built-in Command Detection** - Ready-to-use command start/end detection
- **Performance** - Callbacks are called directly from the terminal engine
- **Extensible** - Easy to build complex analytics on top of basic events

**Live Demo:**
```bash
cd examples/simple_callbacks
go run main.go

cd examples/callback_analytics
go run main.go
# Interactive session with real-time command detection
```

### 5. **Terminal Usage Analytics & Session Intelligence**

Leverage the TermIO engine to capture comprehensive usage statistics and behavioral insights:

```go
// TerminalAnalytics provides detailed insights into shell usage patterns
type TerminalAnalytics struct {
    termio    *termio.TerminalIO
    session   *Session
    commands  map[string]*CommandStats
}

func (ta *TerminalAnalytics) Run() error {
    // Start shell in PTY with analytics wrapper
    cmd := exec.Command("zsh")
    ptyFile, err := pty.Start(cmd)
    if err != nil {
        return err
    }
    defer ptyFile.Close()

    // Process all terminal output through termio engine
    buf := make([]byte, 4096)
    for {
        n, err := ptyFile.Read(buf)
        if err != nil {
            break
        }

        // Engine processes escape sequences and maintains terminal state
        ta.termio.ProcessOutput(buf[:n])

        // Extract commands and analyze patterns from processed output
        ta.analyzeCommandExecution(buf[:n])

        os.Stdout.Write(buf[:n]) // Forward to user
    }

    // Generate comprehensive analytics report
    ta.generateDetailedReport()
    return nil
}

// Example analytics generated:
// - Command frequency and execution times
// - Output pattern analysis (colors, errors, line counts)
// - Productivity metrics and efficiency scores
// - Activity patterns by hour/day
// - Error rate analysis and suggestions
// - Session replay data for debugging
```

**Benefits:**
- **Behavioral Analytics** - Understand terminal usage patterns and productivity
- **Performance Metrics** - Track command execution times and efficiency
- **Error Analysis** - Identify common mistakes and improvement areas
- **Session Recording** - Full session playback with command timing
- **Multi-format Export** - JSON, CSV, and replay data for analysis
- **Productivity Insights** - Actionable recommendations for workflow improvement

**Live Demo:**
```bash
cd examples/analytics
go run main.go
# Use your terminal normally, then exit to see analytics
# Files generated: terminal_analytics.json, .csv, session_replay.json
```

## Use Cases & Benefits

### 1. **Web-Based Terminal Applications**

Build browser-based terminal interfaces with full terminal compatibility:

```go
package main

import (
    "github.com/hnimtadd/termio"
    "github.com/hnimtadd/termio/logger"
)

// WebTerminal handles browser-based terminal sessions
type WebTerminal struct {
    termio *termio.TerminalIO
    pty    *os.File
}

func (wt *WebTerminal) ProcessPTYOutput(data []byte) string {
    // Process PTY output through termio engine
    wt.termio.ProcessOutput(data)

    // Return ANSI-formatted content for browser
    return wt.termio.DumpStringWithFormatting()
}

func (wt *WebTerminal) HandleWebSocketInput(input []byte) {
    // Forward browser input to PTY
    wt.pty.Write(input)
}
```

**Benefits:**
- Server-side terminal processing
- Perfect ANSI formatting preservation for browsers
- Handle complex terminal apps (vim, tmux, htop)
- Maintain terminal state across websocket reconnections
- Real-time collaborative terminal sessions

### 2. **Terminal Recording & Playback Systems**

Create high-fidelity terminal session recordings:

```go
type TerminalRecorder struct {
    termio *termio.TerminalIO
    frames []TerminalFrame
}

type TerminalFrame struct {
    Timestamp time.Time
    Content   string
    CursorX   int
    CursorY   int
}

func (r *TerminalRecorder) Record(ptyData []byte) {
    r.termio.ProcessOutput(ptyData)

    frame := TerminalFrame{
        Timestamp: time.Now(),
        Content:   r.termio.DumpStringWithFormatting(),
        CursorX:   int(r.termio.GetCursor().X),
        CursorY:   int(r.termio.GetCursor().Y),
    }
    r.frames = append(r.frames, frame)
}

func (r *TerminalRecorder) ExportToHTML() string {
    // Convert recorded frames to interactive HTML player
    return generateHTMLPlayer(r.frames)
}
```

**Benefits:**
- Capture exact terminal state with timing
- Perfect for demos, tutorials, and debugging sessions
- Export to multiple formats (HTML, SVG, GIF)
- Interactive playback with pause/rewind controls
- Compress recordings while preserving visual fidelity

### 3. **Terminal Application Testing & Automation**

Programmatically test terminal-based applications:

```go
func TestVimColorScheme(t *testing.T) {
    // Setup terminal emulator
    termio := termio.NewTerminalIO(termio.Options{
        Rows: 24, Cols: 80,
    })

    // Simulate vim startup
    vimOutput := startVimInPTY()
    termio.ProcessOutput(vimOutput)

    // Verify syntax highlighting
    content := termio.DumpStringWithFormatting()
    assert.Contains(t, content, "\033[32m") // Green for comments
    assert.Contains(t, content, "\033[34m") // Blue for keywords

    // Test cursor positioning after command
    sendVimCommand(":set number\n")
    cursor := termio.GetCursor()
    assert.Equal(t, 0, cursor.X) // Cursor at line start
}

func TestShellPromptCustomization(t *testing.T) {
    termio := termio.NewTerminalIO(termio.Options{Rows: 24, Cols: 80})

    // Test different shell prompts
    testCases := []struct{
        prompt string
        expectedColors []string
    }{
        {"[user@host ~]$ ", []string{"\033[34m", "\033[32m"}},
        {"❯ ", []string{"\033[35m"}},
    }

    for _, tc := range testCases {
        termio.ProcessOutput([]byte(tc.prompt))
        content := termio.DumpStringWithFormatting()

        for _, color := range tc.expectedColors {
            assert.Contains(t, content, color)
        }
    }
}
```

**Benefits:**
- Automated testing of terminal UI/UX
- Regression testing for ANSI formatting
- Verify cursor positioning and screen layouts
- Test terminal application compatibility
- Performance benchmarking of terminal apps

### 4. **Custom Terminal Emulators**

Build specialized terminal emulators with enhanced features:

```go
// IDETerminal extends basic terminal with development features
type IDETerminal struct {
    termio          *termio.TerminalIO
    syntaxHighlight bool
    errorHighlight  bool
    autoComplete    bool
}

func (ide *IDETerminal) ProcessWithEnhancements(data []byte) {
    // Process through termio engine first
    ide.termio.ProcessOutput(data)

    if ide.syntaxHighlight {
        ide.applySyntaxHighlighting()
    }

    if ide.errorHighlight {
        ide.highlightErrors()
    }
}

func (ide *IDETerminal) applySyntaxHighlighting() {
    content := ide.termio.DumpString()

    // Enhance with additional syntax highlighting
    enhanced := addLanguageSpecificHighlighting(content)

    // Update terminal with enhanced content
    ide.updateDisplay(enhanced)
}
```

**Benefits:**
- Build IDE-integrated terminals
- Add custom syntax highlighting layers
- Implement smart auto-completion
- Create domain-specific terminal interfaces
- Embed terminals in desktop/mobile applications

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/hnimtadd/termio"
    "github.com/hnimtadd/termio/logger"
)

func main() {
    // Create a logger (optional)
    logger := logger.New(logger.Options{})

    // Create terminal with 80x24 dimensions
    term := termio.NewTerminalIO(termio.Options{
        Rows:   24,
        Cols:   80,
        Logger: logger,
    })

    // Process terminal output byte by byte
    term.Process('H')
    term.Process('e')
    term.Process('l')
    term.Process('l')
    term.Process('o')

    // Or process entire byte slices for better performance
    term.ProcessOutput([]byte("World!"))

    // Get current terminal content as plain text
    content := term.DumpString()
    fmt.Println(content)
}
```

## Usage with PTY

```go
package main

import (
    "io"
    "os"
    "os/exec"

    "github.com/creack/pty"
    "github.com/hnimtadd/termio"
    "github.com/hnimtadd/termio/logger"
)

func main() {
    // Start a shell in a PTY
    cmd := exec.Command("bash")
    ptyFile, err := pty.Start(cmd)
    if err != nil {
        panic(err)
    }
    defer ptyFile.Close()

    // Create terminal emulator
    logger := logger.New(logger.Options{})
    term := termio.NewTerminalIO(termio.Options{
        Rows:   24,
        Cols:   80,
        Logger: logger,
    })

    // Forward stdin to PTY
    go func() { io.Copy(ptyFile, os.Stdin) }()

    // Process PTY output
    buf := make([]byte, 4096)
    for {
        n, err := ptyFile.Read(buf)
        if err != nil {
            break
        }

        // Process the output through terminal emulator
        term.ProcessOutput(buf[:n])

        // Get rendered terminal content
        content := term.DumpString()

        // Clear screen and display
        fmt.Print("\033[H\033[2J")
        fmt.Print(content)
    }
}
```

## API Reference

### Main Types

#### `TerminalIO`
The main terminal emulator instance.

```go
type TerminalIO struct {
    // Internal terminal state and stream parser
}

// Create new terminal emulator
func NewTerminalIO(opts Options) *TerminalIO

// Process single byte
func (t *TerminalIO) Process(c byte) error

// Process byte slice (recommended for performance)
func (t *TerminalIO) ProcessOutput(buf []byte) error

// Get terminal content as plain text
func (t *TerminalIO) DumpString() string

// Resize terminal
func (t *TerminalIO) Resize(cols, rows int)
```

#### `Options`
Configuration for terminal emulator.

```go
type Options struct {
    Rows   int           // Terminal height in rows
    Cols   int           // Terminal width in columns
    Logger logger.Logger // Optional logger instance
}
```

### Terminal Features

#### Supported Escape Sequences
- **CSI (Control Sequence Introducer)** - Cursor movement, erasing, scrolling
- **ESC (Escape)** - Single character sequences
- **DCS (Device Control String)** - Device-specific commands
- **OSC (Operating System Command)** - Window title, semantic prompts

#### Terminal Modes
- **Wrap-around mode** - Automatic line wrapping
- **Insert mode** - Character insertion vs replacement
- **Origin mode** - Cursor positioning relative to margins
- **Line feed mode** - LF behavior (with/without CR)

#### Character Support
- **ASCII and Unicode** - Full UTF-8 support
- **Wide characters** - CJK and other double-width characters
- **Zero-width characters** - Combining characters (limited support)

## Architecture

The library is structured in several key packages:

- **`terminal/`** - Core terminal emulation logic
- **`terminal/parser/`** - Escape sequence parsing state machine
- **`terminal/screen/`** - Screen buffer and cursor management
- **`terminal/page/`** - Memory-efficient page-based storage
- **`logger/`** - Logging infrastructure
- **`io/`** - I/O utilities

## Development Environment

This project uses Nix for reproducible development environments:

```bash
# Enter development shell
nix develop

# Or use legacy nix-shell
nix-shell
```

## Troubleshooting

### Common Issues

1. **Tab completion not working**:
   - Use `Ctrl+D` for completion lists (standard zsh completion)
   - fzf-completion bound to Tab may not work properly yet
   - Check `bindkey | grep "^I"` to see Tab key binding

2. **Colors not displaying**:
   - Ensure terminal supports ANSI colors
   - Check `TERM` environment variable (should be `xterm-256color`)
   - Verify terminal emulator supports true color

3. **Cursor positioning issues**:
   - Verify terminal size matches termio configuration
   - Check for conflicting escape sequences
   - Ensure proper PTY setup and dimensions

4. **Performance issues**:
   - Use `ProcessOutput()` for batch processing instead of `Process()`
   - Consider reducing logging verbosity in production
   - Monitor memory usage with large scrollback buffers

### Debug Logging

Enable detailed logging to troubleshoot issues:

```go
logger := logger.New(logger.Options{
    Buffer: os.Stderr,  // or log file
})

termio := termio.NewTerminalIO(termio.Options{
    Rows: 24, Cols: 80,
    Logger: logger,
})
```

Check logs in `examples/terminal/logs/termio.log` for detailed debugging information.

## Testing

```bash
go test ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

MIT license

## Acknowledgments

This library implements terminal emulation following VT100/xterm specifications and draws inspiration from various terminal emulator implementations.
