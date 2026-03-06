package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModeCreation(t *testing.T) {
	// Test mode entry creation
	mode := entryForMode("test", 42, true, false)
	assert.Equal(t, "test", mode.Name)
	assert.Equal(t, 42, mode.Value)
	assert.True(t, mode.Ansi)
	assert.False(t, mode.Default)
}

func TestModeState(t *testing.T) {
	// Create mode state with default modes
	state := NewModeState(ModePacked, ModePacked)
	require.NotNil(t, state)

	// Test getting default values
	assert.Equal(t, ModeWraparound.Default, state.Get(ModeWraparound))
	assert.Equal(t, ModeInsert.Default, state.Get(ModeInsert))

	// Test setting modes
	state.Set(ModeInsert, true)
	assert.True(t, state.Get(ModeInsert))

	state.Set(ModeInsert, false)
	assert.False(t, state.Get(ModeInsert))
}

func TestModeStateReset(t *testing.T) {
	state := NewModeState(ModePacked, ModePacked)

	// Change some modes
	state.Set(ModeInsert, true)
	state.Set(ModeWraparound, false)

	// Reset should restore defaults
	state.Reset()
	
	assert.Equal(t, ModeInsert.Default, state.Get(ModeInsert))
	assert.Equal(t, ModeWraparound.Default, state.Get(ModeWraparound))
}

func TestModeFromInt(t *testing.T) {
	tests := []struct {
		value    int
		ansi     bool
		expected *Mode
	}{
		{4, true, &ModeInsert},
		{7, false, &ModeWraparound},
		{6, false, &ModeOrigin},
		{999, true, nil}, // Non-existent mode
	}

	for _, tt := range tests {
		result := ModeFromInt(tt.value, tt.ansi)
		if tt.expected == nil {
			assert.Nil(t, result)
		} else {
			require.NotNil(t, result)
			assert.Equal(t, tt.expected.Value, result.Value)
			assert.Equal(t, tt.expected.Ansi, result.Ansi)
			assert.Equal(t, tt.expected.Name, result.Name)
		}
	}
}

func TestModeStateWithNilMaps(t *testing.T) {
	// Test creation with nil maps
	state := NewModeState(nil, nil)
	require.NotNil(t, state)
	require.NotNil(t, state.values)
	require.NotNil(t, state.defaults)

	// Should handle get/set operations safely
	state.Set(ModeInsert, true)
	assert.True(t, state.Get(ModeInsert))
}

func TestAllDefinedModes(t *testing.T) {
	// Ensure all defined modes are accessible
	modes := []Mode{
		ModeDisableKeyboard,
		ModeInsert,
		ModeSendReceiveMode,
		ModeLineFeed,
		ModeWraparound,
		ModeOrigin,
	}

	for _, mode := range modes {
		t.Run(mode.Name, func(t *testing.T) {
			// Verify mode has valid properties
			assert.NotEmpty(t, mode.Name)
			assert.GreaterOrEqual(t, mode.Value, 0)
			
			// Verify mode can be found by value
			found := ModeFromInt(mode.Value, mode.Ansi)
			require.NotNil(t, found, "Mode %s should be findable", mode.Name)
			assert.Equal(t, mode.Value, found.Value)
		})
	}
}

func TestModePackedDefaults(t *testing.T) {
	// Test that ModePacked contains all defined modes
	assert.Len(t, ModePacked, len(entries))

	// Verify each mode exists in ModePacked
	for _, mode := range entries {
		value, exists := ModePacked[mode]
		assert.True(t, exists, "Mode %s should exist in ModePacked", mode.Name)
		assert.Equal(t, mode.Default, value, "Mode %s should have correct default", mode.Name)
	}
}