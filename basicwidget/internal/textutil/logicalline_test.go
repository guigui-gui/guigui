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

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

func newTestFace(t *testing.T) font.Face {
	t.Helper()
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	return font.NewFaceForTest(&text.GoTextFace{Source: source, Size: 16}, font.Attributes{Size: 16})
}

// logicalLineSlices returns the byte slices of each logical line in s, using
// LineByteOffsets to partition. The slices include any trailing line break.
func logicalLineSlices(s string) []string {
	var l textutil.LineByteOffsets
	rebuildFromString(&l, s)
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
		{"multibyte", "一\n二\n"},
	}

	for _, wrapMode := range []textutil.WrapMode{textutil.WrapModeNone, textutil.WrapModeNormal, textutil.WrapModeAnywhere} {
		for _, tc := range cases {
			t.Run(tc.name+wrapModeSuffix(wrapMode), func(t *testing.T) {
				const width = math.MaxInt

				whole := textutil.MeasureHeight(width, tc.str, wrapMode, face, nil, lineHeight, 0, false)

				var sum float64
				for _, line := range logicalLineSlices(tc.str) {
					sum += textutil.MeasureLogicalLineHeight(width, line, wrapMode, face, lineHeight, 0, false)
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
			wholeW, wholeH := textutil.Measure(width, tc.str, textutil.WrapModeNone, face, nil, lineHeight, 0, false, "")

			var maxW, sumH float64
			for _, line := range logicalLineSlices(tc.str) {
				w, h := textutil.MeasureLogicalLine(width, line, textutil.WrapModeNone, face, lineHeight, 0, false, "")
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

func TestCachedVisualLineMaxCaretXIncludesTrailingSpaces(t *testing.T) {
	face := newTestFace(t)
	testCases := []struct {
		name string
		line string
	}{
		{
			name: "one trailing space",
			line: "abcdefghijklmnopqrstuvwxyz ",
		},
		{
			name: "multiple trailing spaces",
			line: "the quick brown fox   ",
		},
	}

	const width = 80
	for _, wrapMode := range []textutil.WrapMode{textutil.WrapModeNormal, textutil.WrapModeAnywhere} {
		for _, tc := range testCases {
			t.Run(tc.name+wrapModeSuffix(wrapMode), func(t *testing.T) {
				style := textutil.Style{
					WrapMode:   wrapMode,
					Face:       face,
					LineHeight: 24,
				}
				position, _, count := textutil.TextPositionFromIndexInLogicalLine(width, tc.line, len(tc.line), &style)
				if count == 0 {
					t.Fatal("TextPositionFromIndexInLogicalLine returned no position")
				}
				got := textutil.CachedVisualLineMaxCaretX(width, tc.line, wrapMode, face, 0, false)
				if got < position.X {
					t.Errorf("CachedVisualLineMaxCaretX = %v, trailing caret X = %v", got, position.X)
				}
				if tc.name == "one trailing space" && wrapMode == textutil.WrapModeNormal {
					rendered, _ := textutil.MeasureLogicalLine(width, tc.line, wrapMode, face, 0, 0, false, "")
					if position.X <= rendered {
						t.Fatalf("test setup: trailing caret X = %v, rendered width = %v", position.X, rendered)
					}
				}
			})
		}
	}
}

// TestCachedVisualLineMaxCaretXExcludesBreakSpace verifies that the final
// space of a non-final visual line does not extend the caret extent: the caret
// after it is rendered at the head of the next visual line. Carets between
// hanging spaces still render on the wrapped line and must stay covered.
func TestCachedVisualLineMaxCaretXExcludesBreakSpace(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)

	// Pick a width that fits "aaa" but not "aaa ", so the break space hangs
	// past the wrapping width.
	wordW, _ := textutil.Measure(math.MaxInt, "aaa", textutil.WrapModeNone, face, nil, lineHeight, 0, false, "")
	wordSpaceW, _ := textutil.Measure(math.MaxInt, "aaa ", textutil.WrapModeNone, face, nil, lineHeight, 0, true, "")
	width := int((wordW + wordSpaceW) / 2)
	if float64(width) < wordW || wordSpaceW <= float64(width) {
		t.Fatalf("test setup: width %d must satisfy %v <= width < %v", width, wordW, wordSpaceW)
	}

	style := textutil.Style{
		WrapMode:   textutil.WrapModeNormal,
		Face:       face,
		LineHeight: lineHeight,
	}

	t.Run("single break space", func(t *testing.T) {
		const line = "aaa bbb"
		if got := textutil.CachedVisualLineCount(width, line, textutil.WrapModeNormal, face, 0, false); got != 2 {
			t.Fatalf("test setup: visual line count = %d, want 2", got)
		}
		tailPos, _, count := textutil.TextPositionFromIndexInLogicalLine(width, line, len("aaa "), &style)
		if count != 2 {
			t.Fatalf("test setup: the caret at the wrap boundary must have two positions, got %d", count)
		}
		got := textutil.CachedVisualLineMaxCaretX(width, line, textutil.WrapModeNormal, face, 0, false)
		if got >= tailPos.X {
			t.Errorf("CachedVisualLineMaxCaretX = %v, want less than the boundary tail X %v", got, tailPos.X)
		}
		if got > float64(width) {
			t.Errorf("CachedVisualLineMaxCaretX = %v, want at most the wrapping width %d", got, width)
		}
	})

	t.Run("multiple hanging spaces", func(t *testing.T) {
		const line = "aaa      bbb"
		boundary := len("aaa      ")
		if got := textutil.CachedVisualLineCount(width, line, textutil.WrapModeNormal, face, 0, false); got != 2 {
			t.Fatalf("test setup: visual line count = %d, want 2", got)
		}
		innerPos, _, count := textutil.TextPositionFromIndexInLogicalLine(width, line, boundary-1, &style)
		if count != 1 {
			t.Fatalf("test setup: the caret between hanging spaces must have one position, got %d", count)
		}
		tailPos, _, count := textutil.TextPositionFromIndexInLogicalLine(width, line, boundary, &style)
		if count != 2 {
			t.Fatalf("test setup: the caret at the wrap boundary must have two positions, got %d", count)
		}
		got := textutil.CachedVisualLineMaxCaretX(width, line, textutil.WrapModeNormal, face, 0, false)
		if got < innerPos.X {
			t.Errorf("CachedVisualLineMaxCaretX = %v, want at least the last in-line caret X %v", got, innerPos.X)
		}
		if got >= tailPos.X {
			t.Errorf("CachedVisualLineMaxCaretX = %v, want less than the boundary tail X %v", got, tailPos.X)
		}
	})
}

// TestMeasureLogicalLineWrapVisualCount verifies that wrapping a long
// line produces multiple visual sublines whose total height equals the line
// height times the visual subline count.
func TestMeasureLogicalLineWrapVisualCount(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)

	// A single logical line, no breaks. Measure at a narrow width to force
	// wrapping into multiple visual sublines.
	logical := "the quick brown fox jumps over the lazy dog"
	const narrowWidth = 80

	advance := func(s string) float64 {
		return text.AdvanceAt(s, len(s), face.TextFace())
	}
	// Sanity-check: this line really does need to wrap at narrowWidth.
	if advance(logical) <= float64(narrowWidth) {
		t.Fatalf("test setup: line fits in %d px (advance=%v); pick a narrower width", narrowWidth, advance(logical))
	}

	h := textutil.MeasureLogicalLineHeight(narrowWidth, logical, textutil.WrapModeNormal, face, lineHeight, 0, false)
	if h <= lineHeight {
		t.Errorf("MeasureLogicalLineHeight with WrapModeNormal = %v, expected > %v (single visual subline)", h, lineHeight)
	}

	// Parity with the whole-document MeasureHeight on the same single line.
	whole := textutil.MeasureHeight(narrowWidth, logical, textutil.WrapModeNormal, face, nil, lineHeight, 0, false)
	if h != whole {
		t.Errorf("WrapModeNormal MeasureLogicalLineHeight = %v, MeasureHeight whole = %v", h, whole)
	}
}

func TestTextPositionFromIndexInLogicalLineMatchesWholeDoc(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)
	s := textutil.Style{
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
			rebuildFromString(&l, tc.str)
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

				lp0, _, lpCount := textutil.TextPositionFromIndexInLogicalLine(width, lines[lineIdx], within, &s)
				if lpCount == 0 {
					t.Errorf("idx=%d: per-logical count=0", idx)
					continue
				}

				// Whole-document positions. When count == 2, one of (pos0, pos1)
				// matches our per-logical query: the head of the next line, at
				// y == originY.
				wp0, wp1, wpCount := textutil.TextPositionFromIndex(&textutil.TextLayoutParams{
					RenderingTextRange:  func(start, end int) string { return tc.str[start:end] },
					RenderingTextLength: len(tc.str),
					Width:               width,
					Style:               s,
				}, idx)

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
	s := textutil.Style{
		Face:       face,
		LineHeight: lineHeight,
	}

	str := "abc\ndef\nghi"
	const width = math.MaxInt

	var l textutil.LineByteOffsets
	rebuildFromString(&l, str)
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
			perLine := textutil.TextIndexFromPositionInLogicalLine(width, image.Pt(q.x, 0), lines[q.lineIndex], &s)
			whole := textutil.TextIndexFromPosition(&textutil.TextLayoutParams{
				RenderingTextRange:  func(start, end int) string { return str[start:end] },
				RenderingTextLength: len(str),
				Width:               width,
				Style:               s,
			}, image.Pt(q.x, int(originY)))
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
	if got, want := textutil.MeasureLogicalLineHeight(math.MaxInt, "abc\n", textutil.WrapModeNone, face, lineHeight, 0, false), lineHeight; got != want {
		t.Errorf("MeasureLogicalLineHeight(\"abc\\n\") = %v, want %v", got, want)
	}
	// The empty trailing line as its own logical line: also one subline.
	if got, want := textutil.MeasureLogicalLineHeight(math.MaxInt, "", textutil.WrapModeNone, face, lineHeight, 0, false), lineHeight; got != want {
		t.Errorf("MeasureLogicalLineHeight(\"\") = %v, want %v", got, want)
	}
	// Whole-document: "abc\n" yields 2 visual sublines (incl. trailing empty).
	if got, want := textutil.MeasureHeight(math.MaxInt, "abc\n", textutil.WrapModeNone, face, nil, lineHeight, 0, false), 2*lineHeight; got != want {
		t.Errorf("MeasureHeight(\"abc\\n\") = %v, want %v", got, want)
	}
}

func wrapModeSuffix(wrapMode textutil.WrapMode) string {
	switch wrapMode {
	case textutil.WrapModeNone:
		return "/noWrap"
	case textutil.WrapModeNormal:
		return "/wrapWord"
	case textutil.WrapModeAnywhere:
		return "/wrapAnywhere"
	}
	return ""
}
