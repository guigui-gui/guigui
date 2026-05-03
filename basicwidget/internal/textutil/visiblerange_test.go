// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
	"image"
	"strings"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// makeLineSource builds a (string, *LineByteOffsets) pair where each
// logical line is exactly lineLen bytes (lineLen-1 'x' chars plus '\n')
// except the last, which has lineLen 'x' chars and no trailing newline.
// The resulting offsets are 0, lineLen, 2*lineLen, ..., (count-1)*lineLen.
func makeLineSource(count, lineLen int) (string, *textutil.LineByteOffsets) {
	var sb strings.Builder
	for i := 0; i < count-1; i++ {
		sb.WriteString(strings.Repeat("x", lineLen-1))
		sb.WriteByte('\n')
	}
	sb.WriteString(strings.Repeat("x", lineLen))
	src := sb.String()
	var lbo textutil.LineByteOffsets
	rebuildFromString(&lbo, src)
	return src, &lbo
}

func TestComputeCompositionInfo_PureInsertion(t *testing.T) {
	// committed: "abc\ndef" (offsets 0, 4). Insert "XYZ" at byte 2.
	var offsets textutil.LineByteOffsets
	rebuildFromString(&offsets, "abc\ndef")
	got, ok := textutil.ComputeCompositionInfo(&textutil.CompositionInfoParams{
		CompositionText: "XYZ",
		LineByteOffsets: &offsets,
		SelectionStart:  2,
		SelectionEnd:    2,
	})
	if !ok {
		t.Fatalf("got ok=false, want true")
	}
	if got.LineIndex != 0 || got.RenderingByteShift != 3 || got.RenderingYShift != 0 {
		t.Errorf("got %+v, want {LineIndex:0, RenderingByteShift:3, RenderingYShift:0}", got)
	}
}

func TestComputeCompositionInfo_LineBreakInComposition(t *testing.T) {
	var offsets textutil.LineByteOffsets
	rebuildFromString(&offsets, "abc\ndef")
	_, ok := textutil.ComputeCompositionInfo(&textutil.CompositionInfoParams{
		CompositionText: "X\nYZ",
		LineByteOffsets: &offsets,
		SelectionStart:  2,
		SelectionEnd:    2,
	})
	if ok {
		t.Errorf("got ok=true, want false (composition contains a line break)")
	}
}

func TestComputeCompositionInfo_CrossLineSelection(t *testing.T) {
	// committed: "abc\ndef\nghi" (offsets 0, 4, 8). Selection 2..6 spans
	// line 0 and line 1.
	var offsets textutil.LineByteOffsets
	rebuildFromString(&offsets, "abc\ndef\nghi")
	_, ok := textutil.ComputeCompositionInfo(&textutil.CompositionInfoParams{
		CompositionText: "XYZ",
		LineByteOffsets: &offsets,
		SelectionStart:  2,
		SelectionEnd:    6,
	})
	if ok {
		t.Errorf("got ok=true, want false (selection spans two logical lines)")
	}
}

func TestComputeCompositionInfo_SameLineReplacement(t *testing.T) {
	// committed: "abcdef\nghi" (offsets 0, 7). Replace bytes 1..4 with "XY".
	var offsets textutil.LineByteOffsets
	rebuildFromString(&offsets, "abcdef\nghi")
	got, ok := textutil.ComputeCompositionInfo(&textutil.CompositionInfoParams{
		CompositionText: "XY",
		LineByteOffsets: &offsets,
		SelectionStart:  1,
		SelectionEnd:    4,
	})
	if !ok {
		t.Fatalf("got ok=false, want true")
	}
	// netDelta = 2 - (4-1) = -1.
	if got.LineIndex != 0 || got.RenderingByteShift != -1 || got.RenderingYShift != 0 {
		t.Errorf("got %+v, want {LineIndex:0, RenderingByteShift:-1, RenderingYShift:0}", got)
	}
}

func TestComputeCompositionInfo_CrossLineSelectionAutoWrap(t *testing.T) {
	// AutoWrap=on with a multi-line selection. The function must reject
	// (ok=false) without ever reading the selection-line fields, since
	// the caller can't safely compute them when ce+byteDelta would
	// underflow. Pass empty selection-line strings to verify the
	// rejection happens before they're consulted.
	face := newTestFace(t)
	var offsets textutil.LineByteOffsets
	rebuildFromString(&offsets, "abc\ndef\nghi")
	_, ok := textutil.ComputeCompositionInfo(&textutil.CompositionInfoParams{
		CompositionText:        "X",
		LineByteOffsets:        &offsets,
		SelectionStart:         2, // line 0
		SelectionEnd:           8, // line 2; byteDelta = 1 - 6 = -5
		AutoWrap:               true,
		CommittedSelectionLine: "",
		RenderingSelectionLine: "",
		Face:                   face,
		LineHeight:             24,
		WrapWidth:              1000,
	})
	if ok {
		t.Errorf("got ok=true, want false (selection spans multiple lines)")
	}
}

func TestComputeCompositionInfo_AutoWrapNoWrapChange(t *testing.T) {
	// AutoWrap=on with a wide enough width that the composition doesn't
	// add any wrap → CompDelta == 0.
	face := newTestFace(t)
	var offsets textutil.LineByteOffsets
	rebuildFromString(&offsets, "abcdef")
	got, ok := textutil.ComputeCompositionInfo(&textutil.CompositionInfoParams{
		CompositionText:        "XY",
		LineByteOffsets:        &offsets,
		SelectionStart:         2,
		SelectionEnd:           2,
		AutoWrap:               true,
		CommittedSelectionLine: "abcdef",
		RenderingSelectionLine: "abXYcdef",
		Face:                   face,
		LineHeight:             24,
		WrapWidth:              1000,
	})
	if !ok {
		t.Fatalf("got ok=false, want true")
	}
	if got.LineIndex != 0 || got.RenderingByteShift != 2 || got.RenderingYShift != 0 {
		t.Errorf("got %+v, want {LineIndex:0, RenderingByteShift:2, RenderingYShift:0}", got)
	}
}

// uniformAutoWrapOffViewportParams returns a params skeleton for
// [textutil.VisibleRangeInViewport] over a 10-line, 10-byte-per-line
// non-autoWrap document.
func uniformAutoWrapOffViewportParams(lineHeight int) textutil.VisibleRangeInViewportParams {
	const n = 10
	src, lbo := makeLineSource(n, 10)
	return textutil.VisibleRangeInViewportParams{
		LineByteOffsets:     lbo,
		RenderingTextRange:  func(start, end int) string { return src[start:end] },
		RenderingTextLength: len(src),
		LineHeight:          float64(lineHeight),
		AutoWrap:            false,
		Composition:         textutil.CompositionInfo{},
	}
}

func TestVisibleRangeInViewport_LineCountZero(t *testing.T) {
	var empty textutil.LineByteOffsets
	_, ok := textutil.VisibleRangeInViewport(&textutil.VisibleRangeInViewportParams{
		LineByteOffsets: &empty,
	})
	if ok {
		t.Errorf("got ok=true, want false for empty input")
	}
}

func TestVisibleRangeInViewport_AutoWrapOff(t *testing.T) {
	cases := []struct {
		name          string
		anchor        int
		visibleHeight int
		wantFirst     int
		wantLast      int
		wantStartByte int
		wantEndByte   int
	}{
		// Anchor at line 0, viewport covers 5 lines + slack.
		{
			name:          "anchor=0, viewport 50px",
			anchor:        0,
			visibleHeight: 50,
			wantFirst:     0,
			wantLast:      6,
			wantStartByte: 0,
			wantEndByte:   70,
		},
		// Anchor at line 5, viewport covers 3 lines + slack.
		{
			name:          "anchor=5, viewport 30px",
			anchor:        5,
			visibleHeight: 30,
			wantFirst:     5,
			wantLast:      9,
			wantStartByte: 50,
			wantEndByte:   100,
		},
		// Anchor at last line: nothing to walk past.
		{
			name:          "anchor=last",
			anchor:        9,
			visibleHeight: 100,
			wantFirst:     9,
			wantLast:      9,
			wantStartByte: 90,
			wantEndByte:   100,
		},
		// Tall viewport: walk runs to the end.
		{
			name:          "anchor=3, viewport 1000px",
			anchor:        3,
			visibleHeight: 1000,
			wantFirst:     3,
			wantLast:      9,
			wantStartByte: 30,
			wantEndByte:   100,
		},
		// Zero visible height: just the slack line beyond anchor.
		{
			name:          "anchor=2, viewport 0px",
			anchor:        2,
			visibleHeight: 0,
			wantFirst:     2,
			wantLast:      3,
			wantStartByte: 20,
			wantEndByte:   40,
		},
		// Anchor past document end clamped to last line.
		{
			name:          "anchor=20 clamps",
			anchor:        20,
			visibleHeight: 0,
			wantFirst:     9,
			wantLast:      9,
			wantStartByte: 90,
			wantEndByte:   100,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := uniformAutoWrapOffViewportParams(10)
			p.FirstLogicalLineInViewport = tc.anchor
			p.ViewportSize.Y = tc.visibleHeight
			r, ok := textutil.VisibleRangeInViewport(&p)
			if !ok {
				t.Fatalf("got ok=false, want true")
			}
			if r.FirstLine != tc.wantFirst || r.LastLine != tc.wantLast {
				t.Errorf("got lines [%d, %d], want [%d, %d]", r.FirstLine, r.LastLine, tc.wantFirst, tc.wantLast)
			}
			if r.StartInBytes != tc.wantStartByte {
				t.Errorf("got StartInBytes=%d, want %d", r.StartInBytes, tc.wantStartByte)
			}
			if r.EndInBytes != tc.wantEndByte {
				t.Errorf("got EndInBytes=%d, want %d", r.EndInBytes, tc.wantEndByte)
			}
			if r.YShift != 0 {
				t.Errorf("got YShift=%d, want 0", r.YShift)
			}
		})
	}
}

func TestVisibleRangeInViewport_AutoWrapOn(t *testing.T) {
	// Walk forward from an anchor, measuring per-line heights with a
	// real face. Use a narrow width to force the long middle line to
	// wrap; verify the walker stops once the cumulative height covers
	// the visible region.
	face := newTestFace(t)
	const lineHeight = 24.0
	const narrowWidth = 80
	str := "first\nthe quick brown fox jumps over the lazy dog\nlast"
	var lbo textutil.LineByteOffsets
	rebuildFromString(&lbo, str)

	// Sanity: the long middle line wraps into multiple visual lines.
	midStart := lbo.ByteOffsetByLineIndex(1)
	midEnd := lbo.ByteOffsetByLineIndex(2)
	wraps := textutil.VisualLineCountForLogicalLine(narrowWidth, str[midStart:midEnd], true, face, 0, false)
	if wraps < 2 {
		t.Fatalf("expected the middle line to wrap; got wraps=%d", wraps)
	}

	p := textutil.VisibleRangeInViewportParams{
		FirstLogicalLineInViewport: 0,
		LineByteOffsets:            &lbo,
		RenderingTextRange:         func(start, end int) string { return str[start:end] },
		RenderingTextLength:        len(str),
		// Cover line 0 (1 visual) + part of line 1 (multiple wraps).
		// The walker must not stop until line 1's full height is
		// accounted for, then include line 2 as the slack.
		ViewportSize: image.Pt(narrowWidth, int(lineHeight+lineHeight*float64(wraps))),
		Face:         face,
		LineHeight:   lineHeight,
		AutoWrap:     true,
	}
	r, ok := textutil.VisibleRangeInViewport(&p)
	if !ok {
		t.Fatalf("got ok=false, want true")
	}
	if r.FirstLine != 0 {
		t.Errorf("got FirstLine=%d, want 0", r.FirstLine)
	}
	if r.LastLine < 2 {
		t.Errorf("got LastLine=%d, want >=2 (covers all three lines)", r.LastLine)
	}
	if r.YShift != 0 {
		t.Errorf("got YShift=%d, want 0", r.YShift)
	}
	// EndInBytes covers up through line LastLine.
	if r.LastLine == 2 && r.EndInBytes != len(str) {
		t.Errorf("got EndInBytes=%d, want %d (full doc)", r.EndInBytes, len(str))
	}
}

func TestVisibleRangeInViewport_FirstLinePastCompositionLine(t *testing.T) {
	// FirstLogicalLineInViewport sits past the composition line, so
	// the rendering byte range for it and subsequent lines must shift
	// by RenderingByteShift.
	p := uniformAutoWrapOffViewportParams(10)
	p.Composition = textutil.CompositionInfo{
		LineIndex:          2,
		RenderingByteShift: 5,
	}
	p.RenderingTextLength = 100 + 5
	p.FirstLogicalLineInViewport = 4
	p.ViewportSize.Y = 30
	r, ok := textutil.VisibleRangeInViewport(&p)
	if !ok {
		t.Fatalf("got ok=false, want true")
	}
	if r.FirstLine != 4 {
		t.Errorf("got FirstLine=%d, want 4", r.FirstLine)
	}
	// FirstLine=4 > CompLine=2: StartInBytes shifted by +5.
	if r.StartInBytes != 4*10+5 {
		t.Errorf("got StartInBytes=%d, want %d", r.StartInBytes, 4*10+5)
	}
}
