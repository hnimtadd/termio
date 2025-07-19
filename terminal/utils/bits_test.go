package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStaticBitSet_Size(t *testing.T) {
	bs := NewStaticBitSet(100)
	assert.Equal(t, 100, bs.size)
}

func TestSetAndIsSet(t *testing.T) {
	bs := NewStaticBitSet(10)
	bs.Set(3)
	assert.True(t, bs.IsSet(3), "bit 3 should be set")
	assert.False(t, bs.IsSet(4), "bit 4 should not be set")
}

func TestSetMultipleBits(t *testing.T) {
	bs := NewStaticBitSet(128)
	bs.Set(0)
	bs.Set(64)
	bs.Set(127)
	assert.True(t, bs.IsSet(0))
	assert.True(t, bs.IsSet(64))
	assert.True(t, bs.IsSet(127))
}

func TestCount(t *testing.T) {
	bs := NewStaticBitSet(10)
	bs.Set(1)
	bs.Set(2)
	bs.Set(3)
	assert.Equal(t, 3, bs.Count())
}

func TestClear(t *testing.T) {
	bs := NewStaticBitSet(10)
	bs.Set(1)
	bs.Set(2)
	bs.Clear()
	assert.Equal(t, 0, bs.Count())
}

func TestNewStaticBitSetFull(t *testing.T) {
	bs := NewStaticBitSetFull(10)
	for i := range 10 {
		assert.True(t, bs.IsSet(i), "bit %d should be set in full bitset", i)
	}
}

func TestSetOutOfBoundsPanics(t *testing.T) {
	bs := NewStaticBitSet(5)
	assert.Panics(t, func() { bs.Set(5) })
}

func TestIsSetOutOfBoundsPanics(t *testing.T) {
	bs := NewStaticBitSet(5)
	assert.Panics(t, func() { bs.IsSet(5) })
}

func TestSetNegativeIndexPanics(t *testing.T) {
	bs := NewStaticBitSet(5)
	assert.Panics(t, func() { bs.Set(-1) })
}

func TestIsSetNegativeIndexPanics(t *testing.T) {
	bs := NewStaticBitSet(5)
	assert.Panics(t, func() { bs.IsSet(-1) })
}
