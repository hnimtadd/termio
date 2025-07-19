package esc

import (
	"fmt"
)

type Command struct {
	Intermediates []uint8
	Final         uint8
}

func (c Command) String() string {
	return fmt.Sprintf("ESC %v %v", c.Intermediates, c.Final)
}
