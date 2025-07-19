package tabstops

import (
	"testing"

	"github.com/hnimtadd/termio/terminal/size"
	"github.com/stretchr/testify/assert"
)

func TestTabstopsBasic(t *testing.T) {
	tab := NewTabstops(16, 0)
	assert.Equal(t, 0, entry(4))
	assert.Equal(t, 1, entry(8))
	assert.Equal(t, 0, index(0))
	assert.Equal(t, 1, index(1))
	assert.Equal(t, 1, index(9))
	assert.EqualValues(t, 0b00001000, masks[3])
	assert.EqualValues(t, 0b00010000, masks[4])
	assert.False(t, tab.Get(4))
	tab.Set(4)
	assert.True(t, tab.Get(4))
	assert.False(t, tab.Get(3))
	tab.Reset(0)
	assert.False(t, tab.Get(4))
	tab.Set(4)
	assert.True(t, tab.Get(4))
	tab.Unset(4)
	assert.False(t, tab.Get(4))
}

func TestTabstopsDynamicAllocations(t *testing.T) {
	tab := NewTabstops(16, 0)
	capacity := tab.Capacity()
	tab.Resize(size.CellCountInt(capacity * 2))
	tab.Set(size.CellCountInt(capacity + 5))
	assert.True(t, tab.Get(size.CellCountInt(capacity+5)), "tab.Get(capacity+5) should be true")

	assert.False(t, tab.Get(size.CellCountInt(capacity+4)))
	assert.False(t, tab.Get(5))
}

func TestTabstopsInterval(t *testing.T) {
	tab := NewTabstops(80, 4)
	if tab.Get(0) {
		t.Errorf("tab.Get(0) = true, want false")
	}
	if !tab.Get(4) {
		t.Errorf("tab.Get(4) = false, want true")
	}
	if tab.Get(5) {
		t.Errorf("tab.Get(5) = true, want false")
	}
	if !tab.Get(8) {
		t.Errorf("tab.Get(8) = false, want true")
	}
}

func TestTabstopsCountOn80(t *testing.T) {
	tab := NewTabstops(80, 8)
	count := 0
	for i := range size.CellCountInt(80) {
		if tab.Get(i) {
			count++
		}
	}
	if count != 9 {
		t.Errorf("tabstops count = %d, want 9", count)
	}
}
