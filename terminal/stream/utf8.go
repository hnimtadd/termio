package stream

import "github.com/hnimtadd/termio/terminal/ansi"

// This UTF8Decoder is a state machine that decodes UTF-8 sequences.
//
// This implementation is mainly based on implementation of Bjoern Hoehrmann
// here:  http://bjoern.hoehrmann.de/utf-8/decoder/dfa
// with support error replacement
type UTF8Decoder struct {
	state       uint8
	accumulator uint32
}

func NewUTF8Decoder() *UTF8Decoder {
	return &UTF8Decoder{
		state:       stateUTF8Accept,
		accumulator: 0,
	}
}

const (
	stateUTF8Accept = 0
	stateUTF8Reject = 12
)

var utf8d = [364]uint8{
	// The first part is maps bytes to character classes
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	8, 8, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
	10, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 4, 3, 3, 11, 6, 6, 6, 5, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8,

	// The second part transition table that maps a combination
	// of a state of the automaton and a character class to a state.
	0, 12, 24, 36, 60, 96, 84, 12, 12, 12, 48, 72, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12,
	12, 0, 12, 12, 12, 12, 12, 0, 12, 0, 12, 12, 12, 24, 12, 12, 12, 12, 12, 24, 12, 24, 12, 12,
	12, 12, 12, 12, 12, 12, 12, 24, 12, 12, 12, 12, 12, 24, 12, 12, 12, 12, 12, 12, 12, 24, 12, 12,
	12, 12, 12, 12, 12, 12, 12, 36, 12, 36, 12, 12, 12, 36, 12, 12, 12, 12, 12, 36, 12, 36, 12, 12,
	12, 36, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12,
}

// Decode the UTF-8 text in input into output until an escape
// character 0x1B is found. This returns the number of bytes consumed
// from input and writes the number of decoded characters into
// output_count.
//
// This may return a value less than len( input ) even with no escape
// character if the input ends with an incomplete UTF-8 sequence.
// The caller should check the next byte manually to determine
// if it is incomplete.
func (d *UTF8Decoder) DecodeUntilControlSeq(
	input []uint8,
	cpBuf []uint32,
) (decoded int, consumed int) {
	decoded = 0
	consumed = 0
	for _, c := range input {
		if c == ansi.C0.ESC {
			// We have an ESC char, decode up to this point. We start by assuming
			// a valid UTF-8 sequence and slow-path into error handling if we find
			// an invalid sequence.
			return decoded, consumed
		}

		cp, generated, isConsumed := d.Next(uint8(c))
		if generated {
			cpBuf[decoded] = cp
			decoded++
		}
		if !isConsumed {
			continue
		}
		consumed++
	}
	// We don't see any escape character
	return decoded, consumed
}

// Takes the next byte in the utf-8 sequence and emits a tuple of
// - The codepoint that was generated, if there is one.
// - The boolean that indicates whether the codepoint was generated.
// - A boolean that indicates whether the provided byte was consumed.
//
// The only case where the byte is not consumed is if an ill-formed
// sequence is reached, in which case a replacement character will be
// emitted and the byte will not be consumed.
//
// If the byte is not consumed, the caller is responsible for calling
// again with the same byte before continuing.
func (d *UTF8Decoder) Next(c uint8) (cp uint32, generated bool, consumed bool) {
	typ := utf8d[c]

	initial := d.state

	if d.state != stateUTF8Accept {
		d.accumulator <<= 6
		d.accumulator |= (uint32(c) & 0x3F)
	} else {
		d.accumulator = (uint32(0xFF) >> typ) & (uint32(c))
	}
	d.state = utf8d[256+int(d.state)+int(typ)]

	switch d.state {
	case stateUTF8Accept:
		defer func() { d.accumulator = 0 }()
		// Emit the fully decoded codepoint.
		return d.accumulator, true, true

	case stateUTF8Reject:
		d.accumulator = 0
		d.state = stateUTF8Accept

		// Emit a replacement character. If we rejected the first byte in
		// a sequence, then it was consumed, otherwise it was not.
		return 0xFFFD, true, initial == stateUTF8Accept

	default:
		return 0, false, true
	}
}
