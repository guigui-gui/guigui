// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
	"math"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// withoutLineOffsets returns a shallow copy of p with the precomputed
// logical-line offset fields cleared. Parity tests use this to drive the
// unrestricted whole-document fallback inside [textutil.TextPositionFromIndex]
// as the reference value they compare the offset-accelerated path against.
func withoutLineOffsets(p *textutil.TextLayoutParams) *textutil.TextLayoutParams {
	q := *p
	q.PrecomputedLineByteOffsets = nil
	q.LogicalLineIndexHint = 0
	q.VisualLineIndexHint = 0
	return &q
}

// precedingVisualLineCountFromString returns a function that gives the
// committed-text visual-line count from line 0 up to lineIdx — used by
// tests to compute VisualLineIndexHint values for non-zero
// LogicalLineIndexHint inputs.
func precedingVisualLineCountFromString(committed string, width int, wrapMode textutil.WrapMode, face font.Face, tabWidth float64, keepTailingSpace bool) func(int) int {
	var l textutil.LineByteOffsets
	rebuildFromString(&l, committed)
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
			sum += textutil.VisualLineCountForLogicalLine(width, committed[start:end], wrapMode, face, tabWidth, keepTailingSpace)
		}
		return sum
	}
}

func TestTextPositionFromIndex(t *testing.T) {
	face := newTestFace(t)
	const lineHeight = 24.0
	s := textutil.Style{
		Font:       face,
		LineHeight: lineHeight,
	}

	// Baseline: position at index 0 of a single-line string sits on visual
	// line 0. Use it to derive line N's Top without hard-coding the face's
	// vertical padding.
	baseline, _, _ := textutil.TextPositionFromIndex(&textutil.TextLayoutParams{
		RenderingTextRange:  func(start, end int) string { return "a"[start:end] },
		RenderingTextLength: 1,
		Width:               1000,
		Style:               s,
	}, 0)
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
			pos0, pos1, count := textutil.TextPositionFromIndex(&textutil.TextLayoutParams{
				RenderingTextRange:  func(start, end int) string { return tc.text[start:end] },
				RenderingTextLength: len(tc.text),
				Width:               1000,
				Style:               s,
			}, tc.index)
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

// TestTextPositionFromIndexLineOffsetsParity sweeps every byte index in a
// variety of inputs and asserts that the offset-accelerated path
// returns the same (pos0, pos1, count) as the unrestricted whole-
// document fallback. Covers all wrap modes and content with multibyte
// runes, trailing breaks, and CRLF.
func TestTextPositionFromIndexLineOffsetsParity(t *testing.T) {
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
		{"multibyte", "一\n二\n三"},
		{"empty trailing", "\n"},
		{"only breaks", "\n\n\n"},
	}

	for _, wrapMode := range []textutil.WrapMode{textutil.WrapModeNone, textutil.WrapModeNormal, textutil.WrapModeAnywhere} {
		for _, tc := range cases {
			t.Run(tc.name+wrapModeSuffix(wrapMode), func(t *testing.T) {
				const width = math.MaxInt
				s := textutil.Style{
					Font:       face,
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

				for idx := 0; idx <= len(tc.str); idx++ {
					wantP0, wantP1, wantCount := textutil.TextPositionFromIndex(withoutLineOffsets(params), idx)
					gotP0, gotP1, gotCount := textutil.TextPositionFromIndex(params, idx)
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

// TestTextPositionFromIndexLineOffsetsWordWrap exercises the [WrapModeNormal]
// path with real width-induced wrapping: a single long logical line that
// wraps at a narrow width into multiple visual sublines. The offset
// path must produce the same Y/X across every visual subline boundary.
func TestTextPositionFromIndexLineOffsetsWordWrap(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)
	s := textutil.Style{
		Font:       face,
		LineHeight: lineHeight,
		WrapMode:   textutil.WrapModeNormal,
	}

	// Multiple logical lines, the middle one wraps.
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

	for idx := 0; idx <= len(str); idx++ {
		wantP0, wantP1, wantCount := textutil.TextPositionFromIndex(withoutLineOffsets(params), idx)
		gotP0, gotP1, gotCount := textutil.TextPositionFromIndex(params, idx)
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

// TestTextPositionFromIndexViewportRelativeHint covers the virtualized
// caller's contract: pass (LogicalLineIndexHint = firstVisibleLine,
// VisualLineIndexHint = 0) so the returned pos.Top is measured from
// the top of the first visible line rather than the document top. The
// walk must step from the hint regardless of wrap mode — a [WrapModeNone]
// shortcut that treated the result as absolute would land cursors far
// above the viewport once the caller scrolls.
func TestTextPositionFromIndexViewportRelativeHint(t *testing.T) {
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
			s := textutil.Style{Font: face, LineHeight: lineHeight, WrapMode: wrapMode}
			var l textutil.LineByteOffsets
			rebuildFromString(&l, str)
			precVL := precedingVisualLineCountFromString(str, math.MaxInt, wrapMode, face, 0, false)

			for _, firstVisible := range []int{0, 1, 10, 30, 49} {
				offset := float64(precVL(firstVisible)) * lineHeight
				for _, idx := range []int{0, 1, 5, 30, 60, len(str) - 1, len(str)} {
					params := &textutil.TextLayoutParams{
						RenderingTextRange:         func(start, end int) string { return str[start:end] },
						RenderingTextLength:        len(str),
						Width:                      math.MaxInt,
						Style:                      s,
						PrecomputedLineByteOffsets: &l,
						LogicalLineIndexHint:       firstVisible,
						VisualLineIndexHint:        0,
					}
					wantP0, wantP1, wantCount := textutil.TextPositionFromIndex(withoutLineOffsets(params), idx)
					gotP0, gotP1, gotCount := textutil.TextPositionFromIndex(params, idx)

					// The hint shifts pos.Top by the visual-line count
					// preceding firstVisible, but X is unaffected.
					adjust := func(p textutil.TextPosition) textutil.TextPosition {
						p.Top -= offset
						p.Bottom -= offset
						return p
					}
					if wantCount >= 1 {
						wantP0 = adjust(wantP0)
					}
					if wantCount == 2 {
						wantP1 = adjust(wantP1)
					}

					if gotCount != wantCount {
						t.Errorf("firstVisible=%d idx=%d: count=%d, want %d", firstVisible, idx, gotCount, wantCount)
						continue
					}
					if gotCount >= 1 && gotP0 != wantP0 {
						t.Errorf("firstVisible=%d idx=%d: pos0=%+v, want %+v", firstVisible, idx, gotP0, wantP0)
					}
					if gotCount == 2 && gotP1 != wantP1 {
						t.Errorf("firstVisible=%d idx=%d: pos1=%+v, want %+v", firstVisible, idx, gotP1, wantP1)
					}
				}
			}
		})
	}
}

// TestTextPositionFromIndexLineOffsetsComposition verifies that an active
// IME composition (without a hard line break) is handled by the
// offset path: results match a from-scratch unrestricted walk of the
// already-spliced rendering text.
func TestTextPositionFromIndexLineOffsetsComposition(t *testing.T) {
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

	for _, wrapMode := range []textutil.WrapMode{textutil.WrapModeNone, textutil.WrapModeNormal} {
		for _, tc := range cases {
			t.Run(tc.name+wrapModeSuffix(wrapMode), func(t *testing.T) {
				const width = math.MaxInt
				s := textutil.Style{
					Font:       face,
					LineHeight: lineHeight,
					WrapMode:   wrapMode,
				}
				rendering := tc.committed[:tc.c.sStart] + tc.c.composition + tc.committed[tc.c.sEnd:]
				if len(tc.c.composition) != tc.c.compLen {
					t.Fatalf("test setup: compLen %d != len(composition) %d", tc.c.compLen, len(tc.c.composition))
				}
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

				for idx := 0; idx <= len(rendering); idx++ {
					wantP0, wantP1, wantCount := textutil.TextPositionFromIndex(withoutLineOffsets(params), idx)
					gotP0, gotP1, gotCount := textutil.TextPositionFromIndex(params, idx)
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

// TestTextPositionFromIndexLineOffsetsCompositionWithLineBreak verifies
// that an IME composition containing a hard line break (which the
// precomputed logical-line offsets can't service without a rebuild) still
// returns correct results — the implementation falls back to the
// unrestricted walk internally.
func TestTextPositionFromIndexLineOffsetsCompositionWithLineBreak(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)
	s := textutil.Style{Font: face, LineHeight: lineHeight}

	committed := "abc\ndef"
	// Composition with an embedded LF: replaces position 4..4 with "X\nY" (3 bytes).
	rendering := "abc\nX\nYdef"
	const width = math.MaxInt

	var l textutil.LineByteOffsets
	rebuildFromString(&l, committed)
	params := &textutil.TextLayoutParams{
		RenderingTextRange:         func(start, end int) string { return rendering[start:end] },
		RenderingTextLength:        len(rendering),
		Width:                      width,
		Style:                      s,
		CommittedTextRange:         func(start, end int) string { return committed[start:end] },
		PrecomputedLineByteOffsets: &l,
		SelectionStart:             4,
		SelectionEnd:               4,
		CompositionLen:             3,
	}

	for idx := 0; idx <= len(rendering); idx++ {
		wantP0, wantP1, wantCount := textutil.TextPositionFromIndex(withoutLineOffsets(params), idx)
		gotP0, gotP1, gotCount := textutil.TextPositionFromIndex(params, idx)
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

// TestTextPositionFromIndexLineOffsetsOutOfRange checks that out-of-range
// indices yield count=0 on the offset path.
func TestTextPositionFromIndexLineOffsetsOutOfRange(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)
	s := textutil.Style{Font: face, LineHeight: lineHeight}

	str := "abc"
	var l textutil.LineByteOffsets
	rebuildFromString(&l, str)
	params := &textutil.TextLayoutParams{
		RenderingTextRange:         func(start, end int) string { return str[start:end] },
		RenderingTextLength:        len(str),
		Width:                      math.MaxInt,
		Style:                      s,
		PrecomputedLineByteOffsets: &l,
	}

	for _, idx := range []int{-1, len(str) + 1, 1000} {
		_, _, c := textutil.TextPositionFromIndex(params, idx)
		if c != 0 {
			t.Errorf("idx=%d: count=%d, want 0", idx, c)
		}
	}
}

// TestPositionWithinLogicalLineParity sweeps every byte index in a variety of
// inputs and asserts that PositionWithinLogicalLine returns the same X and Y
// as TextPositionFromIndex once the line's own Y offset is subtracted. The
// hard-line-break corner case where TextPositionFromIndex collapses the
// cursor's "tail of previous line" + "head of current line" into count=2 is
// the one place the two functions disagree by design — the within-line
// variant reports only count=1 at the head of the cursor's own logical line.
func TestPositionWithinLogicalLineParity(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)

	cases := []struct {
		name string
		str  string
	}{
		{
			name: "single line",
			str:  "abc",
		},
		{
			name: "two lines",
			str:  "abc\ndef",
		},
		{
			name: "three lines",
			str:  "abc\ndef\nghi",
		},
		{
			name: "trailing LF",
			str:  "abc\n",
		},
		{
			name: "multibyte",
			str:  "一\n二\n三",
		},
		{
			name: "only breaks",
			str:  "\n\n\n",
		},
	}

	for _, wrapMode := range []textutil.WrapMode{textutil.WrapModeNone, textutil.WrapModeNormal, textutil.WrapModeAnywhere} {
		for _, tc := range cases {
			t.Run(tc.name+wrapModeSuffix(wrapMode), func(t *testing.T) {
				const width = math.MaxInt
				s := textutil.Style{Font: face, LineHeight: lineHeight, WrapMode: wrapMode}
				var l textutil.LineByteOffsets
				rebuildFromString(&l, tc.str)
				precVL := precedingVisualLineCountFromString(tc.str, width, wrapMode, face, 0, false)
				params := &textutil.TextLayoutParams{
					RenderingTextRange:         func(start, end int) string { return tc.str[start:end] },
					RenderingTextLength:        len(tc.str),
					Width:                      width,
					Style:                      s,
					PrecomputedLineByteOffsets: &l,
				}

				for idx := 0; idx <= len(tc.str); idx++ {
					wantP0, wantP1, wantCount := textutil.TextPositionFromIndex(params, idx)
					gotLine, gotP0, gotP1, gotCount := textutil.PositionWithinLogicalLine(params, idx)

					if wantCount == 0 {
						if gotCount != 0 {
							t.Errorf("idx=%d: count=%d, want 0", idx, gotCount)
						}
						continue
					}

					off := float64(precVL(gotLine)) * lineHeight
					adjust := func(p textutil.TextPosition) textutil.TextPosition {
						p.Top -= off
						p.Bottom -= off
						return p
					}

					// Hard-line-break boundary: TextPositionFromIndex emits two
					// positions spanning two logical lines. PositionWithinLogicalLine
					// reports just the cursor's own line head — count drops to 1.
					if wantCount == 2 && wantP1.Top > wantP0.Top {
						if gotCount != 1 {
							t.Errorf("idx=%d (cross-line c=2): got count=%d, want 1", idx, gotCount)
							continue
						}
						if got, want := gotP0, adjust(wantP1); got != want {
							t.Errorf("idx=%d: head pos=%+v, want %+v (line-relative)", idx, got, want)
						}
						continue
					}

					if gotCount != wantCount {
						t.Errorf("idx=%d: count=%d, want %d", idx, gotCount, wantCount)
						continue
					}
					if got, want := gotP0, adjust(wantP0); got != want {
						t.Errorf("idx=%d: pos0=%+v, want %+v (line-relative)", idx, got, want)
					}
					if gotCount == 2 {
						if got, want := gotP1, adjust(wantP1); got != want {
							t.Errorf("idx=%d: pos1=%+v, want %+v (line-relative)", idx, got, want)
						}
					}
				}
			})
		}
	}
}

// TestTextPositionFromIndexNilLineOffsets verifies that nil PrecomputedLineByteOffsets
// drives the unrestricted whole-document fallback (not a panic) and
// produces results consistent with what the fallback would produce on
// its own.
func TestTextPositionFromIndexNilLineOffsets(t *testing.T) {
	const lineHeight = 24.0
	face := newTestFace(t)
	s := textutil.Style{Font: face, LineHeight: lineHeight}

	str := "abc\ndef"
	const width = math.MaxInt
	params := &textutil.TextLayoutParams{
		RenderingTextRange:  func(start, end int) string { return str[start:end] },
		RenderingTextLength: len(str),
		Width:               width,
		Style:               s,
	}
	noLineOffsets := &textutil.TextLayoutParams{
		RenderingTextRange:  func(start, end int) string { return str[start:end] },
		RenderingTextLength: len(str),
		Width:               width,
		Style:               s,
	}
	for idx := 0; idx <= len(str); idx++ {
		wantP0, wantP1, wantCount := textutil.TextPositionFromIndex(noLineOffsets, idx)
		gotP0, gotP1, gotCount := textutil.TextPositionFromIndex(params, idx)
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
