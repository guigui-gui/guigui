// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
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
	lbo.RebuildFromString(src)
	return src, &lbo
}

func TestComputeCompositionInfo_PureInsertion(t *testing.T) {
	// committed: "abc\ndef" (offsets 0, 4). Insert "XYZ" at byte 2.
	var offsets textutil.LineByteOffsets
	offsets.RebuildFromString("abc\ndef")
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
	offsets.RebuildFromString("abc\ndef")
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
	offsets.RebuildFromString("abc\ndef\nghi")
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
	offsets.RebuildFromString("abcdef\nghi")
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
	offsets.RebuildFromString("abc\ndef\nghi")
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
	offsets.RebuildFromString("abcdef")
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

// uniformAutoWrapOffParams returns a params skeleton describing a 10-line
// document where each logical line is exactly one visual line of height
// lineHeight and 10 bytes long. Composition is set to NoComposition.
// TotalHeight matches the actual text height.
func uniformAutoWrapOffParams(lineHeight int) textutil.VisibleRangeParams {
	const n = 10
	src, lbo := makeLineSource(n, 10)
	return textutil.VisibleRangeParams{
		LineByteOffsets: lbo,
		RenderingLength: len(src),
		LineHeight:      lineHeight,
		AutoWrap:        false,
		VerticalAlign:   textutil.VerticalAlignTop,
		BoundsHeight:    n * lineHeight,
		TotalHeight:     n * lineHeight,
		Composition:     textutil.CompositionInfo{},
	}
}

func TestComputeVisibleRange_LineCountZero(t *testing.T) {
	var empty textutil.LineByteOffsets // LineCount() == 0
	_, ok := textutil.ComputeVisibleRange(&textutil.VisibleRangeParams{
		LineByteOffsets: &empty,
		Composition:     textutil.CompositionInfo{},
	})
	if ok {
		t.Errorf("got ok=true, want false for empty input")
	}
}

func TestComputeVisibleRange_AutoWrapOffTopAlign(t *testing.T) {
	cases := []struct {
		name        string
		visMinY     int
		visMaxY     int
		wantFirst   int
		wantLast    int
		wantByteEnd int
		wantYShift  int
	}{
		// Visible window covers exactly one line. With one line of slack
		// on each side, FirstLine = max(0, line-1), LastLine = line+1
		// (capped at last line index).
		{"viewport on line 0", 0, 9, 0, 1, 20, 0},
		{"viewport on line 5", 50, 59, 4, 6, 70, 40},
		// Whole document visible.
		{"full document", 0, 100, 0, 9, 100, 0},
		// Viewport above the text: empty visible range, slack still
		// includes line 0 (capped at min) and extends LastLine to 1.
		{"viewport above text", -50, -1, 0, 1, 20, 0},
		// Viewport spans last few lines. relMinY=80 maps to line 8,
		// firstLine = 8-1 = 7 with slack.
		{"viewport bottom", 80, 100, 7, 9, 100, 70},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := uniformAutoWrapOffParams(10)
			p.VisibleMinY = tc.visMinY
			p.VisibleMaxY = tc.visMaxY
			r, ok := textutil.ComputeVisibleRange(&p)
			if !ok {
				t.Fatalf("got Restricted=false, want true")
			}
			if r.FirstLine != tc.wantFirst || r.LastLine != tc.wantLast {
				t.Errorf("got lines [%d, %d], want [%d, %d]", r.FirstLine, r.LastLine, tc.wantFirst, tc.wantLast)
			}
			if r.StartInBytes != tc.wantFirst*10 {
				t.Errorf("got StartInBytes=%d, want %d", r.StartInBytes, tc.wantFirst*10)
			}
			if r.EndInBytes != tc.wantByteEnd {
				t.Errorf("got EndInBytes=%d, want %d", r.EndInBytes, tc.wantByteEnd)
			}
			if r.YShift != tc.wantYShift {
				t.Errorf("got YShift=%d, want %d", r.YShift, tc.wantYShift)
			}
		})
	}
}

func TestComputeVisibleRange_AutoWrapOffMiddleAlign(t *testing.T) {
	// 10 lines × 10px = 100px content, in a 200px tall bounds → text is
	// centered, alignOffset = 50. Line 0 is at screen Y=50, line 9 ends at
	// Y=150. Visible window [80, 130] covers content rows [30, 80] which
	// is lines 3..8 (with slack: 2..9, then capped to 9).
	p := uniformAutoWrapOffParams(10)
	p.VerticalAlign = textutil.VerticalAlignMiddle
	p.BoundsHeight = 200
	p.VisibleMinY = 80
	p.VisibleMaxY = 130
	r, ok := textutil.ComputeVisibleRange(&p)
	if !ok {
		t.Fatalf("got Restricted=false")
	}
	if r.FirstLine != 2 {
		t.Errorf("got FirstLine=%d, want 2", r.FirstLine)
	}
	// AlignOffset = 50, lineY of line 2 = 20 → YShift = 70.
	if r.YShift != 70 {
		t.Errorf("got YShift=%d, want 70", r.YShift)
	}
}

func TestComputeVisibleRange_AutoWrapOffBottomAlign(t *testing.T) {
	// 10 lines × 10px in a 200px bounds, bottom-aligned → alignOffset = 100.
	// Visible [110, 130] → relMinY=10, relMaxY=30 → lines 1..3 with slack.
	p := uniformAutoWrapOffParams(10)
	p.VerticalAlign = textutil.VerticalAlignBottom
	p.BoundsHeight = 200
	p.VisibleMinY = 110
	p.VisibleMaxY = 130
	r, ok := textutil.ComputeVisibleRange(&p)
	if !ok {
		t.Fatalf("got Restricted=false")
	}
	if r.FirstLine != 0 || r.LastLine != 4 {
		t.Errorf("got lines [%d, %d], want [0, 4]", r.FirstLine, r.LastLine)
	}
	// AlignOffset = 100, lineY of line 0 = 0 → YShift = 100.
	if r.YShift != 100 {
		t.Errorf("got YShift=%d, want 100", r.YShift)
	}
}

func TestComputeVisibleRange_AutoWrapOnUniform(t *testing.T) {
	// Uniform-height autoWrap: cumulativeYs is the same as autoWrap-off
	// but consulted via binary search. 10 lines, each 10px tall.
	p := uniformAutoWrapOffParams(10)
	p.AutoWrap = true
	p.CumulativeYs = []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	p.VisibleMinY = 35
	p.VisibleMaxY = 65
	r, ok := textutil.ComputeVisibleRange(&p)
	if !ok {
		t.Fatalf("got Restricted=false")
	}
	// relMaxY=65: smallest i with cumYs[i]>=65 is 7 (cumYs[7]=70). lastLine=7.
	if r.FirstLine != 2 || r.LastLine != 7 {
		t.Errorf("got lines [%d, %d], want [2, 7]", r.FirstLine, r.LastLine)
	}
	if r.YShift != 20 {
		t.Errorf("got YShift=%d, want 20", r.YShift)
	}
}

func TestComputeVisibleRange_AutoWrapOnHeterogeneous(t *testing.T) {
	// Lines have heights [10, 30, 10, 30, 10] (autoWrap creates wider
	// visual heights for some logical lines). cumulativeYs = prefix sums.
	src, lbo := makeLineSource(5, 10)
	p := textutil.VisibleRangeParams{
		LineByteOffsets: lbo,
		RenderingLength: len(src),
		CumulativeYs:    []int{0, 10, 40, 50, 80, 90},
		LineHeight:      10,
		AutoWrap:        true,
		VerticalAlign:   textutil.VerticalAlignTop,
		BoundsHeight:    90,
		TotalHeight:     90,
		Composition:     textutil.CompositionInfo{},
	}
	// Visible [25, 75]: line 1 spans [10, 40), line 2 [40, 50), line 3
	// [50, 80). All three intersect; with slack, also includes 0 and 4.
	p.VisibleMinY = 25
	p.VisibleMaxY = 75
	r, ok := textutil.ComputeVisibleRange(&p)
	if !ok {
		t.Fatalf("got Restricted=false")
	}
	if r.FirstLine != 0 || r.LastLine != 4 {
		t.Errorf("got lines [%d, %d], want [0, 4]", r.FirstLine, r.LastLine)
	}
	if r.YShift != 0 {
		t.Errorf("got YShift=%d, want 0", r.YShift)
	}
}

func TestComputeVisibleRange_CompositionInsertionAutoWrapOff(t *testing.T) {
	// 10 lines × 10px. Composition inserts 5 bytes at line 3 (no
	// replacement), so netDelta=5 and rendering bytes past line 3
	// shift by +5. Line heights unchanged (autoWrap=off).
	p := uniformAutoWrapOffParams(10)
	p.Composition.LineIndex = 3
	p.Composition.RenderingByteShift = 5
	p.RenderingLength = 100 + 5
	p.VisibleMinY = 50
	p.VisibleMaxY = 70
	r, ok := textutil.ComputeVisibleRange(&p)
	if !ok {
		t.Fatalf("got Restricted=false")
	}
	// relMinY=50 → line 5, firstLine=4 with slack. relMaxY=70 → line 7,
	// lastLine=8 with slack.
	if r.FirstLine != 4 || r.LastLine != 8 {
		t.Errorf("got lines [%d, %d], want [4, 8]", r.FirstLine, r.LastLine)
	}
	// Lines 4..8 are all past CompLine=3, so byte offsets shift by +5.
	if r.StartInBytes != 4*10+5 {
		t.Errorf("got StartInBytes=%d, want %d", r.StartInBytes, 45)
	}
	if r.EndInBytes != 9*10+5 {
		t.Errorf("got EndInBytes=%d, want %d", r.EndInBytes, 95)
	}
	if r.YShift != 40 {
		t.Errorf("got YShift=%d, want 40", r.YShift)
	}
}

func TestComputeVisibleRange_CompositionStraddlesCompLine(t *testing.T) {
	// Slice starts before compLine and ends after it. ByteStart should
	// not include netDelta (FirstLine <= CompLine), ByteEnd should
	// include it (LastLine+1 > CompLine).
	p := uniformAutoWrapOffParams(10)
	p.Composition.LineIndex = 5
	p.Composition.RenderingByteShift = 7
	p.RenderingLength = 100 + 7
	// Visible [40, 70] → lines 4..7, slack to 3..8.
	p.VisibleMinY = 40
	p.VisibleMaxY = 70
	r, ok := textutil.ComputeVisibleRange(&p)
	if !ok {
		t.Fatalf("got Restricted=false")
	}
	if r.FirstLine != 3 || r.LastLine != 8 {
		t.Errorf("got lines [%d, %d], want [3, 8]", r.FirstLine, r.LastLine)
	}
	// FirstLine=3 <= CompLine=5: ByteStart = offsets[3] = 30 (no shift).
	if r.StartInBytes != 30 {
		t.Errorf("got StartInBytes=%d, want 30", r.StartInBytes)
	}
	// LastLine+1 = 9 > CompLine=5: ByteEnd = offsets[9] + netDelta = 90 + 7 = 97.
	if r.EndInBytes != 97 {
		t.Errorf("got EndInBytes=%d, want 97", r.EndInBytes)
	}
	// FirstLine=3 <= CompLine=5: no compDelta added to YShift.
	if r.YShift != 30 {
		t.Errorf("got YShift=%d, want 30", r.YShift)
	}
}

func TestComputeVisibleRange_CompositionAutoWrapOnDelta(t *testing.T) {
	// AutoWrap on. Composition on line 2 makes that line wrap into one
	// extra visual sub-line, captured as compDelta = 10. Lines 3..n in
	// rendering are at cumulativeYs[i] + 10.
	src, lbo := makeLineSource(5, 10)
	p := textutil.VisibleRangeParams{
		LineByteOffsets: lbo,
		RenderingLength: len(src) + 3, // composition added 3 bytes
		CumulativeYs:    []int{0, 10, 20, 30, 40, 50},
		LineHeight:      10,
		AutoWrap:        true,
		VerticalAlign:   textutil.VerticalAlignTop,
		BoundsHeight:    60,
		TotalHeight:     60,
		Composition: textutil.CompositionInfo{
			LineIndex:          2,
			RenderingByteShift: 3,
			RenderingYShift:    10,
		},
	}
	// Rendering Ys: line 0 at 0, 1 at 10, 2 at 20, 3 at 40, 4 at 50.
	// (Line 2 is now 20px tall in rendering.)
	// Visible [35, 55]: line 3 spans [40, 50), line 4 [50, 60). Both
	// included; with slack, also 2 and 4 (capped).
	p.VisibleMinY = 35
	p.VisibleMaxY = 55
	r, ok := textutil.ComputeVisibleRange(&p)
	if !ok {
		t.Fatalf("got Restricted=false")
	}
	// relMinY=35: smallest i with renderingYAt(i)>=36 is 3
	// (renderingYAt(3)=40). FirstLine = max(0, 3-2) = 1.
	if r.FirstLine != 1 || r.LastLine != 4 {
		t.Errorf("got lines [%d, %d], want [1, 4]", r.FirstLine, r.LastLine)
	}
	// FirstLine=1 < CompLine=2: no compDelta in YShift, lineY = 10.
	if r.YShift != 10 {
		t.Errorf("got YShift=%d, want 10", r.YShift)
	}
}
