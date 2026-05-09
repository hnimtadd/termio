package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParserNext(t *testing.T) {
	tcs := []struct {
		name     string
		previous []uint8
		curr     uint8
		expected func(*testing.T, [3]*Action)
	}{
		{
			name:     "esc: ESC ( B -- 0x1B 0x28 0x42",
			previous: []uint8{0x1B, '('},
			curr:     'B',
			expected: func(t *testing.T, actions [3]*Action) {
				assert.Nil(t, actions[0])
				assert.NotNil(t, actions[1].ESCDispatchData)
				assert.Nil(t, actions[2])

				d := actions[1].ESCDispatchData
				assert.EqualValues(t, 'B', d.Final)
				assert.EqualValues(t, 1, len(d.Intermediates))
				assert.EqualValues(t, '(', d.Intermediates[0])
			},
		},
		{
			name:     "csi: CSI ( B",
			previous: []uint8{0x9B, '('},
			curr:     'B',
			expected: func(t *testing.T, actions [3]*Action) {
				assert.Nil(t, actions[0])
				assert.NotNil(t, actions[1].CSIDispatchData)
				assert.Nil(t, actions[2])

				d := actions[1].CSIDispatchData
				assert.EqualValues(t, 'B', d.Final)
				assert.EqualValues(t, 1, len(d.Intermediates))
				assert.EqualValues(t, '(', d.Intermediates[0])
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			p := NewParser()
			for _, prev := range tc.previous {
				p.Next(prev)
			}
			actions := p.Next(tc.curr)
			tc.expected(t, actions)
		})
	}
}

func TestParserOSCPathIsInitializedAndTerminates(t *testing.T) {
	p := NewParser()

	input := []uint8{0x9D, '0', ';', 'x', 0x9C}
	var endAction *Action
	for _, b := range input {
		actions := p.Next(b)
		for _, action := range actions {
			if action != nil && action.Type == ActionOSCEnd {
				endAction = action
			}
		}
	}

	assert.NotNil(t, endAction)
	assert.NotNil(t, endAction.OSCDispatchData)
	assert.Equal(t, StateGround, p.State)
}
