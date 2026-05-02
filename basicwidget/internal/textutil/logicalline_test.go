// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
	"bytes"
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

func newTestFace(t *testing.T) text.Face {
	t.Helper()
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	return &text.GoTextFace{Source: source, Size: 16}
}

// logicalLineSlices returns the byte slices of each logical line in s, using
// LineByteOffsets to partition. The slices include any trailing line break.
func logicalLineSlices(s string) []string {
	var l textutil.LineByteOffsets
	l.RebuildFromString(s)
	out := make([]string, l.LineCount())
	for i := range out {
		start := l.ByteOffsetByLineIndex(i)
		end := len(s)
		if i+1 < l.LineCount() {
			end = l.ByteOffsetByLineIndex(i + 1)
		}
		out[i] = s[start:end]
	}
	return out
}

func TestMeasureLogicalLineHeightParity(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)

	cases := []struct {
		name string
		str  string
	}{
		{"empty", ""},
		{"single", "abc"},
		{"two lines", "abc\ndef"},
		{"trailing LF", "abc\n"},
		{"two lines trailing", "abc\ndef\n"},
		{"three lines", "abc\ndef\nghi"},
		{"CRLF", "abc\r\ndef"},
		{"multibyte", "下\n中\n"},
	}

	for _, autoWrap := range []bool{false, true} {
		for _, tc := range cases {
			t.Run(tc.name+autoWrapSuffix(autoWrap), func(t *testing.T) {
				const width = math.MaxInt

				whole := textutil.MeasureHeight(width, tc.str, autoWrap, face, lineHeight, 0, false)

				var sum float64
				for _, line := range logicalLineSlices(tc.str) {
					sum += textutil.MeasureLogicalLineHeight(width, line, autoWrap, face, lineHeight, 0, false)
				}

				if whole != sum {
					t.Errorf("MeasureHeight = %v, sum of MeasureLogicalLineHeight = %v", whole, sum)
				}
			})
		}
	}
}

func TestMeasureLogicalLineParity(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)

	cases := []struct {
		name string
		str  string
	}{
		{"empty", ""},
		{"single", "abc"},
		{"two lines", "abc\ndef"},
		{"trailing LF", "abc\n"},
		{"three lines", "abc\ndef\nghi"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const width = math.MaxInt
			wholeW, wholeH := textutil.Measure(width, tc.str, false, face, lineHeight, 0, false, "")

			var maxW, sumH float64
			for _, line := range logicalLineSlices(tc.str) {
				w, h := textutil.MeasureLogicalLine(width, line, false, face, lineHeight, 0, false, "")
				maxW = max(maxW, w)
				sumH += h
			}

			if wholeW != maxW {
				t.Errorf("Measure width = %v, max per-line width = %v", wholeW, maxW)
			}
			if wholeH != sumH {
				t.Errorf("Measure height = %v, sum per-line height = %v", wholeH, sumH)
			}
		})
	}
}

// TestMeasureLogicalLineAutoWrapVisualCount verifies that wrapping a long
// line produces multiple visual sublines whose total height equals the line
// height times the visual subline count.
func TestMeasureLogicalLineAutoWrapVisualCount(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)

	// A single logical line, no breaks. Measure at a narrow width to force
	// wrapping into multiple visual sublines.
	logical := "the quick brown fox jumps over the lazy dog"
	const narrowWidth = 80

	advance := func(s string) float64 {
		return text.Advance(s, face)
	}
	// Sanity-check: this line really does need to wrap at narrowWidth.
	if advance(logical) <= float64(narrowWidth) {
		t.Fatalf("test setup: line fits in %d px (advance=%v); pick a narrower width", narrowWidth, advance(logical))
	}

	h := textutil.MeasureLogicalLineHeight(narrowWidth, logical, true, face, lineHeight, 0, false)
	if h <= lineHeight {
		t.Errorf("MeasureLogicalLineHeight with autoWrap = %v, expected > %v (single visual subline)", h, lineHeight)
	}

	// Parity with the whole-document MeasureHeight on the same single line.
	whole := textutil.MeasureHeight(narrowWidth, logical, true, face, lineHeight, 0, false)
	if h != whole {
		t.Errorf("autoWrap MeasureLogicalLineHeight = %v, MeasureHeight whole = %v", h, whole)
	}
}

func TestTextPositionFromIndexInLogicalLineMatchesWholeDoc(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)
	op := &textutil.Options{
		Face:       face,
		LineHeight: lineHeight,
	}

	cases := []struct {
		name string
		str  string
	}{
		{"single line", "abc"},
		{"two lines", "abc\ndef"},
		{"trailing LF", "abc\n"},
		{"two lines trailing", "abc\ndef\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const width = math.MaxInt
			var l textutil.LineByteOffsets
			l.RebuildFromString(tc.str)
			lines := logicalLineSlices(tc.str)

			for idx := 0; idx <= len(tc.str); idx++ {
				// Pick the logical line that starts at or before idx but where
				// idx is strictly inside, OR (when idx is exactly a line start)
				// the line that begins at idx. LineIndexForByteOffset already
				// implements that mapping.
				lineIdx := l.LineIndexForByteOffset(idx)
				lineStart := l.ByteOffsetByLineIndex(lineIdx)
				within := idx - lineStart
				originY := float64(lineIdx) * lineHeight

				lp0, _, lpCount := textutil.TextPositionFromIndexInLogicalLine(width, lines[lineIdx], within, op)
				if lpCount == 0 {
					t.Errorf("idx=%d: per-logical count=0", idx)
					continue
				}

				// Whole-document positions. When count == 2, one of (pos0, pos1)
				// matches our per-logical query: the head of the next line, at
				// y == originY.
				wp0, wp1, wpCount := textutil.TextPositionFromIndex(&textutil.TextPositionFromIndexParams{
					Index:               idx,
					RenderingTextRange:  func(start, end int) string { return tc.str[start:end] },
					RenderingTextLength: len(tc.str),
					Width:               width,
					Options:             op,
				})

				expectedTop := lp0.Top + originY

				switch wpCount {
				case 1:
					if got, want := wp0.Top, expectedTop; got != want {
						t.Errorf("idx=%d: wholeDoc.Top=%v want %v (per-logical %v + originY %v)",
							idx, got, want, lp0.Top, originY)
					}
					if got, want := wp0.X, lp0.X; got != want {
						t.Errorf("idx=%d: wholeDoc.X=%v want %v", idx, got, want)
					}
				case 2:
					// idx is at the boundary between two visual sublines. The
					// "head of next line" position must match our per-logical
					// query (which mapped idx to the line beginning at idx).
					if got, want := wp1.Top, expectedTop; got != want {
						t.Errorf("idx=%d: wholeDoc.pos1.Top=%v want %v",
							idx, got, want)
					}
					if got, want := wp1.X, lp0.X; got != want {
						t.Errorf("idx=%d: wholeDoc.pos1.X=%v want %v", idx, got, want)
					}
				default:
					t.Errorf("idx=%d: wholeDoc count=%d", idx, wpCount)
				}
			}
		})
	}
}

func TestTextIndexFromPositionInLogicalLineMatchesWholeDoc(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)
	op := &textutil.Options{
		Face:       face,
		LineHeight: lineHeight,
	}

	str := "abc\ndef\nghi"
	const width = math.MaxInt

	var l textutil.LineByteOffsets
	l.RebuildFromString(str)
	lines := logicalLineSlices(str)

	// Query a few positions that fall inside specific logical lines and
	// verify that the per-logical answer matches the whole-document answer
	// after translating Y by the line's origin.
	queries := []struct {
		name      string
		x         int
		lineIndex int
	}{
		{"line 0 / left", 0, 0},
		{"line 0 / right", 1000, 0},
		{"line 1 / left", 0, 1},
		{"line 2 / right", 1000, 2},
	}

	for _, q := range queries {
		t.Run(q.name, func(t *testing.T) {
			originY := float64(q.lineIndex) * lineHeight
			// Local Y near the top of the per-logical line maps to a whole-doc
			// Y of originY.
			perLine := textutil.TextIndexFromPositionInLogicalLine(width, image.Pt(q.x, 0), lines[q.lineIndex], op)
			whole := textutil.TextIndexFromPosition(&textutil.TextIndexFromPositionParams{
				Position:            image.Pt(q.x, int(originY)),
				RenderingTextRange:  func(start, end int) string { return str[start:end] },
				RenderingTextLength: len(str),
				Width:               width,
				Options:             op,
			})
			lineStart := l.ByteOffsetByLineIndex(q.lineIndex)
			if got, want := perLine+lineStart, whole; got != want {
				t.Errorf("perLine(%d)+lineStart(%d) = %d, whole = %d",
					perLine, lineStart, got, want)
			}
		})
	}
}

// TestLinesInLogicalLineNoTrailingEmpty verifies that, unlike the
// whole-document iterator, a per-logical-line measurement of "abc\n" yields
// exactly one visual subline (the trailing empty line is the next logical
// line, queried separately).
func TestLinesInLogicalLineNoTrailingEmpty(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)

	// "abc\n" as one logical line: should be exactly one visual subline tall.
	if got, want := textutil.MeasureLogicalLineHeight(math.MaxInt, "abc\n", false, face, lineHeight, 0, false), lineHeight; got != want {
		t.Errorf("MeasureLogicalLineHeight(\"abc\\n\") = %v, want %v", got, want)
	}
	// The empty trailing line as its own logical line: also one subline.
	if got, want := textutil.MeasureLogicalLineHeight(math.MaxInt, "", false, face, lineHeight, 0, false), lineHeight; got != want {
		t.Errorf("MeasureLogicalLineHeight(\"\") = %v, want %v", got, want)
	}
	// Whole-document: "abc\n" yields 2 visual sublines (incl. trailing empty).
	if got, want := textutil.MeasureHeight(math.MaxInt, "abc\n", false, face, lineHeight, 0, false), 2*lineHeight; got != want {
		t.Errorf("MeasureHeight(\"abc\\n\") = %v, want %v", got, want)
	}
}

func autoWrapSuffix(autoWrap bool) string {
	if autoWrap {
		return "/autoWrap"
	}
	return "/noWrap"
}
