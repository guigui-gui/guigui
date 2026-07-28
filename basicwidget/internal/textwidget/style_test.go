// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget_test

import (
	"image/color"
	"slices"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
	"github.com/guigui-gui/guigui/basicwidget/internal/textwidget"
)

func equalStyleRuns(a, b []textstyle.Run) bool {
	return slices.EqualFunc(a, b, func(x, y textstyle.Run) bool {
		return x.Start == y.Start && x.End == y.End && x.Style.Equal(y.Style)
	})
}

func TestTextSetStyleInRange(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetValue("hello world")
	txt.SetColorInRange(0, 5, red)
	txt.SetUnderlineInRange(6, 11, true)

	got := txt.StyleRuns()
	if len(got) != 2 {
		t.Fatalf("got %d runs, want 2: %+v", len(got), got)
	}
	if got[0].Start != 0 || got[0].End != 5 {
		t.Errorf("got run range [%d, %d), want [0, 5)", got[0].Start, got[0].End)
	}
	if clr, ok := got[0].Style.Color(); !ok || clr != color.Color(red) {
		t.Errorf("Color(): got: %v, %t, want: %v, true", clr, ok, red)
	}
	if underline, ok := got[1].Style.Underline(); !ok || !underline {
		t.Errorf("Underline(): got: %t, %t, want: true, true", underline, ok)
	}
}

func TestTextStyleRunsLifetime(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	t.Run("equal SetValue keeps styles", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetValue("hello")
		txt.SetColorInRange(0, 5, red)
		txt.SetValue("hello")
		if got := txt.StyleRuns(); len(got) != 1 {
			t.Errorf("got %d runs, want 1: %+v", len(got), got)
		}
	})

	t.Run("changed SetValue clears styles", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetValue("hello")
		txt.SetColorInRange(0, 5, red)
		txt.SetValue("goodbye")
		if got := txt.StyleRuns(); got != nil {
			t.Errorf("got: %+v, want: nil", got)
		}
	})

	t.Run("replacement adopts the replaced text's style", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetEditable(true)
		txt.ForceSetValue("hello")
		txt.SetColorInRange(0, 5, red)
		txt.SetSelection(0, 2)
		txt.ReplaceValueAtSelection("xy")
		var wantRuns textstyle.Runs
		wantRuns.SetColor(0, 5, red)
		want := slices.Collect(wantRuns.All())
		if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}
	})

	t.Run("insertion shifts styles", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetEditable(true)
		txt.ForceSetValue("hello")
		txt.SetColorInRange(2, 4, red)
		txt.SetSelection(0, 0)
		txt.ReplaceValueAtSelection("ab")
		var wantRuns textstyle.Runs
		wantRuns.SetColor(4, 6, red)
		want := slices.Collect(wantRuns.All())
		if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}
	})

	t.Run("insertion inside a style extends it", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetEditable(true)
		txt.ForceSetValue("hello")
		txt.SetColorInRange(1, 4, red)
		txt.SetSelection(2, 2)
		txt.ReplaceValueAtSelection("ab")
		var wantRuns textstyle.Runs
		wantRuns.SetColor(1, 6, red)
		want := slices.Collect(wantRuns.All())
		if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}
	})

	t.Run("deletion merges adjacent equal styles", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetEditable(true)
		txt.ForceSetValue("hello world")
		txt.SetColorInRange(0, 3, red)
		txt.SetColorInRange(5, 8, red)
		txt.ReplaceTextAt("", 3, 5)
		var wantRuns textstyle.Runs
		wantRuns.SetColor(0, 6, red)
		want := slices.Collect(wantRuns.All())
		if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}
	})

	t.Run("unset and reset remove overrides", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetValue("hello")
		txt.SetColorInRange(0, 5, red)
		txt.SetUnderlineInRange(0, 5, true)
		txt.UnsetColorInRange(0, 5)
		var wantRuns textstyle.Runs
		wantRuns.SetUnderline(0, 5, true)
		want := slices.Collect(wantRuns.All())
		if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}
		txt.ResetStylesInRange(0, 5)
		if got := txt.StyleRuns(); got != nil {
			t.Errorf("got: %+v, want: nil", got)
		}
	})
}

func TestAppendFaceRunsThroughComposition(t *testing.T) {
	src := []textutil.FaceRun{
		{Start: 5, End: 10},
		{Start: 20, End: 25},
	}
	tests := []struct {
		name             string
		selStart, selEnd int
		compLen          int
		want             []textutil.FaceRun
	}{
		{
			name:     "insertion before the runs shifts them",
			selStart: 2, selEnd: 2, compLen: 3,
			want: []textutil.FaceRun{{Start: 8, End: 13}, {Start: 23, End: 28}},
		},
		{
			name:     "insertion at a run start shifts the run",
			selStart: 5, selEnd: 5, compLen: 3,
			want: []textutil.FaceRun{{Start: 8, End: 13}, {Start: 23, End: 28}},
		},
		{
			name:     "insertion inside a run extends it",
			selStart: 7, selEnd: 7, compLen: 3,
			want: []textutil.FaceRun{{Start: 5, End: 13}, {Start: 23, End: 28}},
		},
		{
			name:     "insertion at a run end extends it",
			selStart: 10, selEnd: 10, compLen: 3,
			want: []textutil.FaceRun{{Start: 5, End: 13}, {Start: 23, End: 28}},
		},
		{
			name:     "insertion between the runs",
			selStart: 15, selEnd: 15, compLen: 3,
			want: []textutil.FaceRun{{Start: 5, End: 10}, {Start: 23, End: 28}},
		},
		{
			name:     "replacement covering a run drops it",
			selStart: 4, selEnd: 11, compLen: 2,
			want: []textutil.FaceRun{{Start: 15, End: 20}},
		},
		{
			name:     "replacement starting inside a run stays part of it",
			selStart: 8, selEnd: 12, compLen: 4,
			want: []textutil.FaceRun{{Start: 5, End: 12}, {Start: 20, End: 25}},
		},
		{
			name:     "inverted selection is normalized",
			selStart: 12, selEnd: 8, compLen: 4,
			want: []textutil.FaceRun{{Start: 5, End: 12}, {Start: 20, End: 25}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := textwidget.AppendFaceRunsThroughComposition(nil, src, tt.selStart, tt.selEnd, tt.compLen)
			if !slices.EqualFunc(got, tt.want, func(x, y textutil.FaceRun) bool {
				return x.Start == y.Start && x.End == y.End
			}) {
				t.Errorf("got: %+v, want: %+v", got, tt.want)
			}
		})
	}
}
