// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
	"image"
	"math"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

const insertionLineHeight = 24

// insertionFixture lays out "aaaa bbbb cccc" at a width that wraps it into one
// visual line per word, with no face override on the text itself.
func insertionFixture(t *testing.T) (str string, width int, style textutil.Style) {
	t.Helper()
	small, _ := testFaces(t)
	str = "aaaa bbbb cccc"
	width = int(math.Ceil(advanceOf("aaaa ", small)))
	style = textutil.Style{
		WrapMode:       textutil.WrapModeNormal,
		Face:           small,
		LineHeight:     insertionLineHeight,
		LineHeightMode: textutil.LineHeightModeFlexible,
	}
	return str, width, style
}

func TestInsertionScalesItsVisualLineOnly(t *testing.T) {
	str, width, style := insertionFixture(t)
	_, large := testFaces(t)

	// The three visual lines are "aaaa ", "bbbb " and "cccc".
	for _, tc := range []struct {
		name         string
		indexInBytes int
		wantHeight   float64
	}{
		{name: "no insertion", indexInBytes: -1, wantHeight: 3 * insertionLineHeight},
		{name: "first line", indexInBytes: 2, wantHeight: 4 * insertionLineHeight},
		{name: "second line", indexInBytes: 6, wantHeight: 4 * insertionLineHeight},
		// A wrap boundary belongs to the following line, where the caret
		// renders.
		{name: "wrap boundary", indexInBytes: 5, wantHeight: 4 * insertionLineHeight},
		{name: "text end", indexInBytes: len(str), wantHeight: 4 * insertionLineHeight},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var insertion textutil.Insertion
			if tc.indexInBytes >= 0 {
				insertion = textutil.Insertion{Face: large, IndexInBytes: tc.indexInBytes}
			}
			if got := textutil.MeasureHeight(width, str, style.WrapMode, style.Face, nil, insertion, style.LineHeight, style.LineHeightMode, 0, false); got != tc.wantHeight {
				t.Errorf("MeasureHeight() = %v, want %v", got, tc.wantHeight)
			}
		})
	}
}

func TestInsertionAtLineBreak(t *testing.T) {
	small, large := testFaces(t)

	for _, tc := range []struct {
		name         string
		str          string
		indexInBytes int
		wantHeights  []float64
	}{
		// The position right before a break sits on the line the break ends,
		// the one right after it on the following line.
		{name: "before break", str: "abc\ndef", indexInBytes: 3, wantHeights: []float64{2 * insertionLineHeight, insertionLineHeight}},
		{name: "after break", str: "abc\ndef", indexInBytes: 4, wantHeights: []float64{insertionLineHeight, 2 * insertionLineHeight}},
		// A trailing break makes the last logical line an empty one, which
		// owns the position past the break.
		{name: "before trailing break", str: "abc\n", indexInBytes: 3, wantHeights: []float64{2 * insertionLineHeight, insertionLineHeight}},
		{name: "after trailing break", str: "abc\n", indexInBytes: 4, wantHeights: []float64{insertionLineHeight, 2 * insertionLineHeight}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			style := textutil.Style{
				WrapMode:       textutil.WrapModeNone,
				Face:           small,
				LineHeight:     insertionLineHeight,
				LineHeightMode: textutil.LineHeightModeFlexible,
			}
			insertion := textutil.Insertion{Face: large, IndexInBytes: tc.indexInBytes}

			// Each logical line is measured on its own by virtualized layout,
			// which stacks the lines by those heights.
			var total float64
			var start int
			for i, line := range logicalLines(tc.str) {
				h := textutil.MeasureLogicalLineHeight(math.MaxInt, line, style.WrapMode, style.Face, nil, insertion, start, style.LineHeight, style.LineHeightMode, 0, false)
				if want := tc.wantHeights[i]; h != want {
					t.Errorf("MeasureLogicalLineHeight(line %d) = %v, want %v", i, h, want)
				}
				total += h
				start += len(line)
			}

			var want float64
			for _, h := range tc.wantHeights {
				want += h
			}
			if got := textutil.MeasureHeight(math.MaxInt, tc.str, style.WrapMode, style.Face, nil, insertion, style.LineHeight, style.LineHeightMode, 0, false); got != want {
				t.Errorf("MeasureHeight() = %v, want %v", got, want)
			}
			if total != want {
				t.Errorf("sum of MeasureLogicalLineHeight() = %v, want %v", total, want)
			}
		})
	}
}

// logicalLines splits str after each hard line break, the way the widget's
// precomputed logical-line offsets do: a trailing break leaves an empty last
// line.
func logicalLines(str string) []string {
	lines := []string{""}
	for len(str) > 0 {
		p, l := textutil.FirstLineBreakPositionAndLen(str)
		if p < 0 {
			lines[len(lines)-1] = str
			break
		}
		lines[len(lines)-1] = str[:p+l]
		lines = append(lines, "")
		str = str[p+l:]
	}
	return lines
}

func TestInsertionScalesTheCaret(t *testing.T) {
	str, width, style := insertionFixture(t)
	_, large := testFaces(t)

	const indexInBytes = 6
	base, _, count := textutil.TextPositionFromIndexInLogicalLine(width, str, 0, indexInBytes, &style)
	if count == 0 {
		t.Fatal("TextPositionFromIndexInLogicalLine found no position")
	}

	style.Insertion = textutil.Insertion{Face: large, IndexInBytes: indexInBytes}
	scaled, _, count := textutil.TextPositionFromIndexInLogicalLine(width, str, 0, indexInBytes, &style)
	if count == 0 {
		t.Fatal("TextPositionFromIndexInLogicalLine found no position")
	}

	// The caret spans its line's content area, which the insertion scales by
	// the ratio of the face sizes.
	if got, want := scaled.Bottom-scaled.Top, 2*(base.Bottom-base.Top); math.Abs(got-want) > 1e-9 {
		t.Errorf("caret height = %v, want %v", got, want)
	}

	// The line below the insertion moves down with it.
	belowBase, _, count := textutil.TextPositionFromIndexInLogicalLine(width, str, 0, 12, &style)
	if count == 0 {
		t.Fatal("TextPositionFromIndexInLogicalLine found no position")
	}
	if want := 3 * style.LineHeight; belowBase.Top < want {
		t.Errorf("third line caret top = %v, want at least %v", belowBase.Top, want)
	}
}

// TestInsertionHitTestFollowsTheScaledLine asserts that hit testing resolves
// against the scaled line boxes, so a click lands on the line it is drawn on.
func TestInsertionHitTestFollowsTheScaledLine(t *testing.T) {
	str, width, style := insertionFixture(t)
	_, large := testFaces(t)
	style.Insertion = textutil.Insertion{Face: large, IndexInBytes: 6}

	// The second line's box spans [24, 72) once scaled, so this Y resolves
	// into "bbbb" rather than the third line.
	const y = 60
	if got, want := textutil.TextIndexFromPositionInLogicalLine(width, image.Pt(0, y), str, 0, &style), 5; got != want {
		t.Errorf("TextIndexFromPositionInLogicalLine(y=%d) = %d, want %d", y, got, want)
	}
}

func TestInsertionIgnoredUnderFixedLineHeightMode(t *testing.T) {
	str, width, style := insertionFixture(t)
	_, large := testFaces(t)
	style.LineHeightMode = textutil.LineHeightModeFixed
	style.Insertion = textutil.Insertion{Face: large, IndexInBytes: 6}

	if got, want := textutil.MeasureHeight(width, str, style.WrapMode, style.Face, nil, style.Insertion, style.LineHeight, style.LineHeightMode, 0, false), 3*style.LineHeight; got != want {
		t.Errorf("MeasureHeight() = %v, want %v", got, want)
	}
	pos, _, count := textutil.TextPositionFromIndexInLogicalLine(width, str, 0, 6, &style)
	if count == 0 {
		t.Fatal("TextPositionFromIndexInLogicalLine found no position")
	}
	if got, want := pos.Bottom-pos.Top, style.LineHeight; got > want {
		t.Errorf("caret height = %v, want at most %v", got, want)
	}
}

// TestInsertionSmallerThanTheLineKeepsTheHeight asserts that an insertion
// carrying a smaller face than the text around it leaves the line as tall as
// its largest face.
func TestInsertionSmallerThanTheLineKeepsTheHeight(t *testing.T) {
	small, large := testFaces(t)
	const str = "abc"
	style := textutil.Style{
		WrapMode:       textutil.WrapModeNone,
		Face:           small,
		FaceRuns:       []textutil.FaceRun{{Start: 0, End: len(str), Face: large}},
		LineHeight:     insertionLineHeight,
		LineHeightMode: textutil.LineHeightModeFlexible,
	}
	insertion := textutil.Insertion{Face: small, IndexInBytes: 1}

	if got, want := textutil.MeasureHeight(math.MaxInt, str, style.WrapMode, style.Face, style.FaceRuns, insertion, style.LineHeight, style.LineHeightMode, 0, false), 2*style.LineHeight; got != want {
		t.Errorf("MeasureHeight() = %v, want %v", got, want)
	}
}
