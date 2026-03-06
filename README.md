# TermIO

A Go library providing a complete terminal emulator engine for processing terminal output and managing terminal state.

## Overview

TermIO is a terminal emulation library that processes escape sequences, manages screen state, and provides an abstraction layer for terminal operations. It's designed to be embedded in terminal applications, emulators, or any software that needs to process and render terminal output.

## Features

### Core Terminal Emulation
- **VT100/xterm compatibility** - Supports standard terminal escape sequences
- **Complete terminal state management** - Cursor position, scrolling regions, modes
- **Screen buffer management** - Efficient screen rendering with dirty tracking
- **Multi-cell character support** - Full Unicode and wide character (CJK) support

### Advanced Features
- **Scrollback buffer** - Historical terminal output preservation
- **Semantic prompts** - Shell integration support (OSC 133 sequences)
- **Terminal modes** - Wrap-around, insert, origin, and other VT modes
- **Tabstop management** - Configurable tab stops and navigation
- **Scrolling regions** - Partial screen scrolling support

### Stream Processing
- **Escape sequence parsing** - Handles CSI, ESC, DCS, OSC sequences
- **UTF-8 stream processing** - Proper Unicode handling in byte streams
- **Error recovery** - Robust parsing with panic recovery

## Installation

```bash
go get github.com/hnimtadd/termio
```

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

[Add your license information here]

## Acknowledgments

This library implements terminal emulation following VT100/xterm specifications and draws inspiration from various terminal emulator implementations.