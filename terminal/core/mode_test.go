package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ModeState(t *testing.T) {
	// Create a new mode state
	state := NewModeState(nil, nil)

	assert.False(
		t,
		state.Get(ModeDisableKeyboard),
		"Expected ModeDisableKeyboard to be false by default",
	)

	// Set the mode
	state.Set(ModeDisableKeyboard, true)

	// Check if the mode is set correctly
	assert.True(
		t,
		state.Get(ModeDisableKeyboard),
		"Expected ModeDisableKeyboard to be set to true",
	)

	// Unset the mode
	state.Set(ModeDisableKeyboard, false)

	// Check if the mode is unset correctly
	assert.False(
		t,
		state.Get(ModeDisableKeyboard),
		"Expected ModeDisableKeyboard to be set to false",
	)
}

func TestModeFromInput(t *testing.T) {
	mode := ModeFromInt(2, true)
	assert.NotNil(t, mode)
	assert.True(t, *mode == ModeDisableKeyboard)

	mode = ModeFromInt(4, true)
	assert.NotNil(t, mode)
	assert.True(t, *mode == ModeInsert)
}
