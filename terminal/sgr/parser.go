// SGR (Selective Graphic Rendition) attribute parsing and types
//
// This is implemented based on: https://vt100.net/docs/vt510-rm/SGR.html
package sgr

import (
	"iter"
	"math"

	"github.com/hnimtadd/termio/terminal/color"
	"github.com/hnimtadd/termio/terminal/utils"
)

type AttributeType uint16

const (
	AttributeTypeUnset AttributeType = iota
	// Bold the text.
	AttributeTypeBold
	AttributeTypeResetBold

	// Italic the text.
	AttributeTypeItalic
	AttributeTypeResetItalic

	// Faint/dim text.
	AttributeTypeFaint
	AttributeTypeResetFaint

	// Underline the text.
	AttributeTypeUnderline
	AttributeTypeResetUnderline
	AttributeTypeUnderlineColor
	AttributeTypeResetUnderlineColor

	// Overline the text.
	AttributeTypeOverline
	AttributeTypeResetOverline

	// Blink the text.
	AttributeTypeBlink
	AttributeTypeResetBlink

	// Invert fg/bg colors.
	AttributeTypeInverse
	AttributeTypeResetInverse

	// Invisible text.
	AttributeTypeInvisible
	AttributeTypeResetInvisible

	// Fg direct color
	AttributeTypeDirectColorFg
	// Bg direct color
	AttributeTypeDirectColorBg

	// Strikethrough the text.
	AttributeTypeStrikethrough
	AttributeTypeResetStrikethrough

	// Reset fg colors.
	AttributeTypeResetFg
	// Reset bg colors.
	AttributeTypeResetBg

	// Unkown
	AttributeTypeUnknown
)

type UnderlineType uint8

const (
	UnderlineTypeNone UnderlineType = iota
	UnderlineTypeSingle
	UnderlineTypeDouble
	UnderlineTypeCurly
	UnderlineTypeDotted
	UnderlineTypedashed
)

type unknown struct {
	Full    []uint16
	Partial []uint16
}

type Attribute struct {
	Type           AttributeType
	Underline      UnderlineType
	UnderlineColor color.RGB
	Unknown        unknown
	DirectColorFg  color.RGB
	DirectColorBg  color.RGB
}
type Parser struct {
	Params    []uint16
	ParamsSep *utils.StaticBitSet
	idx       int
}

// next return pull function that could be used to get attr parsed by this
// parser.
// Result of pull function:
//   - attr: parsed value
//   - ok: bool value indicated pull is availabe next time or not.
func (p *Parser) next() func() (attr *Attribute, ok bool) {
	p.idx = 0
	return func() (*Attribute, bool) {
		if p.idx >= len(p.Params) {
			// If we are at the index zero, it means we must have an empty
			// list and an empty list implicitly means nothings.
			if p.idx == 0 {
				p.idx += 1
				return &Attribute{Type: AttributeTypeUnset}, false
			}
			return nil, false
		}
		slice := p.Params[p.idx:]
		colon := p.ParamsSep.IsSet(p.idx)
		p.idx += 1
		// Our last one will have an idx be the last value.
		if colon {
			switch slice[0] {
			// Underline, FG colored, BG colored is support, Set Underline colored
			case 4, 38, 48:
				// we need colon separated value for colors
				break
			default:
				// otherwise, consume all the colon separated values.
				start := p.idx
				for p.ParamsSep.IsSet(p.idx) {
					p.idx += 1
				}
				p.idx += 1
				return &Attribute{
					Type: AttributeTypeUnknown,
					Unknown: unknown{
						Full:    p.Params[start:p.idx],
						Partial: slice[0 : p.idx-start+1],
					},
				}, true
			}
		}
		// Based on: https://en.wikipedia.org/wiki/ANSI_escape_code
		switch slice[0] {
		case 0:
			return &Attribute{Type: AttributeTypeUnset}, true
		case 1:
			return &Attribute{Type: AttributeTypeBold}, true
		case 2:
			return &Attribute{Type: AttributeTypeFaint}, true
		case 3:
			return &Attribute{Type: AttributeTypeItalic}, true
		case 4:
			if colon {
				utils.Assert(len(slice) > 2)
				if p.isColon() {
					p.consumeUnknownColon()
					return nil, true
				}

				p.idx += 1
				// Get the underlineType
				// based on: https://gitlab.com/gnachman/iterm2/-/issues/6382
				switch slice[1] {
				case 0:
					return &Attribute{Type: AttributeTypeResetUnderline}, true
				case 1:
					return &Attribute{
						Type:      AttributeTypeUnderline,
						Underline: UnderlineTypeSingle,
					}, true
				case 2:
					return &Attribute{
						Type:      AttributeTypeUnderline,
						Underline: UnderlineTypeDouble,
					}, true
				case 3:
					return &Attribute{
						Type:      AttributeTypeUnderline,
						Underline: UnderlineTypeCurly,
					}, true
				case 4:
					return &Attribute{
						Type:      AttributeTypeUnderline,
						Underline: UnderlineTypeDotted,
					}, true
				case 5:
					return &Attribute{
						Type:      AttributeTypeUnderline,
						Underline: UnderlineTypedashed,
					}, true
				default:
					// For unknown underline styles, just render
					// a single underline.
					return &Attribute{
						Type:      AttributeTypeUnderline,
						Underline: UnderlineTypeSingle,
					}, true
				}

			}
			return &Attribute{Type: AttributeTypeUnderline, Underline: UnderlineTypeSingle}, true
		case 5, 6:
			return &Attribute{Type: AttributeTypeBlink}, true
		case 7:
			return &Attribute{Type: AttributeTypeInverse}, true
		case 8:
			return &Attribute{Type: AttributeTypeInvisible}, true
		case 9:
			return &Attribute{Type: AttributeTypeStrikethrough}, true
		case 21:
			return &Attribute{Type: AttributeTypeUnderline, Underline: UnderlineTypeDouble}, true
		case 22:
			return &Attribute{Type: AttributeTypeResetBold}, true
		case 23:
			return &Attribute{Type: AttributeTypeResetItalic}, true
		case 24:
			return &Attribute{Type: AttributeTypeResetUnderline}, true
		case 25:
			return &Attribute{Type: AttributeTypeResetBlink}, true
		case 27:
			return &Attribute{Type: AttributeTypeResetInverse}, true
		case 28:
			return &Attribute{Type: AttributeTypeResetInvisible}, true
		case 29:
			return &Attribute{Type: AttributeTypeResetStrikethrough}, true
		case 38:
			if len(slice) >= 2 {
				switch slice[1] {
				// direct-color (r, g, b)
				case 2:
					color := p.parseDirectColor(slice, colon)
					if color != nil {
						return &Attribute{
							Type:          AttributeTypeDirectColorFg,
							DirectColorFg: *color,
						}, true
					} else {
						return nil, true
					}
				// case 5 we don't support indexed color yet.
				default:
					return nil, true
				}
			}
		case 48:
			if len(slice) >= 2 {
				switch slice[1] {
				// direct-color (r, g, b)
				case 2:
					color := p.parseDirectColor(slice, colon)
					if color != nil {
						return &Attribute{
							Type:          AttributeTypeDirectColorBg,
							DirectColorBg: *color,
						}, true
					} else {
						return nil, true
					}
				// case 5 we don't support indexed color yet.
				default:
					return nil, true
				}
			}
		case 49:
			// Reset the background color)
			return &Attribute{Type: AttributeTypeResetBg}, true
		case 53:
			return &Attribute{Type: AttributeTypeOverline}, true
		case 55:
			return &Attribute{Type: AttributeTypeResetOverline}, true
		case 58:
			// underline color
			if len(slice) >= 2 {
				switch slice[1] {
				// direct-color (r, g, b)
				case 2:
					if color := p.parseDirectColor(slice, colon); color != nil {
						return &Attribute{
							Type:           AttributeTypeUnderlineColor,
							UnderlineColor: *color,
						}, true
					} else {
						return nil, true
					}
				// case 5 we don't support indexed color yet.
				default:
					return nil, true
				}
			}
		case 59:
			return &Attribute{Type: AttributeTypeResetUnderlineColor}, true
		}
		return &Attribute{
			Type:    AttributeTypeUnknown,
			Unknown: unknown{Full: p.Params, Partial: slice},
		}, true
	}
}

// Iter returns iter.Sew[*Attribute] iterator that yields the attributes
func (p *Parser) Iter() iter.Seq[*Attribute] {
	next := p.next()
	return func(yield func(*Attribute) bool) {
		for {
			attr, ok := next()
			if !yield(attr) {
				return
			}
			if !ok {
				return
			}
		}
	}
}

// Iter2 returns iter.Seq2[int, *Attribute] that yields the attributes with idx
func (p *Parser) Iter2() iter.Seq2[int, *Attribute] {
	return func(yield func(int, *Attribute) bool) {
		pull, close := iter.Pull(p.Iter())
		defer close()
		for idx := 0; ; idx++ {
			attr, isOk := pull()
			if !yield(idx, attr) {
				return
			}
			if !isOk {
				return
			}
		}
	}
}

// parseDirectColor parses the direct color from the parameters.
// Any direct color style must have at least 5 values.
func (p *Parser) parseDirectColor(slice []uint16, colon bool) *color.RGB {
	if len(slice) < 5 {
		return nil
	}
	// Assert this method only used for direct color sets (38, 48, 58) and subparam 2.
	utils.Assert(slice[1] == 2)
	if !colon {
		p.idx += 4
		// perform truncate data as we are working with uint16
		// the value should be 0 to 255, we don't know the behavior of term if
		// the value is out of range.
		return &color.RGB{
			R: uint8(min(math.MaxUint8, slice[2])),
			G: uint8(min(math.MaxUint8, slice[3])),
			B: uint8(min(math.MaxUint8, slice[4])),
		}
	}

	// we have a colon, we might have either 5 or 6 values depending
	// on the color space is present or not.
	count := p.countColon()
	switch count {
	case 3:
		// rgb
		p.idx += 4
		return &color.RGB{
			R: uint8(min(math.MaxUint8, slice[2])),
			G: uint8(min(math.MaxUint8, slice[3])),
			B: uint8(min(math.MaxUint8, slice[4])),
		}
	case 4:
		p.idx += 5
		return &color.RGB{
			R: uint8(min(math.MaxUint8, slice[3])),
			G: uint8(min(math.MaxUint8, slice[4])),
			B: uint8(min(math.MaxUint8, slice[5])),
		}
	default:
		// consume remaining colon, as we have ill-formed data.
		p.consumeUnknownColon()
		return nil
	}
}

// Returns true if the present position has a colon separator.
// This always returns false for the last value since it has no
// separator.
func (p *Parser) isColon() bool {
	// The `- 1` here is because the last value has no separator.
	if p.idx >= len(p.Params)-1 {
		return false
	}
	return p.ParamsSep.IsSet(p.idx)
}

// Consumes all the remaining parameters separated by a colon and
// returns an unknown attribute.
func (p *Parser) consumeUnknownColon() {
	count := p.countColon()
	p.idx += count + 1
}

func (p *Parser) countColon() int {
	count := 0
	for count, idx := 0, p.idx; idx < len(p.Params) && p.ParamsSep.IsSet(idx); idx, count = idx+1, count+1 {
	}
	return count
}
