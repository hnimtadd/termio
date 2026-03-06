package termio

import (
	"github.com/hnimtadd/termio/terminal/sgr"
)

// EventType represents different types of terminal events
type EventType int

const (
	EventTypeCharacter EventType = iota
	EventTypeCSI
	EventTypeESC
	EventTypeDCS
	EventTypeOSC
	EventTypeSGR
	EventTypeCarriageReturn
	EventTypeLineFeed
	EventTypeCursorMove
	EventTypeErase
	EventTypeMode
	EventTypePrompt
	EventTypeCommandStart
	EventTypeCommandEnd
)

// Event represents a terminal event with its associated data
type Event struct {
	Type EventType
	Data interface{}
}

// Character event data
type CharacterEvent struct {
	Char     rune
	Position struct{ X, Y int }
}

// CSI event data
type CSIEvent struct {
	Function string
	Params   []uint16
}

// ESC event data
type ESCEvent struct {
	Function string
}

// DCS event data
type DCSEvent struct {
	Command string
}

// OSC event data
type OSCEvent struct {
	Command string
	Data    string
}

// SGR event data
type SGREvent struct {
	Attribute *sgr.Attribute
}

// Cursor movement event data
type CursorMoveEvent struct {
	FromX, FromY int
	ToX, ToY     int
	Direction    string // "up", "down", "left", "right", "position"
}

// Erase event data
type EraseEvent struct {
	Type string // "line", "display", "chars"
	Mode interface{}
}

// Mode change event data
type ModeEvent struct {
	ModeName string
	Value    uint16
	Enabled  bool
	ANSI     bool
}

// Prompt detection event data
type PromptEvent struct {
	Content  string
	Position struct{ X, Y int }
	Type     string // "primary", "secondary", "continuation"
}

// Command execution event data
type CommandStartEvent struct {
	Content   string
	Position  struct{ X, Y int }
	Timestamp int64
}

type CommandEndEvent struct {
	Duration  int64 // milliseconds
	ExitCode  int
	Timestamp int64
}

// EventCallback is a function that handles terminal events
type EventCallback func(event *Event)

// EventManager manages event callbacks and dispatching
type EventManager struct {
	callbacks map[EventType][]EventCallback
}

// NewEventManager creates a new event manager
func NewEventManager() *EventManager {
	return &EventManager{
		callbacks: make(map[EventType][]EventCallback),
	}
}

// RegisterCallback registers a callback for a specific event type
func (em *EventManager) RegisterCallback(eventType EventType, callback EventCallback) {
	if em.callbacks[eventType] == nil {
		em.callbacks[eventType] = make([]EventCallback, 0)
	}
	em.callbacks[eventType] = append(em.callbacks[eventType], callback)
}

// UnregisterCallback removes a callback for a specific event type
func (em *EventManager) UnregisterCallback(eventType EventType, callback EventCallback) {
	if callbacks, exists := em.callbacks[eventType]; exists {
		// Note: This is a simple implementation that doesn't handle function pointer comparison
		// In a production system, you might want to use callback IDs or a more sophisticated approach
		for i, cb := range callbacks {
			if &cb == &callback {
				em.callbacks[eventType] = append(callbacks[:i], callbacks[i+1:]...)
				break
			}
		}
	}
}

// EmitEvent dispatches an event to all registered callbacks
func (em *EventManager) EmitEvent(event *Event) {
	if callbacks, exists := em.callbacks[event.Type]; exists {
		for _, callback := range callbacks {
			callback(event)
		}
	}
}

// RegisterAllEvents is a convenience method to register the same callback for all event types
func (em *EventManager) RegisterAllEvents(callback EventCallback) {
	eventTypes := []EventType{
		EventTypeCharacter, EventTypeCSI, EventTypeESC, EventTypeDCS, EventTypeOSC,
		EventTypeSGR, EventTypeCarriageReturn, EventTypeLineFeed, EventTypeCursorMove,
		EventTypeErase, EventTypeMode, EventTypePrompt, EventTypeCommandStart, EventTypeCommandEnd,
	}
	
	for _, eventType := range eventTypes {
		em.RegisterCallback(eventType, callback)
	}
}