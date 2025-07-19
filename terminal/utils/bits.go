package utils

import (
	"math/bits"
)

const bitSetSize = 64 // Number of bits in a uint64

// StaticBitSet is a simple bit set implementation
type StaticBitSet struct {
	bits      []uint64
	size      int
	sliceSize int // Number of uint64s needed to store the bits
}

// Changes the value of all bits in the specifed range to 1.
func (s *StaticBitSet) SetRange(start int, end int) {
	Assert(0 <= start)
	Assert(start <= end)
	Assert(end <= s.size, "End index out of bounds")
	startAddr, startOffset := s.addr(start)
	endAddr, endOffset := s.addr(end - 1)

	if startAddr == endAddr {
		// If both start and end are in the same uint64, set the bits directly.
		// clear the bits first

		// Mark all bits from startOffset to endOffset as 1.
		// 1 << startOffset = 0000 ... 10000...000
		//                             |startOffset
		// ( 1<< startOffset ) - 1 = 0000 ... 01111...111
		//                                     |startOffset
		var mask1 uint64 = (1 << startOffset) - 1

		// Mark all bits from 0 to endOffset as 1.
		// 1 << (endOffset + 1) = 0000 ... 10000...0000
		//                                 | endOffset + 1
		// (1 << (endOffset + 1)) - 1 = 0000 ... 01111...1111
		//                                        | endOffset + 1
		// ^((1 << (endOffset + 1)) - 1) = 1111 ... 10000...0000
		var mask2 uint64 = ^((1 << (endOffset + 1)) - 1)

		// clear all the bits in the range first
		// mask1 & mask2 = 0000 ... 01110...0000
		//                startOffset| |endOffset
		// ^(mask1 & mask2) = 1111 ... 10001...1111
		//                   startOffset| |endOffset
		s.bits[startAddr] &= ^(mask1 & mask2)

		// set all the bits in range to 1
		s.bits[startAddr] |= mask1 & mask2
	} else {
		var mask1 uint64 = (1 << startOffset) - 1 // Mask for bits before startOffset
		// Set bits in the first uint64 from startOffset to the end of that uint64.
		s.bits[startAddr] |= ^mask1 // Set all bits from startOffset to the end of the uint64

		var mask2 uint64 = ^((1 << (endOffset + 1)) - 1) // Mask for bits after endOffset
		// Set bits in the last uint64 from the beginning to endOffset.
		s.bits[endAddr] |= mask2 // Set all bits from the beginning to endOffset

		// Set all bits in between to 1.
		for i := startAddr + 1; i < endAddr; i++ {
			s.bits[i] = ^uint64(0) // Set all bits to 1
		}
	}
}

// NewStaticBitSet creates a new StaticBitSet with the given size.
func NewStaticBitSet(size int) *StaticBitSet {
	set := &StaticBitSet{size: size}
	set.init()
	return set
}

// NewStaticBitSetFull creates a StaticBitSet with all bits set to 1.
func NewStaticBitSetFull(size int) *StaticBitSet {
	set := &StaticBitSet{
		size: size,
	}
	set.init()
	for i := range set.bits {
		set.bits[i] = ^uint64(0) // Set all bits to 1
	}
	return set
}

// Set sets the bit at the given idx to 1
func (s *StaticBitSet) Set(idx int) {
	Assert(idx >= 0 && idx < s.size, "Index out of bounds")
	idx, offset := s.addr(idx)
	s.bits[idx] |= 1 << offset
}

// Unset clears the bit at the given idx
func (s *StaticBitSet) Unset(idx int) {
	Assert(idx >= 0 && idx < s.size, "Index out of bounds")
	idx, offset := s.addr(idx)
	s.bits[idx] &^= 1 << offset // Clear the bit at idx
}

// addr return the index of bit array containing the bit at idx and offset of
// givent bit in that array.
func (s *StaticBitSet) addr(idx int) (int, int) {
	return idx / bitSetSize, idx % bitSetSize
}

// IsSet returns if bit at given idx is set
func (s *StaticBitSet) IsSet(idx int) bool {
	Assert(idx >= 0 && idx < s.size, "Index out of bounds")
	idx, offset := s.addr(idx)
	return s.bits[idx]&(1<<offset) != 0
}

// Count counts the number of bits set
func (s *StaticBitSet) Count() int {
	total := 0
	for i := range s.sliceSize {
		// Count the number of bits set
		total += bits.OnesCount64(s.bits[i])
	}
	return total
}

func (s *StaticBitSet) init() {
	s.sliceSize = (s.size + 63) / 64 // Calculate how many uint64s we need
	s.bits = make([]uint64, s.sliceSize)
}

// Clear clears the bits set
func (s *StaticBitSet) Clear() {
	s.init()
}
