// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
	"image"
	"math"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// withoutIndexLineOffsets returns a shallow copy of p with the
// precomputed logical-line offset fields cleared. Parity tests use this
// to drive the unrestricted whole-document fallback inside
// [textutil.TextIndexFromPosition] as the reference value they compare
// the offset-accelerated path against.
func withoutIndexLineOffsets(p *textutil.TextLayoutParams) *textutil.TextLayoutParams {
	q := *p
	q.PrecomputedLineByteOffsets = nil
	q.LogicalLineIndexHint = 0
	q.VisualLineIndexHint = 0
	return &q
}

// TestTextIndexFromPositionLineOffsetsParity sweeps a grid of positions
// over a variety of inputs and asserts the offset-accelerated path
// matches the unrestricted whole-document fallback.
func TestTextIndexFromPositionLineOffsetsParity(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)

	cases := []struct {
		name string
		str  string
	}{
		{"empty", ""},
		{"single line", "abc"},
		{"two lines", "abc\ndef"},
		{"trailing LF", "abc\n"},
		{"three lines", "abc\ndef\nghi"},
		{"multibyte", "一\n二\n三"},
	}

	for _, wrapMode := range []textutil.WrapMode{textutil.WrapModeNone, textutil.WrapModeNormal, textutil.WrapModeAnywhere} {
		for _, tc := range cases {
			t.Run(tc.name+wrapModeSuffix(wrapMode), func(t *testing.T) {
				const width = math.MaxInt
				s := textutil.Style{
					Face:       face,
					LineHeight: lineHeight,
					WrapMode:   wrapMode,
				}
				var l textutil.LineByteOffsets
				rebuildFromString(&l, tc.str)
				params := &textutil.TextLayoutParams{
					RenderingTextRange:         func(start, end int) string { return tc.str[start:end] },
					RenderingTextLength:        len(tc.str),
					Width:                      width,
					Style:                      s,
					PrecomputedLineByteOffsets: &l,
				}

				lineCount := l.LineCount()
				if lineCount == 0 {
					lineCount = 1
				}
				for line := 0; line < lineCount+1; line++ {
					for _, x := range []int{-100, 0, 5, 50, 1000} {
						y := int(float64(line) * lineHeight)
						position := image.Pt(x, y)
						want := textutil.TextIndexFromPosition(withoutIndexLineOffsets(params), position)
						got := textutil.TextIndexFromPosition(params, position)
						if got != want {
							t.Errorf("line=%d x=%d: idx=%d, want %d", line, x, got, want)
						}
					}
				}
			})
		}
	}
}

// TestTextIndexFromPositionLineOffsetsWordWrap exercises the [WrapModeNormal]
// path with real width-induced wrapping in the middle line.
func TestTextIndexFromPositionLineOffsetsWordWrap(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)
	s := textutil.Style{Face: face, LineHeight: lineHeight, WrapMode: textutil.WrapModeNormal}

	const narrowWidth = 80
	str := "first\nthe quick brown fox jumps over the lazy dog\nlast"

	var l textutil.LineByteOffsets
	rebuildFromString(&l, str)
	params := &textutil.TextLayoutParams{
		RenderingTextRange:         func(start, end int) string { return str[start:end] },
		RenderingTextLength:        len(str),
		Width:                      narrowWidth,
		Style:                      s,
		PrecomputedLineByteOffsets: &l,
	}

	totalVL := textutil.MeasureHeight(narrowWidth, str, textutil.WrapModeNormal, face, lineHeight, 0, false) / lineHeight
	for vl := 0; vl < int(totalVL)+1; vl++ {
		for _, x := range []int{-10, 0, 30, 200} {
			position := image.Pt(x, int(float64(vl)*lineHeight))
			want := textutil.TextIndexFromPosition(withoutIndexLineOffsets(params), position)
			got := textutil.TextIndexFromPosition(params, position)
			if got != want {
				t.Errorf("vl=%d x=%d: idx=%d, want %d", vl, x, got, want)
			}
		}
	}
}

// TestTextIndexFromPositionHintParity sweeps non-zero hint values
// across the document and asserts the hint-walk path matches the
// fallback without precomputed offsets. This exercises forward walk (hint before
// the click), backward walk (hint past the click), and the document
// boundaries — paths that the default zero-hint sweeps don't cover.
func TestTextIndexFromPositionHintParity(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)

	cases := []struct {
		name     string
		str      string
		width    int
		wrapMode textutil.WrapMode
	}{
		{"three lines no wrap", "abc\ndef\nghi", math.MaxInt, textutil.WrapModeNone},
		{"three lines wordWrap no wrap", "abc\ndef\nghi", math.MaxInt, textutil.WrapModeNormal},
		{"middle line wraps", "first\nthe quick brown fox jumps over the lazy dog\nlast", 80, textutil.WrapModeNormal},
		{"middle line wraps anywhere", "first\nthequickbrownfoxjumpsoverthelazydog\nlast", 80, textutil.WrapModeAnywhere},
		{"trailing LF", "abc\ndef\n", math.MaxInt, textutil.WrapModeNormal},
	}

	for _, tc := range cases {
		t.Run(tc.name+wrapModeSuffix(tc.wrapMode), func(t *testing.T) {
			s := textutil.Style{Face: face, LineHeight: lineHeight, WrapMode: tc.wrapMode}
			var l textutil.LineByteOffsets
			rebuildFromString(&l, tc.str)
			n := l.LineCount()
			if n == 0 {
				n = 1
			}
			precVL := precedingVisualLineCountFromString(tc.str, tc.width, tc.wrapMode, face, 0, false)

			totalVL := int(textutil.MeasureHeight(tc.width, tc.str, tc.wrapMode, face, lineHeight, 0, false) / lineHeight)
			for hint := 0; hint < n; hint++ {
				params := &textutil.TextLayoutParams{
					RenderingTextRange:         func(start, end int) string { return tc.str[start:end] },
					RenderingTextLength:        len(tc.str),
					Width:                      tc.width,
					Style:                      s,
					PrecomputedLineByteOffsets: &l,
					LogicalLineIndexHint:       hint,
					VisualLineIndexHint:        precVL(hint),
				}
				for vl := 0; vl < totalVL+2; vl++ {
					for _, x := range []int{-10, 0, 30, 200} {
						position := image.Pt(x, int(float64(vl)*lineHeight))
						want := textutil.TextIndexFromPosition(withoutIndexLineOffsets(params), position)
						got := textutil.TextIndexFromPosition(params, position)
						if got != want {
							t.Errorf("hint=%d vl=%d x=%d: idx=%d, want %d", hint, vl, x, got, want)
						}
					}
				}
			}
		})
	}
}

// TestTextIndexFromPositionViewportRelativeHint covers the virtualized
// caller's contract: pass (LogicalLineIndexHint = firstVisibleLine,
// VisualLineIndexHint = 0) so position.Y is measured from the top of
// the first visible line rather than the document top. The walk must
// step from the hint instead of treating target as an absolute line
// index — otherwise [WrapModeNone] clicks always resolve to lines near
// the document start regardless of how far the user has scrolled.
func TestTextIndexFromPositionViewportRelativeHint(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)

	var sb []byte
	for i := range 50 {
		if i > 0 {
			sb = append(sb, '\n')
		}
		sb = append(sb, byte('a'+i%26))
	}
	str := string(sb)

	for _, wrapMode := range []textutil.WrapMode{textutil.WrapModeNone, textutil.WrapModeNormal} {
		t.Run(wrapModeSuffix(wrapMode), func(t *testing.T) {
			s := textutil.Style{Face: face, LineHeight: lineHeight, WrapMode: wrapMode}
			var l textutil.LineByteOffsets
			rebuildFromString(&l, str)

			for _, firstVisible := range []int{0, 1, 10, 30, 49} {
				for _, vlInViewport := range []int{-50, -5, -1, 0, 1, 5} {
					params := &textutil.TextLayoutParams{
						RenderingTextRange:         func(start, end int) string { return str[start:end] },
						RenderingTextLength:        len(str),
						Width:                      math.MaxInt,
						Style:                      s,
						PrecomputedLineByteOffsets: &l,
						LogicalLineIndexHint:       firstVisible,
						VisualLineIndexHint:        0,
					}
					position := image.Pt(0, int(float64(vlInViewport)*lineHeight))
					// Reference: the same click in absolute coords (visual
					// line firstVisible+vlInViewport from the document top)
					// resolved by the fallback without precomputed offsets.
					refPosition := image.Pt(0, int(float64(firstVisible+vlInViewport)*lineHeight))
					want := textutil.TextIndexFromPosition(withoutIndexLineOffsets(params), refPosition)
					got := textutil.TextIndexFromPosition(params, position)
					if got != want {
						t.Errorf("firstVisible=%d vlInViewport=%d: idx=%d, want %d", firstVisible, vlInViewport, got, want)
					}
				}
			}
		})
	}
}

// TestTextIndexFromPositionLineOffsetsComposition verifies an active IME
// composition is handled correctly (committed-text logical-line offsets +
// composition shifts vs the slow path on the already-spliced text).
func TestTextIndexFromPositionLineOffsetsComposition(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)

	type comp struct {
		sStart, sEnd, compLen int
		composition           string
	}
	cases := []struct {
		name      string
		committed string
		c         comp
	}{
		{"insert at line0 start", "abc\ndef", comp{0, 0, 1, "X"}},
		{"insert mb in line1", "abc\ndef\nghi", comp{5, 5, 3, "中"}},
		{"replace in line0", "abcdef\nghi", comp{1, 3, 4, "WXYZ"}},
		{"insert at end", "abc\ndef", comp{7, 7, 2, "YZ"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const width = math.MaxInt
			s := textutil.Style{Face: face, LineHeight: lineHeight}
			rendering := tc.committed[:tc.c.sStart] + tc.c.composition + tc.committed[tc.c.sEnd:]
			var l textutil.LineByteOffsets
			rebuildFromString(&l, tc.committed)
			params := &textutil.TextLayoutParams{
				RenderingTextRange:         func(start, end int) string { return rendering[start:end] },
				RenderingTextLength:        len(rendering),
				Width:                      width,
				Style:                      s,
				CommittedTextRange:         func(start, end int) string { return tc.committed[start:end] },
				PrecomputedLineByteOffsets: &l,
				SelectionStart:             tc.c.sStart,
				SelectionEnd:               tc.c.sEnd,
				CompositionLen:             tc.c.compLen,
			}
			renderingLineCount := 1
			for _, c := range rendering {
				if c == '\n' {
					renderingLineCount++
				}
			}
			for line := 0; line < renderingLineCount+1; line++ {
				for _, x := range []int{0, 5, 50, 1000} {
					position := image.Pt(x, int(float64(line)*lineHeight))
					want := textutil.TextIndexFromPosition(withoutIndexLineOffsets(params), position)
					got := textutil.TextIndexFromPosition(params, position)
					if got != want {
						t.Errorf("line=%d x=%d: idx=%d, want %d", line, x, got, want)
					}
				}
			}
		})
	}
}
