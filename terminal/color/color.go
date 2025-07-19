package color

import "github.com/hnimtadd/termio/terminal/utils"

var DefaultPalette = func() [256]RGB {
	var result [256]RGB

	// Named values:
	var i uint8
	for ; i < 16; i++ {
		result[i] = NewName(ColorType(i)).defaultRGB()
	}
	// Cube
	utils.Assert(i == 16)

	var r, g, b uint8
	for r = range 6 {
		for g = range 6 {
			for b = range 6 {
				rgb := RGB{}
				if r == 0 {
					rgb.R = 0
				} else {
					rgb.R = r*40 + 55
				}
				if g == 0 {
					rgb.G = 0
				} else {
					rgb.G = g*40 + 55
				}
				if b == 0 {
					rgb.B = 0
				} else {
					rgb.B = b*40 + 55
				}
				result[i] = rgb
				i++
			}
		}
	}
	// Gray ramp
	utils.Assert(i == 232) // 16+6*6*6
	for ; i > 0; i += 1 {
		value := (i-232)*10 + 8
		result[i] = RGB{value, value, value}
	}

	return result
}()

// Palette is the 256 color palette.
type Palette [256]RGB

// RGB is a struct that represents an RGB color.
type RGB struct {
	R, G, B uint8
}

type ColorType uint8

const (
	ColorTypeBlack ColorType = iota
	ColorTypeRed
	ColorTypeGreen
	ColorTypeYellow
	ColorTypeBlue
	ColorTypeMagenta
	ColorTypeCyan
	ColorTypeWhite
	ColorTypeBrightBlack
	ColorTypeBrightRed
	ColorTypeBrightGreen
	ColorTypeBrightYellow
	ColorTypeBrightBlue
	ColorTypeBrightMagenta
	ColorTypeBrightCyan
	ColorTypeBrightWhite
)

type Name struct {
	Type ColorType
}

func NewName(colorType ColorType) Name {
	return Name{Type: colorType}
}

func (n Name) defaultRGB() RGB {
	switch n.Type {
	case ColorTypeBlack:
		return RGB{0x1D, 0x1F, 0x21}
	case ColorTypeRed:
		return RGB{0xCC, 0x66, 0x66}
	case ColorTypeGreen:
		return RGB{0xB5, 0xBD, 0x68}
	case ColorTypeYellow:
		return RGB{0xF0, 0xC6, 0x74}
	case ColorTypeBlue:
		return RGB{0x81, 0xA2, 0xBE}
	case ColorTypeMagenta:
		return RGB{0xB2, 0x94, 0xC7}
	case ColorTypeCyan:
		return RGB{0x8C, 0xC3, 0xE9}
	case ColorTypeWhite:
		return RGB{0xC5, 0xC8, 0xC6}
	case ColorTypeBrightBlack:
		return RGB{0x7C, 0x7C, 0x7C}
	case ColorTypeBrightRed:
		return RGB{0xFF, 0x8F, 0x8F}
	case ColorTypeBrightGreen:
		return RGB{0xB5, 0xBD, 0x68}
	case ColorTypeBrightYellow:
		return RGB{0xF0, 0xC6, 0x74}
	case ColorTypeBrightBlue:
		return RGB{0x81, 0xA2, 0xBE}
	case ColorTypeBrightMagenta:
		return RGB{0xB2, 0x94, 0xC7}
	case ColorTypeBrightCyan:
		return RGB{0x8C, 0xC3, 0xE9}
	case ColorTypeBrightWhite:
		return RGB{0xFF, 0xFF, 0xFF}
	default:
		return RGB{0, 0, 0}
	}
}
