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

// TestFaceRunsCompositionParity asserts the offset-accelerated hit-testing
// and caret paths match the whole-document fallbacks when face runs and an
// IME composition are both present. Style.FaceRuns carries rendering-text
// offsets and CommittedFaceRuns the same runs in committed-text offsets.
func TestFaceRunsCompositionParity(t *testing.T) {
	const lineHeight = 24.0
	small, large := testFaces(t)
	committed := "abcdef ghij klmno\npqr stu vwx yz\n\nthe last line"
	committedFaceRuns := []textutil.FaceRun{
		{Start: 3, End: 10, Face: large},
		{Start: 22, End: 30, Face: large},
	}

	cases := []struct {
		name              string
		selStart, selEnd  int
		comp              string
		renderingFaceRuns []textutil.FaceRun
	}{
		{
			// Insertion inside the first run extends it; the second run
			// shifts by the composition length.
			name:     "insertion inside a run",
			selStart: 5, selEnd: 5,
			comp: "XYZ",
			renderingFaceRuns: []textutil.FaceRun{
				{Start: 3, End: 13, Face: large},
				{Start: 25, End: 33, Face: large},
			},
		},
		{
			// Insertion at the first run's end extends it.
			name:     "insertion at a run end",
			selStart: 10, selEnd: 10,
			comp: "XY",
			renderingFaceRuns: []textutil.FaceRun{
				{Start: 3, End: 12, Face: large},
				{Start: 24, End: 32, Face: large},
			},
		},
		{
			// Replacement starting inside the first run stays part of it.
			name:     "replacement over a run",
			selStart: 8, selEnd: 12,
			comp: "0123456",
			renderingFaceRuns: []textutil.FaceRun{
				{Start: 3, End: 15, Face: large},
				{Start: 25, End: 33, Face: large},
			},
		},
	}

	for _, tc := range cases {
		for _, wrapMode := range []textutil.WrapMode{textutil.WrapModeNone, textutil.WrapModeNormal, textutil.WrapModeAnywhere} {
			for _, width := range []int{math.MaxInt, 80} {
				t.Run(fmt.Sprintf("%s width=%d%s", tc.name, width, wrapModeSuffix(wrapMode)), func(t *testing.T) {
					rendering := committed[:tc.selStart] + tc.comp + committed[tc.selEnd:]
					s := textutil.Style{
						Face:       small,
						FaceRuns:   tc.renderingFaceRuns,
						LineHeight: lineHeight,
						WrapMode:   wrapMode,
					}
					var l textutil.LineByteOffsets
					rebuildFromString(&l, committed)
					params := &textutil.TextLayoutParams{
						RenderingTextRange:         func(start, end int) string { return rendering[start:end] },
						RenderingTextLength:        len(rendering),
						Width:                      width,
						Style:                      s,
						CommittedTextRange:         func(start, end int) string { return committed[start:end] },
						CommittedFaceRuns:          committedFaceRuns,
						PrecomputedLineByteOffsets: &l,
						SelectionStart:             tc.selStart,
						SelectionEnd:               tc.selEnd,
						CompositionLen:             len(tc.comp),
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
}

// TestComputeCompositionInfoFaceRuns asserts the splice line's committed and
// rendering heights are measured with their respective face runs.
func TestComputeCompositionInfoFaceRuns(t *testing.T) {
	const (
		lineHeight = 24.0
		width      = 80
	)
	small, large := testFaces(t)
	committed := "xxxx\naaaa bbbb cccc"
	// The selection line starts at byte 5; the runs use whole-text offsets.
	const lineStart = 5
	committedFaceRuns := []textutil.FaceRun{{Start: 5, End: 9, Face: large}}
	const selStart, selEnd = 7, 7
	comp := "zzzz zzzz zzzz zzzz zzzz zzzz"
	renderingFaceRuns := []textutil.FaceRun{{Start: 5, End: 9 + len(comp), Face: large}}
	committedLine := committed[lineStart:]
	renderingLine := committed[lineStart:selStart] + comp + committed[selEnd:]

	var l textutil.LineByteOffsets
	rebuildFromString(&l, committed)
	info, ok := textutil.ComputeCompositionInfo(&textutil.CompositionInfoParams{
		CompositionText:           comp,
		LineByteOffsets:           &l,
		SelectionStart:            selStart,
		SelectionEnd:              selEnd,
		WrapMode:                  textutil.WrapModeNormal,
		CommittedSelectionLine:    committedLine,
		RenderingSelectionLine:    renderingLine,
		Face:                      small,
		LineHeight:                lineHeight,
		CommittedFaceRuns:         committedFaceRuns,
		RenderingFaceRuns:         renderingFaceRuns,
		SelectionLineStartInBytes: lineStart,
		WrapWidth:                 width,
	})
	if !ok {
		t.Fatal("ComputeCompositionInfo: ok = false, want true")
	}
	committedH := textutil.MeasureLogicalLineHeight(width, committedLine, textutil.WrapModeNormal, small, committedFaceRuns, lineStart, lineHeight, 0, false)
	renderingH := textutil.MeasureLogicalLineHeight(width, renderingLine, textutil.WrapModeNormal, small, renderingFaceRuns, lineStart, lineHeight, 0, false)
	want := int(math.Ceil(renderingH)) - int(math.Ceil(committedH))
	if info.RenderingYShift != want {
		t.Errorf("RenderingYShift = %d, want %d", info.RenderingYShift, want)
	}
	// Guard that the assertion has teeth: the same delta measured without
	// face runs must differ, or the run plumbing is unobservable here.
	runlessCommittedH := textutil.MeasureLogicalLineHeight(width, committedLine, textutil.WrapModeNormal, small, nil, 0, lineHeight, 0, false)
	runlessRenderingH := textutil.MeasureLogicalLineHeight(width, renderingLine, textutil.WrapModeNormal, small, nil, 0, lineHeight, 0, false)
	if runless := int(math.Ceil(runlessRenderingH)) - int(math.Ceil(runlessCommittedH)); runless == want {
		t.Fatalf("test data has no teeth: run-aware and runless deltas are both %d", want)
	}
}

// TestFaceRunsCompositionHintParity asserts the committed-text hint
// translation accounts for face runs: a hint placed past the splice line
// resolves like the whole-document walk anchored at the hint line's
// committed placement.
func TestFaceRunsCompositionHintParity(t *testing.T) {
	const (
		lineHeight = 24.0
		width      = 80
	)
	small, large := testFaces(t)
	committed := "abcdef ghij klmno\npqr stu vwx yz\n\nthe last line"
	committedFaceRuns := []textutil.FaceRun{{Start: 3, End: 10, Face: large}}
	const selStart, selEnd = 5, 5
	comp := "XYZ XYZ XYZ XYZ"
	renderingFaceRuns := []textutil.FaceRun{{Start: 3, End: 10 + len(comp), Face: large}}
	rendering := committed[:selStart] + comp + committed[selEnd:]

	var l textutil.LineByteOffsets
	rebuildFromString(&l, committed)
	params := &textutil.TextLayoutParams{
		RenderingTextRange:  func(start, end int) string { return rendering[start:end] },
		RenderingTextLength: len(rendering),
		Width:               width,
		Style: textutil.Style{
			Face:       small,
			FaceRuns:   renderingFaceRuns,
			LineHeight: lineHeight,
			WrapMode:   textutil.WrapModeNormal,
		},
		CommittedTextRange:         func(start, end int) string { return committed[start:end] },
		CommittedFaceRuns:          committedFaceRuns,
		PrecomputedLineByteOffsets: &l,
		SelectionStart:             selStart,
		SelectionEnd:               selEnd,
		CompositionLen:             len(comp),
		LogicalLineIndexHint:       2,
	}

	// The hint line's committed-anchored origin: the visual lines of the
	// committed lines before it, measured with the committed face runs.
	var originVL int
	for i := range 2 {
		cs := l.ByteOffsetByLineIndex(i)
		ce := l.ByteOffsetByLineIndex(i + 1)
		originVL += textutil.VisualLineCountForLogicalLine(width, committed[cs:ce], textutil.WrapModeNormal, small, committedFaceRuns, cs, 0, false)
	}
	originY := originVL * int(lineHeight)

	for y := -8 - originY; y < 6*int(lineHeight); y += 5 {
		for x := -8; x < 240; x += 7 {
			pos := image.Pt(x, y)
			got := textutil.TextIndexFromPosition(params, pos)
			want := textutil.TextIndexFromPosition(withoutIndexLineOffsets(params), pos.Add(image.Pt(0, originY)))
			if got != want {
				t.Fatalf("position=%v: index=%d, want %d", pos, got, want)
			}
		}
	}
}
