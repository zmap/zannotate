/*
 * ZAnnotate Copyright 2025 Regents of the University of Michigan
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
 * implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

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
