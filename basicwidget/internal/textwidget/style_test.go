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

	got := txt.OverrideStyleRuns()
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

func TestTextOverrideStyleRunsLifetime(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	t.Run("equal SetValue keeps styles", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetValue("hello")
		txt.SetColorInRange(0, 5, red)
		txt.SetValue("hello")
		if got := txt.OverrideStyleRuns(); len(got) != 1 {
			t.Errorf("got %d runs, want 1: %+v", len(got), got)
		}
	})

	t.Run("changed SetValue clears styles", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetValue("hello")
		txt.SetColorInRange(0, 5, red)
		txt.SetValue("goodbye")
		if got := txt.OverrideStyleRuns(); got != nil {
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
		if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, want) {
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
		if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, want) {
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
		if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}
	})

	t.Run("deletion merges adjacent equal styles", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetEditable(true)
		txt.ForceSetValue("hello world")
		txt.SetColorInRange(0, 3, red)
		txt.SetColorInRange(5, 8, red)
		txt.ReplaceTextAt("", 3, 5, nil)
		var wantRuns textstyle.Runs
		wantRuns.SetColor(0, 6, red)
		want := slices.Collect(wantRuns.All())
		if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, want) {
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
		if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}
		txt.ResetStylesInRange(0, 5)
		if got := txt.OverrideStyleRuns(); got != nil {
			t.Errorf("got: %+v, want: nil", got)
		}
	})
}

func TestTextReadBaseStyle(t *testing.T) {
	var txt textwidget.Text
	var base textstyle.Style
	base = base.WithItalic(true).WithVariation(font.TagWght, float32(text.WeightBold))
	txt.SetBaseStyle(base)

	var s textstyle.Style
	txt.ReadBaseStyle(&s)
	if italic, ok := s.Italic(); !ok || !italic {
		t.Errorf("Italic(): got: %t, %t, want: true, true", italic, ok)
	}
	if v, ok := s.Variation(font.TagWght); !ok || v != float32(text.WeightBold) {
		t.Errorf("Variation(wght): got: %v, %t, want: %v, true", v, ok, float32(text.WeightBold))
	}
	if _, ok := s.Color(); ok {
		t.Error("Color() must be unset without a base text color")
	}
}

func TestTextReadOverrideStyleRunsInRange(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetValue("hello world")
	txt.SetColorInRange(3, 8, red)

	var runs textstyle.Runs
	txt.ReadOverrideStyleRunsInRange(&runs, 5, 11)
	var wantRuns textstyle.Runs
	wantRuns.SetColor(0, 3, red)
	got := slices.Collect(runs.All())
	want := slices.Collect(wantRuns.All())
	if !equalStyleRuns(got, want) {
		t.Errorf("got: %+v, want: %+v", got, want)
	}
}

func TestTextReplaceOverrideStyleRunsInRange(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetValue("hello world")
	txt.SetColorInRange(0, 11, red)
	txt.SetUnderlineInRange(2, 4, true)

	var src textstyle.Runs
	src.SetColor(0, 3, blue)
	txt.ReplaceOverrideStyleRunsInRange(&src, 2, 6, false)

	var wantRuns textstyle.Runs
	wantRuns.SetColor(0, 2, red)
	wantRuns.SetColor(2, 5, blue)
	wantRuns.SetColor(6, 11, red)
	want := slices.Collect(wantRuns.All())
	if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, want) {
		t.Errorf("got: %+v, want: %+v", got, want)
	}
}

func TestTextReadEffectiveStyleRunsInRange(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetValue("hello world")
	var base textstyle.Style
	txt.SetBaseStyle(base.WithItalic(true))
	txt.SetColorInRange(3, 5, red)

	var runs textstyle.Runs
	txt.ReadEffectiveStyleRunsInRange(&runs, 2, 6)

	// [2, 3) has no override: the effective style resolves the base style
	// and the rendering defaults.
	s := runs.StyleAt(0)
	if italic, ok := s.Italic(); !ok || !italic {
		t.Errorf("Italic() at 0: got: %t, %t, want: true, true", italic, ok)
	}
	if v, ok := s.Variation(font.TagWght); !ok || v != float32(text.WeightMedium) {
		t.Errorf("Variation(wght) at 0: got: %v, %t, want: %v, true", v, ok, float32(text.WeightMedium))
	}
	if scale, ok := s.Scale(); !ok || scale != 1 {
		t.Errorf("Scale() at 0: got: %v, %t, want: 1, true", scale, ok)
	}
	if clr, ok := s.Color(); !ok || clr != nil {
		t.Errorf("Color() at 0: got: %v, %t, want: nil, true", clr, ok)
	}
	if underline, ok := s.Underline(); !ok || underline {
		t.Errorf("Underline() at 0: got: %t, %t, want: false, true", underline, ok)
	}

	// [3, 5) has the color override merged over the base style.
	s = runs.StyleAt(1)
	if clr, ok := s.Color(); !ok || clr != color.Color(red) {
		t.Errorf("Color() at 1: got: %v, %t, want: %v, true", clr, ok, red)
	}
	if italic, ok := s.Italic(); !ok || !italic {
		t.Errorf("Italic() at 1: got: %t, %t, want: true, true", italic, ok)
	}

	// The result is rebased to [0, end-start); bytes past it are uncovered.
	if got := runs.StyleAt(4); !got.IsZero() {
		t.Errorf("style at 4 = %+v; want zero", got)
	}
}

func TestTextEffectiveStyleAt(t *testing.T) {
	var txt textwidget.Text
	txt.SetValue("hello")
	txt.SetUnderlineInRange(1, 4, true)

	// The effective style at an index is the style that text typed there
	// adopts: the style of the byte right before the index, except at a
	// logical line's head, where it is the byte at the index.
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
		underline, ok := txt.EffectiveStyleAt(tt.caret).Underline()
		if !ok || underline != tt.want {
			t.Errorf("EffectiveStyleAt(%d).Underline(): got: %t, %t, want: %t, true", tt.caret, underline, ok, tt.want)
		}
	}

	// The base style resolves at the start of the value, and unset weights
	// resolve to the default medium weight.
	var boldText textwidget.Text
	boldText.SetValue("hello")
	var boldBase textstyle.Style
	boldText.SetBaseStyle(boldBase.WithVariation(font.TagWght, float32(text.WeightBold)))
	if v, ok := boldText.EffectiveStyleAt(0).Variation(font.TagWght); !ok || v != float32(text.WeightBold) {
		t.Errorf("Variation(wght): got: %v, %t, want: %v, true", v, ok, float32(text.WeightBold))
	}
	if v, ok := txt.EffectiveStyleAt(0).Variation(font.TagWght); !ok || v != float32(text.WeightMedium) {
		t.Errorf("Variation(wght): got: %v, %t, want: %v, true", v, ok, float32(text.WeightMedium))
	}
}

// TestRenderingStyleRunsThroughComposition asserts that the committed
// overrides move with the composition splice, so the rendering text keeps
// carrying them.
func TestRenderingStyleRunsThroughComposition(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	tests := []struct {
		name             string
		selStart, selEnd int
		composition      string
		wantStarts       []int
		wantEnds         []int
	}{
		{
			name:     "composition before the runs shifts them",
			selStart: 2, selEnd: 2, composition: "abc",
			wantStarts: []int{8, 23}, wantEnds: []int{13, 28},
		},
		{
			name:     "composition at a run start shifts the run",
			selStart: 5, selEnd: 5, composition: "abc",
			wantStarts: []int{8, 23}, wantEnds: []int{13, 28},
		},
		{
			name:     "composition inside a run extends it",
			selStart: 7, selEnd: 7, composition: "abc",
			wantStarts: []int{5, 23}, wantEnds: []int{13, 28},
		},
		{
			name:     "composition at a run end extends it",
			selStart: 10, selEnd: 10, composition: "abc",
			wantStarts: []int{5, 23}, wantEnds: []int{13, 28},
		},
		{
			name:     "composition between the runs",
			selStart: 15, selEnd: 15, composition: "abc",
			wantStarts: []int{5, 23}, wantEnds: []int{10, 28},
		},
		{
			name:     "composition replacing a run drops it",
			selStart: 4, selEnd: 11, composition: "ab",
			wantStarts: []int{15}, wantEnds: []int{20},
		},
		{
			name:     "composition starting inside a run stays part of it",
			selStart: 8, selEnd: 12, composition: "abcd",
			wantStarts: []int{5, 20}, wantEnds: []int{12, 25},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var txt textwidget.Text
			txt.SetEditable(true)
			txt.ForceSetValue(strings.Repeat("a", 30))
			txt.SetColorInRange(5, 10, red)
			txt.SetColorInRange(20, 25, red)
			txt.HandleFocusChanged(true)
			txt.SetSelection(tt.selStart, tt.selEnd)
			txt.SetCompositionByIME(tt.composition, 0, len(tt.composition))

			var wantRuns textstyle.Runs
			for i := range tt.wantStarts {
				wantRuns.SetColor(tt.wantStarts[i], tt.wantEnds[i], red)
			}
			if got := txt.RenderingStyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
				t.Errorf("got: %+v, want: %+v", got, runsSlice(&wantRuns))
			}
		})
	}
}

// TestRenderingStyleRunsApplyInsertionStyle asserts that the composition
// renders with the insertion style it will carry once committed, on top of
// the style it adopts from the text around it.
func TestRenderingStyleRunsApplyInsertionStyle(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	t.Run("without committed overrides", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetEditable(true)
		txt.ForceSetValue("hello")
		txt.HandleFocusChanged(true)
		txt.SetSelection(3, 3)
		txt.SetInsertionStyle(textstyle.Style{}.WithColor(red))
		txt.SetCompositionByIME("abc", 0, 3)

		var wantRuns textstyle.Runs
		wantRuns.SetColor(3, 6, red)
		if got := txt.RenderingStyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
			t.Errorf("got: %+v, want: %+v", got, runsSlice(&wantRuns))
		}
	})

	t.Run("over an adopted override", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetEditable(true)
		txt.ForceSetValue(strings.Repeat("a", 30))
		txt.SetColorInRange(5, 10, red)
		txt.HandleFocusChanged(true)
		txt.SetSelection(7, 7)
		txt.SetInsertionStyle(textstyle.Style{}.WithUnderline(true))
		txt.SetCompositionByIME("abc", 0, 3)

		// The composition adopts the run it is typed into and carries the
		// insertion style on top of it.
		var wantRuns textstyle.Runs
		wantRuns.SetColor(5, 13, red)
		wantRuns.SetUnderline(7, 10, true)
		if got := txt.RenderingStyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
			t.Errorf("got: %+v, want: %+v", got, runsSlice(&wantRuns))
		}
	})

	t.Run("committing keeps the style", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetEditable(true)
		txt.ForceSetValue("hello")
		txt.HandleFocusChanged(true)
		txt.SetSelection(3, 3)
		txt.SetInsertionStyle(textstyle.Style{}.WithColor(red))
		txt.SetCompositionByIME("abc", 0, 3)
		txt.SetCompositionByIME("", 0, 0)
		txt.CommitTextByIME("abc")

		var wantRuns textstyle.Runs
		wantRuns.SetColor(3, 6, red)
		if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
			t.Errorf("got: %+v, want: %+v", got, runsSlice(&wantRuns))
		}
	})
}

func TestOverrideStyleRunsRoundTrip(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetValue("hello world")
	txt.SetColorInRange(0, 5, red)
	txt.ReplaceTextAt("!!", 5, 5, nil)

	// The copied-out runs reflect the edit; installing them into another
	// text reproduces the same overrides.
	var runs textstyle.Runs
	txt.ReadOverrideStyleRuns(&runs)
	var txt2 textwidget.Text
	txt2.SetValue("hello!! world")
	txt2.CopyOverrideStyleRunsFrom(&runs, false)

	var wantRuns textstyle.Runs
	wantRuns.SetColor(0, 7, red)
	want := slices.Collect(wantRuns.All())
	if got := txt2.OverrideStyleRuns(); !equalStyleRuns(got, want) {
		t.Errorf("got: %+v, want: %+v", got, want)
	}
}

func TestTextUndoRedoRestoresOverrideStyleRuns(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	t.Run("undo of an insertion", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetEditable(true)
		txt.ForceSetValue("hello")
		txt.SetColorInRange(1, 4, red)
		want := txt.OverrideStyleRuns()

		txt.SetSelection(2, 2)
		txt.ReplaceValueAtSelection("ab")
		if !txt.Undo() {
			t.Fatal("Undo must return true")
		}
		if got, wantValue := txt.Value(), "hello"; got != wantValue {
			t.Errorf("got: %q, want: %q", got, wantValue)
		}
		if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}
	})

	t.Run("undo resurrects a deleted uniquely-styled span", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetEditable(true)
		txt.ForceSetValue("hello world")
		txt.SetColorInRange(6, 11, red)
		want := txt.OverrideStyleRuns()

		// Deleting the styled span removes its run entirely; positional
		// adjustment alone cannot bring it back.
		txt.ReplaceTextAt("", 5, 11, nil)
		if got := txt.OverrideStyleRuns(); got != nil {
			t.Fatalf("got: %+v, want: nil", got)
		}

		if !txt.Undo() {
			t.Fatal("Undo must return true")
		}
		if got, wantValue := txt.Value(), "hello world"; got != wantValue {
			t.Errorf("got: %q, want: %q", got, wantValue)
		}
		if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, want) {
			t.Errorf("got: %+v, want: %+v", got, want)
		}
	})

	t.Run("redo restores the post-edit styles", func(t *testing.T) {
		var txt textwidget.Text
		txt.SetEditable(true)
		txt.ForceSetValue("hello")
		txt.SetColorInRange(0, 5, red)

		// Deleting the head truncates the run to [0, 3).
		txt.ReplaceTextAt("", 0, 2, nil)
		afterEdit := txt.OverrideStyleRuns()

		if !txt.Undo() {
			t.Fatal("Undo must return true")
		}
		if !txt.Redo() {
			t.Fatal("Redo must return true")
		}
		if got, wantValue := txt.Value(), "llo"; got != wantValue {
			t.Errorf("got: %q, want: %q", got, wantValue)
		}
		if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, afterEdit) {
			t.Errorf("got: %+v, want: %+v", got, afterEdit)
		}
	})
}

func TestTextUndoCoalescedEditsRestoresOverrideStyleRuns(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")
	txt.SetColorInRange(0, 5, red)
	want := txt.OverrideStyleRuns()

	// Backspace-like consecutive deletes coalesce into one undo entry.
	txt.ReplaceTextAt("", 4, 5, nil)
	txt.ReplaceTextAt("", 3, 4, nil)
	txt.ReplaceTextAt("", 2, 3, nil)
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
	if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, want) {
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
	want := txt.OverrideStyleRuns()

	// A whole-value replacement clears the styles.
	txt.ForceSetValue("goodbye")
	if got := txt.OverrideStyleRuns(); got != nil {
		t.Fatalf("got: %+v, want: nil", got)
	}

	// Undoing the replacement restores the previous value and its styles.
	if !txt.Undo() {
		t.Fatal("Undo must return true")
	}
	if got, wantValue := txt.Value(), "hello"; got != wantValue {
		t.Errorf("got: %q, want: %q", got, wantValue)
	}
	if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, want) {
		t.Errorf("got: %+v, want: %+v", got, want)
	}
}

func TestTextResetClearsHistoryAndStyles(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")
	txt.SetColorInRange(0, 5, red)
	txt.ReplaceTextAt("x", 0, 0, nil)

	// Resetting the value clears the undo history and the styles.
	if _, err := txt.ReadValueFrom(strings.NewReader("goodbye")); err != nil {
		t.Fatal(err)
	}
	if got := txt.OverrideStyleRuns(); got != nil {
		t.Errorf("got: %+v, want: nil", got)
	}
	if txt.CanUndo() {
		t.Error("CanUndo must return false after a reset")
	}
	if txt.Undo() {
		t.Error("Undo must return false after a reset")
	}
}

func TestTextStyleOnlyUndoRedo(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello world")
	if txt.CanUndo() {
		t.Fatal("CanUndo must return false before the style change")
	}

	var runs textstyle.Runs
	runs.SetColor(0, 5, red)
	txt.ReplaceOverrideStyleRunsInRange(&runs, 0, 5, true)
	afterChange := txt.OverrideStyleRuns()
	if !txt.CanUndo() {
		t.Fatal("CanUndo must return true after the style change")
	}

	// Undo leaves the value unchanged, restores the previous styles, and
	// selects the styled range.
	if !txt.Undo() {
		t.Fatal("Undo must return true")
	}
	if got, want := txt.Value(), "hello world"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
	if got := txt.OverrideStyleRuns(); got != nil {
		t.Errorf("got: %+v, want: nil", got)
	}
	if start, end := txt.Selection(); start != 0 || end != 5 {
		t.Errorf("Selection: got (%d, %d), want (0, 5)", start, end)
	}
	if txt.CanUndo() {
		t.Error("CanUndo must return false after undoing the style change")
	}

	if !txt.Redo() {
		t.Fatal("Redo must return true")
	}
	if got, want := txt.Value(), "hello world"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
	if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, afterChange) {
		t.Errorf("got: %+v, want: %+v", got, afterChange)
	}
	if start, end := txt.Selection(); start != 0 || end != 5 {
		t.Errorf("Selection: got (%d, %d), want (0, 5)", start, end)
	}
}

func TestTextStyleOnlyUndoUnchangedReplacement(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")

	// A replacement equal to the current overrides records nothing, so
	// reinstalling the same styles repeatedly leaves one undo entry.
	var runs textstyle.Runs
	runs.SetColor(0, 5, red)
	txt.ReplaceOverrideStyleRunsInRange(&runs, 0, 5, true)
	txt.ReplaceOverrideStyleRunsInRange(&runs, 0, 5, true)
	txt.ReplaceOverrideStyleRunsInRange(&runs, 0, 5, true)

	if !txt.Undo() {
		t.Fatal("Undo must return true")
	}
	if got := txt.OverrideStyleRuns(); got != nil {
		t.Errorf("got: %+v, want: nil", got)
	}
	if txt.CanUndo() {
		t.Error("CanUndo must return false after undoing the only style change")
	}

	// An empty replacement over unstyled text records nothing.
	var txt2 textwidget.Text
	txt2.SetEditable(true)
	txt2.ForceSetValue("hello")
	var empty textstyle.Runs
	txt2.ReplaceOverrideStyleRunsInRange(&empty, 0, 5, true)
	if txt2.CanUndo() {
		t.Error("CanUndo must return false after an empty replacement of unstyled text")
	}
}

func TestTextStyleOnlyUndoInterleavedWithEdits(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")

	txt.ReplaceTextAt(" world", 5, 5, nil)
	var runs textstyle.Runs
	runs.SetColor(0, 5, red)
	txt.ReplaceOverrideStyleRunsInRange(&runs, 0, 5, true)
	styled := txt.OverrideStyleRuns()
	txt.ReplaceTextAt("!", 11, 11, nil)

	// Undo walks back the text edit, the style change, and the first edit.
	if !txt.Undo() {
		t.Fatal("Undo must return true")
	}
	if got, want := txt.Value(), "hello world"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
	if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, styled) {
		t.Errorf("got: %+v, want: %+v", got, styled)
	}

	if !txt.Undo() {
		t.Fatal("Undo must return true")
	}
	if got, want := txt.Value(), "hello world"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
	if got := txt.OverrideStyleRuns(); got != nil {
		t.Errorf("got: %+v, want: nil", got)
	}

	if !txt.Undo() {
		t.Fatal("Undo must return true")
	}
	if got, want := txt.Value(), "hello"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
	if txt.CanUndo() {
		t.Error("CanUndo must return false after undoing every entry")
	}

	// Redo walks forward through the same entries.
	if !txt.Redo() {
		t.Fatal("Redo must return true")
	}
	if !txt.Redo() {
		t.Fatal("Redo must return true")
	}
	if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, styled) {
		t.Errorf("got: %+v, want: %+v", got, styled)
	}
	if !txt.Redo() {
		t.Fatal("Redo must return true")
	}
	if got, want := txt.Value(), "hello world!"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}

	// A new style change after an undo truncates the redo tail.
	if !txt.Undo() {
		t.Fatal("Undo must return true")
	}
	var underline textstyle.Runs
	underline.SetUnderline(0, 5, true)
	txt.ReplaceOverrideStyleRunsInRange(&underline, 6, 11, true)
	if txt.CanRedo() {
		t.Error("CanRedo must return false after a style change")
	}
}

func TestTextCopyOverrideStyleRunsFromRecordsNoHistory(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")

	// Unrecorded whole-value replacements restore model state on every build
	// and must not grow the undo history.
	var runs textstyle.Runs
	runs.SetColor(0, 5, red)
	for range 3 {
		txt.CopyOverrideStyleRunsFrom(&runs, false)
	}
	if txt.CanUndo() {
		t.Error("CanUndo must return false after whole-value replacements")
	}
}

func TestTextStyleOnlyUndoKeepsHotspots(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello world")
	hotspots := []textwidget.TextRange{{StartInBytes: 6, EndInBytes: 11}}
	txt.SetHotspotRanges(hotspots)

	var runs textstyle.Runs
	runs.SetColor(0, 5, red)
	txt.ReplaceOverrideStyleRunsInRange(&runs, 0, 5, true)

	if !txt.Undo() {
		t.Fatal("Undo must return true")
	}
	if got := txt.AppendHotspotRanges(nil); !slices.Equal(got, hotspots) {
		t.Errorf("got: %+v, want: %+v", got, hotspots)
	}
	if !txt.Redo() {
		t.Fatal("Redo must return true")
	}
	if got := txt.AppendHotspotRanges(nil); !slices.Equal(got, hotspots) {
		t.Errorf("got: %+v, want: %+v", got, hotspots)
	}
}

func TestTextRecordedWholeValueStyleUndoRedo(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello world")

	var runs textstyle.Runs
	runs.SetColor(0, 5, red)
	txt.CopyOverrideStyleRunsFrom(&runs, true)
	afterChange := txt.OverrideStyleRuns()
	if !txt.CanUndo() {
		t.Fatal("CanUndo must return true after the recorded replacement")
	}

	// A recorded replacement equal to the current overrides records nothing.
	txt.CopyOverrideStyleRunsFrom(&runs, true)

	// Undo restores the previous overrides and selects the whole value.
	if !txt.Undo() {
		t.Fatal("Undo must return true")
	}
	if got := txt.OverrideStyleRuns(); got != nil {
		t.Errorf("got: %+v, want: nil", got)
	}
	if start, end := txt.Selection(); start != 0 || end != 11 {
		t.Errorf("Selection: got (%d, %d), want (0, 11)", start, end)
	}
	if txt.CanUndo() {
		t.Error("CanUndo must return false after undoing the only replacement")
	}

	if !txt.Redo() {
		t.Fatal("Redo must return true")
	}
	if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, afterChange) {
		t.Errorf("got: %+v, want: %+v", got, afterChange)
	}
}

func TestTextUnrecordedRangedStyleReplacement(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello world")

	// Unrecorded ranged replacements apply without growing the undo history,
	// even when they change the overrides.
	var runs textstyle.Runs
	runs.SetColor(0, 5, red)
	txt.ReplaceOverrideStyleRunsInRange(&runs, 0, 5, false)
	var runs2 textstyle.Runs
	runs2.SetColor(0, 5, blue)
	txt.ReplaceOverrideStyleRunsInRange(&runs2, 0, 5, false)

	want := slices.Collect(func(yield func(textstyle.Run) bool) {
		yield(textstyle.Run{Start: 0, End: 5, Style: textstyle.Style{}.WithColor(blue)})
	})
	if got := txt.OverrideStyleRuns(); !equalStyleRuns(got, want) {
		t.Errorf("got: %+v, want: %+v", got, want)
	}
	if txt.CanUndo() {
		t.Error("CanUndo must return false after unrecorded replacements")
	}
}
