package utils

func AddWithOverflow(a int, b int) (int, bool) {
	// Check for overflow
	if (a > 0 && b > 0 && a > (1<<16)-1-b) ||
		(a < 0 && b < 0 && a < -(1<<16)-b) {
		return 0, true
	}

	return a + b, false
}
