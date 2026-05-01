// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
	"image"
	"math"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// withoutIndexSidecar returns a shallow copy of p with the sidecar
// fields cleared. Parity tests use this to drive the unrestricted
// whole-document fallback inside [textutil.TextIndexFromPosition] as
// the reference value they compare the sidecar-accelerated path
// against.
func withoutIndexSidecar(p *textutil.TextIndexFromPositionParams) *textutil.TextIndexFromPositionParams {
	q := *p
	q.LineByteOffsets = nil
	q.LogicalLineIndexHint = 0
	q.VisualLineIndexHint = 0
	return &q
}

// TestTextIndexFromPositionSidecarParity sweeps a grid of positions
// over a variety of inputs and asserts the sidecar-accelerated path
// matches the unrestricted whole-document fallback.
func TestTextIndexFromPositionSidecarParity(t *testing.T) {
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
		{"multibyte", "下\n中\n上"},
	}

	for _, autoWrap := range []bool{false, true} {
		for _, tc := range cases {
			t.Run(tc.name+autoWrapSuffix(autoWrap), func(t *testing.T) {
				const width = math.MaxInt
				op := &textutil.Options{
					Face:       face,
					LineHeight: lineHeight,
					AutoWrap:   autoWrap,
				}
				var l textutil.LineByteOffsets
				l.RebuildFromString(tc.str)
				params := &textutil.TextIndexFromPositionParams{
					RenderingText:   tc.str,
					Width:           width,
					Options:         op,
					LineByteOffsets: &l,
				}

				lineCount := l.LineCount()
				if lineCount == 0 {
					lineCount = 1
				}
				for line := 0; line < lineCount+1; line++ {
					for _, x := range []int{-100, 0, 5, 50, 1000} {
						y := int(float64(line) * lineHeight)
						params.Position = image.Pt(x, y)
						want := textutil.TextIndexFromPosition(withoutIndexSidecar(params))
						got := textutil.TextIndexFromPosition(params)
						if got != want {
							t.Errorf("line=%d x=%d: idx=%d, want %d", line, x, got, want)
						}
					}
				}
			})
		}
	}
}

// TestTextIndexFromPositionSidecarAutoWrap exercises the autoWrap path
// with real width-induced wrapping in the middle line.
func TestTextIndexFromPositionSidecarAutoWrap(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)
	op := &textutil.Options{Face: face, LineHeight: lineHeight, AutoWrap: true}

	const narrowWidth = 80
	str := "first\nthe quick brown fox jumps over the lazy dog\nlast"

	var l textutil.LineByteOffsets
	l.RebuildFromString(str)
	params := &textutil.TextIndexFromPositionParams{
		RenderingText:   str,
		Width:           narrowWidth,
		Options:         op,
		LineByteOffsets: &l,
	}

	totalVL := textutil.MeasureHeight(narrowWidth, str, true, face, lineHeight, 0, false) / lineHeight
	for vl := 0; vl < int(totalVL)+1; vl++ {
		for _, x := range []int{-10, 0, 30, 200} {
			params.Position = image.Pt(x, int(float64(vl)*lineHeight))
			want := textutil.TextIndexFromPosition(withoutIndexSidecar(params))
			got := textutil.TextIndexFromPosition(params)
			if got != want {
				t.Errorf("vl=%d x=%d: idx=%d, want %d", vl, x, got, want)
			}
		}
	}
}

// TestTextIndexFromPositionHintParity sweeps non-zero hint values
// across the document and asserts the hint-walk path matches the
// sidecar-less fallback. This exercises forward walk (hint before
// the click), backward walk (hint past the click), and the document
// boundaries — paths that the default zero-hint sweeps don't cover.
func TestTextIndexFromPositionHintParity(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)

	cases := []struct {
		name     string
		str      string
		width    int
		autoWrap bool
	}{
		{"three lines no wrap", "abc\ndef\nghi", math.MaxInt, false},
		{"three lines autoWrap no wrap", "abc\ndef\nghi", math.MaxInt, true},
		{"middle line wraps", "first\nthe quick brown fox jumps over the lazy dog\nlast", 80, true},
		{"trailing LF", "abc\ndef\n", math.MaxInt, true},
	}

	for _, tc := range cases {
		t.Run(tc.name+autoWrapSuffix(tc.autoWrap), func(t *testing.T) {
			op := &textutil.Options{Face: face, LineHeight: lineHeight, AutoWrap: tc.autoWrap}
			var l textutil.LineByteOffsets
			l.RebuildFromString(tc.str)
			n := l.LineCount()
			if n == 0 {
				n = 1
			}
			precVL := precedingVisualLineCountFromString(tc.str, tc.width, tc.autoWrap, face, 0, false)

			totalVL := int(textutil.MeasureHeight(tc.width, tc.str, tc.autoWrap, face, lineHeight, 0, false) / lineHeight)
			for hint := 0; hint < n; hint++ {
				params := &textutil.TextIndexFromPositionParams{
					RenderingText:        tc.str,
					Width:                tc.width,
					Options:              op,
					LineByteOffsets:      &l,
					LogicalLineIndexHint: hint,
					VisualLineIndexHint:  precVL(hint),
				}
				for vl := 0; vl < totalVL+2; vl++ {
					for _, x := range []int{-10, 0, 30, 200} {
						params.Position = image.Pt(x, int(float64(vl)*lineHeight))
						want := textutil.TextIndexFromPosition(withoutIndexSidecar(params))
						got := textutil.TextIndexFromPosition(params)
						if got != want {
							t.Errorf("hint=%d vl=%d x=%d: idx=%d, want %d", hint, vl, x, got, want)
						}
					}
				}
			}
		})
	}
}

// TestTextIndexFromPositionSidecarComposition verifies an active IME
// composition is handled correctly (committed sidecar + composition
// shifts vs the slow path on the already-spliced text).
func TestTextIndexFromPositionSidecarComposition(t *testing.T) {
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
			op := &textutil.Options{Face: face, LineHeight: lineHeight}
			rendering := tc.committed[:tc.c.sStart] + tc.c.composition + tc.committed[tc.c.sEnd:]
			var l textutil.LineByteOffsets
			l.RebuildFromString(tc.committed)
			params := &textutil.TextIndexFromPositionParams{
				RenderingText:   rendering,
				Width:           width,
				Options:         op,
				CommittedText:   tc.committed,
				LineByteOffsets: &l,
				SelectionStart:  tc.c.sStart,
				SelectionEnd:    tc.c.sEnd,
				CompositionLen:  tc.c.compLen,
			}
			renderingLineCount := 1
			for _, c := range rendering {
				if c == '\n' {
					renderingLineCount++
				}
			}
			for line := 0; line < renderingLineCount+1; line++ {
				for _, x := range []int{0, 5, 50, 1000} {
					params.Position = image.Pt(x, int(float64(line)*lineHeight))
					want := textutil.TextIndexFromPosition(withoutIndexSidecar(params))
					got := textutil.TextIndexFromPosition(params)
					if got != want {
						t.Errorf("line=%d x=%d: idx=%d, want %d", line, x, got, want)
					}
				}
			}
		})
	}
}
