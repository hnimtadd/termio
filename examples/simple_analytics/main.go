package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
	"github.com/hnimtadd/termio"
	"github.com/hnimtadd/termio/logger"
)

// SimpleAnalyticsCollector demonstrates termio callbacks with basic event counting
type SimpleAnalyticsCollector struct {
	termio    *termio.TerminalIO
	session   *SessionData
	startTime time.Time
}

// SessionData represents basic session analytics
type SessionData struct {
	ID           string        `json:"id"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time,omitempty"`
	Duration     time.Duration `json:"duration"`
	EventCounts  EventCounts   `json:"event_counts"`
	CharStats    CharStats     `json:"char_stats"`
}

// EventCounts tracks different types of terminal events
type EventCounts struct {
	Characters     int `json:"characters"`
	CarriageReturn int `json:"carriage_return"`
	LineFeed       int `json:"line_feed"`
	CursorMoves    int `json:"cursor_moves"`
	ColorChanges   int `json:"color_changes"`
	ModeChanges    int `json:"mode_changes"`
	Total          int `json:"total"`
}

// CharStats provides character-level statistics
type CharStats struct {
	PrintableChars   int     `json:"printable_chars"`
	ControlChars     int     `json:"control_chars"`
	AlphaChars       int     `json:"alpha_chars"`
	NumericChars     int     `json:"numeric_chars"`
	SpecialChars     int     `json:"special_chars"`
	TotalChars       int     `json:"total_chars"`
	AvgCharsPerEvent float64 `json:"avg_chars_per_event"`
}

func NewSimpleAnalyticsCollector() *SimpleAnalyticsCollector {
	logger := logger.New(logger.Options{})
	termioInstance := termio.NewTerminalIO(termio.Options{
		Rows:   24,
		Cols:   80,
		Logger: logger,
	})

	collector := &SimpleAnalyticsCollector{
		termio:    termioInstance,
		startTime: time.Now(),
		session: &SessionData{
			ID:        fmt.Sprintf("session_%d", time.Now().Unix()),
			StartTime: time.Now(),
		},
	}

	// Register callbacks for different event types
	termioInstance.RegisterCallback(termio.EventTypeCharacter, collector.onCharacterEvent)
	termioInstance.RegisterCallback(termio.EventTypeCarriageReturn, collector.onCarriageReturnEvent)
	termioInstance.RegisterCallback(termio.EventTypeLineFeed, collector.onLineFeedEvent)
	termioInstance.RegisterCallback(termio.EventTypeCursorMove, collector.onCursorMoveEvent)
	termioInstance.RegisterCallback(termio.EventTypeSGR, collector.onColorEvent)
	termioInstance.RegisterCallback(termio.EventTypeMode, collector.onModeEvent)

	return collector
}

func (sac *SimpleAnalyticsCollector) onCharacterEvent(event *termio.Event) {
	sac.session.EventCounts.Characters++
	sac.session.EventCounts.Total++
	
	if charEvent, ok := event.Data.(termio.CharacterEvent); ok {
		sac.analyzeCharacter(charEvent.Char)
	}
}

func (sac *SimpleAnalyticsCollector) onCarriageReturnEvent(event *termio.Event) {
	sac.session.EventCounts.CarriageReturn++
	sac.session.EventCounts.Total++
}

func (sac *SimpleAnalyticsCollector) onLineFeedEvent(event *termio.Event) {
	sac.session.EventCounts.LineFeed++
	sac.session.EventCounts.Total++
}

func (sac *SimpleAnalyticsCollector) onCursorMoveEvent(event *termio.Event) {
	sac.session.EventCounts.CursorMoves++
	sac.session.EventCounts.Total++
}

func (sac *SimpleAnalyticsCollector) onColorEvent(event *termio.Event) {
	sac.session.EventCounts.ColorChanges++
	sac.session.EventCounts.Total++
}

func (sac *SimpleAnalyticsCollector) onModeEvent(event *termio.Event) {
	sac.session.EventCounts.ModeChanges++
	sac.session.EventCounts.Total++
}

func (sac *SimpleAnalyticsCollector) analyzeCharacter(char rune) {
	sac.session.CharStats.TotalChars++
	
	if char >= 32 && char <= 126 {
		sac.session.CharStats.PrintableChars++
		
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			sac.session.CharStats.AlphaChars++
		} else if char >= '0' && char <= '9' {
			sac.session.CharStats.NumericChars++
		} else {
			sac.session.CharStats.SpecialChars++
		}
	} else {
		sac.session.CharStats.ControlChars++
	}
}

func (sac *SimpleAnalyticsCollector) Run() error {
	fmt.Println("Terminal Event Analytics")
	fmt.Println("========================")
	fmt.Println("Tracking characters and events with termio callbacks")
	fmt.Println("Type commands and watch the real-time event counting")
	fmt.Println("Press Ctrl+C or type 'exit' when done")
	fmt.Println("")

	// Start shell in PTY
	cmd := exec.Command("zsh")
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start PTY: %v", err)
	}
	defer ptyFile.Close()

	// Forward stdin to PTY
	go func() {
		io.Copy(ptyFile, os.Stdin)
	}()

	// Process PTY output through termio (which will trigger callbacks)
	buf := make([]byte, 4096)
	for {
		n, err := ptyFile.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("PTY read error: %v", err)
		}

		// Process through termio engine (this triggers all callbacks)
		sac.termio.ProcessOutput(buf[:n])
		
		// Write to stdout for user interaction
		os.Stdout.Write(buf[:n])
	}

	// Finalize session
	sac.session.EndTime = time.Now()
	sac.session.Duration = sac.session.EndTime.Sub(sac.session.StartTime)
	sac.calculateStats()
	sac.generateReport()
	sac.saveSession()

	return nil
}

func (sac *SimpleAnalyticsCollector) calculateStats() {
	if sac.session.EventCounts.Total > 0 {
		sac.session.CharStats.AvgCharsPerEvent = float64(sac.session.CharStats.TotalChars) / float64(sac.session.EventCounts.Total)
	}
}

func (sac *SimpleAnalyticsCollector) generateReport() {
	fmt.Printf("\n\n")
	fmt.Println("TERMINAL EVENT ANALYTICS")
	fmt.Println("========================")
	
	fmt.Printf("Session Duration: %v\n", sac.session.Duration.Truncate(time.Second))
	fmt.Printf("Total Events: %d\n", sac.session.EventCounts.Total)
	fmt.Println("")
	
	fmt.Println("EVENT BREAKDOWN")
	fmt.Println("---------------")
	fmt.Printf("Characters: %d\n", sac.session.EventCounts.Characters)
	fmt.Printf("Carriage Returns: %d\n", sac.session.EventCounts.CarriageReturn)
	fmt.Printf("Line Feeds: %d\n", sac.session.EventCounts.LineFeed)
	fmt.Printf("Cursor Moves: %d\n", sac.session.EventCounts.CursorMoves)
	fmt.Printf("Color Changes: %d\n", sac.session.EventCounts.ColorChanges)
	fmt.Printf("Mode Changes: %d\n", sac.session.EventCounts.ModeChanges)
	fmt.Println("")
	
	fmt.Println("CHARACTER ANALYSIS")
	fmt.Println("------------------")
	fmt.Printf("Total Characters: %d\n", sac.session.CharStats.TotalChars)
	fmt.Printf("Printable: %d (%.1f%%)\n", 
		sac.session.CharStats.PrintableChars,
		float64(sac.session.CharStats.PrintableChars)/float64(sac.session.CharStats.TotalChars)*100)
	fmt.Printf("Control Chars: %d (%.1f%%)\n", 
		sac.session.CharStats.ControlChars,
		float64(sac.session.CharStats.ControlChars)/float64(sac.session.CharStats.TotalChars)*100)
	fmt.Printf("Alphabetic: %d\n", sac.session.CharStats.AlphaChars)
	fmt.Printf("Numeric: %d\n", sac.session.CharStats.NumericChars)
	fmt.Printf("Special: %d\n", sac.session.CharStats.SpecialChars)
	fmt.Printf("Avg Chars per Event: %.1f\n", sac.session.CharStats.AvgCharsPerEvent)
	
	fmt.Println("")
	fmt.Println("PERFORMANCE METRICS")
	fmt.Println("-------------------")
	eventsPerSecond := float64(sac.session.EventCounts.Total) / sac.session.Duration.Seconds()
	charsPerSecond := float64(sac.session.CharStats.TotalChars) / sac.session.Duration.Seconds()
	fmt.Printf("Events per second: %.1f\n", eventsPerSecond)
	fmt.Printf("Characters per second: %.1f\n", charsPerSecond)
}

func (sac *SimpleAnalyticsCollector) saveSession() {
	// Save detailed session data
	file, err := os.Create("event_analytics.json")
	if err != nil {
		fmt.Printf("Warning: Could not save session: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(sac.session)
	
	fmt.Println("")
	fmt.Printf("Session data saved to: event_analytics.json\n")
	fmt.Printf("Powered by termio event callbacks!\n")
}

func main() {
	collector := NewSimpleAnalyticsCollector()
	if err := collector.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}