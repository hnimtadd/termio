package parser

// State for the state machine
type State int

const (
	StateGround State = iota
	StateEscape
	StateEscapeIntermediate
	StateCSIEntry
	StateCSIParam
	StateCSIIntermediate
	StateCsiIgnore
	StateDCSEntry
	StateDCSParam
	StateDCSIntermediate
	StateDCSPassthrough
	StateDCSIgnore
	StateOSCString
	StateSosPmApcString
)
