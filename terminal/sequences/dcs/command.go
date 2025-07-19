package dcs

import "fmt"

type DCS struct {
	Intermediates []uint8
	Params        []uint16
	Final         uint8
}

func (c *DCS) String() string {
	return fmt.Sprintf("DCS %v %v %v", c.Intermediates, c.Params, c.Final)
}

type (
	CommandType int
	Command     struct {
		Type CommandType
	}
)
