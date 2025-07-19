package ansi

// we ignore SOH/STX: https://github.com/microsoft/terminal/issues/10786
// and XTERM control sequence doesn't support them too:
// https://www.x.org/docs/xterm/ctlseqs.pdf
// TODO implement XON, XOFF, CAN, SUB
type c0 struct {
	NUL uint8 // NUL is the null character (Caret: ^@, Char: \0).
	BEL uint8 // BEL is the bell character (Caret: ^G, Char: \a).
	BS  uint8 // BS is the backspace character (Caret: ^H, Char: \b).
	CR  uint8 // CR is the carriage return character (Caret: ^M, Char: \r).
	ENQ uint8 // ENQ is the enquiry character (Caret: ^E).
	EOT uint8 // EOT is the end of transmission character (Caret: ^D).
	ESC uint8 // ESC is the Escape character (Caret: ^[).
	FF  uint8 // FF is the form feed character (Caret: ^L, Char: \f).
	HT  uint8 // HT is the horizontal tab character (Caret: ^I, Char: \t).
	LF  uint8 // LF is the line feed character (Caret: ^J, Char: \n).
	SI  uint8 // SI is the shift in character (Caret: ^O).
	SO  uint8 // SO is the shift out character (Caret: ^N).
	VT  uint8 // VT is the vertical tab character (Caret: ^K, Char: \v).
}

// C0 (7-bit) control character from ANSI.
//
// This is not complete, control character are only added to this
// as the terminal emulator handles them.
//
// see chapter 3 for detail information about control characters
// supported by KAI based on VT100, which is compatiable with ANSI standard:
// https://vt100.net/docs/vt100-ug/chapter3.html#S3.2
var C0 = c0{
	BEL: 0x07,
	BS:  0x08,
	CR:  0x0D,
	ENQ: 0x05,
	EOT: 0x04,
	ESC: 0x1b,
	FF:  0x0C,
	NUL: 0x00,
	HT:  0x09,
	LF:  0x0A,
	SI:  0x0F,
	SO:  0x0E,
	VT:  0x0B,
}
