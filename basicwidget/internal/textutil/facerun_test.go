// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

func testFaces(t *testing.T) (small, large font.Face) {
	t.Helper()
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	smallFace := &text.GoTextFace{Source: source, Size: 16}
	largeFace := &text.GoTextFace{Source: source, Size: 32}
	return font.NewFaceForTest(smallFace, font.Attributes{Size: 16}),
		font.NewFaceForTest(largeFace, font.Attributes{Size: 32})
}

func advanceOf(str string, face font.Face) float64 {
	return text.AdvanceAt(str, len(str), face.TextFace())
}

func TestFaceAt(t *testing.T) {
	small, large := testFaces(t)
	faceRuns := []textutil.FaceRun{
		{Start: 2, End: 5, Face: large},
		{Start: 8, End: 10, Face: large},
	}

	tests := []struct {
		offset     int
		wantFace   font.Face
		wantChange int
	}{
		{offset: 0, wantFace: small, wantChange: 2},
		{offset: 1, wantFace: small, wantChange: 2},
		{offset: 2, wantFace: large, wantChange: 5},
		{offset: 4, wantFace: large, wantChange: 5},
		{offset: 5, wantFace: small, wantChange: 8},
		{offset: 7, wantFace: small, wantChange: 8},
		{offset: 8, wantFace: large, wantChange: 10},
		{offset: 10, wantFace: small, wantChange: math.MaxInt},
	}

	for _, tt := range tests {
		face, change := textutil.FaceAt(faceRuns, small, tt.offset)
		if face != tt.wantFace || change != tt.wantChange {
			t.Errorf("FaceAt(%d): got: %v, %d, want: %v, %d", tt.offset, face.Attributes().Size, change, tt.wantFace.Attributes().Size, tt.wantChange)
		}
	}
}

func TestAdvanceWithFaces(t *testing.T) {
	small, large := testFaces(t)

	t.Run("no runs matches single face", func(t *testing.T) {
		str := "Hello, world"
		got := textutil.AdvanceWithFaces(str, 0, len(str), small, nil, 0, false)
		want := advanceOf(str, small)
		if got != want {
			t.Errorf("got: %f, want: %f", got, want)
		}
	})

	t.Run("segments measured standalone per face", func(t *testing.T) {
		// "abcdef" with "cd" in the large face.
		faceRuns := []textutil.FaceRun{
			{Start: 2, End: 4, Face: large},
		}
		got := textutil.AdvanceWithFaces("abcdef", 0, 6, small, faceRuns, 0, false)
		want := advanceOf("ab", small) + advanceOf("cd", large) + advanceOf("ef", small)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("got: %f, want: %f", got, want)
		}
	})

	t.Run("base rebases run offsets", func(t *testing.T) {
		// The same text as above, but str starts at offset 2 within the
		// run coordinate space.
		faceRuns := []textutil.FaceRun{
			{Start: 2, End: 4, Face: large},
		}
		got := textutil.AdvanceWithFaces("cdef", 2, 4, small, faceRuns, 0, false)
		want := advanceOf("cd", large) + advanceOf("ef", small)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("got: %f, want: %f", got, want)
		}
	})

	t.Run("prefix measurement stops at index", func(t *testing.T) {
		faceRuns := []textutil.FaceRun{
			{Start: 2, End: 4, Face: large},
		}
		got := textutil.AdvanceWithFaces("abcdef", 0, 3, small, faceRuns, 0, false)
		want := advanceOf("ab", small) + advanceOf("c", large)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("got: %f, want: %f", got, want)
		}
	})

	t.Run("tab snaps to the next stop between faces", func(t *testing.T) {
		const tabWidth = 100
		faceRuns := []textutil.FaceRun{
			{Start: 3, End: 5, Face: large},
		}
		got := textutil.AdvanceWithFaces("ab\tcd", 0, 5, small, faceRuns, tabWidth, false)
		abWidth := advanceOf("ab", small)
		afterTab := float64(int(abWidth/tabWidth)+1) * tabWidth
		want := afterTab + advanceOf("cd", large)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("got: %f, want: %f", got, want)
		}
	})

	t.Run("trailing spaces are trimmed", func(t *testing.T) {
		faceRuns := []textutil.FaceRun{
			{Start: 0, End: 2, Face: large},
		}
		got := textutil.AdvanceWithFaces("ab  ", 0, 4, small, faceRuns, 0, false)
		want := advanceOf("ab", large)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("got: %f, want: %f", got, want)
		}
	})
}

// TestFaceRunsFastPathParity asserts the offset-accelerated hit-testing and
// caret paths match the whole-document fallbacks when face runs are present.
func TestFaceRunsFastPathParity(t *testing.T) {
	const lineHeight = 24.0
	small, large := testFaces(t)
	str := "abcdef ghij klmno\npqr stu vwx yz\n\nthe last line"
	faceRuns := []textutil.FaceRun{
		{Start: 3, End: 10, Face: large},
		{Start: 22, End: 30, Face: large},
	}
	for _, wrapMode := range []textutil.WrapMode{textutil.WrapModeNone, textutil.WrapModeNormal, textutil.WrapModeAnywhere} {
		for _, width := range []int{math.MaxInt, 80} {
			t.Run(fmt.Sprintf("width=%d%s", width, wrapModeSuffix(wrapMode)), func(t *testing.T) {
				s := textutil.Style{
					Face:       small,
					FaceRuns:   faceRuns,
					LineHeight: lineHeight,
					WrapMode:   wrapMode,
				}
				var l textutil.LineByteOffsets
				rebuildFromString(&l, str)
				params := &textutil.TextLayoutParams{
					RenderingTextRange:         func(start, end int) string { return str[start:end] },
					RenderingTextLength:        len(str),
					Width:                      width,
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

				for y := -8; y < 8*int(lineHeight); y += 7 {
					for x := -8; x < 240; x += 7 {
						pos := image.Pt(x, y)
						want := textutil.TextIndexFromPosition(withoutIndexLineOffsets(params), pos)
						got := textutil.TextIndexFromPosition(params, pos)
						if got != want {
							t.Fatalf("position=%v: index=%d, want %d", pos, got, want)
						}
					}
				}
			})
		}
	}
}

// TestFaceRunsMeasureHeightParity asserts that per-logical-line height
// measurement with face runs sums to the whole-document measurement.
func TestFaceRunsMeasureHeightParity(t *testing.T) {
	const lineHeight = 24.0
	small, large := testFaces(t)
	str := "abcdef ghij klmno\npqr stu vwx yz\n\nthe last line"
	faceRuns := []textutil.FaceRun{
		{Start: 3, End: 10, Face: large},
		{Start: 22, End: 30, Face: large},
	}
	for _, wrapMode := range []textutil.WrapMode{textutil.WrapModeNone, textutil.WrapModeNormal, textutil.WrapModeAnywhere} {
		for _, width := range []int{math.MaxInt, 80} {
			t.Run(fmt.Sprintf("width=%d%s", width, wrapModeSuffix(wrapMode)), func(t *testing.T) {
				whole := textutil.MeasureHeight(width, str, wrapMode, small, faceRuns, lineHeight, 0, false)
				var sum float64
				var start int
				for _, line := range strings.SplitAfter(str, "\n") {
					sum += textutil.MeasureLogicalLineHeight(width, line, wrapMode, small, faceRuns, start, lineHeight, 0, false)
					start += len(line)
				}
				if sum != whole {
					t.Errorf("sum of MeasureLogicalLineHeight = %v, MeasureHeight = %v", sum, whole)
				}
			})
		}
	}
}
