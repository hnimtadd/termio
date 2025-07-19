package parser

import (
	"fmt"
	"strings"

	"github.com/hnimtadd/termio/terminal/sequences/csi"
	"github.com/hnimtadd/termio/terminal/sequences/dcs"
	"github.com/hnimtadd/termio/terminal/sequences/esc"
	"github.com/hnimtadd/termio/terminal/sequences/osc"
)

// ActionType is an action that taked when event or
// state transition occurs
type ActionType int

const (
	ActionNone ActionType = iota
	ActionIgnore
	ActionPrint
	ActionExecute
	ActionCollect
	ActionParam
	ActionESCDispatch
	ActionCSIDispatch
	ActionDCSHook
	ActionDCSPut
	ActionDCSUnHook
	ActionOSCStart
	ActionOSCPut
	ActionOSCEnd
)

func (a ActionType) String() string {
	switch a {
	case ActionNone:
		return "None"
	case ActionIgnore:
		return "Ignore"
	case ActionPrint:
		return "Print"
	case ActionExecute:
		return "Execute"
	case ActionCollect:
		return "Collect"
	case ActionParam:
		return "Param"
	case ActionESCDispatch:
		return "ESCDispatch"
	case ActionCSIDispatch:
		return "CSIDispatch"
	case ActionDCSHook:
		return "DCSHook"
	case ActionDCSPut:
		return "DCSPut"
	case ActionDCSUnHook:
		return "DCSUnHook"
	case ActionOSCStart:
		return "OSCStart"
	case ActionOSCPut:
		return "OSCPut"
	case ActionOSCEnd:
		return "OSCEnd"
	default:
		return "Unknown"
	}
}

// Action is the action that a caller of the parser is expected to
// take as a result of some input character
type Action struct {
	Type ActionType

	// Draw character to the screen. This is a unicode codepoint.
	PrintData uint8

	// ExecuteData the C0 or C1 function.
	ExecuteData uint8

	// execute the CSI command.
	CSIDispatchData *csi.Command

	// execute the ECS command.
	ESCDispatchData *esc.Command

	// execute the OSC command.
	OSCDispatchData *osc.Command

	// DCS-related events
	DCSHookData *dcs.DCS
	DCSPutData  uint8

	// APC data
	APCPutData uint8
}

func (a *Action) String() string {
	if a == nil {
		return "{nil}"
	}
	builder := new(strings.Builder)
	fmt.Fprintf(builder, "{ .%s = ", a.Type.String())
	switch a.Type {
	case ActionPrint:
		fmt.Fprintf(builder, "0x%x", a.PrintData)
	case ActionExecute:
		fmt.Fprintf(builder, "0x%x", a.ExecuteData)
	case ActionCSIDispatch:
		if a.CSIDispatchData != nil {
			fmt.Fprintf(builder, "%s", a.CSIDispatchData.String())
		} else {
			fmt.Fprintf(builder, "nil")
		}
	case ActionESCDispatch:
		if a.ESCDispatchData != nil {
			fmt.Fprintf(builder, "%s", a.ESCDispatchData.String())
		} else {
			fmt.Fprintf(builder, "nil")
		}
	case ActionOSCStart:
		if a.OSCDispatchData != nil {
			fmt.Fprintf(builder, "osc")
		} else {
			fmt.Fprintf(builder, "nil")
		}
	case ActionDCSHook:
		if a.DCSHookData != nil {
			fmt.Fprintf(builder, "%s", a.DCSHookData.String())
		} else {
			fmt.Fprintf(builder, "nil")
		}
	case ActionDCSPut:
		if a.DCSPutData != 0 {
			fmt.Fprintf(builder, "0x%x", a.DCSPutData)
		} else {
			fmt.Fprintf(builder, "nil")
		}
	}
	fmt.Fprintf(builder, "}")
	return builder.String()
}
