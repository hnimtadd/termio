package osc

type Parser struct{}

func (p *Parser) End() *Command {
	return &Command{}
}

func (p *Parser) Next(c uint8) {
}

func (p *Parser) Reset() {}
