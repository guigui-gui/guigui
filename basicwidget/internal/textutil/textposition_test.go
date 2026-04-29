// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
	"bytes"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// withoutSidecar returns a shallow copy of p with the sidecar fields
// cleared. Parity tests use this to drive the unrestricted whole-document
// fallback inside [textutil.TextPositionFromIndex] as the reference value
// they compare the sidecar-accelerated path against.
func withoutSidecar(p *textutil.TextPositionFromIndexParams) *textutil.TextPositionFromIndexParams {
	q := *p
	q.LineByteOffsets = nil
	q.PrecedingVisualLineCount = nil
	return &q
}

// precedingVisualLineCountFromString returns a PrecedingVisualLineCount
// implementation backed by a fresh visual-line-count walk over committed.
// The Text widget caches this; tests just walk on each call since inputs
// are tiny.
func precedingVisualLineCountFromString(committed string, width int, autoWrap bool, face text.Face, tabWidth float64, keepTailingSpace bool) func(int) int {
	var l textutil.LineByteOffsets
	l.RebuildFromString(committed)
	return func(lineIdx int) int {
		if lineIdx <= 0 {
			return 0
		}
		n := l.LineCount()
		if lineIdx > n {
			lineIdx = n
		}
		var sum int
		for i := 0; i < lineIdx; i++ {
			start := l.ByteOffsetByLineIndex(i)
			end := len(committed)
			if i+1 < n {
				end = l.ByteOffsetByLineIndex(i + 1)
			}
			sum += textutil.VisualLineCountForLogicalLine(width, committed[start:end], autoWrap, face, tabWidth, keepTailingSpace)
		}
		return sum
	}
}

func TestTextPositionFromIndex(t *testing.T) {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	face := &text.GoTextFace{Source: source, Size: 16}
	const lineHeight = 24.0
	op := &textutil.Options{
		Face:       face,
		LineHeight: lineHeight,
	}

	// Baseline: position at index 0 of a single-line string sits on visual
	// line 0. Use it to derive line N's Top without hard-coding the face's
	// vertical padding.
	baseline, _, _ := textutil.TextPositionFromIndex(&textutil.TextPositionFromIndexParams{
		Index:         0,
		RenderingText: "a",
		Width:         1000,
		Options:       op,
	})
	topOfLine := func(n int) float64 {
		return baseline.Top + float64(n)*lineHeight
	}

	testCases := []struct {
		name      string
		text      string
		index     int
		wantCount int
		// Visual line index for each returned position (0 = first line).
		// -1 means "don't check".
		wantLine0 int
		wantLine1 int
		// Whether pos0.X / pos1.X must be 0 (line start).
		wantPos0XZero bool
		wantPos1XZero bool
	}{
		{
			name:          "single-line/start",
			text:          "abc",
			index:         0,
			wantCount:     1,
			wantLine0:     0,
			wantLine1:     -1,
			wantPos0XZero: true,
		},
		{
			name:      "single-line/end",
			text:      "abc",
			index:     3,
			wantCount: 1,
			wantLine0: 0,
			wantLine1: -1,
		},
		{
			name:          "trailing-newline/end",
			text:          "a\n",
			index:         2,
			wantCount:     2,
			wantLine0:     0, // tail of "a\n" — must be on line 0 with X > 0
			wantLine1:     1, // head of empty line — at start of line 1
			wantPos1XZero: true,
		},
		{
			name:          "trailing-newline/start",
			text:          "a\n",
			index:         0,
			wantCount:     1,
			wantLine0:     0,
			wantLine1:     -1,
			wantPos0XZero: true,
		},
		{
			name:          "mid-newline-boundary",
			text:          "a\nb",
			index:         2,
			wantCount:     2,
			wantLine0:     0,
			wantLine1:     1,
			wantPos1XZero: true,
		},
		{
			name:          "empty",
			text:          "",
			index:         0,
			wantCount:     1,
			wantLine0:     0,
			wantLine1:     -1,
			wantPos0XZero: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pos0, pos1, count := textutil.TextPositionFromIndex(&textutil.TextPositionFromIndexParams{
				Index:         tc.index,
				RenderingText: tc.text,
				Width:         1000,
				Options:       op,
			})
			if count != tc.wantCount {
				t.Fatalf("count: got %d, want %d", count, tc.wantCount)
			}
			if tc.wantLine0 >= 0 {
				if want := topOfLine(tc.wantLine0); pos0.Top != want {
					t.Errorf("pos0.Top: got %v, want %v (line %d)", pos0.Top, want, tc.wantLine0)
				}
			}
			if tc.wantLine1 >= 0 && count == 2 {
				if want := topOfLine(tc.wantLine1); pos1.Top != want {
					t.Errorf("pos1.Top: got %v, want %v (line %d)", pos1.Top, want, tc.wantLine1)
				}
			}
			if tc.wantPos0XZero && pos0.X != 0 {
				t.Errorf("pos0.X: got %v, want 0", pos0.X)
			}
			if tc.wantPos1XZero && count == 2 && pos1.X != 0 {
				t.Errorf("pos1.X: got %v, want 0", pos1.X)
			}
			// For "trailing-newline/end" specifically, the tail (pos0) must have
			// a non-zero X (after "a"), otherwise the selection rendering would
			// draw width=0 — the bug this regression test is guarding.
			if tc.name == "trailing-newline/end" && pos0.X <= 0 {
				t.Errorf("pos0.X for trailing newline tail: got %v, want > 0", pos0.X)
			}
		})
	}
}

// TestTextPositionFromIndexSidecarParity sweeps every byte index in a
// variety of inputs and asserts that the sidecar-accelerated path
// returns the same (pos0, pos1, count) as the unrestricted whole-
// document fallback. Covers both autoWrap modes and content with
// multibyte runes, trailing breaks, and CRLF.
func TestTextPositionFromIndexSidecarParity(t *testing.T) {
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
		{"two lines trailing", "abc\ndef\n"},
		{"three lines", "abc\ndef\nghi"},
		{"CRLF", "abc\r\ndef"},
		{"multibyte", "下\n中\n上"},
		{"empty trailing", "\n"},
		{"only breaks", "\n\n\n"},
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
				params := &textutil.TextPositionFromIndexParams{
					RenderingText:            tc.str,
					Width:                    width,
					Options:                  op,
					LineByteOffsets:          &l,
					PrecedingVisualLineCount: precedingVisualLineCountFromString(tc.str, width, autoWrap, face, 0, false),
				}

				for idx := 0; idx <= len(tc.str); idx++ {
					params.Index = idx
					wantP0, wantP1, wantCount := textutil.TextPositionFromIndex(withoutSidecar(params))
					gotP0, gotP1, gotCount := textutil.TextPositionFromIndex(params)
					if gotCount != wantCount {
						t.Errorf("idx=%d: count=%d, want %d", idx, gotCount, wantCount)
						continue
					}
					if gotCount >= 1 && gotP0 != wantP0 {
						t.Errorf("idx=%d: pos0=%+v, want %+v", idx, gotP0, wantP0)
					}
					if gotCount == 2 && gotP1 != wantP1 {
						t.Errorf("idx=%d: pos1=%+v, want %+v", idx, gotP1, wantP1)
					}
				}
			})
		}
	}
}

// TestTextPositionFromIndexSidecarAutoWrap exercises the autoWrap-with-
// real-wrapping path: a single long logical line that wraps at a narrow
// width into multiple visual sublines. The sidecar path must produce
// the same Y/X across every visual subline boundary.
func TestTextPositionFromIndexSidecarAutoWrap(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)
	op := &textutil.Options{
		Face:       face,
		LineHeight: lineHeight,
		AutoWrap:   true,
	}

	// Multiple logical lines, the middle one wraps.
	const narrowWidth = 80
	str := "first\nthe quick brown fox jumps over the lazy dog\nlast"

	var l textutil.LineByteOffsets
	l.RebuildFromString(str)
	params := &textutil.TextPositionFromIndexParams{
		RenderingText:            str,
		Width:                    narrowWidth,
		Options:                  op,
		LineByteOffsets:          &l,
		PrecedingVisualLineCount: precedingVisualLineCountFromString(str, narrowWidth, true, face, 0, false),
	}

	for idx := 0; idx <= len(str); idx++ {
		params.Index = idx
		wantP0, wantP1, wantCount := textutil.TextPositionFromIndex(withoutSidecar(params))
		gotP0, gotP1, gotCount := textutil.TextPositionFromIndex(params)
		if gotCount != wantCount {
			t.Errorf("idx=%d: count=%d, want %d", idx, gotCount, wantCount)
			continue
		}
		if gotCount >= 1 && gotP0 != wantP0 {
			t.Errorf("idx=%d: pos0=%+v, want %+v", idx, gotP0, wantP0)
		}
		if gotCount == 2 && gotP1 != wantP1 {
			t.Errorf("idx=%d: pos1=%+v, want %+v", idx, gotP1, wantP1)
		}
	}
}

// TestTextPositionFromIndexSidecarComposition verifies that an active
// IME composition (without a hard line break) is handled by the
// sidecar path: results match a from-scratch unrestricted walk of the
// already-spliced rendering text.
func TestTextPositionFromIndexSidecarComposition(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)

	type comp struct {
		sStart, sEnd, compLen int
		composition           string // inserted at sStart in rendering
	}
	cases := []struct {
		name      string
		committed string
		c         comp
	}{
		// Insert a single ASCII char at the start of line 0.
		{"insert at line0 start", "abc\ndef", comp{sStart: 0, sEnd: 0, compLen: 1, composition: "X"}},
		// Insert a 3-byte UTF-8 char inside line 1.
		{"insert mb in line1", "abc\ndef\nghi", comp{sStart: 5, sEnd: 5, compLen: 3, composition: "中"}},
		// Replace a 2-byte selection inside line 0 with 4 bytes.
		{"replace in line0", "abcdef\nghi", comp{sStart: 1, sEnd: 3, compLen: 4, composition: "WXYZ"}},
		// Composition at the very end of the document.
		{"insert at end", "abc\ndef", comp{sStart: 7, sEnd: 7, compLen: 2, composition: "YZ"}},
		// Composition at the start of a line that starts immediately after a hard break.
		{"insert at line1 start", "abc\ndef", comp{sStart: 4, sEnd: 4, compLen: 2, composition: "YZ"}},
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
				rendering := tc.committed[:tc.c.sStart] + tc.c.composition + tc.committed[tc.c.sEnd:]
				if len(tc.c.composition) != tc.c.compLen {
					t.Fatalf("test setup: compLen %d != len(composition) %d", tc.c.compLen, len(tc.c.composition))
				}
				var l textutil.LineByteOffsets
				l.RebuildFromString(tc.committed)
				params := &textutil.TextPositionFromIndexParams{
					RenderingText:            rendering,
					Width:                    width,
					Options:                  op,
					CommittedText:            tc.committed,
					LineByteOffsets:          &l,
					SelectionStart:           tc.c.sStart,
					SelectionEnd:             tc.c.sEnd,
					CompositionLen:           tc.c.compLen,
					PrecedingVisualLineCount: precedingVisualLineCountFromString(tc.committed, width, autoWrap, face, 0, false),
				}

				for idx := 0; idx <= len(rendering); idx++ {
					params.Index = idx
					wantP0, wantP1, wantCount := textutil.TextPositionFromIndex(withoutSidecar(params))
					gotP0, gotP1, gotCount := textutil.TextPositionFromIndex(params)
					if gotCount != wantCount {
						t.Errorf("idx=%d: count=%d, want %d", idx, gotCount, wantCount)
						continue
					}
					if gotCount >= 1 && gotP0 != wantP0 {
						t.Errorf("idx=%d: pos0=%+v, want %+v", idx, gotP0, wantP0)
					}
					if gotCount == 2 && gotP1 != wantP1 {
						t.Errorf("idx=%d: pos1=%+v, want %+v", idx, gotP1, wantP1)
					}
				}
			})
		}
	}
}

// TestTextPositionFromIndexSidecarCompositionWithLineBreak verifies
// that an IME composition containing a hard line break (which the
// sidecar can't service without a rebuild) still returns correct
// results — the implementation falls back to the unrestricted walk
// internally.
func TestTextPositionFromIndexSidecarCompositionWithLineBreak(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)
	op := &textutil.Options{Face: face, LineHeight: lineHeight}

	committed := "abc\ndef"
	// Composition with an embedded LF: replaces position 4..4 with "X\nY" (3 bytes).
	rendering := "abc\nX\nYdef"
	const width = math.MaxInt

	var l textutil.LineByteOffsets
	l.RebuildFromString(committed)
	params := &textutil.TextPositionFromIndexParams{
		RenderingText:            rendering,
		Width:                    width,
		Options:                  op,
		CommittedText:            committed,
		LineByteOffsets:          &l,
		SelectionStart:           4,
		SelectionEnd:             4,
		CompositionLen:           3,
		PrecedingVisualLineCount: precedingVisualLineCountFromString(committed, width, false, face, 0, false),
	}

	for idx := 0; idx <= len(rendering); idx++ {
		params.Index = idx
		wantP0, wantP1, wantCount := textutil.TextPositionFromIndex(withoutSidecar(params))
		gotP0, gotP1, gotCount := textutil.TextPositionFromIndex(params)
		if gotCount != wantCount {
			t.Errorf("idx=%d: count=%d, want %d", idx, gotCount, wantCount)
			continue
		}
		if gotCount >= 1 && gotP0 != wantP0 {
			t.Errorf("idx=%d: pos0=%+v, want %+v", idx, gotP0, wantP0)
		}
		if gotCount == 2 && gotP1 != wantP1 {
			t.Errorf("idx=%d: pos1=%+v, want %+v", idx, gotP1, wantP1)
		}
	}
}

// TestTextPositionFromIndexSidecarOutOfRange checks that out-of-range
// indices yield count=0 on the sidecar path.
func TestTextPositionFromIndexSidecarOutOfRange(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)
	op := &textutil.Options{Face: face, LineHeight: lineHeight}

	str := "abc"
	var l textutil.LineByteOffsets
	l.RebuildFromString(str)
	params := &textutil.TextPositionFromIndexParams{
		RenderingText:            str,
		Width:                    math.MaxInt,
		Options:                  op,
		LineByteOffsets:          &l,
		PrecedingVisualLineCount: func(int) int { return 0 },
	}

	for _, idx := range []int{-1, len(str) + 1, 1000} {
		params.Index = idx
		_, _, c := textutil.TextPositionFromIndex(params)
		if c != 0 {
			t.Errorf("idx=%d: count=%d, want 0", idx, c)
		}
	}
}

// TestTextPositionFromIndexNilSidecar verifies that nil LineByteOffsets
// and nil PrecedingVisualLineCount drive the unrestricted whole-document
// fallback (not a panic) and produce results consistent with what the
// fallback would produce on its own.
func TestTextPositionFromIndexNilSidecar(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)
	op := &textutil.Options{Face: face, LineHeight: lineHeight}

	str := "abc\ndef"
	const width = math.MaxInt
	for _, tc := range []struct {
		name   string
		params *textutil.TextPositionFromIndexParams
	}{
		{
			"nil offsets",
			&textutil.TextPositionFromIndexParams{
				RenderingText:            str,
				Width:                    width,
				Options:                  op,
				PrecedingVisualLineCount: func(int) int { return 0 },
			},
		},
		{
			"nil count fn",
			&textutil.TextPositionFromIndexParams{
				RenderingText: str,
				Width:         width,
				Options:       op,
				LineByteOffsets: func() *textutil.LineByteOffsets {
					var l textutil.LineByteOffsets
					l.RebuildFromString(str)
					return &l
				}(),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			noSidecar := &textutil.TextPositionFromIndexParams{
				RenderingText: str,
				Width:         width,
				Options:       op,
			}
			for idx := 0; idx <= len(str); idx++ {
				noSidecar.Index = idx
				tc.params.Index = idx
				wantP0, wantP1, wantCount := textutil.TextPositionFromIndex(noSidecar)
				gotP0, gotP1, gotCount := textutil.TextPositionFromIndex(tc.params)
				if gotCount != wantCount {
					t.Errorf("idx=%d: count=%d, want %d", idx, gotCount, wantCount)
					continue
				}
				if gotCount >= 1 && gotP0 != wantP0 {
					t.Errorf("idx=%d: pos0=%+v, want %+v", idx, gotP0, wantP0)
				}
				if gotCount == 2 && gotP1 != wantP1 {
					t.Errorf("idx=%d: pos1=%+v, want %+v", idx, gotP1, wantP1)
				}
			}
		})
	}
}
