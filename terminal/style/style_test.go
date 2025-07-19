package style

import (
	"testing"

	"github.com/hnimtadd/termio/terminal/color"
	"github.com/hnimtadd/termio/terminal/page"
	"github.com/stretchr/testify/assert"
)

func TestColorString(t *testing.T) {
	cNone := Color{Type: ColorTypeNone}
	assert.Equal(t, "Color.none", cNone.String())

	cPalette := Color{Type: ColorTypePalette, Palette: 5}
	assert.Equal(t, "Color.palette{{ 5 }}", cPalette.String())

	cRGB := Color{Type: ColorTypeRGB, RGB: color.RGB{R: 1, G: 2, B: 3}}
	assert.Equal(t, "Color.rgb{{ 1, 2, 3 }}", cRGB.String())
}

func TestStyle_BG(t *testing.T) {
	palette := color.Palette{}
	palette[3] = color.RGB{R: 10, G: 20, B: 30}
	cell := &page.Cell{
		ContentTag:          page.ContentTagBGColorPalette,
		ContentColorPalette: 3,
	}
	style := &Style{}
	bg := style.BG(cell, &palette)
	assert.NotNil(t, bg)
	assert.Equal(t, &palette[3], bg)

	cell2 := &page.Cell{
		ContentTag:      page.ContentTagBGColorRGB,
		ContentColorRGB: color.RGB{R: 7, G: 8, B: 9},
	}
	bg2 := style.BG(cell2, &palette)
	assert.NotNil(t, bg2)
	assert.Equal(t, &color.RGB{R: 7, G: 8, B: 9}, bg2)

	style.BackgroundColor = Color{Type: ColorTypePalette, Palette: 3}
	bg3 := style.BG(&page.Cell{}, &palette)
	assert.Equal(t, &palette[3], bg3)

	style.BackgroundColor = Color{Type: ColorTypeRGB, RGB: color.RGB{R: 1, G: 2, B: 3}}
	bg4 := style.BG(&page.Cell{}, &palette)
	assert.Equal(t, &style.BackgroundColor.RGB, bg4)

	style.BackgroundColor = Color{Type: ColorTypeNone}
	bg5 := style.BG(&page.Cell{}, &palette)
	assert.Nil(t, bg5)
}

func TestStyle_FG(t *testing.T) {
	palette := color.Palette{}
	palette[2] = color.RGB{R: 100, G: 101, B: 102}
	style := &Style{ForegroundColor: Color{Type: ColorTypePalette, Palette: 2}}
	fg := style.FG(&page.Cell{}, &palette, false)
	assert.Nil(t, fg) // Only returns non-nil if boldIsBright && Bold

	style.Bold = true
	fg2 := style.FG(&page.Cell{}, &palette, true)
	assert.NotNil(t, fg2)

	style.ForegroundColor = Color{Type: ColorTypeRGB, RGB: color.RGB{R: 1, G: 2, B: 3}}
	fg3 := style.FG(&page.Cell{}, &palette, false)
	assert.Equal(t, &style.ForegroundColor.RGB, fg3)

	style.ForegroundColor = Color{Type: ColorTypeNone}
	fg4 := style.FG(&page.Cell{}, &palette, false)
	assert.Nil(t, fg4)
}

func TestStyle_UColor(t *testing.T) {
	palette := color.Palette{}
	palette[1] = color.RGB{R: 11, G: 12, B: 13}
	style := &Style{UnderlineColor: Color{Type: ColorTypePalette, Palette: 1}}
	uc := style.UColor(&palette)
	assert.Equal(t, &palette[1], uc)

	style.UnderlineColor = Color{Type: ColorTypeRGB, RGB: color.RGB{R: 2, G: 3, B: 4}}
	uc2 := style.UColor(&palette)
	assert.Equal(t, &style.UnderlineColor.RGB, uc2)

	style.UnderlineColor = Color{Type: ColorTypeNone}
	uc3 := style.UColor(&palette)
	assert.Nil(t, uc3)
}

func TestStyle_BGCell(t *testing.T) {
	style := &Style{BackgroundColor: Color{Type: ColorTypeNone}}
	assert.Nil(t, style.BGCell())

	style.BackgroundColor = Color{Type: ColorTypePalette, Palette: 2}
	cell := style.BGCell()
	assert.NotNil(t, cell)
	assert.Equal(t, page.ContentTagBGColorPalette, cell.ContentTag)
	assert.Equal(t, uint8(2), cell.ContentColorPalette)

	style.BackgroundColor = Color{Type: ColorTypeRGB, RGB: color.RGB{R: 1, G: 2, B: 3}}
	cell2 := style.BGCell()
	assert.NotNil(t, cell2)
	assert.Equal(t, page.ContentTagBGColorRGB, cell2.ContentTag)
	assert.Equal(t, color.RGB{R: 1, G: 2, B: 3}, cell2.ContentColorRGB)
}

func TestStyle_ResetAndIsDefault(t *testing.T) {
	style := &Style{
		ForegroundColor: Color{Type: ColorTypePalette, Palette: 1},
		Bold:            true,
	}
	assert.False(t, style.IsDefault())
	style.Reset()
	assert.True(t, style.IsDefault())
}

func TestStyle_Hash(t *testing.T) {
	style1 := Style{
		Bold:            true,
		ForegroundColor: Color{Type: ColorTypePalette, Palette: 1},
	}
	// This is hacky, but we want to ensure that the hash is consistent
	assert.Equal(t, uint64(0x28dd36ff04042d), style1.Hash())
	style2 := Style{
		Bold:            false,
		ForegroundColor: Color{Type: ColorTypePalette, Palette: 1},
	}

	// This is hacky, but we want to ensure that the hash is consistent
	assert.Equal(t, uint64(0x8a13dc570c294d48), style2.Hash())
	style3 := Style{
		Italic:          true,
		ForegroundColor: Color{Type: ColorTypePalette, Palette: 2},
	}
	// This is hacky, but we want to ensure that the hash is consistent
	assert.Equal(t, uint64(0x947cdad87ab69e04), style3.Hash())
}

func TestStyle_HashAndEquals(t *testing.T) {
	style1 := Style{ForegroundColor: Color{Type: ColorTypePalette, Palette: 1}}
	style2 := Style{ForegroundColor: Color{Type: ColorTypePalette, Palette: 1}}
	style3 := Style{ForegroundColor: Color{Type: ColorTypePalette, Palette: 2}}

	assert.Equal(t, style1.Hash(), style2.Hash())
	assert.NotEqual(t, style1.Hash(), style3.Hash())
	assert.True(t, style1.Equals(style2))
	assert.False(t, style1.Equals(style3))
}

func TestStyle_DeletePanics(t *testing.T) {
	style := Style{}
	assert.Panics(t, func() { style.Delete() })
}
