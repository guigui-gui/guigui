// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
	"slices"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// TestVisualLinesFromCachedStartsMatchesVisualLines verifies that the
// cache-backed Draw build reproduces the shaping packer's visual lines exactly,
// across wrap modes, widths, and trailing-break / empty-line cases.
func TestVisualLinesFromCachedStartsMatchesVisualLines(t *testing.T) {
	face := newTestFace(t)

	strs := []string{
		"",
		"abc",
		"abc\n",
		"abc\ndef",
		"abc\ndef\n",
		"the quick brown fox jumps over the lazy dog",
		"the quick brown fox\njumps over the lazy dog\n",
		"一二三四五六七八九十\n",
		"a\nb\nc\n",
		"word",
		"\n",
		"\n\n",
		"trailing spaces   \nmore",
		"col\tone\ttwo three four five six seven eight",
		"\ta\tb\tcdef ghij klmn opqr stuv wxyz more words here\n",
		"abc אבג def דה ghi", // LTR with embedded RTL (Hebrew) runs
		"אבג דהו זח",         // RTL (Hebrew) text that wraps
	}
	widths := []int{40, 80, 200, 100000}
	wrapModes := []textutil.WrapMode{textutil.WrapModeNormal, textutil.WrapModeAnywhere}
	tabWidths := []float64{0, 32}
	keepTailings := []bool{false, true}

	for _, wrapMode := range wrapModes {
		for _, width := range widths {
			for _, tabWidth := range tabWidths {
				for _, keepTailingSpace := range keepTailings {
					for _, str := range strs {
						adv := func(s string, idx int) float64 {
							return textutil.AdvanceForTestParams(s, idx, face, tabWidth, keepTailingSpace)
						}
						var want []textutil.VisualLine
						for vl := range textutil.VisualLines(width, str, wrapMode, adv) {
							want = append(want, vl)
						}
						got, ok := textutil.VisualLinesFromCachedStarts(width, str, wrapMode, face, tabWidth, keepTailingSpace)
						if !ok {
							t.Errorf("VisualLinesFromCachedStarts ok=false for str=%q width=%d wrap=%v tab=%v keep=%v", str, width, wrapMode, tabWidth, keepTailingSpace)
							continue
						}
						if !slices.Equal(got, want) {
							t.Errorf("mismatch str=%q width=%d wrap=%v tab=%v keep=%v\n got=%v\nwant=%v", str, width, wrapMode, tabWidth, keepTailingSpace, got, want)
						}
					}
				}
			}
		}
	}
}

// TestTextPositionFromIndexMatchesShaping verifies that TextPositionFromIndex,
// which resolves a wrapped line through the content-keyed layout cache
// (visual-line starts), produces the same caret positions as the shaping packer
// (TextPositionFromIndexInLogicalLine), across every byte index of several
// single-logical-line wrapped strings. A face with a recipe is used so the cache
// applies.
func TestTextPositionFromIndexMatchesShaping(t *testing.T) {
	face := newTestFace(t)
	const width = 80
	style := textutil.Style{
		Face:       face,
		LineHeight: 24,
		WrapMode:   textutil.WrapModeNormal,
	}

	strs := []string{
		"the quick brown fox jumps over the lazy dog and then some more text to wrap",
		"wrap me wrap me wrap me and another long line that should wrap a few times",
		"一二三四五六七八九十一二三四五六七八九十短い行をここに置く",
	}

	for _, str := range strs {
		var lbo textutil.LineByteOffsets
		rebuildFromString(&lbo, str)
		rng := func(start, end int) string { return str[start:end] }

		for idx := 0; idx <= len(str); idx++ {
			p0a, p1a, ca := textutil.TextPositionFromIndex(&textutil.TextLayoutParams{
				RenderingTextRange:         rng,
				RenderingTextLength:        len(str),
				Width:                      width,
				Style:                      style,
				PrecomputedLineByteOffsets: &lbo,
			}, idx)
			p0b, p1b, cb := textutil.TextPositionFromIndexInLogicalLine(width, str, idx, &style)
			if ca != cb || p0a != p0b || p1a != p1b {
				t.Errorf("str=%q idx=%d: cache=(%v,%v,%d) shaping=(%v,%v,%d)",
					str, idx, p0a, p1a, ca, p0b, p1b, cb)
			}
		}
	}
}

// TestLayoutCacheEviction verifies that idle entries are swept once the cache is
// over its soft limit, while recently-used entries survive.
func TestLayoutCacheEviction(t *testing.T) {
	var tick int64
	c := textutil.NewLayoutCacheForTest(2, func() int64 { return tick })

	c.Touch("a")
	c.Touch("b")
	if got := c.Len(); got != 2 {
		t.Fatalf("Len = %d, want 2", got)
	}

	// Past the alive window, keep "a" warm and add a third entry to trip the
	// over-limit sweep: "b" (idle) is evicted, "a" and "c" survive.
	tick = int64(textutil.EntryAliveTicks) + 1
	c.Touch("a")
	c.Touch("c")
	if c.Has("b") {
		t.Error("idle entry b should have been evicted")
	}
	if !c.Has("a") {
		t.Error("recently-used entry a should survive")
	}
	if !c.Has("c") {
		t.Error("fresh entry c should survive")
	}
}
