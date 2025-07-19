package csi

import (
	"fmt"

	"github.com/hnimtadd/termio/terminal/utils"
)

type Command struct {
	Intermediates []uint8
	Params        []uint16
	ParamsSet     *utils.StaticBitSet
	Final         uint8
}

func (c Command) String() string {
	return fmt.Sprintf("CSI %v %v %v", c.Intermediates, c.Params, c.Final)
}

// Erase in Display mode
type EDMode uint8

const (
	EDModeBelow      EDMode = 0
	EDModeAbove      EDMode = 1
	EDModeComplete   EDMode = 2
	EDModeScrollback EDMode = 3
)

// Erase in Line mode
type ELMode uint8

const (
	ELModeRight ELMode = 0
	ELModeLeft  ELMode = 1
	ELModeAll   ELMode = 2
)

type SGR uint8
