package rtree

// Node is a node in an R-Tree. Nodes can either be leaf nodes holding entries
// for terminal items, or intermediate nodes holding entries for more nodes.
type Node struct {
	IsLeaf  bool
	Entries []Entry
}

// Entry is an entry under a node, leading either to terminal items, or more nodes.
type Entry struct {
	BBox  BBox
	Index int
}

// RTree is an in-memory R-Tree data structure. Its zero value is an empty R-Tree.
type RTree struct {
	RootIndex int
	Nodes     []Node
}

// Search looks for any items in the tree that overlap with the the given
// bounding box. The callback is called with the item index for each found
// item.
func (t *RTree) Search(bb BBox, callback func(index int)) {
	if len(t.Nodes) == 0 {
		return
	}
	var recurse func(*Node)
	recurse = func(n *Node) {
		for _, entry := range n.Entries {
			if !overlap(entry.BBox, bb) {
				continue
			}
			if n.IsLeaf {
				callback(entry.Index)
			} else {
				recurse(&t.Nodes[entry.Index])
			}
		}
	}
	recurse(&t.Nodes[t.RootIndex])
}
