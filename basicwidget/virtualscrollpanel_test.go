// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget_test

import (
	"testing"

	"github.com/guigui-gui/guigui/basicwidget"
)

func TestTopItemAfterPixelScroll(t *testing.T) {
	type testCase struct {
		name        string
		heights     []int
		startIndex  int
		startOffset int
		deltaPx     int
		wantIndex   int
		wantOffset  int
	}

	uniform := func(n, h int) []int {
		s := make([]int, n)
		for i := range s {
			s[i] = h
		}
		return s
	}

	testCases := []testCase{
		{
			name:        "zero delta is a no-op",
			heights:     uniform(10, 20),
			startIndex:  3,
			startOffset: -5,
			deltaPx:     0,
			wantIndex:   3,
			wantOffset:  -5,
		},
		{
			name:        "uniform forward, exact viewport",
			heights:     uniform(20, 20),
			startIndex:  0,
			startOffset: 0,
			deltaPx:     100,
			wantIndex:   5,
			wantOffset:  0,
		},
		{
			name:        "uniform forward, mid-item landing",
			heights:     uniform(20, 20),
			startIndex:  0,
			startOffset: 0,
			deltaPx:     105,
			wantIndex:   5,
			wantOffset:  -5,
		},
		{
			name:        "uniform forward from non-zero offset",
			heights:     uniform(20, 20),
			startIndex:  2,
			startOffset: -10,
			deltaPx:     100,
			wantIndex:   7,
			wantOffset:  -10,
		},
		{
			name:        "uniform backward overshoots top, clamped",
			heights:     uniform(20, 20),
			startIndex:  3,
			startOffset: 0,
			deltaPx:     -200,
			wantIndex:   0,
			wantOffset:  0,
		},
		{
			name:        "uniform backward, exact",
			heights:     uniform(20, 20),
			startIndex:  5,
			startOffset: 0,
			deltaPx:     -60,
			wantIndex:   2,
			wantOffset:  0,
		},
		{
			name:        "uniform backward, mid-item landing",
			heights:     uniform(20, 20),
			startIndex:  5,
			startOffset: 0,
			deltaPx:     -65,
			wantIndex:   1,
			wantOffset:  -15,
		},
		{
			// Forward inside an item taller than the viewport: index
			// stays put, only offset advances. The original bug.
			name:        "tall single item, forward keeps same index",
			heights:     []int{500},
			startIndex:  0,
			startOffset: 0,
			deltaPx:     100,
			wantIndex:   0,
			wantOffset:  -100,
		},
		{
			name:        "tall single item, repeated forward accumulates offset",
			heights:     []int{500},
			startIndex:  0,
			startOffset: -100,
			deltaPx:     100,
			wantIndex:   0,
			wantOffset:  -200,
		},
		{
			// Backward by less than the offset stays inside the same item.
			name:        "tall single item, backward partial",
			heights:     []int{500},
			startIndex:  0,
			startOffset: -200,
			deltaPx:     -100,
			wantIndex:   0,
			wantOffset:  -100,
		},
		{
			name:        "heterogeneous heights forward",
			heights:     []int{10, 20, 30, 40, 50},
			startIndex:  0,
			startOffset: 0,
			deltaPx:     35,
			wantIndex:   2,
			wantOffset:  -5,
		},
		{
			name:        "heterogeneous heights backward",
			heights:     []int{10, 20, 30, 40, 50},
			startIndex:  4,
			startOffset: 0,
			deltaPx:     -55,
			wantIndex:   2,
			wantOffset:  -15,
		},
		{
			// Zero is a valid height (e.g. a collapsed item). The walk
			// must step past it without consuming any of the offset.
			name:        "zero-height items are walked past",
			heights:     []int{0, 0, 20, 20},
			startIndex:  0,
			startOffset: 0,
			deltaPx:     5,
			wantIndex:   2,
			wantOffset:  -5,
		},
		{
			// Layout's bottom clamp finishes the job; the walk just stops.
			name:        "forward stops at totalCount-1",
			heights:     uniform(5, 20),
			startIndex:  0,
			startOffset: 0,
			deltaPx:     1000,
			wantIndex:   4,
			wantOffset:  -920,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			measure := func(i int) int {
				return tc.heights[i]
			}
			gotIdx, gotOff := basicwidget.TopItemAfterPixelScroll(measure, len(tc.heights), tc.startIndex, tc.startOffset, tc.deltaPx)
			if gotIdx != tc.wantIndex || gotOff != tc.wantOffset {
				t.Errorf("TopItemAfterPixelScroll start=(%d,%d) delta=%d => (%d,%d); want (%d,%d)",
					tc.startIndex, tc.startOffset, tc.deltaPx, gotIdx, gotOff, tc.wantIndex, tc.wantOffset)
			}
		})
	}
}
