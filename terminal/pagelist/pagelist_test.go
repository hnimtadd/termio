package pagelist

import (
	"testing"

	"github.com/hnimtadd/termio/coordinate"
	"github.com/hnimtadd/termio/terminal/page"
	"github.com/hnimtadd/termio/terminal/point"
	"github.com/hnimtadd/termio/terminal/size"
	"github.com/stretchr/testify/assert"
)

func TestPageList(t *testing.T) {
	s := NewPageList(80, 24)
	assert.Equal(t, s.ViewPort, ViewportTagActive)
	assert.NotNil(t, s.Pages.First)
	assert.EqualValues(t, s.Rows, s.totalRows())
	// Active area should be the top
	assert.EqualValues(t, &Pin{
		Node: s.Pages.First,
		Y:    0,
		X:    0,
	}, s.GetTopLeft(point.TagActive))
}

func TestPageListInitRowsAcrossTwoPages(t *testing.T) {
	const rows = 100
	cap := page.StandardCapacity
	err := cap.Adjust(page.Adjustment{
		Cols: 50,
	})
	assert.NoError(t, err)
	for cap.Rows >= rows && err == nil {
		err = cap.Adjust(page.Adjustment{
			Cols: cap.Cols + 50,
		})
	}
	assert.NoError(t, err)
	// Init

	s := NewPageList(cap.Cols, rows)
	assert.Equal(t, s.ViewPort, ViewportTagActive)
	assert.NotNil(t, s.Pages.First)
	assert.NotNil(t, s.Pages.First.Next)
	assert.EqualValues(t, s.Rows, s.totalRows())
}

func TestPageListPointFromPinActive(t *testing.T) {
	s := NewPageList(80, 24)

	// Active area should be the top
	assert.EqualValues(t, &point.Point{
		Tag: point.TagActive,
		Coordinate: coordinate.Point[size.CellCountInt]{
			Y: 0,
			X: 0,
		},
	}, s.pointFromPin(point.TagActive, Pin{
		Node: s.Pages.First,
		Y:    0,
		X:    0,
	}))

	assert.EqualValues(t, &point.Point{
		Tag: point.TagActive,
		Coordinate: coordinate.Point[size.CellCountInt]{
			Y: 2,
			X: 4,
		},
	}, s.pointFromPin(point.TagActive, Pin{
		Node: s.Pages.First,
		Y:    2,
		X:    4,
	}))
}

func TestPageListPointFromPinActiveWithHistory(t *testing.T) {
	s := NewPageList(80, 24)
	err := s.growRows(30)
	assert.NoError(t, err)

	// Active area should be the top
	assert.EqualValues(t, &point.Point{
		Tag: point.TagActive,
		Coordinate: coordinate.Point[size.CellCountInt]{
			Y: 0,
			X: 2,
		},
	}, s.pointFromPin(point.TagActive, Pin{
		Node: s.Pages.First,
		Y:    30,
		X:    2,
	}))

	// In history, invalid
	assert.Nil(t, s.pointFromPin(point.TagActive, Pin{
		Node: s.Pages.First,
		Y:    21,
		X:    2,
	}))
}
