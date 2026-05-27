// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
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
		"\t\t\t\t",              // only tabs
		"word word word\tend\t", // trailing bare tab (trimmed unless KeepTailingSpace)
		"abc אבג def דה ghi",    // LTR with embedded RTL (Hebrew) runs
		"אבג דהו זח",            // RTL (Hebrew) text that wraps
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

// TestVisualLinesFromCachedStartsIncrementalEdits checks that incremental edits
// of a long wrapping line (insert/delete/replace, incl. multibyte) match the
// shaping packer, exercising the patch-the-edited-span path on the last-measured line.
func TestVisualLinesFromCachedStartsIncrementalEdits(t *testing.T) {
	face := newTestFace(t)

	var sb strings.Builder
	for range 20 {
		sb.WriteString("the quick brown lazy dog runs and jumps over many small hills today ")
	}
	base := strings.TrimSpace(sb.String())

	type edit struct {
		at  int
		del int
		ins string
	}

	for _, wrapMode := range []textutil.WrapMode{textutil.WrapModeNormal, textutil.WrapModeAnywhere} {
		for _, width := range []int{40, 80, 200} {
			cur := base
			assertCachedMatchesShaping(t, face, width, wrapMode, cur)

			edits := []edit{
				// insert at start
				{
					at:  0,
					ins: "X",
				},
				// insert near start
				{
					at:  7,
					ins: "hello ",
				},
				// delete in the middle
				{
					at:  len(cur) / 2,
					del: 5,
				},
				// same-length replace
				{
					at:  len(cur) / 3,
					del: 3,
					ins: "ZZZ",
				},
				// insert multibyte mid
				{
					at:  len(cur) / 2,
					ins: "一二三",
				},
				// grow in place
				{
					at:  len(cur) / 4,
					del: 2,
					ins: "longer ",
				},
				// append at end
				{
					at:  len(cur),
					ins: " appended end",
				},
			}
			for _, e := range edits {
				at := snapToRuneStart(cur, e.at)
				del := e.del
				for at+del < len(cur) && !utf8.RuneStart(cur[at+del]) {
					del++
				}
				cur = cur[:at] + e.ins + cur[at+del:]
				assertCachedMatchesShaping(t, face, width, wrapMode, cur)
			}
		}
	}
}

// TestCachedVisualLineStartsFullAndSubstringAlternation alternates laying out the
// full logical line and a visible substring of it (the widget's per-tick pattern)
// and checks both keep matching the shaping packer.
func TestCachedVisualLineStartsFullAndSubstringAlternation(t *testing.T) {
	face := newTestFace(t)
	const width = 200
	wrapMode := textutil.WrapModeNormal

	var sb strings.Builder
	for range 60 {
		sb.WriteString("the quick brown lazy dog runs and jumps over many small hills today ")
	}
	cur := strings.TrimSpace(sb.String())

	for i := range 12 {
		// One keystroke near the middle.
		at := snapToRuneStart(cur, len(cur)/2+i)
		cur = cur[:at] + string(rune('a'+i%26)) + cur[at:]

		// Full logical line (caret / height path).
		assertCachedMatchesShaping(t, face, width, wrapMode, cur)

		// Visible substring around the edit (virtualized draw path): a rune-
		// aligned window containing the edit, so its content changes too.
		vs := snapToRuneStart(cur, max(0, at-300))
		ve := min(len(cur), at+300)
		for ve < len(cur) && !utf8.RuneStart(cur[ve]) {
			ve++
		}
		assertCachedMatchesShaping(t, face, width, wrapMode, cur[vs:ve])
	}
}

// TestCachedVisualLineStartsLongLineSurvivesShortLine checks that a short line
// laid out at the same parameters (e.g. an empty trailing logical line) does not
// evict the long line as the last-measured line, which would reshape it whole
// next keystroke.
func TestCachedVisualLineStartsLongLineSurvivesShortLine(t *testing.T) {
	const width = 200
	wrapMode := textutil.WrapModeNormal

	var sb strings.Builder
	for range 50 {
		sb.WriteString("the quick brown lazy dog runs and jumps over many small hills today ")
	}
	long := strings.TrimSpace(sb.String())
	edited := long[:len(long)/2] + "X" + long[len(long)/2:]

	for _, short := range []string{"", "Line 1, Column 5", "a short line"} {
		// A fresh face per case gives this case distinct cache keys, so the long
		// line becomes the last-measured line from scratch (a relayout that
		// populates it), independent of other cases and tests — no reset needed.
		face := newTestFace(t)

		// Make the long line the last-measured one.
		assertCachedMatchesShaping(t, face, width, wrapMode, long)
		if got := textutil.LastMeasuredLineLenForTest(); got != len(long) {
			t.Fatalf("long line not stored as last-measured: len=%d, want %d", got, len(long))
		}

		// A short line at the same parameters must not evict the long line.
		assertCachedMatchesShaping(t, face, width, wrapMode, short)
		if got := textutil.LastMeasuredLineLenForTest(); got != len(long) {
			t.Fatalf("short line %q evicted the long line: last-measured len=%d, want %d", short, got, len(long))
		}

		// A single-character edit of the long line must then update the
		// last-measured line (small patch) and still match the shaping packer.
		assertCachedMatchesShaping(t, face, width, wrapMode, edited)
		if got := textutil.LastMeasuredLineLenForTest(); got != len(edited) {
			t.Fatalf("last-measured not updated to the edited long line: len=%d, want %d", got, len(edited))
		}
	}
}

// snapToRuneStart returns the largest index ≤ i that begins a rune in s.
func snapToRuneStart(s string, i int) int {
	if i > len(s) {
		i = len(s)
	}
	for i > 0 && i < len(s) && !utf8.RuneStart(s[i]) {
		i--
	}
	return i
}

// assertCachedMatchesShaping fails t when the cache-backed Draw build of str
// differs from the shaping packer at the given parameters.
func assertCachedMatchesShaping(t *testing.T, face font.Face, width int, wrapMode textutil.WrapMode, str string) {
	t.Helper()
	adv := func(s string, idx int) float64 {
		return textutil.AdvanceForTestParams(s, idx, face, 0, false)
	}
	var want []textutil.VisualLine
	for vl := range textutil.VisualLines(width, str, wrapMode, adv) {
		want = append(want, vl)
	}
	got, ok := textutil.VisualLinesFromCachedStarts(width, str, wrapMode, face, 0, false)
	if !ok {
		t.Fatalf("VisualLinesFromCachedStarts ok=false for width=%d wrap=%v len=%d", width, wrapMode, len(str))
	}
	if !slices.Equal(got, want) {
		t.Fatalf("mismatch width=%d wrap=%v len=%d\n got=%v\nwant=%v", width, wrapMode, len(str), got, want)
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

	// Past the alive window, touch "a" again and add a third entry to trip the
	// over-limit sweep: "b" (idle) is evicted, "a" (recently used) and "c" survive.
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
