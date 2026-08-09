// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
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
