// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
	"image/color"
	"math"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

const decorationLineHeight = 48.0

// decorationBaseline returns the baseline of the visual line at
// visualLineIndex, in the coordinates the decorations are reported in. base is
// the widget's face, on whose baseline every face of a visual line is drawn.
func decorationBaseline(base font.Face, visualLineIndex int) float64 {
	m := base.TextFace().Metrics()
	padding := (decorationLineHeight - (m.HAscent + m.HDescent)) / 2
	return float64(visualLineIndex)*decorationLineHeight + padding + m.HAscent
}

// wantUnderline returns the underline position and thickness of a span
// resolved to face on the visual line at visualLineIndex.
func wantUnderline(base, face font.Face, visualLineIndex int) (y, thickness float64) {
	m := face.TextFace().Metrics()
	thickness = max(1, (m.HAscent+m.HDescent)*textutil.DecorationThicknessRatio)
	y = decorationBaseline(base, visualLineIndex) + textutil.UnderlineOffsetRatio*m.HDescent - thickness/2
	return y, thickness
}

// wantStrikethrough returns the strikethrough position and thickness of a span
// resolved to face on the visual line at visualLineIndex.
func wantStrikethrough(base, face font.Face, visualLineIndex int) (y, thickness float64) {
	m := face.TextFace().Metrics()
	thickness = max(1, (m.HAscent+m.HDescent)*textutil.DecorationThicknessRatio)
	y = decorationBaseline(base, visualLineIndex) - textutil.StrikethroughOffsetRatio*m.HAscent - thickness/2
	return y, thickness
}

// backgroundDrawOptions returns the options a background test draws with. The
// tab width is nonzero, as a widget's is, so that a line break is laid out
// with a space's advance.
func backgroundDrawOptions(face font.Face, runs []textutil.StyleRun) *textutil.DrawOptions {
	return &textutil.DrawOptions{
		Style: textutil.Style{
			WrapMode:         textutil.WrapModeNone,
			Face:             face,
			LineHeight:       decorationLineHeight,
			TabWidth:         advanceOf("    ", face),
			KeepTailingSpace: true,
		},
		TextColor: color.White,
		StyleRuns: runs,
	}
}

// TestBackgroundStopsAtLineBreak asserts that a background covering a line
// break stops at the last character of the line, not over the space-wide
// advance the line break is laid out with.
func TestBackgroundStopsAtLineBreak(t *testing.T) {
	small, _ := testFaces(t)
	const str = "ab\ncd"
	yellow := color.RGBA{R: 0xff, G: 0xff, A: 0xff}
	options := backgroundDrawOptions(small, []textutil.StyleRun{{Start: 0, End: len(str), BackgroundColor: yellow}})

	got := textutil.BackgroundsPerVisualLine(1000, str, options)
	if len(got) != 2 || len(got[0]) != 1 || len(got[1]) != 1 {
		t.Fatalf("got: %v, want one background on each of two visual lines", got)
	}

	if wantWidth := advanceOf("ab", small); math.Abs(got[0][0].Width-wantWidth) > 1e-9 {
		t.Errorf("width: got: %f, want: %f (the line break's advance is %f)", got[0][0].Width, wantWidth, advanceOf(" ", small))
	}
	if wantWidth := advanceOf("cd", small); math.Abs(got[1][0].Width-wantWidth) > 1e-9 {
		t.Errorf("width of the last line: got: %f, want: %f", got[1][0].Width, wantWidth)
	}
}

// TestBackgroundOfLineBreakOnly asserts that a background covering nothing but
// a line break is not drawn at all.
func TestBackgroundOfLineBreakOnly(t *testing.T) {
	small, _ := testFaces(t)
	const str = "ab\ncd"
	yellow := color.RGBA{R: 0xff, G: 0xff, A: 0xff}
	options := backgroundDrawOptions(small, []textutil.StyleRun{{Start: 2, End: 3, BackgroundColor: yellow}})

	got := textutil.BackgroundsPerVisualLine(1000, str, options)
	if len(got) != 2 {
		t.Fatalf("got: %v, want two visual lines", got)
	}
	for i, bs := range got {
		if len(bs) != 0 {
			t.Errorf("visual line %d: got: %v, want no background", i, bs)
		}
	}
}

// TestBackgroundStopsAtLineBreakWhenWrapped asserts that the rule holds for the
// last visual line of a wrapped logical line too.
func TestBackgroundStopsAtLineBreakWhenWrapped(t *testing.T) {
	small, _ := testFaces(t)
	const str = "AAAA BB\ncd"
	yellow := color.RGBA{R: 0xff, G: 0xff, A: 0xff}
	options := backgroundDrawOptions(small, []textutil.StyleRun{{Start: 0, End: len(str), BackgroundColor: yellow}})
	options.Style.WrapMode = textutil.WrapModeNormal
	layoutWidth := int(math.Ceil(advanceOf("AAAA ", small))) + 1

	got := textutil.BackgroundsPerVisualLine(layoutWidth, str, options)
	if len(got) != 3 || len(got[1]) != 1 {
		t.Fatalf("got: %v, want one background on the second of three visual lines", got)
	}

	if wantWidth := advanceOf("BB", small); math.Abs(got[1][0].Width-wantWidth) > 1e-9 {
		t.Errorf("width: got: %f, want: %f (the line break's advance is %f)", got[1][0].Width, wantWidth, advanceOf(" ", small))
	}
}

// TestDecorationInsideLargerFaceRun asserts that a decoration covering only
// bytes drawn with a larger face takes its metrics from that face, not from
// the widget's base face.
func TestDecorationInsideLargerFaceRun(t *testing.T) {
	small, large := testFaces(t)
	const str = "abcdef"
	// "cdef" is drawn with the large face and struck through.
	options := &textutil.DrawOptions{
		Style: textutil.Style{
			WrapMode:   textutil.WrapModeNone,
			Face:       small,
			FaceRuns:   []textutil.FaceRun{{Start: 2, End: 6, Face: large}},
			LineHeight: decorationLineHeight,
		},
		TextColor: color.White,
		StyleRuns: []textutil.StyleRun{{Start: 2, End: 6, Strikethrough: true}},
	}

	got := textutil.DecorationsPerVisualLine(1000, str, options)
	if len(got) != 1 || len(got[0]) != 1 {
		t.Fatalf("got: %v, want one decoration on one visual line", got)
	}

	wantY, wantThickness := wantStrikethrough(small, large, 0)
	baseY, _ := wantStrikethrough(small, small, 0)
	if wantY == baseY {
		t.Fatal("the faces do not differ in strikethrough position; the test cannot tell them apart")
	}
	d := got[0][0]
	if math.Abs(d.Y-wantY) > 1e-9 {
		t.Errorf("Y: got: %f, want: %f (the base face would give %f)", d.Y, wantY, baseY)
	}
	if math.Abs(d.Thickness-wantThickness) > 1e-9 {
		t.Errorf("Thickness: got: %f, want: %f", d.Thickness, wantThickness)
	}
	if wantX := advanceOf("ab", small); math.Abs(d.X-wantX) > 1e-9 {
		t.Errorf("X: got: %f, want: %f", d.X, wantX)
	}
}

// TestDecorationSpanningMixedFaceSizes asserts that a span crossing faces of
// different sizes is drawn at one position and thickness, taken from the
// smallest face, even when it is split into several runs by color.
func TestDecorationSpanningMixedFaceSizes(t *testing.T) {
	small, large := testFaces(t)
	const str = "abcdef"
	red := color.RGBA{R: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}
	// "def" is drawn with the large face, and the underline over "abcdef"
	// changes color at "d".
	options := &textutil.DrawOptions{
		Style: textutil.Style{
			WrapMode:   textutil.WrapModeNone,
			Face:       small,
			FaceRuns:   []textutil.FaceRun{{Start: 3, End: 6, Face: large}},
			LineHeight: decorationLineHeight,
		},
		TextColor: color.White,
		StyleRuns: []textutil.StyleRun{
			{Start: 0, End: 3, Underline: true, Color: red},
			{Start: 3, End: 6, Underline: true, Color: blue},
		},
	}

	got := textutil.DecorationsPerVisualLine(1000, str, options)
	if len(got) != 1 || len(got[0]) != 2 {
		t.Fatalf("got: %v, want two decorations on one visual line", got)
	}

	wantY, wantThickness := wantUnderline(small, small, 0)
	for i, d := range got[0] {
		if math.Abs(d.Y-wantY) > 1e-9 {
			t.Errorf("decoration %d Y: got: %f, want: %f", i, d.Y, wantY)
		}
		if math.Abs(d.Thickness-wantThickness) > 1e-9 {
			t.Errorf("decoration %d Thickness: got: %f, want: %f", i, d.Thickness, wantThickness)
		}
	}
	if got[0][0].Color != color.Color(red) || got[0][1].Color != color.Color(blue) {
		t.Errorf("colors: got: %v, %v, want: %v, %v", got[0][0].Color, got[0][1].Color, red, blue)
	}
	if end, next := got[0][0].X+got[0][0].Width, got[0][1].X; math.Abs(end-next) > 1e-9 {
		t.Errorf("the color parts are not contiguous: %f ends at %f, %f starts", got[0][0].X, end, next)
	}
}

// TestDecorationWrappedSpanResolvesPerVisualLine asserts that each visual line
// of a wrapped span resolves its metrics from the faces on that line alone.
func TestDecorationWrappedSpanResolvesPerVisualLine(t *testing.T) {
	small, large := testFaces(t)
	const str = "AAAA BBBB"
	// "AAAA " is drawn with the large face and wraps onto its own visual
	// line; "BBBB" is drawn with the base face.
	options := &textutil.DrawOptions{
		Style: textutil.Style{
			WrapMode:   textutil.WrapModeNormal,
			Face:       small,
			FaceRuns:   []textutil.FaceRun{{Start: 0, End: 5, Face: large}},
			LineHeight: decorationLineHeight,
		},
		TextColor: color.White,
		StyleRuns: []textutil.StyleRun{{Start: 0, End: len(str), Underline: true}},
	}
	layoutWidth := int(math.Ceil(advanceOf("AAAA", large))) + 1

	got := textutil.DecorationsPerVisualLine(layoutWidth, str, options)
	if len(got) != 2 || len(got[0]) != 1 || len(got[1]) != 1 {
		t.Fatalf("got: %v, want one decoration on each of two visual lines", got)
	}

	tests := []struct {
		visualLineIndex int
		face            font.Face
	}{
		{visualLineIndex: 0, face: large},
		{visualLineIndex: 1, face: small},
	}
	for _, tt := range tests {
		wantY, wantThickness := wantUnderline(small, tt.face, tt.visualLineIndex)
		d := got[tt.visualLineIndex][0]
		if math.Abs(d.Y-wantY) > 1e-9 {
			t.Errorf("visual line %d Y: got: %f, want: %f", tt.visualLineIndex, d.Y, wantY)
		}
		if math.Abs(d.Thickness-wantThickness) > 1e-9 {
			t.Errorf("visual line %d Thickness: got: %f, want: %f", tt.visualLineIndex, d.Thickness, wantThickness)
		}
	}
}
