// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget_test

import (
	"image/color"
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
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

func TestTextIsBoldInRange(t *testing.T) {
	bold := float32(text.WeightBold)

	t.Run("default base style", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetValue("hello world")
		if txt.IsBoldInRange(0, 5) {
			t.Error("IsBoldInRange must return false without a bold weight")
		}
		txt.SetVariationInRange(0, 5, font.TagWght, bold)
		if !txt.IsBoldInRange(0, 5) {
			t.Error("IsBoldInRange must return true for a fully overridden range")
		}
		if txt.IsBoldInRange(0, 8) {
			t.Error("IsBoldInRange must return false for a partially bold range")
		}
	})

	t.Run("bold base style", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetValue("hello world")
		txt.SetVariation(font.TagWght, bold)
		if !txt.IsBoldInRange(0, 5) {
			t.Error("IsBoldInRange must return true with a bold base style")
		}
		// A partial bold override on a bold base style keeps the range
		// uniformly bold.
		txt.SetVariationInRange(0, 3, font.TagWght, bold)
		if !txt.IsBoldInRange(0, 5) {
			t.Error("IsBoldInRange must fold the base style into uncovered bytes")
		}
		txt.SetVariationInRange(0, 3, font.TagWght, float32(text.WeightLight))
		if txt.IsBoldInRange(0, 5) {
			t.Error("IsBoldInRange must return false for a non-bold override")
		}
	})
}

func TestTextApplyBoldInRange(t *testing.T) {
	bold := float32(text.WeightBold)

	t.Run("default base style", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetValue("hello world")
		txt.ApplyBoldInRange(0, 5, true)
		var wantRuns textstyle.Runs
		wantRuns.SetVariation(0, 5, font.TagWght, bold)
		want := slices.Collect(wantRuns.All())
		if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}

		// Applying bold is idempotent.
		txt.ApplyBoldInRange(0, 5, true)
		if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}

		// Applying bold over a partially bold range makes it all bold.
		txt.ApplyBoldInRange(0, 8, true)
		wantRuns.SetVariation(0, 8, font.TagWght, bold)
		want = slices.Collect(wantRuns.All())
		if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}

		// Applying not bold removes the overrides.
		txt.ApplyBoldInRange(0, 8, false)
		if got := txt.StyleRuns(); got != nil {
			t.Errorf("got: %+v, want: nil", got)
		}
	})

	t.Run("bold base style", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetValue("hello world")
		txt.SetVariation(font.TagWght, bold)

		// Applying not bold on a bold base style needs an explicit override.
		txt.ApplyBoldInRange(0, 5, false)
		if txt.IsBoldInRange(0, 5) {
			t.Error("IsBoldInRange must return false after applying not bold")
		}
		var wantRuns textstyle.Runs
		wantRuns.SetVariation(0, 5, font.TagWght, float32(text.WeightMedium))
		want := slices.Collect(wantRuns.All())
		if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}

		// Applying bold again removes the override so the base style shows.
		txt.ApplyBoldInRange(0, 5, true)
		if !txt.IsBoldInRange(0, 5) {
			t.Error("IsBoldInRange must return true after applying bold")
		}
		if got := txt.StyleRuns(); got != nil {
			t.Errorf("got: %+v, want: nil", got)
		}
	})

	t.Run("empty range is a no-op", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetValue("hello world")
		txt.ApplyBoldInRange(3, 3, true)
		txt.ApplyBoldInRange(3, 3, false)
		if got := txt.StyleRuns(); got != nil {
			t.Errorf("got: %+v, want: nil", got)
		}
	})
}

func TestTextApplyItalicInRange(t *testing.T) {
	t.Run("default base style", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetValue("hello world")
		if txt.IsItalicInRange(0, 5) {
			t.Error("IsItalicInRange must return false without an italic style")
		}
		txt.ApplyItalicInRange(0, 5, true)
		if !txt.IsItalicInRange(0, 5) {
			t.Error("IsItalicInRange must return true after applying italic")
		}
		var wantRuns textstyle.Runs
		wantRuns.SetItalic(0, 5, true)
		want := slices.Collect(wantRuns.All())
		if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}
		txt.ApplyItalicInRange(0, 5, false)
		if got := txt.StyleRuns(); got != nil {
			t.Errorf("got: %+v, want: nil", got)
		}
	})

	t.Run("italic base style", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetValue("hello world")
		txt.SetItalic(true)
		if !txt.IsItalicInRange(0, 5) {
			t.Error("IsItalicInRange must return true with an italic base style")
		}

		// Applying not italic on an italic base style needs an explicit
		// override.
		txt.ApplyItalicInRange(0, 5, false)
		if txt.IsItalicInRange(0, 5) {
			t.Error("IsItalicInRange must return false after applying not italic")
		}
		var wantRuns textstyle.Runs
		wantRuns.SetItalic(0, 5, false)
		want := slices.Collect(wantRuns.All())
		if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}

		// Applying italic again removes the override so the base style shows.
		txt.ApplyItalicInRange(0, 5, true)
		if !txt.IsItalicInRange(0, 5) {
			t.Error("IsItalicInRange must return true after applying italic")
		}
		if got := txt.StyleRuns(); got != nil {
			t.Errorf("got: %+v, want: nil", got)
		}
	})
}

func TestTextApplyUnderlineInRange(t *testing.T) {
	var txt textwidget.Text
	txt.SetValue("hello world")

	txt.SetUnderlineInRange(0, 3, true)
	if txt.IsUnderlineInRange(0, 5) {
		t.Error("IsUnderlineInRange must return false for a partially underlined range")
	}

	// Applying an underline to a partially underlined range underlines it
	// all.
	txt.ApplyUnderlineInRange(0, 5, true)
	if !txt.IsUnderlineInRange(0, 5) {
		t.Error("IsUnderlineInRange must return true after applying an underline")
	}
	var wantRuns textstyle.Runs
	wantRuns.SetUnderline(0, 5, true)
	want := slices.Collect(wantRuns.All())
	if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
		t.Errorf("got: %+v, want: %+v", got, want)
	}

	// Applying no underline removes the overrides instead of setting false
	// overrides.
	txt.ApplyUnderlineInRange(0, 5, false)
	if txt.IsUnderlineInRange(0, 5) {
		t.Error("IsUnderlineInRange must return false after removing the underline")
	}
	if got := txt.StyleRuns(); got != nil {
		t.Errorf("got: %+v, want: nil", got)
	}
}

func TestTextApplyStrikethroughInRange(t *testing.T) {
	var txt textwidget.Text
	txt.SetValue("hello world")

	txt.ApplyStrikethroughInRange(0, 5, true)
	if !txt.IsStrikethroughInRange(0, 5) {
		t.Error("IsStrikethroughInRange must return true after applying a strikethrough")
	}
	txt.ApplyStrikethroughInRange(0, 5, false)
	if txt.IsStrikethroughInRange(0, 5) {
		t.Error("IsStrikethroughInRange must return false after removing the strikethrough")
	}
	if got := txt.StyleRuns(); got != nil {
		t.Errorf("got: %+v, want: nil", got)
	}
}

func TestTextStyleQueriesAtCaret(t *testing.T) {
	var txt textwidget.Text
	txt.SetValue("hello")
	txt.SetUnderlineInRange(1, 4, true)

	// An empty range reports the style that text typed at the caret adopts:
	// the style of the byte right before the caret.
	tests := []struct {
		caret int
		want  bool
	}{
		{caret: 0, want: false},
		{caret: 1, want: false},
		{caret: 2, want: true},
		{caret: 4, want: true},
		{caret: 5, want: false},
	}
	for _, tt := range tests {
		if got := txt.IsUnderlineInRange(tt.caret, tt.caret); got != tt.want {
			t.Errorf("IsUnderlineInRange(%d, %d): got: %t, want: %t", tt.caret, tt.caret, got, tt.want)
		}
	}

	var boldText textwidget.Text
	boldText.SetValue("hello")
	boldText.SetVariation(font.TagWght, float32(text.WeightBold))
	if !boldText.IsBoldInRange(0, 0) {
		t.Error("IsBoldInRange must report the base style at the start of the value")
	}
}

func TestTextColorInRange(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetValue("hello world")
	if clr, uniform := txt.ColorInRange(0, 5); !uniform || clr != nil {
		t.Errorf("got: %v, %t, want: nil, true", clr, uniform)
	}

	txt.SetColorInRange(0, 5, red)
	if clr, uniform := txt.ColorInRange(0, 5); !uniform || clr != color.Color(red) {
		t.Errorf("got: %v, %t, want: %v, true", clr, uniform, red)
	}
	if _, uniform := txt.ColorInRange(0, 8); uniform {
		t.Error("a partially colored range must not be uniform")
	}
	if clr, uniform := txt.ColorInRange(2, 2); !uniform || clr != color.Color(red) {
		t.Errorf("got: %v, %t, want: %v, true", clr, uniform, red)
	}
}

func TestTextScaleInRange(t *testing.T) {
	var txt textwidget.Text
	txt.SetValue("hello world")
	if scale, uniform := txt.ScaleInRange(0, 5); !uniform || scale != 1 {
		t.Errorf("got: %v, %t, want: 1, true", scale, uniform)
	}

	txt.SetScaleInRange(0, 5, 2)
	if scale, uniform := txt.ScaleInRange(0, 5); !uniform || scale != 2 {
		t.Errorf("got: %v, %t, want: 2, true", scale, uniform)
	}
	if _, uniform := txt.ScaleInRange(0, 8); uniform {
		t.Error("a partially scaled range must not be uniform")
	}
	if scale, uniform := txt.ScaleInRange(2, 2); !uniform || scale != 2 {
		t.Errorf("got: %v, %t, want: 2, true", scale, uniform)
	}
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

func TestStyleRunsRoundTrip(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetValue("hello world")
	txt.SetColorInRange(0, 5, red)
	txt.ReplaceTextAt("!!", 5, 5)

	// The copied-out runs reflect the edit; installing them into another
	// text reproduces the same overrides.
	var runs textstyle.Runs
	txt.ReadStyleRuns(&runs)
	var txt2 textwidget.Text
	txt2.SetValue("hello!! world")
	txt2.CopyStyleRunsFrom(&runs)

	var wantRuns textstyle.Runs
	wantRuns.SetColor(0, 7, red)
	want := slices.Collect(wantRuns.All())
	if got := txt2.StyleRuns(); !equalStyleRuns(got, want) {
		t.Errorf("got: %+v, want: %+v", got, want)
	}
}

func TestTextUndoRedoRestoresStyleRuns(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	t.Run("undo of an insertion", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetEditable(true)
		txt.ForceSetValue("hello")
		txt.SetColorInRange(1, 4, red)
		want := txt.StyleRuns()

		txt.SetSelection(2, 2)
		txt.ReplaceValueAtSelection("ab")
		if !txt.Undo() {
			t.Fatal("Undo must return true")
		}
		if got, wantValue := txt.Value(), "hello"; got != wantValue {
			t.Errorf("got: %q, want: %q", got, wantValue)
		}
		if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}
	})

	t.Run("undo resurrects a deleted uniquely-styled span", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetEditable(true)
		txt.ForceSetValue("hello world")
		txt.SetColorInRange(6, 11, red)
		want := txt.StyleRuns()

		// Deleting the styled span removes its run entirely; positional
		// adjustment alone cannot bring it back.
		txt.ReplaceTextAt("", 5, 11)
		if got := txt.StyleRuns(); got != nil {
			t.Fatalf("got: %+v, want: nil", got)
		}

		if !txt.Undo() {
			t.Fatal("Undo must return true")
		}
		if got, wantValue := txt.Value(), "hello world"; got != wantValue {
			t.Errorf("got: %q, want: %q", got, wantValue)
		}
		if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}
	})

	t.Run("redo restores the post-edit styles", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetEditable(true)
		txt.ForceSetValue("hello")
		txt.SetColorInRange(0, 5, red)

		// Deleting the head truncates the run to [0, 3).
		txt.ReplaceTextAt("", 0, 2)
		afterEdit := txt.StyleRuns()

		if !txt.Undo() {
			t.Fatal("Undo must return true")
		}
		if !txt.Redo() {
			t.Fatal("Redo must return true")
		}
		if got, wantValue := txt.Value(), "llo"; got != wantValue {
			t.Errorf("got: %q, want: %q", got, wantValue)
		}
		if got := txt.StyleRuns(); !equalStyleRuns(got, afterEdit) {
			t.Errorf("got: %+v, want: %+v", got, afterEdit)
		}
	})
}

func TestTextUndoCoalescedEditsRestoresStyleRuns(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")
	txt.SetColorInRange(0, 5, red)
	want := txt.StyleRuns()

	// Backspace-like consecutive deletes coalesce into one undo entry.
	txt.ReplaceTextAt("", 4, 5)
	txt.ReplaceTextAt("", 3, 4)
	txt.ReplaceTextAt("", 2, 3)
	if got, wantValue := txt.Value(), "he"; got != wantValue {
		t.Fatalf("got: %q, want: %q", got, wantValue)
	}

	// One undo reverts the whole group and restores the group-start styles.
	if !txt.Undo() {
		t.Fatal("Undo must return true")
	}
	if got, wantValue := txt.Value(), "hello"; got != wantValue {
		t.Errorf("got: %q, want: %q", got, wantValue)
	}
	if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
		t.Errorf("got: %+v, want: %+v", got, want)
	}
	if txt.CanUndo() {
		t.Error("CanUndo must return false after undoing the coalesced entry")
	}
}

func TestTextUndoAfterWholeValueReplacement(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")
	txt.SetColorInRange(0, 5, red)
	want := txt.StyleRuns()

	// A whole-value replacement clears the styles.
	txt.ForceSetValue("goodbye")
	if got := txt.StyleRuns(); got != nil {
		t.Fatalf("got: %+v, want: nil", got)
	}

	// Undoing the replacement restores the previous value and its styles.
	if !txt.Undo() {
		t.Fatal("Undo must return true")
	}
	if got, wantValue := txt.Value(), "hello"; got != wantValue {
		t.Errorf("got: %q, want: %q", got, wantValue)
	}
	if got := txt.StyleRuns(); !equalStyleRuns(got, want) {
		t.Errorf("got: %+v, want: %+v", got, want)
	}
}

func TestTextResetClearsHistoryAndStyles(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")
	txt.SetColorInRange(0, 5, red)
	txt.ReplaceTextAt("x", 0, 0)

	// Resetting the value clears the undo history and the styles.
	if _, err := txt.ReadValueFrom(strings.NewReader("goodbye")); err != nil {
		t.Fatal(err)
	}
	if got := txt.StyleRuns(); got != nil {
		t.Errorf("got: %+v, want: nil", got)
	}
	if txt.CanUndo() {
		t.Error("CanUndo must return false after a reset")
	}
	if txt.Undo() {
		t.Error("Undo must return false after a reset")
	}
}
