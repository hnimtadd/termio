package style

import (
	"fmt"

	"github.com/hnimtadd/termio/terminal/color"
	"github.com/hnimtadd/termio/terminal/page"
	"github.com/hnimtadd/termio/terminal/set"
	"github.com/hnimtadd/termio/terminal/sgr"
	"github.com/hnimtadd/termio/terminal/utils"
	"github.com/mitchellh/hashstructure/v2"
)

// Style attribute for a cell.
type Style struct {
	// Various colors, self-explanatory
	ForegroundColor Color
	BackgroundColor Color
	UnderlineColor  Color

	Bold          bool
	Italic        bool
	Faint         bool
	Blink         bool
	Inverse       bool
	Invisible     bool
	Strikethrough bool
	Overline      bool
	Underline     sgr.UnderlineType
}

// BG returns the bg color for a cell with this style given the cell
// that has this style and the palette to use.
//
// Note that generally if a cell is a color-only cell, it SHOULD
// only have the default style, but this is meant to work with the
// default style as well.
func (s *Style) BG(cell *page.Cell, palette *color.Palette) *color.RGB {
	switch cell.ContentTag {
	case page.ContentTagBGColorPalette:
		return &palette[cell.ContentColorPalette]
	case page.ContentTagBGColorRGB:
		return &color.RGB{
			R: cell.ContentColorRGB.R,
			G: cell.ContentColorRGB.G,
			B: cell.ContentColorRGB.B,
		}
	default:
		switch s.BackgroundColor.Type {
		case ColorTypeNone:
			return nil
		case ColorTypePalette:
			return &palette[s.BackgroundColor.Palette]
		case ColorTypeRGB:
			return &s.BackgroundColor.RGB
		}
	}
	return nil
}

// FG returns the fg color for a cell with this style given the palette.
func (s *Style) FG(
	cell *page.Cell,
	palette *color.Palette,
	boldIsBright bool,
) *color.RGB {
	switch s.ForegroundColor.Type {
	case ColorTypeNone:
		return nil
	case ColorTypePalette:
		if boldIsBright && s.Bold {
			brightOffset := color.ColorTypeBrightBlack
			if color.ColorType(s.ForegroundColor.Palette) < brightOffset {
				return &palette[color.ColorType(s.ForegroundColor.Palette)+
					brightOffset]
			}
		}
	case ColorTypeRGB:
		return &s.ForegroundColor.RGB
	}
	return nil
}

// UCloer returns the underline color for this style.
func (s *Style) UColor(
	palette *color.Palette,
) *color.RGB {
	switch s.UnderlineColor.Type {
	case ColorTypeNone:
		return nil
	case ColorTypePalette:
		return &palette[s.UnderlineColor.Palette]
	case ColorTypeRGB:
		return &s.UnderlineColor.RGB
	default:
		// we should never get here, but if we do, just return nil
		return nil
	}
}

// Returns a bg-color only cell from this style, if it exists.
func (s *Style) BGCell() *page.Cell {
	switch s.BackgroundColor.Type {
	case ColorTypeNone:
		return nil
	case ColorTypePalette:
		return &page.Cell{
			ContentTag:          page.ContentTagBGColorPalette,
			ContentColorPalette: s.BackgroundColor.Palette,
		}
	case ColorTypeRGB:
		return &page.Cell{
			ContentTag:      page.ContentTagBGColorRGB,
			ContentColorRGB: s.BackgroundColor.RGB,
		}
	default:
		return nil
	}
}

func (s *Style) Reset() {
	*s = Style{
		ForegroundColor: Color{Type: ColorTypeNone},
		BackgroundColor: Color{Type: ColorTypeNone},
		UnderlineColor:  Color{Type: ColorTypeNone},
		Bold:            false,
		Italic:          false,
		Faint:           false,
		Blink:           false,
		Inverse:         false,
		Invisible:       false,
		Strikethrough:   false,
		Overline:        false,
		Underline:       sgr.UnderlineTypeNone,
	}
}

func (s *Style) IsDefault() bool {
	return *s == Style{}
}

func (s Style) Hash() uint64 {
	hashed, err := hashstructure.Hash(s, hashstructure.FormatV2, nil)
	utils.Assert(err == nil, fmt.Sprintf("failed to hash style: %v", err))
	return hashed
}

func (s Style) Equals(other set.Hashable) bool {
	this := s.Hash()
	that := other.Hash()
	return this == that
}

func (s Style) Delete() {
	panic("Not implemented")
}

// The color for an SGR attribute. A color can come from multiple sources
// so we use this to track the source plus color value so that we can properly
// react to things like palette changes.
type Color struct {
	Type    ColorType
	Palette uint8
	RGB     color.RGB
}

func (c Color) String() string {
	switch c.Type {
	case ColorTypeNone:
		return "Color.none"
	case ColorTypePalette:
		return fmt.Sprintf("Color.palette{{ %d }}", c.Palette)
	case ColorTypeRGB:
		return fmt.Sprintf("Color.rgb{{ %d, %d, %d }}", c.RGB.R, c.RGB.G, c.RGB.B)
	default:
		return "Color.unknown"
	}
}

type ColorType int

const (
	ColorTypeNone ColorType = iota
	ColorTypePalette
	ColorTypeRGB
)
