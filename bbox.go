package rtree

import "math"

// BBox is an axis-aligned bounding box.
type BBox struct {
	MinX, MinY, MaxX, MaxY float64
}

// calculate bound calculates the smallest bounding box that fits a node.
func (t *RTree) calculateBound(n int) BBox {
	bb := t.Nodes[n].Entries[0].BBox
	for _, entry := range t.Nodes[n].Entries[1:] {
		bb = combine(bb, entry.BBox)
	}
	return bb
}

// combine gives the smallest bounding box containing both bbox1 and bbox2.
func combine(bbox1, bbox2 BBox) BBox {
	return BBox{
		MinX: math.Min(bbox1.MinX, bbox2.MinX),
		MinY: math.Min(bbox1.MinY, bbox2.MinY),
		MaxX: math.Max(bbox1.MaxX, bbox2.MaxX),
		MaxY: math.Max(bbox1.MaxY, bbox2.MaxY),
	}
}

// enlargment returns how much additional area the existing BBox would have to
// enlarge by to accomodate the additional BBox.
func enlargement(existing, additional BBox) float64 {
	return area(combine(existing, additional)) - area(existing)
}

func area(bb BBox) float64 {
	return (bb.MaxX - bb.MinX) * (bb.MaxY - bb.MinY)
}

func overlap(bbox1, bbox2 BBox) bool {
	return true &&
		(bbox1.MinX <= bbox2.MaxX) && (bbox1.MaxX >= bbox2.MinX) &&
		(bbox1.MinY <= bbox2.MaxY) && (bbox1.MaxY >= bbox2.MinY)
}
