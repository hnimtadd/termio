// Provide handler type for control characters
//
// Since we might not have handlers to implement all the control chracters handlers
// Caller to terminal.Stream only needs to implement the handlers they want to
//
// I ported these handler into separate interfaces and use type assertion to
// detect if specific handler is implemented
//
// 1 potential approachs is to move these handlers into struct field
//
// Handler is named by control sequence abbr and
// contains method name is control sequence name
//
// E.g:
//
// - ehandler.CUU with CursorUp method
//
// - handler.CUD with CursorDown method
package handler
