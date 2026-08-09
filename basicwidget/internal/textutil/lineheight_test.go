// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
	"image"
	"math"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// flexibleFixture lays out "aaaa bbbb cccc" at a width that wraps it into
// one visual line per word, with the middle one covered by a face twice the
// base size.
func flexibleFixture(t *testing.T) (str string, width int, style textutil.Style) {
	t.Helper()
	small, large := testFaces(t)
	str = "aaaa bbbb cccc"
	width = int(math.Ceil(advanceOf("aaaa ", small)))
	style = textutil.Style{
		WrapMode:   textutil.WrapModeNormal,
		Face:       small,
		FaceRuns:   []textutil.FaceRun{{Start: 5, End: 9, Face: large}},
		LineHeight: 24,
	}
	return str, width, style
}

func TestLineHeightModeScalesOnlyTheLineWithTheLargerFace(t *testing.T) {
	str, width, style := flexibleFixture(t)

	fixed := textutil.MeasureHeight(width, str, style.WrapMode, style.Face, style.FaceRuns, style.LineHeight, textutil.LineHeightModeFixed, 0, false)
	if want := 3 * style.LineHeight; fixed != want {
		t.Fatalf("MeasureHeight(fixed) = %v, want %v", fixed, want)
	}

	// The middle visual line carries a face of twice the base size, so it is
	// twice as tall; the lines around it keep the base height.
	flexible := textutil.MeasureHeight(width, str, style.WrapMode, style.Face, style.FaceRuns, style.LineHeight, textutil.LineHeightModeFlexible, 0, false)
	if want := 4 * style.LineHeight; flexible != want {
		t.Errorf("MeasureHeight(flexible) = %v, want %v", flexible, want)
	}
}

func TestLineHeightModeFlexibleWithoutLargerFace(t *testing.T) {
	small, large := testFaces(t)
	const str = "aaaa bbbb"
	width := int(math.Ceil(advanceOf("aaaa ", small)))

	for _, tc := range []struct {
		name     string
		faceRuns []textutil.FaceRun
	}{
		{name: "no runs"},
		// A run smaller than the base face does not shrink its line: the
		// base face takes part in every line.
		{name: "smaller run", faceRuns: []textutil.FaceRun{{Start: 5, End: 9, Face: small}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixed := textutil.MeasureHeight(width, str, textutil.WrapModeNormal, large, tc.faceRuns, 24, textutil.LineHeightModeFixed, 0, false)
			flexible := textutil.MeasureHeight(width, str, textutil.WrapModeNormal, large, tc.faceRuns, 24, textutil.LineHeightModeFlexible, 0, false)
			if fixed != flexible {
				t.Errorf("MeasureHeight(flexible) = %v, MeasureHeight(fixed) = %v", flexible, fixed)
			}
		})
	}
}

func TestLineHeightModeFlexibleCaretPositions(t *testing.T) {
	str, width, style := flexibleFixture(t)
	style.LineHeightMode = textutil.LineHeightModeFlexible

	// Index 0 sits on the base-height first line, index 6 inside the scaled
	// second line.
	first, _, count := textutil.TextPositionFromIndexInLogicalLine(width, str, 0, 0, &style)
	if count == 0 {
		t.Fatal("TextPositionFromIndexInLogicalLine(0) found no position")
	}
	second, _, count := textutil.TextPositionFromIndexInLogicalLine(width, str, 0, 6, &style)
	if count == 0 {
		t.Fatal("TextPositionFromIndexInLogicalLine(6) found no position")
	}

	if first.Top < 0 || first.Bottom > style.LineHeight {
		t.Errorf("first line caret = [%v, %v], want within [0, %v]", first.Top, first.Bottom, style.LineHeight)
	}
	if second.Top < style.LineHeight {
		t.Errorf("second line caret top = %v, want at least %v", second.Top, style.LineHeight)
	}
	// The caret spans the content area, which scales with the line.
	if got, want := second.Bottom-second.Top, 2*(first.Bottom-first.Top); math.Abs(got-want) > 1e-9 {
		t.Errorf("second line caret height = %v, want %v", got, want)
	}
}

func TestLineHeightModeFlexibleHitTest(t *testing.T) {
	str, width, style := flexibleFixture(t)
	style.LineHeightMode = textutil.LineHeightModeFlexible

	// The second line's box spans [24, 72) once scaled, so this Y resolves
	// into "bbbb". Under the fixed mode the same Y lands on the third line.
	const y = 60
	// X=0 resolves to the head of the line covering y: "bbbb" at 5 under the
	// flexible mode, "cccc" at 10 under the fixed one.
	if got, want := textutil.TextIndexFromPositionInLogicalLine(width, image.Pt(0, y), str, 0, &style), 5; got != want {
		t.Errorf("TextIndexFromPositionInLogicalLine(flexible, y=%d) = %d, want %d", y, got, want)
	}

	fixedStyle := style
	fixedStyle.LineHeightMode = textutil.LineHeightModeFixed
	if got, want := textutil.TextIndexFromPositionInLogicalLine(width, image.Pt(0, y), str, 0, &fixedStyle), 10; got != want {
		t.Errorf("TextIndexFromPositionInLogicalLine(fixed, y=%d) = %d, want %d", y, got, want)
	}

	// The first line's box spans [0, 24): a Y inside it resolves to the
	// first line whatever the mode is.
	if got := textutil.TextIndexFromPositionInLogicalLine(width, image.Pt(0, 12), str, 0, &style); got > 5 {
		t.Errorf("TextIndexFromPositionInLogicalLine(y=12) = %d, want an offset in the first line", got)
	}
}
