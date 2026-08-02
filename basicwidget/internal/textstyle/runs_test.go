// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textstyle_test

import (
	"image/color"
	"math"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
)

func equalRuns(a, b []textstyle.Run) bool {
	return slices.EqualFunc(a, b, func(x, y textstyle.Run) bool {
		return x.Start == y.Start && x.End == y.End && x.Style.Equal(y.Style)
	})
}

func TestRunsSet(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}

	tests := []struct {
		name string
		ops  func(runs *textstyle.Runs)
		want []textstyle.Run
	}{
		{
			name: "single range",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 5, red)
			},
			want: []textstyle.Run{
				{Start: 0, End: 5, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "empty range is a no-op",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(5, 5, red)
				runs.SetColor(7, 3, red)
			},
			want: nil,
		},
		{
			name: "negative start is clamped",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(-3, 5, red)
			},
			want: []textstyle.Run{
				{Start: 0, End: 5, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "explicit nil color is an override",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 5, nil)
			},
			want: []textstyle.Run{
				{Start: 0, End: 5, Style: textstyle.Style{}.WithColor(nil)},
			},
		},
		{
			name: "partial overlap merges properties",
			ops: func(runs *textstyle.Runs) {
				runs.SetVariation(0, 9, font.TagWght, float32(text.WeightBold))
				runs.SetUnderline(5, 16, true)
			},
			want: []textstyle.Run{
				{Start: 0, End: 5, Style: textstyle.Style{}.WithVariation(font.TagWght, float32(text.WeightBold))},
				{Start: 5, End: 9, Style: textstyle.Style{}.WithVariation(font.TagWght, float32(text.WeightBold)).WithUnderline(true)},
				{Start: 9, End: 16, Style: textstyle.Style{}.WithUnderline(true)},
			},
		},
		{
			name: "later call wins per property",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 10, red)
				runs.SetUnderline(0, 10, true)
				runs.SetColor(3, 6, blue)
			},
			want: []textstyle.Run{
				{Start: 0, End: 3, Style: textstyle.Style{}.WithColor(red).WithUnderline(true)},
				{Start: 3, End: 6, Style: textstyle.Style{}.WithColor(blue).WithUnderline(true)},
				{Start: 6, End: 10, Style: textstyle.Style{}.WithColor(red).WithUnderline(true)},
			},
		},
		{
			name: "explicit false overrides true",
			ops: func(runs *textstyle.Runs) {
				runs.SetUnderline(0, 10, true)
				runs.SetUnderline(3, 6, false)
			},
			want: []textstyle.Run{
				{Start: 0, End: 3, Style: textstyle.Style{}.WithUnderline(true)},
				{Start: 3, End: 6, Style: textstyle.Style{}.WithUnderline(false)},
				{Start: 6, End: 10, Style: textstyle.Style{}.WithUnderline(true)},
			},
		},
		{
			name: "adjacent equal runs coalesce",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 5, red)
				runs.SetColor(5, 10, red)
			},
			want: []textstyle.Run{
				{Start: 0, End: 10, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "identical reapplication coalesces with split neighbors",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 10, red)
				runs.SetColor(3, 6, red)
			},
			want: []textstyle.Run{
				{Start: 0, End: 10, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "set over a gap between runs",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 3, red)
				runs.SetColor(6, 9, red)
				runs.SetUnderline(1, 8, true)
			},
			want: []textstyle.Run{
				{Start: 0, End: 1, Style: textstyle.Style{}.WithColor(red)},
				{Start: 1, End: 3, Style: textstyle.Style{}.WithColor(red).WithUnderline(true)},
				{Start: 3, End: 6, Style: textstyle.Style{}.WithUnderline(true)},
				{Start: 6, End: 8, Style: textstyle.Style{}.WithColor(red).WithUnderline(true)},
				{Start: 8, End: 9, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "whole-text set then a narrower one",
			ops: func(runs *textstyle.Runs) {
				runs.SetVariation(0, math.MaxInt, font.TagWght, float32(text.WeightBold))
				runs.SetColor(2, 4, red)
			},
			want: []textstyle.Run{
				{Start: 0, End: 2, Style: textstyle.Style{}.WithVariation(font.TagWght, float32(text.WeightBold))},
				{Start: 2, End: 4, Style: textstyle.Style{}.WithVariation(font.TagWght, float32(text.WeightBold)).WithColor(red)},
				{Start: 4, End: math.MaxInt, Style: textstyle.Style{}.WithVariation(font.TagWght, float32(text.WeightBold))},
			},
		},
		{
			name: "scale and lang",
			ops: func(runs *textstyle.Runs) {
				runs.SetScale(0, 4, 1.5)
				runs.SetLang(2, 6, language.Japanese)
			},
			want: []textstyle.Run{
				{Start: 0, End: 2, Style: textstyle.Style{}.WithScale(1.5)},
				{Start: 2, End: 4, Style: textstyle.Style{}.WithScale(1.5).WithLang(language.Japanese)},
				{Start: 4, End: 6, Style: textstyle.Style{}.WithLang(language.Japanese)},
			},
		},
		{
			name: "features merge by tag with the later value winning",
			ops: func(runs *textstyle.Runs) {
				runs.SetFeature(0, 10, font.TagLiga, 1)
				runs.SetFeature(5, 15, font.TagLiga, 0)
				runs.SetFeature(5, 15, font.TagTnum, 1)
			},
			want: []textstyle.Run{
				{Start: 0, End: 5, Style: textstyle.Style{}.WithFeature(font.TagLiga, 1)},
				{Start: 5, End: 15, Style: textstyle.Style{}.WithFeature(font.TagLiga, 0).WithFeature(font.TagTnum, 1)},
			},
		},
		{
			name: "feature lists stay canonical across merges",
			ops: func(runs *textstyle.Runs) {
				runs.SetFeature(0, 10, font.TagTnum, 1)
				runs.SetFeature(0, 10, font.TagLiga, 1)
				runs.SetFeature(0, 10, font.TagLiga, 0)
				runs.SetFeature(0, 10, font.TagLiga, 1)
			},
			want: []textstyle.Run{
				{Start: 0, End: 10, Style: textstyle.Style{}.WithFeature(font.TagLiga, 1).WithFeature(font.TagTnum, 1)},
			},
		},
		{
			name: "variations merge by tag",
			ops: func(runs *textstyle.Runs) {
				runs.SetVariation(0, 10, font.TagWght, float32(text.WeightNormal))
				runs.SetVariation(5, 10, font.TagWght, float32(text.WeightBold))
			},
			want: []textstyle.Run{
				{Start: 0, End: 5, Style: textstyle.Style{}.WithVariation(font.TagWght, float32(text.WeightNormal))},
				{Start: 5, End: 10, Style: textstyle.Style{}.WithVariation(font.TagWght, float32(text.WeightBold))},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var runs textstyle.Runs
			tt.ops(&runs)
			if got := slices.Collect(runs.All()); !equalRuns(got, tt.want) {
				t.Errorf("got: %+v, want: %+v", got, tt.want)
			}
		})
	}
}

func TestRunsUnset(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	tests := []struct {
		name string
		ops  func(runs *textstyle.Runs)
		want []textstyle.Run
	}{
		{
			name: "unset one property keeps the others",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 10, red)
				runs.SetUnderline(0, 10, true)
				runs.UnsetUnderline(0, 10)
			},
			want: []textstyle.Run{
				{Start: 0, End: 10, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "unset the only property removes the run",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 10, red)
				runs.UnsetColor(0, 10)
			},
			want: nil,
		},
		{
			name: "unset a partial range splits the run",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 10, red)
				runs.SetUnderline(0, 10, true)
				runs.UnsetColor(3, 6)
			},
			want: []textstyle.Run{
				{Start: 0, End: 3, Style: textstyle.Style{}.WithColor(red).WithUnderline(true)},
				{Start: 3, End: 6, Style: textstyle.Style{}.WithUnderline(true)},
				{Start: 6, End: 10, Style: textstyle.Style{}.WithColor(red).WithUnderline(true)},
			},
		},
		{
			name: "unset merges runs that become equal",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 10, red)
				runs.SetUnderline(3, 6, true)
				runs.UnsetUnderline(0, 10)
			},
			want: []textstyle.Run{
				{Start: 0, End: 10, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "unset a feature keeps other features",
			ops: func(runs *textstyle.Runs) {
				runs.SetFeature(0, 10, font.TagLiga, 1)
				runs.SetFeature(0, 10, font.TagTnum, 1)
				runs.UnsetFeature(0, 10, font.TagLiga)
			},
			want: []textstyle.Run{
				{Start: 0, End: 10, Style: textstyle.Style{}.WithFeature(font.TagTnum, 1)},
			},
		},
		{
			name: "unset a variation",
			ops: func(runs *textstyle.Runs) {
				runs.SetVariation(0, 10, font.TagWght, float32(text.WeightBold))
				runs.UnsetVariation(0, 10, font.TagWght)
			},
			want: nil,
		},
		{
			name: "unset an unrelated property is a no-op",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 10, red)
				runs.UnsetUnderline(0, 10)
				runs.UnsetFeature(0, 10, font.TagLiga)
			},
			want: []textstyle.Run{
				{Start: 0, End: 10, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "unset an empty range is a no-op",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 10, red)
				runs.UnsetColor(5, 5)
			},
			want: []textstyle.Run{
				{Start: 0, End: 10, Style: textstyle.Style{}.WithColor(red)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var runs textstyle.Runs
			tt.ops(&runs)
			if got := slices.Collect(runs.All()); !equalRuns(got, tt.want) {
				t.Errorf("got: %+v, want: %+v", got, tt.want)
			}
		})
	}
}

func TestRunsReset(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	type opRange struct {
		start int
		end   int
	}

	tests := []struct {
		name   string
		resets []opRange
		want   []textstyle.Run
	}{
		{
			name:   "middle split",
			resets: []opRange{{start: 3, end: 6}},
			want: []textstyle.Run{
				{Start: 0, End: 3, Style: textstyle.Style{}.WithColor(red)},
				{Start: 6, End: 10, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name:   "whole range",
			resets: []opRange{{start: 0, end: math.MaxInt}},
			want:   nil,
		},
		{
			name:   "outside range is a no-op",
			resets: []opRange{{start: 10, end: 20}},
			want: []textstyle.Run{
				{Start: 0, End: 10, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name:   "empty range is a no-op",
			resets: []opRange{{start: 5, end: 5}},
			want: []textstyle.Run{
				{Start: 0, End: 10, Style: textstyle.Style{}.WithColor(red)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var runs textstyle.Runs
			runs.SetColor(0, 10, red)
			for _, r := range tt.resets {
				runs.Reset(r.start, r.end)
			}
			if got := slices.Collect(runs.All()); !equalRuns(got, tt.want) {
				t.Errorf("got: %+v, want: %+v", got, tt.want)
			}
		})
	}
}

func TestRunsClear(t *testing.T) {
	var runs textstyle.Runs
	runs.SetUnderline(0, 10, true)
	runs.Clear()
	if got := slices.Collect(runs.All()); got != nil {
		t.Errorf("got: %+v, want: nil", got)
	}
}

func TestRunsStyleAt(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var runs textstyle.Runs
	runs.SetColor(2, 5, red)
	runs.SetUnderline(7, 9, true)

	tests := []struct {
		index int
		want  textstyle.Style
	}{
		{index: 0, want: textstyle.Style{}},
		{index: 1, want: textstyle.Style{}},
		{index: 2, want: textstyle.Style{}.WithColor(red)},
		{index: 4, want: textstyle.Style{}.WithColor(red)},
		{index: 5, want: textstyle.Style{}},
		{index: 6, want: textstyle.Style{}},
		{index: 7, want: textstyle.Style{}.WithUnderline(true)},
		{index: 8, want: textstyle.Style{}.WithUnderline(true)},
		{index: 9, want: textstyle.Style{}},
	}

	for _, tt := range tests {
		if got := runs.StyleAt(tt.index); !got.Equal(tt.want) {
			t.Errorf("StyleAt(%d): got: %+v, want: %+v", tt.index, got, tt.want)
		}
	}
}

func TestRunsStyleGetters(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var runs textstyle.Runs
	runs.SetColor(0, 10, red)
	runs.SetFeature(0, 10, font.TagTnum, 1)

	style := runs.StyleAt(0)
	if got, ok := style.Color(); !ok || got != color.Color(red) {
		t.Errorf("Color(): got: %v, %t, want: %v, true", got, ok, red)
	}
	if _, ok := style.Underline(); ok {
		t.Errorf("Underline(): got set, want unset")
	}
	if got, want := style.Features(), []font.Feature{{Tag: font.TagTnum, Value: 1}}; !slices.Equal(got, want) {
		t.Errorf("Features(): got: %+v, want: %+v", got, want)
	}
	if got := style.Variations(); got != nil {
		t.Errorf("Variations(): got: %+v, want: nil", got)
	}
}

func TestRunsReplace(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}

	tests := []struct {
		name string
		ops  func(runs *textstyle.Runs)
		want []textstyle.Run
	}{
		{
			name: "insertion before a run shifts it",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(5, 10, red)
				runs.Replace(2, 2, 3)
			},
			want: []textstyle.Run{
				{Start: 8, End: 13, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "insertion after a run keeps it",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(5, 10, red)
				runs.Replace(12, 12, 3)
			},
			want: []textstyle.Run{
				{Start: 5, End: 10, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "insertion strictly inside a run extends it",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(5, 10, red)
				runs.Replace(7, 7, 3)
			},
			want: []textstyle.Run{
				{Start: 5, End: 13, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "insertion at a run's start shifts it",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(5, 10, red)
				runs.Replace(5, 5, 3)
			},
			want: []textstyle.Run{
				{Start: 8, End: 13, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "insertion at a run's end extends it",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(5, 10, red)
				runs.Replace(10, 10, 3)
			},
			want: []textstyle.Run{
				{Start: 5, End: 13, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "insertion between adjacent runs extends the former",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 3, red)
				runs.SetColor(3, 6, blue)
				runs.Replace(3, 3, 2)
			},
			want: []textstyle.Run{
				{Start: 0, End: 5, Style: textstyle.Style{}.WithColor(red)},
				{Start: 5, End: 8, Style: textstyle.Style{}.WithColor(blue)},
			},
		},
		{
			name: "deletion before a run shifts it",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(5, 10, red)
				runs.Replace(0, 3, 0)
			},
			want: []textstyle.Run{
				{Start: 2, End: 7, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "deletion inside a run shrinks it",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(5, 10, red)
				runs.Replace(6, 8, 0)
			},
			want: []textstyle.Run{
				{Start: 5, End: 8, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "deletion covering a run removes it",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(5, 10, red)
				runs.Replace(4, 11, 0)
			},
			want: nil,
		},
		{
			name: "deletion overlapping a run's head truncates it",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(5, 10, red)
				runs.Replace(3, 7, 0)
			},
			want: []textstyle.Run{
				{Start: 3, End: 6, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "deletion overlapping a run's tail truncates it",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(5, 10, red)
				runs.Replace(8, 12, 0)
			},
			want: []textstyle.Run{
				{Start: 5, End: 8, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "deletion between equal runs merges them",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 3, red)
				runs.SetColor(5, 8, red)
				runs.Replace(3, 5, 0)
			},
			want: []textstyle.Run{
				{Start: 0, End: 6, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "deletion between different runs keeps them apart",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 3, red)
				runs.SetColor(5, 8, blue)
				runs.Replace(3, 5, 0)
			},
			want: []textstyle.Run{
				{Start: 0, End: 3, Style: textstyle.Style{}.WithColor(red)},
				{Start: 3, End: 6, Style: textstyle.Style{}.WithColor(blue)},
			},
		},
		{
			name: "replacement inside a run adopts the run's style",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(5, 10, red)
				runs.Replace(6, 8, 2)
			},
			want: []textstyle.Run{
				{Start: 5, End: 10, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "replacement across runs adopts the first replaced byte's style",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 5, red)
				runs.SetColor(5, 10, blue)
				runs.Replace(3, 7, 1)
			},
			want: []textstyle.Run{
				{Start: 0, End: 4, Style: textstyle.Style{}.WithColor(red)},
				{Start: 4, End: 7, Style: textstyle.Style{}.WithColor(blue)},
			},
		},
		{
			name: "replacement starting on unstyled text stays unstyled",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(5, 10, red)
				runs.Replace(3, 7, 2)
			},
			want: []textstyle.Run{
				{Start: 5, End: 8, Style: textstyle.Style{}.WithColor(red)},
			},
		},
		{
			name: "zero-length replacement is a no-op",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(5, 10, red)
				runs.Replace(3, 3, 0)
			},
			want: []textstyle.Run{
				{Start: 5, End: 10, Style: textstyle.Style{}.WithColor(red)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var runs textstyle.Runs
			tt.ops(&runs)
			if got := slices.Collect(runs.All()); !equalRuns(got, tt.want) {
				t.Errorf("got: %+v, want: %+v", got, tt.want)
			}
		})
	}
}

func TestRunsCopyRangeFrom(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}

	var src textstyle.Runs
	src.SetColor(2, 6, red)
	src.SetUnderline(8, 12, true)
	src.SetColor(12, 14, blue)

	tests := []struct {
		name       string
		start, end int
		want       []textstyle.Run
	}{
		{
			name:  "whole coverage rebases to 0",
			start: 2,
			end:   14,
			want: []textstyle.Run{
				{Start: 0, End: 4, Style: textstyle.Style{}.WithColor(red)},
				{Start: 6, End: 10, Style: textstyle.Style{}.WithUnderline(true)},
				{Start: 10, End: 12, Style: textstyle.Style{}.WithColor(blue)},
			},
		},
		{
			name:  "clipping keeps the overlapping parts",
			start: 4,
			end:   10,
			want: []textstyle.Run{
				{Start: 0, End: 2, Style: textstyle.Style{}.WithColor(red)},
				{Start: 4, End: 6, Style: textstyle.Style{}.WithUnderline(true)},
			},
		},
		{
			name:  "range without overrides is empty",
			start: 6,
			end:   8,
			want:  nil,
		},
		{
			name:  "empty range is empty",
			start: 5,
			end:   5,
			want:  nil,
		},
		{
			name:  "negative start is clamped",
			start: -3,
			end:   4,
			want: []textstyle.Run{
				{Start: 2, End: 4, Style: textstyle.Style{}.WithColor(red)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var runs textstyle.Runs
			// Preload runs to verify the copy replaces existing overrides.
			runs.SetColor(0, 100, blue)
			runs.CopyRangeFrom(&src, tt.start, tt.end)
			if got := slices.Collect(runs.All()); !equalRuns(got, tt.want) {
				t.Errorf("got: %+v, want: %+v", got, tt.want)
			}
		})
	}
}

func TestRunsApplyAt(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}

	var src textstyle.Runs
	src.SetColor(0, 3, red)
	src.SetUnderline(5, 8, true)

	t.Run("shifted apply on empty runs", func(t *testing.T) {
		var runs textstyle.Runs
		runs.ApplyAt(&src, 10)
		want := []textstyle.Run{
			{Start: 10, End: 13, Style: textstyle.Style{}.WithColor(red)},
			{Start: 15, End: 18, Style: textstyle.Style{}.WithUnderline(true)},
		}
		if got := slices.Collect(runs.All()); !equalRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}
	})

	t.Run("applies on top of existing overrides", func(t *testing.T) {
		var runs textstyle.Runs
		runs.SetColor(0, 20, blue)
		runs.ApplyAt(&src, 10)
		want := []textstyle.Run{
			{Start: 0, End: 10, Style: textstyle.Style{}.WithColor(blue)},
			{Start: 10, End: 13, Style: textstyle.Style{}.WithColor(red)},
			{Start: 13, End: 15, Style: textstyle.Style{}.WithColor(blue)},
			{Start: 15, End: 18, Style: textstyle.Style{}.WithColor(blue).WithUnderline(true)},
			{Start: 18, End: 20, Style: textstyle.Style{}.WithColor(blue)},
		}
		if got := slices.Collect(runs.All()); !equalRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}
	})

	t.Run("zero offset round-trips a copied range", func(t *testing.T) {
		var runs textstyle.Runs
		runs.ApplyAt(&src, 0)
		if got, want := slices.Collect(runs.All()), slices.Collect(src.All()); !equalRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}
	})
}

func TestRunsIsEmpty(t *testing.T) {
	var runs textstyle.Runs
	if !runs.IsEmpty() {
		t.Errorf("IsEmpty() = false, want true")
	}
	runs.SetUnderline(0, 5, true)
	if runs.IsEmpty() {
		t.Errorf("IsEmpty() = true, want false")
	}
	runs.Clear()
	if !runs.IsEmpty() {
		t.Errorf("IsEmpty() = false, want true")
	}
}

func TestRunsReplaceRange(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}

	var r textstyle.Runs
	r.SetColor(0, 10, red)
	r.SetUnderline(2, 4, true)

	var src textstyle.Runs
	src.SetColor(0, 3, blue)

	// [2, 6) is replaced: the underline override is dropped, blue covers
	// [2, 5), and red keeps covering the outside of the range.
	r.ReplaceRange(&src, 2, 6)

	if got, want := styleAtColor(&r, 1), color.Color(red); got != want {
		t.Errorf("color at 1 = %v; want %v", got, want)
	}
	for i := 2; i < 5; i++ {
		if got, want := styleAtColor(&r, i), color.Color(blue); got != want {
			t.Errorf("color at %d = %v; want %v", i, got, want)
		}
	}
	if got := r.StyleAt(5); !got.IsZero() {
		t.Errorf("style at 5 = %+v; want zero", got)
	}
	if got, want := styleAtColor(&r, 6), color.Color(red); got != want {
		t.Errorf("color at 6 = %v; want %v", got, want)
	}
	if underline, _ := r.StyleAt(2).Underline(); underline {
		t.Errorf("underline at 2 survived the replacement")
	}

	// Src overrides beyond the range length are ignored.
	var r2 textstyle.Runs
	var src2 textstyle.Runs
	src2.SetColor(0, 10, blue)
	r2.ReplaceRange(&src2, 0, 3)
	if got := r2.StyleAt(3); !got.IsZero() {
		t.Errorf("style at 3 = %+v; want zero", got)
	}
}

func TestRunsCopyMergedFrom(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}

	base := textstyle.Style{}.WithColor(red).WithItalic(false)

	var src textstyle.Runs
	src.SetColor(3, 5, blue)
	src.SetItalic(4, 6, true)

	var dst textstyle.Runs
	dst.CopyMergedFrom(&src, base, 2, 8)

	// [2, 3) has no override: base only, rebased to 0.
	s := dst.StyleAt(0)
	if clr, _ := s.Color(); clr != color.Color(red) {
		t.Errorf("color at 0 = %v; want %v", clr, red)
	}
	// [3, 5) has the blue override; [4, 6) is also italic.
	if clr, _ := dst.StyleAt(1).Color(); clr != color.Color(blue) {
		t.Errorf("color at 1 = %v; want %v", clr, blue)
	}
	if italic, _ := dst.StyleAt(1).Italic(); italic {
		t.Errorf("italic at 1 = true; want false")
	}
	if italic, _ := dst.StyleAt(2).Italic(); !italic {
		t.Errorf("italic at 2 = false; want true")
	}
	// [5, 6): italic override only, base color shows through the merge.
	if clr, _ := dst.StyleAt(3).Color(); clr != color.Color(red) {
		t.Errorf("color at 3 = %v; want %v", clr, red)
	}
	// [6, 8): base only.
	if clr, _ := dst.StyleAt(5).Color(); clr != color.Color(red) {
		t.Errorf("color at 5 = %v; want %v", clr, red)
	}
	if got := dst.StyleAt(6); !got.IsZero() {
		t.Errorf("style at 6 = %+v; want zero (past the range)", got)
	}
}

func styleAtColor(r *textstyle.Runs, i int) color.Color {
	clr, _ := r.StyleAt(i).Color()
	return clr
}
