package ansi

import "fmt"

// table is a map of ANSI control characters to their names.
// any unsupported ansi characters will have hex value key.
var table = map[uint8]string{
	C0.NUL: "NUL", // Null
	0x01:   "SOH", // Start of Heading
	0x02:   "STX", // Start of Text
	0x03:   "ETX", // End of Text
	C0.EOT: "EOT", // End of Transmission
	C0.ENQ: "ENQ", // Enquiry
	0x06:   "ACK", // Acknowledge
	C0.BEL: "BEL", // Bell
	C0.BS:  "BS",  // Backspace
	C0.HT:  "HT",  // Horizontal Tab
	C0.LF:  "LF",  // Line Feed
	C0.VT:  "VT",  // Vertical Tab
	C0.FF:  "FF",  // Form Feed
	C0.CR:  "CR",  // Carriage Return
	C0.SO:  "SO",  // Shift Out
	C0.SI:  "SI",  // Shift In
	0x10:   "DLE", // Data Link Escape
	0x11:   "DC1", // Device Control 1
	0x12:   "DC2", // Device Control 2
	0x13:   "DC3", // Device Control 3
	0x14:   "DC4", // Device Control 4
	0x15:   "NAK", // Negative Acknowledge
	0x16:   "SYN", // Synchronous Idle
	0x17:   "ETB", // End of Transmission Block
	0x18:   "CAN", // Cancel
	0x19:   "EM",  // End of Medium
	0x1A:   "SUB", // Substitute
	0x1B:   "ESC", // Escape
	0x1C:   "FS",  // File Separator
	0x1D:   "GS",  // Group Separator
	0x1E:   "RS",  // Record Separator
	0x1F:   "US",  // Unit Separator
	0x7F:   "DEL", // Delete
}

func String(val uint8) string {
	if name, ok := table[val]; ok {
		return fmt.Sprintf("%s (0x%02X) (%q)", name, val, rune(val))
	}
	return fmt.Sprintf("0x%02X (%q)", val, rune(val))
}
