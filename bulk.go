package rtree

import "sort"

// InsertItem is an item that can be inserted for bulk loading.
type InsertItem struct {
	BBox      BBox
	DataIndex int
}

// BulkLoad bulk loads multiple items into a new R-Tree. The bulk load
// operation is optimised for creating R-Trees with minimal node overlap. This
// allows for fast searching.
func BulkLoad(inserts []InsertItem) RTree {
	var tr RTree
	// Find any existing entries, and add them to the new list.
	items := make([]InsertItem, len(inserts))
	copy(items, inserts)
	for _, node := range tr.Nodes {
		if !node.IsLeaf {
			continue
		}
		for _, entry := range node.Entries {
			items = append(items, InsertItem{
				entry.BBox, entry.Index,
			})
		}
	}

	n := tr.bulkInsert(items)
	tr.RootIndex = n
	return tr
}

func (t *RTree) bulkInsert(items []InsertItem) int {
	if len(items) <= 2 {
		node := Node{IsLeaf: true, Parent: -1}
		for _, item := range items {
			node.Entries = append(node.Entries, Entry{
				BBox:  item.BBox,
				Index: item.DataIndex,
			})
		}
		t.Nodes = append(t.Nodes, node)
		return len(t.Nodes) - 1
	}

	bbox := items[0].BBox
	for _, item := range items[1:] {
		bbox = combine(bbox, item.BBox)
	}

	horizontal := bbox.MaxX-bbox.MinX > bbox.MaxY-bbox.MinY
	sort.Slice(items, func(i, j int) bool {
		bi := items[i].BBox
		bj := items[j].BBox
		if horizontal {
			return bi.MinX+bi.MaxX < bj.MinX+bj.MaxX
		} else {
			return bi.MinY+bi.MaxY < bj.MinY+bj.MaxY
		}
	})

	split := len(items) / 2
	n1 := t.bulkInsert(items[:split])
	n2 := t.bulkInsert(items[split:])

	parent := Node{IsLeaf: false, Parent: -1, Entries: []Entry{
		Entry{BBox: t.calculateBound(n1), Index: n1},
		Entry{BBox: t.calculateBound(n2), Index: n2},
	}}
	t.Nodes = append(t.Nodes, parent)
	t.Nodes[n1].Parent = len(t.Nodes) - 1
	t.Nodes[n2].Parent = len(t.Nodes) - 1
	return len(t.Nodes) - 1
}
