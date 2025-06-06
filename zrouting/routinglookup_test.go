package zrouting

import (
	"testing"

	"gotest.tools/assert"
)

type pathFilterTest struct {
	path     []uint32
	filter   PathFilter
	expected []uint32
}

// 64512 - 65534

func TestIBGPFilter(t *testing.T) {
	tests := []pathFilterTest{
		{
			path:     []uint32{1, 2, 3, 4},
			filter:   IdentityPathFilter,
			expected: []uint32{1, 2, 3, 4},
		},
		{
			path:     []uint32{1, 65000, 38},
			filter:   InternalBGPPathFilter(38),
			expected: []uint32{1, 38},
		},
		{
			path:     []uint32{1, 65000},
			filter:   InternalBGPPathFilter(38),
			expected: []uint32{1, 38},
		},
		{
			path:     []uint32{1, 65000, 64512, 65118, 38, 27},
			filter:   InternalBGPPathFilter(38),
			expected: []uint32{1, 38, 27},
		},
		{
			path:     []uint32{1, 65000, 64512, 65118},
			filter:   InternalBGPPathFilter(38),
			expected: []uint32{1, 38},
		},
		{
			path:     []uint32{1, 2, 2, 64512, 64512, 38, 3, 3},
			filter:   InternalBGPPathFilter(38),
			expected: []uint32{1, 2, 2, 38, 3, 3},
		},
		{
			path:     []uint32{1, 2, 2, 64512, 64512, 3, 3},
			filter:   InternalBGPPathFilter(38),
			expected: []uint32{1, 2, 2, 38, 3, 3},
		},
	}
	for _, test := range tests {
		actual := test.filter(test.path)
		assert.DeepEqual(t, test.expected, actual)
	}
}
