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
