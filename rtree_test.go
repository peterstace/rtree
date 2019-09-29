package rtree_test

import (
	"reflect"
	"testing"

	"github.com/peterstace/rtree"
)

func SearchSingleNotFound(t *testing.T) {
}

func TestRTree(t *testing.T) {
	type insert func(*rtree.RTree)
	empty := func(*rtree.RTree) {}

	for _, tt := range []struct {
		name   string
		insert insert
		bbox   rtree.BBox
		want   []int
	}{
		{
			name:   "empty with search 0 0 0 0",
			insert: empty,
			bbox:   rtree.BBox{MinX: 0, MinY: 0, MaxX: 0, MaxY: 0},
			want:   nil,
		},
		{
			name:   "empty with search 0 0 1 1",
			insert: empty,
			bbox:   rtree.BBox{MinX: 0, MinY: 0, MaxX: 1, MaxY: 1},
			want:   nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rt, err := rtree.New(2, 4)
			if err != nil {
				t.Fatal(err)
			}
			tt.insert(&rt)
			var got []int
			rt.Search(tt.bbox, func(idx int) error {
				got = append(got, idx)
				return nil
			})
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got:%v want: %v", got, tt.want)
			}
		})
	}
}

// TODO: test for error propagation
