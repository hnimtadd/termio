package sgr

import (
	"iter"
	"testing"

	"github.com/hnimtadd/termio/terminal/color"
	"github.com/hnimtadd/termio/terminal/utils"
	"github.com/stretchr/testify/assert"
)

func TestParserNext(t *testing.T) {
	tests := []struct {
		name      string
		params    []uint16
		paramsSep *utils.StaticBitSet
		expected  *Attribute
	}{
		{
			name:      "[]: unset",
			params:    []uint16{},
			paramsSep: utils.NewStaticBitSet(1),
			expected:  &Attribute{Type: AttributeTypeUnset},
		},
		{
			name:      "[0]: unset",
			params:    []uint16{0},
			paramsSep: utils.NewStaticBitSet(1),
			expected:  &Attribute{Type: AttributeTypeUnset},
		},
		{
			name:      "[38, 2, 40, 44, 52]: direct color fg",
			params:    []uint16{38, 2, 40, 44, 52},
			paramsSep: utils.NewStaticBitSet(5),
			expected: &Attribute{
				Type:          AttributeTypeDirectColorFg,
				DirectColorFg: color.RGB{R: 40, G: 44, B: 52},
			},
		},
		{
			name:      "[38, 2, 44, 52]: unknown",
			params:    []uint16{38, 2, 44, 52},
			paramsSep: utils.NewStaticBitSet(4),
			expected:  nil,
		},
		{
			name:      "[48, 2, 40, 44, 52]: direct color bg",
			params:    []uint16{48, 2, 40, 44, 52},
			paramsSep: utils.NewStaticBitSet(5),
			expected: &Attribute{
				Type:          AttributeTypeDirectColorBg,
				DirectColorBg: color.RGB{R: 40, G: 44, B: 52},
			},
		},
		{
			name:      "[38, 2, 44, 52]: unknown",
			params:    []uint16{38, 2, 44, 52},
			paramsSep: utils.NewStaticBitSet(4),
			expected:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Parser{
				Params:    tc.params,
				ParamsSep: tc.paramsSep,
			}
			pull, stop := iter.Pull(p.Iter())
			defer stop()
			got, ok := pull()
			assert.True(t, ok)
			assert.EqualValues(t, tc.expected, got)
		})
	}
}

func TestParserNextMultiple(t *testing.T) {
	t.Run("[0, 38, 2, 40, 44, 52]: unset, DirectColorFg", func(t *testing.T) {
		parser := Parser{
			Params:    []uint16{0, 38, 2, 40, 44, 52},
			ParamsSep: utils.NewStaticBitSet(6),
		}
		pull, cls := iter.Pull(parser.Iter())
		defer cls()
		attr, ok := pull()
		assert.True(t, ok)
		assert.NotNil(t, attr)
		assert.Equal(t, AttributeTypeUnset, attr.Type)

		attr, ok = pull()
		assert.True(t, ok)
		assert.NotNil(t, attr)
		assert.Equal(t, AttributeTypeDirectColorFg, attr.Type)
		assert.Equal(t, color.RGB{R: 40, G: 44, B: 52}, attr.DirectColorFg)

		attr, ok = pull()
		assert.True(t, ok) // We don't mark the parsernext to done here
		assert.Nil(t, attr)

		attr, ok = pull()
		assert.False(t, ok)
		assert.Nil(t, attr)
	})
}

func TestUnsupportedWithColon(t *testing.T) {
	t.Run("sgr: unsupported with colon", func(t *testing.T) {
		sepList := utils.NewStaticBitSet(3)
		sepList.Set(0)
		parser := Parser{
			Params:    []uint16{0, 4, 1},
			ParamsSep: sepList,
		}
		pull, cls := iter.Pull(parser.Iter())
		defer cls()

		attr, ok := pull()
		assert.True(t, ok)
		assert.NotNil(t, attr)
		assert.Equal(t, AttributeTypeUnknown, attr.Type)

		attr, ok = pull()
		assert.True(t, ok)
		assert.NotNil(t, attr)
		assert.Equal(t, AttributeTypeBold, attr.Type)

		attr, ok = pull()
		assert.True(t, ok) // We don't mark the parsernext to done here
		assert.Nil(t, attr)

		attr, ok = pull()
		assert.False(t, ok)
		assert.Nil(t, attr)
	})
}

func TestParserWithSingleAttribute(t *testing.T) {
	tests := []struct {
		name      string
		params    []uint16
		paramsSep *utils.StaticBitSet
		expected  AttributeType
	}{
		{
			name:      "sgr: bold",
			params:    []uint16{1},
			paramsSep: utils.NewStaticBitSet(1),
			expected:  AttributeTypeBold,
		},
		{
			name:      "sgr: reset bold",
			params:    []uint16{22},
			paramsSep: utils.NewStaticBitSet(1),
			expected:  AttributeTypeResetBold,
		},
		{
			name:      "sgr: italic",
			params:    []uint16{3},
			paramsSep: utils.NewStaticBitSet(1),
			expected:  AttributeTypeItalic,
		},
		{
			name:      "sgr: reset italic",
			params:    []uint16{23},
			paramsSep: utils.NewStaticBitSet(1),
			expected:  AttributeTypeResetItalic,
		},
		{
			name:      "sgr: underline",
			params:    []uint16{4},
			paramsSep: utils.NewStaticBitSet(1),
			expected:  AttributeTypeUnderline,
		},
		{
			name:      "sgr: resetUnderLine",
			params:    []uint16{24},
			paramsSep: utils.NewStaticBitSet(1),
			expected:  AttributeTypeResetUnderline,
		},
		{
			name:      "sgr: overline",
			params:    []uint16{53},
			paramsSep: utils.NewStaticBitSet(1),
			expected:  AttributeTypeOverline,
		},
		{
			name:      "sgr: reset overline",
			params:    []uint16{55},
			paramsSep: utils.NewStaticBitSet(1),
			expected:  AttributeTypeResetOverline,
		},
		{
			name:      "sgr: invisible",
			params:    []uint16{8},
			paramsSep: utils.NewStaticBitSet(1),
			expected:  AttributeTypeInvisible,
		},
		{
			name:      "sgr: reset invisible",
			params:    []uint16{28},
			paramsSep: utils.NewStaticBitSet(1),
			expected:  AttributeTypeResetInvisible,
		},
		{
			name:      "sgr: blink",
			params:    []uint16{5},
			paramsSep: utils.NewStaticBitSet(1),
			expected:  AttributeTypeBlink,
		},
		{
			name:      "sgr: blink",
			params:    []uint16{6},
			paramsSep: utils.NewStaticBitSet(1),
			expected:  AttributeTypeBlink,
		},
		{
			name:      "sgr: reset Blink",
			params:    []uint16{25},
			paramsSep: utils.NewStaticBitSet(1),
			expected:  AttributeTypeResetBlink,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Parser{
				Params:    tc.params,
				ParamsSep: tc.paramsSep,
			}
			pull, stop := iter.Pull(p.Iter())
			defer stop()
			got, ok := pull()
			assert.True(t, ok)
			assert.EqualValues(t, tc.expected, got.Type)
		})
	}
}
