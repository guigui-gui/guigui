// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget_test

import (
	"math"
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

func TestVirtualScrollPanelScrollPositionRoundTrip(t *testing.T) {
	var p basicwidget.VirtualScrollPanel

	// ForceSetTopItem (cancelAnimation=true) writes the vertical position
	// instantly; this is the path List.ForceSetScrollPosition uses.
	p.ForceSetTopItem(7, -12, true)
	if idx, off := p.TopItem(); idx != 7 || off != -12 {
		t.Errorf("after ForceSetTopItem, TopItem() = (%d, %d); want (7, -12)", idx, off)
	}

	// SetTopItem falls back to a pending instant change before the first draw;
	// this is the path List.SetScrollPosition uses.
	p.SetTopItem(3, -4)
	if idx, off := p.TopItem(); idx != 7 || off != -12 {
		t.Errorf("after SetTopItem before apply, TopItem() = (%d, %d); want (7, -12)", idx, off)
	}
	p.ApplyPendingScrollOffset()
	if idx, off := p.TopItem(); idx != 3 || off != -4 {
		t.Errorf("after SetTopItem and apply, TopItem() = (%d, %d); want (3, -4)", idx, off)
	}

	// forceSetScrollOffsetX queues a horizontal change applied on the next
	// layout/tick; scrollOffset reports it only once applied.
	p.ForceSetScrollOffsetX(-34)
	if x, _ := p.ScrollOffset(); x != 0 {
		t.Errorf("ScrollOffset() before apply = %v; want 0", x)
	}
	p.ApplyPendingScrollOffset()
	if x, y := p.ScrollOffset(); x != -34 || y != 0 {
		t.Errorf("ScrollOffset() after apply = (%v, %v); want (-34, 0)", x, y)
	}
}

func TestBottomFracIdx(t *testing.T) {
	uniform := func(n, h int) []int {
		s := make([]int, n)
		for i := range s {
			s[i] = h
		}
		return s
	}

	testCases := []struct {
		name           string
		heights        []int
		viewportHeight int
		want           float64
	}{
		{
			name:           "empty list",
			heights:        nil,
			viewportHeight: 100,
			want:           0,
		},
		{
			name:           "zero viewport",
			heights:        uniform(10, 20),
			viewportHeight: 0,
			want:           0,
		},
		{
			name:           "content fits without scrolling",
			heights:        uniform(5, 20),
			viewportHeight: 200,
			want:           0,
		},
		{
			// 5 items fit in the viewport, so the canonical-bottom top
			// item is index 5 with offset 0.
			name:           "uniform heights, exact viewport-multiple",
			heights:        uniform(10, 20),
			viewportHeight: 100,
			want:           5,
		},
		{
			// At canonical bottom, the last item's bottom hits the
			// viewport bottom; walking up by 95px lands 5px above the
			// top of item 5. topIdx=5, topOff=-5, fracIdx=5.25.
			name:           "uniform heights, mid-item bottom",
			heights:        uniform(10, 20),
			viewportHeight: 95,
			want:           5.25,
		},
		{
			// Single item taller than the viewport: max fracIdx is the
			// fraction of the item scrolled off-screen at canonical bottom.
			name:           "single tall item",
			heights:        []int{500},
			viewportHeight: 200,
			want:           0.6,
		},
		{
			// One huge item dominates the average. The old
			// totalCount - viewport/avgH approximation gave ~6.17 here,
			// well above any reachable fracIdx, so the thumb saturated
			// partway down the track even at canonical bottom.
			name:           "tall item among small ones",
			heights:        []int{30, 30, 30, 1000, 30, 30, 30, 30, 30, 30, 30},
			viewportHeight: 570,
			want:           3 + 640.0/1000.0,
		},
		{
			// Zero-height items at the tail are walked past without
			// consuming viewport space, so the canonical top is the
			// first non-zero ancestor.
			name:           "zero-height items at tail",
			heights:        []int{20, 20, 20, 20, 20, 0, 0},
			viewportHeight: 60,
			want:           2,
		},
	}

	const epsilon = 1e-9
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			measure := func(i int) int {
				return tc.heights[i]
			}
			got := basicwidget.BottomFracIdx(measure, len(tc.heights), tc.viewportHeight)
			if math.Abs(got-tc.want) > epsilon {
				t.Errorf("BottomFracIdx(viewport=%d) = %v; want %v",
					tc.viewportHeight, got, tc.want)
			}
		})
	}
}
