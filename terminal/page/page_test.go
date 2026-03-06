package page

import (
	"testing"

	"github.com/hnimtadd/termio/terminal/size"
	styleid "github.com/hnimtadd/termio/terminal/style/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageCreation(t *testing.T) {
	cap := Capacity{
		Cols:   80,
		Rows:   24,
		Styles: 100,
	}

	page := NewPage(cap)
	require.NotNil(t, page)
	assert.Equal(t, cap.Cols, page.Size.Cols)
	assert.Equal(t, cap.Rows, page.Size.Rows)
	assert.Len(t, page.Rows, int(cap.Rows))
	
	// Check that rows are properly initialized
	for i, row := range page.Rows {
		assert.NotNil(t, row, "Row %d should not be nil", i)
		assert.Len(t, row.Cells, int(cap.Cols))
		
		// Check that cells are properly initialized
		for j, cell := range row.Cells {
			assert.NotNil(t, cell, "Cell [%d][%d] should not be nil", i, j)
			assert.Equal(t, ContentTagCP, cell.ContentTag)
			assert.Equal(t, uint32(0), cell.ContentCP)
			assert.Equal(t, styleid.DefaultID, cell.StyleID)
		}
	}
}

func TestCapacitySize(t *testing.T) {
	tests := []struct {
		name string
		cap  Capacity
	}{
		{
			name: "Small terminal",
			cap:  Capacity{Cols: 80, Rows: 24, Styles: 100},
		},
		{
			name: "Large terminal", 
			cap:  Capacity{Cols: 200, Rows: 60, Styles: 1000},
		},
		{
			name: "Minimal terminal",
			cap:  Capacity{Cols: 1, Rows: 1, Styles: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := tt.cap.Size()
			assert.Greater(t, size, uint64(0))
			
			// Size should increase with more cells/styles
			expectedCells := uint64(tt.cap.Cols) * uint64(tt.cap.Rows)
			assert.Greater(t, size, expectedCells) // Should be at least the cell count
		})
	}
}

func TestCapacityAdjust(t *testing.T) {
	cap := Capacity{
		Cols:   80,
		Rows:   24,
		Styles: 100,
	}

	// Test valid adjustment
	err := cap.Adjust(Adjustment{Cols: 100})
	assert.NoError(t, err)
	assert.Equal(t, size.CellCountInt(100), cap.Cols)
	
	// Rows should be adjusted to maintain total cells
	expectedRows := (80 * 24) / 100
	assert.Equal(t, size.CellCountInt(expectedRows), cap.Rows)
}

func TestCapacityAdjustOutOfMemory(t *testing.T) {
	cap := Capacity{
		Cols:   2,
		Rows:   1,
		Styles: 100,
	}

	// This should fail because we can't fit any rows with such large columns
	err := cap.Adjust(Adjustment{Cols: 1000})
	assert.Error(t, err)
	assert.Equal(t, ErrOutOfMemory, err)
}

func TestMoveCells(t *testing.T) {
	page := NewPage(Capacity{Cols: 10, Rows: 5, Styles: 100})
	
	// Set up source row with test data
	srcRow := page.Rows[0]
	dstRow := page.Rows[1]
	
	// Fill source cells with test data
	for i := 0; i < 5; i++ {
		srcRow.Cells[i].ContentCP = uint32('A' + i)
		srcRow.Cells[i].StyleID = styleid.ID(i + 1)
	}
	
	// Move cells
	page.MoveCells(srcRow, 1, dstRow, 2, 3)
	
	// Verify destination cells received the data
	assert.Equal(t, uint32('B'), dstRow.Cells[2].ContentCP)
	assert.Equal(t, uint32('C'), dstRow.Cells[3].ContentCP)
	assert.Equal(t, uint32('D'), dstRow.Cells[4].ContentCP)
	
	// Verify source cells were cleared
	assert.Equal(t, uint32(0), srcRow.Cells[1].ContentCP)
	assert.Equal(t, uint32(0), srcRow.Cells[2].ContentCP)
	assert.Equal(t, uint32(0), srcRow.Cells[3].ContentCP)
}

func TestSwapCells(t *testing.T) {
	page := NewPage(Capacity{Cols: 10, Rows: 5, Styles: 100})
	
	cell1 := page.Rows[0].Cells[0]
	cell2 := page.Rows[0].Cells[1]
	
	// Set up test data
	cell1.ContentCP = 'A'
	cell1.StyleID = 1
	cell2.ContentCP = 'B'
	cell2.StyleID = 2
	
	// Swap cells
	page.SwapCells(cell1, cell2)
	
	// Verify swap occurred
	assert.Equal(t, uint32('B'), cell1.ContentCP)
	assert.Equal(t, styleid.ID(2), cell1.StyleID)
	assert.Equal(t, uint32('A'), cell2.ContentCP)
	assert.Equal(t, styleid.ID(1), cell2.StyleID)
	
	// Both should be marked dirty
	assert.True(t, cell1.IsDirty)
	assert.True(t, cell2.IsDirty)
}

func TestClonePartialRowFrom(t *testing.T) {
	srcPage := NewPage(Capacity{Cols: 10, Rows: 5, Styles: 100})
	dstPage := NewPage(Capacity{Cols: 10, Rows: 5, Styles: 100})
	
	srcRow := srcPage.Rows[0]
	dstRow := dstPage.Rows[0]
	
	// Fill source row with test data
	for i := 0; i < 5; i++ {
		srcRow.Cells[i].ContentCP = uint32('A' + i)
		srcRow.Cells[i].StyleID = styleid.ID(i + 1)
	}
	
	// Clone partial row
	err := dstPage.ClonePartialRowFrom(srcPage, dstRow, srcRow, 1, 4)
	assert.NoError(t, err)
	
	// Verify cloned data
	assert.Equal(t, uint32('B'), dstRow.Cells[1].ContentCP)
	assert.Equal(t, uint32('C'), dstRow.Cells[2].ContentCP)
	assert.Equal(t, uint32('D'), dstRow.Cells[3].ContentCP)
	
	// Verify cells outside range weren't affected
	assert.Equal(t, uint32(0), dstRow.Cells[0].ContentCP)
	assert.Equal(t, uint32(0), dstRow.Cells[4].ContentCP)
}

func TestClonePartialRowFromBounds(t *testing.T) {
	page := NewPage(Capacity{Cols: 5, Rows: 5, Styles: 100})
	
	srcRow := page.Rows[0]
	dstRow := page.Rows[1]
	
	// Test invalid bounds
	err := page.ClonePartialRowFrom(page, dstRow, srcRow, 3, 2) // left >= right
	assert.Error(t, err)
	
	err = page.ClonePartialRowFrom(page, dstRow, srcRow, 0, 10) // right > cols
	assert.Error(t, err)
}

func TestPageAssertIntegrity(t *testing.T) {
	page := NewPage(Capacity{Cols: 80, Rows: 24, Styles: 100})
	
	// Should not panic with valid page
	assert.NotPanics(t, func() {
		page.AssertIntegrity()
	})
}

func TestCellMethods(t *testing.T) {
	cell := &Cell{
		ContentTag: ContentTagCP,
		ContentCP:  'A',
		Wide:       WideNarrow,
		StyleID:    1,
	}
	
	// Test basic methods
	assert.Equal(t, uint32('A'), cell.Codepoint())
	assert.Equal(t, uint8(1), cell.Width())
	assert.False(t, cell.IsEmpty())
	assert.True(t, cell.HasText())
	
	// Test empty cell
	emptyCell := &Cell{
		ContentTag: ContentTagCP,
		ContentCP:  0,
	}
	assert.True(t, emptyCell.IsEmpty())
	assert.False(t, emptyCell.HasText())
	
	// Test wide cell
	wideCell := &Cell{
		ContentTag: ContentTagCP,
		ContentCP:  'あ',
		Wide:       WideWide,
	}
	assert.Equal(t, uint8(2), wideCell.Width())
}