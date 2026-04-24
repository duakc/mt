package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	type Case struct {
		InputSize  int
		OutputSize int
		OutputCap  int

		Msg string
	}
	newCase := func(i, o, oc int, msg string) Case {
		return Case{i, o, oc, msg}
	}

	cases := []Case{
		newCase(0, 0, 0, "zero"),
		newCase((1<<6)-1, (1<<6)-1, 1<<6, "minimal size"),
		newCase(1<<6, 1<<6, 1<<6, "minimal size"),
		newCase((1<<7)-1, (1<<7)-1, 1<<7, ""),
		newCase((1<<7)+1, (1<<7)+1, 1<<8, ""),

		newCase(MaxAllocatableSize, MaxAllocatableSize, MaxAllocatableSize, "maxsize"),
		newCase(MaxAllocatableSize+1, MaxAllocatableSize+1, MaxAllocatableSize+1, "maxsize plus one"),
	}

	for i := 0; i < len(cases); i++ {
		cc := cases[i]
		got := Get(cc.InputSize)
		assert.Equalf(t, cc.OutputSize, len(got),
			"InputSize=%d OutputSize: testCase.Index=%d "+cc.Msg, cc.InputSize, i)
		assert.Equalf(t, cc.OutputCap, cap(got),
			"InputSize=%d OutputCap: testCase.Index=%d "+cc.Msg, cc.InputSize, i)
		Put(got)
	}
}

func TestPut(t *testing.T) {
	type Case struct {
		Input int
		Can   bool
	}
	cases := []Case{
		{Input: 0, Can: false},
		{Input: 63, Can: true},
		{Input: 1 << 10, Can: true},
		{Input: MaxAllocatableSize, Can: true},
		{Input: MaxAllocatableSize + 1, Can: false},
	}
	for i := 0; i < len(cases); i++ {
		cc := cases[i]
		assert.Equalf(t, cc.Can, Put(Get(cc.Input)),
			"Input=%d testCase.Index=%d", cc.Input, i)
	}
}
