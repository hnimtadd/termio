package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/creack/pty"
	"github.com/hnimtadd/termio"
	"github.com/hnimtadd/termio/logger"
	"golang.org/x/term"
)

func main() {
	// Parse command-line flags
	shell := flag.String("shell", "zsh", "Shell to spawn (bash or zsh)")
	debug := flag.Bool("debug", false, "Enable debug mode to log escape sequences")
	flag.Parse()

	// Setup raw mode immediately and defer restore to ensure it's always reverted
	restore, err := setupRawMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup raw mode: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := restore(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to restore terminal: %v\n", err)
		}
	}()

	// Get current terminal size, fallback to 80x24 if detection fails
	cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil || cols == 0 || rows == 0 {
		cols = 80
		rows = 24
	}

	// Setup debug logging if enabled
	var debugLog *os.File
	if *debug {
		// Create logs directory if it doesn't exist
		logDir := filepath.Join(os.TempDir(), "termio-debug")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create debug log directory: %v\n", err)
			os.Exit(1)
		}

		// Create debug log file
		debugLogPath := filepath.Join(logDir, "emulator-debug.log")
		debugLog, err = os.Create(debugLogPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create debug log file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(debugLog, "Starting terminal emulator debug log - %dx%d terminal\n", cols, rows)
	}

	// Create and start the shell process with PTY
	cmd := exec.Command(*shell)
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start PTY with %s: %v\n", *shell, err)
		os.Exit(1)
	}
	defer ptyFile.Close()

	// Initialize termio with logger
	logFile, err := os.Create(filepath.Join(os.TempDir(), "termio.log"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create termio log: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	termioLogger := logger.New(logger.Options{Buffer: logFile})
	termioInstance := termio.NewTerminalIO(
		termio.Options{
			Rows:   rows,
			Cols:   cols,
			Logger: termioLogger,
		})
	defer termioInstance.Close()

	// Create channel for input goroutine completion and resize handler shutdown
	inputDone := make(chan struct{})
	resizeDone := make(chan struct{})

	// Start goroutine to handle PTY resize events
	go handleResize(ptyFile, termioInstance, resizeDone)

	// Start goroutine to copy stdin to PTY
	go copyInputToPTY(ptyFile, inputDone)

	// Process PTY output (main loop)
	if err := processPTYOutput(ptyFile, termioInstance, *debug, debugLog); err != nil {
		fmt.Fprintf(os.Stderr, "Error processing PTY output: %v\n", err)
		os.Exit(1)
	}

	// After shell exits (processPTYOutput returns), close stdin to unblock copyInputToPTY
	os.Stdin.Close()

	// Signal resize handler to stop
	close(resizeDone)

	// Wait for input goroutine to finish
	<-inputDone

	// Close debug log file if it was opened
	if debugLog != nil {
		if err := debugLog.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close debug log: %v\n", err)
		}
	}

	// Wait for shell to exit
	cmd.Wait()
}

// setupRawMode configures the terminal to raw mode and returns a cleanup function
func setupRawMode() (restore func() error, err error) {
	fd := int(os.Stdin.Fd())

	// Save current terminal state
	state, err := term.GetState(fd)
	if err != nil {
		return nil, fmt.Errorf("failed to get terminal state: %w", err)
	}

	// Set terminal to raw mode
	_, err = term.MakeRaw(fd)
	if err != nil {
		return nil, fmt.Errorf("failed to set raw mode: %w", err)
	}

	// Return restore function
	restore = func() error {
		return term.Restore(fd, state)
	}

	return restore, nil
}

// handleResize sets up SIGWINCH handler to resize PTY on terminal size changes
func handleResize(ptyFile *os.File, termioInstance *termio.TerminalIO, done <-chan struct{}) {
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	defer signal.Stop(sigwinch)

	for {
		select {
		case <-sigwinch:
			// Get new terminal size
			cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
			if err != nil || cols == 0 || rows == 0 {
				// Skip resize if we can't get size
				continue
			}

			// Resize PTY with proper Winsize struct
			winsize := &pty.Winsize{
				Rows: uint16(rows),
				Cols: uint16(cols),
			}
			if err := pty.Setsize(ptyFile, winsize); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to resize PTY: %v\n", err)
			}

			// Resize termio engine
			termioInstance.Resize(cols, rows)
		case <-done:
			// Exit when signaled
			return
		}
	}
}

// copyInputToPTY forwards stdin to PTY (in goroutine)
func copyInputToPTY(ptyFile *os.File, done chan struct{}) {
	_, _ = io.Copy(ptyFile, os.Stdin)
	done <- struct{}{}
}

// processPTYOutput reads PTY output, processes through termio, writes to stdout
func processPTYOutput(ptyFile *os.File, termioInstance *termio.TerminalIO, debug bool, debugLog *os.File) error {
	// Allocate buffer outside loop for reuse
	buf := make([]byte, 4096)

	for {
		// Read from PTY
		n, err := ptyFile.Read(buf)

		// Handle PTY EOF - shell has exited
		if err == io.EOF {
			return nil
		}

		// Handle other read errors
		if err != nil {
			return fmt.Errorf("failed to read from PTY: %w", err)
		}

		// If debug mode is enabled, log raw bytes before processing
		if debug && debugLog != nil {
			if _, err := debugLog.Write(buf[:n]); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write debug log: %v\n", err)
			}
		}

		// Process through termio and get output to display
		output, err := termioInstance.ProcessForOutput(buf[:n])
		if err != nil {
			return fmt.Errorf("failed to process PTY output: %w", err)
		}

		// Write processed output to stdout
		if _, err := os.Stdout.Write(output); err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}
	}
}
