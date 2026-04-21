package mtmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeMap(t *testing.T) {
	type Case struct {
		Msg string

		Input  []map[int]bool
		Output map[int]bool
	}
	cases := []Case{
		{
			Msg: "the map at the behind will override the previous map value if has the same key",
			Input: []map[int]bool{
				{1: false, 2: true},
				{1: true, 2: false},
			}, Output: map[int]bool{
				1: true, 2: false,
			},
		},
		{
			Input: []map[int]bool{
				{1: true, 2: false},
				{5: true, 6: false},
			},
			Output: map[int]bool{
				1: true, 2: false, 5: true, 6: false,
			},
		},
		{
			Input: []map[int]bool{
				{1: true, 2: false},
				{2: true, 6: false},
			},
			Output: map[int]bool{
				1: true, 2: true, 6: false,
			},
		},
	}
	for i := 0; i < len(cases); i++ {
		c := cases[i]
		merged := MergeMap(c.Input...)
		assert.Equalf(t, c.Output, merged, c.Msg)
	}
}
