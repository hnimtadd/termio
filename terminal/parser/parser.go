package parser

import (
	"github.com/hnimtadd/termio/logger"
	"github.com/hnimtadd/termio/terminal/sequences/csi"
	"github.com/hnimtadd/termio/terminal/sequences/dcs"
	"github.com/hnimtadd/termio/terminal/sequences/esc"
	"github.com/hnimtadd/termio/terminal/sequences/osc"
	"github.com/hnimtadd/termio/terminal/utils"
)

const (
	MaxParams        = 24
	MaxIntermediates = 4
)

// VT-series parser for escape and control sequences.
//
// This is implemented directly as the state machine described on
// vt100.net: https://vt100.net/emu/dec_ansi_parser
type Parser struct {
	State State

	// intermediate tracking
	intermediates    [MaxIntermediates]uint8
	intermediatesIdx int

	// param tracking
	params      [MaxParams]uint16
	paramsIdx   int
	paramsSet   *utils.StaticBitSet
	paramAcc    uint16
	paramAccIdx int

	oscParser *osc.Parser
	table     parserTable

	logger logger.Logger
}

func NewParser() *Parser {
	return &Parser{
		State:            StateGround,
		intermediates:    [MaxIntermediates]uint8{},
		intermediatesIdx: 0,
		params:           [MaxParams]uint16{},
		paramsIdx:        0,
		paramAcc:         0,
		paramAccIdx:      0,
		table:            newParserTable(),
		paramsSet:        utils.NewStaticBitSet(MaxParams),
	}
}

// Next consumes the next character c and returns the actions to execute.
//
// # Up to 3 actions may need to be executed
//
// When going from one state to another state, the actions take place
// in this order
//
// 1. exit action from old state
//
// 2. transition action
//
// 3. entry action to new state
func (p *Parser) Next(c uint8) [3]*Action {
	effect := p.table[c][p.State]

	nextState := effect.state
	action := effect.action

	// after generating the actions, we set our next state
	defer func() {
		p.State = nextState
	}()

	// When going from one state to another state, the actions take place
	// in this order:
	//
	// 1. Exit action from old state
	//
	// 2. Transition action
	//
	// 3. Entry action to new state
	actions := [3]*Action{}

	// Exit action from old state
	{
		var exitAction *Action = nil
		if p.State != nextState {
			switch p.State {
			case StateOSCString:
				// oscEnd
				if cmd := p.oscParser.End(); cmd != nil {
					exitAction = &Action{
						Type:            ActionOSCEnd,
						OSCDispatchData: cmd,
					}
				}
			case StateDCSPassthrough:
				// DCSUnhook
				exitAction = &Action{
					Type: ActionDCSUnHook,
				}
			}
		}
		actions[0] = exitAction
	}

	// transtion action
	{
		actions[1] = p.doAction(action, c)
	}

	// entry action
	{
		var entryAction *Action = nil
		if p.State != nextState {
			switch nextState {
			case StateEscape, StateDCSEntry, StateCSIEntry:
				p.Clear()
			case StateOSCString:
				// entry/osc_start
				p.oscParser.Reset()
			case StateDCSPassthrough:
				// entry/ hook
				// This action is invoked when a final character arrives
				// in the first part of a DCS

				// finalize parameters
				if p.paramsIdx > 0 {
					p.params[p.paramsIdx] = p.paramAcc
					p.paramsIdx += 1
				}
				entryAction = &Action{
					Type: ActionDCSHook,
					DCSHookData: &dcs.DCS{
						Intermediates: p.intermediates[:p.intermediatesIdx],
						Params:        p.params[:p.paramsIdx],
						Final:         c,
					},
				}
				// TODO: handle StateSosPmApcString once we have a use-case
			}
		}
		actions[2] = entryAction
	}

	return actions
}

func (p *Parser) doAction(actionType ActionType, c uint8) (action *Action) {
	switch actionType {
	case ActionIgnore, ActionNone:
		return
	case ActionPrint:
		return &Action{Type: ActionPrint, PrintData: c}
	case ActionExecute:
		return &Action{Type: ActionExecute, ExecuteData: c}
	case ActionCollect:
		p.Collect(c)
		return
	case ActionParam:
		// Semicolon separates parameters. If we encounter a semicolon
		// we need to store and move on to the next parameter.
		if c == ';' || c == ':' {
			// ignore too many parameters
			if p.paramsIdx >= MaxParams {
				return
			}

			// set param final value
			p.params[p.paramsIdx] = p.paramAcc
			if c == ':' {
				p.paramsSet.Set(p.paramsIdx)
			}
			p.paramsIdx += 1

			// reset current params value to 0
			p.paramAcc = 0
			p.paramAccIdx = 0
			return
		}

		// A numeric value. Add it to our accumulator
		if p.paramAccIdx > 0 {
			p.paramAcc *= 10
			p.paramAcc += uint16(c - '0')
		} else {
			p.paramAcc = uint16(c - '0')
		}

		// Increment our accumulator index. If we overflow then
		// we're out of bounds and we exit immediately.
		nextParamsIdx, overflow := utils.AddWithOverflow(p.paramAccIdx, 1)
		if overflow {
			return
		}
		p.paramAccIdx = nextParamsIdx

		// The client is expected to perform no action.
		return
	case ActionESCDispatch:
		return &Action{
			Type: ActionESCDispatch,
			ESCDispatchData: &esc.Command{
				Intermediates: p.intermediates[:p.intermediatesIdx],
				Final:         c,
			},
		}
	case ActionCSIDispatch:
		// Ignore too many parameters
		if p.paramsIdx >= MaxParams {
			return
		}

		// Finalize parameters if we have one
		if p.paramAccIdx > 0 {
			p.params[p.paramsIdx] = p.paramAcc
			p.paramsIdx += 1
		}
		action = &Action{
			Type: ActionCSIDispatch,
			CSIDispatchData: &csi.Command{
				Intermediates: p.intermediates[:p.intermediatesIdx],
				Params:        p.params[:p.paramsIdx],
				ParamsSet:     p.paramsSet,
				Final:         c,
			},
		}

		// We only allow colon or mixed separators for the 'm' command.
		if c != 'm' && p.paramsSet.Count() > 0 {
			p.logger.Warn(
				"CSI colon or mixed separators only allowed for 'm' command",
				"got",
				action,
			)
			return nil
		}
		return
	case ActionDCSPut:
		// dcsPut event inside StateDCSPassthrough
		return &Action{
			Type:       ActionDCSPut,
			DCSPutData: c,
		}
	case ActionOSCPut:
		p.oscParser.Next(c)
		return
	default:
		p.logger.Warn("Unknown action", "type", actionType)
		return nil
	}
}

func (p *Parser) Collect(c uint8) {
	if p.intermediatesIdx > MaxIntermediates {
		p.logger.Warn("Too many intermediates, ignoring", "codepoint", c)
		return
	}
	p.intermediates[p.intermediatesIdx] = c
	p.intermediatesIdx += 1
}

func (p *Parser) Clear() {
	p.paramsIdx = 0
	p.paramAcc = 0
	p.paramAccIdx = 0
	p.paramsSet.Clear()
	p.intermediatesIdx = 0
}
