package styleid

// The unique identifier for a style. This is at most the number of cells
// that can fit into a terminal page.
type ID uint64

const DefaultID ID = 0
