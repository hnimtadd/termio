package stream

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestASCIIUTF8Decoder(t *testing.T) {
	d := NewUTF8Decoder()
	out := make([]byte, 13)
	for i, b := range []byte("Hello, World!") {
		cp, _, consumed := d.Next(b)
		if consumed {
			out[i] = byte(cp)
		}
	}
	assert.Equal(t, "Hello, World!", string(out))
}

func TestWellFormedUTF8Decoder(t *testing.T) {
	d := NewUTF8Decoder()
	out := []uint32{}

	for _, b := range []byte("😄✤ÁA") {
		consumed := false
		for !consumed {
			var cp uint32
			var generated bool

			cp, generated, consumed = d.Next(b)
			fmt.Printf(
				"byte: %x, cp: %x, generated: %v, consumed: %v\n",
				b,
				cp,
				generated,
				consumed,
			)
			if generated {
				out = append(out, cp)
			}
		}
	}
	assert.EqualValues(t, []uint32{0x1F604, 0x2724, 0xC1, 0x41}, out)
}

func TestPartiallyInvalidUTF8Decoder(t *testing.T) {
	d := NewUTF8Decoder()
	out := []uint32{}

	for _, b := range []byte("\xF0\x9F😄\xED\xA0\x80") {
		consumed := false
		for !consumed {
			var cp uint32
			var generated bool
			cp, generated, consumed = d.Next(b)
			if generated {
				out = append(out, cp)
			}
		}
	}
	assert.EqualValues(t, []uint32{0xFFFD, 0x1F604, 0xFFFD, 0xFFFD, 0xFFFD}, out)
}
