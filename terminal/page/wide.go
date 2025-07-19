package page

type Wide int

const (
	// Not a wide character, cell width 1
	WideNarrow Wide = iota

	// WideWide character, cell width 2
	WideWide

	// Spacer after wide character. Do not render
	WideSpacerTail

	// Spacer before wide character. Do not render.
	//
	// This is Spacer at the end of a soft-wrapped line to indicate that a
	// wide character is continued on the next line..
	WideSpacerHead
)
