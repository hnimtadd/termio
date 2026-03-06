package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/hnimtadd/termio"
	"github.com/hnimtadd/termio/logger"
)

func main() {
	fmt.Println("Hello world!")
	cmd := exec.Command("zsh")
	pty, err := pty.Start(cmd)
	if err != nil {
		fmt.Printf("failed to start pty, err: %v", err)
		os.Exit(1)
	}
	go func() { io.Copy(pty, os.Stdin) }()

	f, err := os.Create("./logs/termio.log")
	if err != nil {
		fmt.Printf("failed to create file, err: %v", err)
		os.Exit(1)
	}
	logger := logger.New(logger.Options{Buffer: f})
	termio := termio.NewTerminalIO(
		termio.Options{
			Rows:   24,
			Cols:   80,
			Logger: logger,
		})
		
	for {
		buf := make([]byte, 4096)
		n, err := pty.Read(buf)
		if err != nil {
			break
		}
		
		// Process input through termio and get output that should be displayed
		output, err := termio.ProcessForOutput(buf[:n])
		if err != nil {
			fmt.Printf("Failed to process pty output: %v\n", err)
			break
		}
		
		// Write the processed output
		os.Stdout.Write(output)
	}
}