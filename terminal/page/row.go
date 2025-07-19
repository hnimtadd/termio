package page

import "github.com/hnimtadd/termio/terminal/size"

type Row struct {
	// The cells in the row offset from the page.
	Cells []*Cell
	// Whether the row is wrapped
	Wrap bool
	// Whether the row is a continuation of a wrapped line
	WrapContinuation bool
	Y                size.CellCountInt

	// True if any of the cells in this row have a ref-counted style.
	// This can have false positives but never a false negative. Meaning:
	// this will be set to true the first time a style is used, but it
	// will not be set to false if the style is no longer used, because
	// checking for that condition is too expensive.
	//
	// Why have this weird false positive flag at all? This makes VT operations
	// that erase cells (such as insert lines, delete lines, erase chars,
	// etc.) MUCH MUCH faster in the case that the row was never styled.
	// At the time of writing this, the speed difference is around 4x.
	Styled bool

	// The semantic prompt type for this row as specified by the running
	// program, or "unknow" if it was never set.
	SemanticPrompt SemanticPromptType
}

type RAC struct {
	Row  *Row
	Cell *Cell
}
