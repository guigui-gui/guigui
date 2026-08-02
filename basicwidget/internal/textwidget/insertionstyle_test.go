// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget_test

import (
	"image/color"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
	"github.com/guigui-gui/guigui/basicwidget/internal/textwidget"
)

func runsSlice(r *textstyle.Runs) []textstyle.Run {
	return slices.Collect(r.All())
}

func TestTextInsertionStyleMaterializesOnInsertion(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")
	txt.SetSelection(5, 5)
	txt.SetInsertionStyle(textstyle.Style{}.WithColor(red))

	txt.ReplaceValueAtSelection("ab")

	if got := txt.Value(); got != "helloab" {
		t.Fatalf("Value(): got: %q, want: %q", got, "helloab")
	}
	var wantRuns textstyle.Runs
	wantRuns.SetColor(5, 7, red)
	if got := txt.StyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
		t.Errorf("got: %+v, want: %+v", got, runsSlice(&wantRuns))
	}
	if got := txt.InsertionStyle(); !got.IsZero() {
		t.Errorf("InsertionStyle() after insertion: got: %+v, want: zero", got)
	}

	// Typing right after the materialized span adopts its style.
	txt.ReplaceValueAtSelection("c")
	wantRuns.Clear()
	wantRuns.SetColor(5, 8, red)
	if got := txt.StyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
		t.Errorf("got: %+v, want: %+v", got, runsSlice(&wantRuns))
	}
}

func TestTextInsertionStyleMaterializesOnIMECommit(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")
	txt.SetSelection(5, 5)
	txt.SetInsertionStyle(textstyle.Style{}.WithColor(red))

	txt.CommitTextByIME("ab")

	if got := txt.Value(); got != "helloab" {
		t.Fatalf("Value(): got: %q, want: %q", got, "helloab")
	}
	var wantRuns textstyle.Runs
	wantRuns.SetColor(5, 7, red)
	if got := txt.StyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
		t.Errorf("got: %+v, want: %+v", got, runsSlice(&wantRuns))
	}
	if got := txt.InsertionStyle(); !got.IsZero() {
		t.Errorf("InsertionStyle() after commit: got: %+v, want: zero", got)
	}
}

func TestTextInsertionStyleOverridesAdoptedStyle(t *testing.T) {
	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")
	var runs textstyle.Runs
	runs.SetVariation(0, 5, font.TagWght, float32(text.WeightBold))
	txt.CopyStyleRunsFrom(&runs)
	txt.SetSelection(5, 5)

	// Typing at the bold run's end with an insertion-style medium weight must
	// produce medium text, overriding the adopted bold.
	txt.SetInsertionStyle(textstyle.Style{}.WithVariation(font.TagWght, float32(text.WeightMedium)))
	txt.ReplaceValueAtSelection("x")

	var wantRuns textstyle.Runs
	wantRuns.SetVariation(0, 5, font.TagWght, float32(text.WeightBold))
	wantRuns.SetVariation(5, 6, font.TagWght, float32(text.WeightMedium))
	if got := txt.StyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
		t.Errorf("got: %+v, want: %+v", got, runsSlice(&wantRuns))
	}
}

func TestTextInsertionStyleClearedOnSelectionChange(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")
	txt.SetSelection(5, 5)
	txt.SetInsertionStyle(textstyle.Style{}.WithColor(red))

	txt.SetSelection(2, 2)

	if got := txt.InsertionStyle(); !got.IsZero() {
		t.Fatalf("InsertionStyle() after selection change: got: %+v, want: zero", got)
	}
	txt.ReplaceValueAtSelection("x")
	if got := txt.StyleRuns(); got != nil {
		t.Errorf("got: %+v, want: nil", got)
	}
}

func TestTextInsertionStyleClearedOnDeletion(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")
	txt.SetSelection(5, 5)
	txt.SetInsertionStyle(textstyle.Style{}.WithColor(red))

	txt.ReplaceTextAt("", 4, 5, nil)

	if got := txt.InsertionStyle(); !got.IsZero() {
		t.Errorf("InsertionStyle() after deletion: got: %+v, want: zero", got)
	}
}

func TestTextInsertionStyleClearedOnValueReplacement(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")
	txt.SetSelection(5, 5)
	txt.SetInsertionStyle(textstyle.Style{}.WithColor(red))

	txt.ForceSetValue("goodbye")

	if got := txt.InsertionStyle(); !got.IsZero() {
		t.Errorf("InsertionStyle() after value replacement: got: %+v, want: zero", got)
	}
}

func TestTextInsertionStyleClearedByRichPaste(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}

	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")
	txt.SetSelection(5, 5)
	txt.SetInsertionStyle(textstyle.Style{}.WithColor(red))

	var pasted textstyle.Runs
	pasted.SetColor(0, 2, blue)
	txt.ReplaceTextAt("ab", 5, 5, &pasted)

	if got := txt.InsertionStyle(); !got.IsZero() {
		t.Fatalf("InsertionStyle() after rich paste: got: %+v, want: zero", got)
	}
	var wantRuns textstyle.Runs
	wantRuns.SetColor(5, 7, blue)
	if got := txt.StyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
		t.Errorf("got: %+v, want: %+v", got, runsSlice(&wantRuns))
	}
}

func TestTextInsertionStyleResetEvent(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt textwidget.Text
	var resets int
	txt.OnInsertionStyleReset(func(context *guigui.Context) {
		resets++
	})
	txt.SetEditable(true)
	txt.ForceSetValue("hello")
	txt.SetSelection(5, 5)
	if resets != 0 {
		t.Fatalf("resets after setup: got: %d, want: 0", resets)
	}

	// Setting the insertion style dispatches nothing.
	txt.SetInsertionStyle(textstyle.Style{}.WithColor(red))
	if resets != 0 {
		t.Fatalf("resets after SetInsertionStyle: got: %d, want: 0", resets)
	}

	// Consuming the insertion style by an insertion dispatches once.
	txt.ReplaceValueAtSelection("a")
	if resets != 1 {
		t.Fatalf("resets after insertion: got: %d, want: 1", resets)
	}

	// A selection change with a zero insertion style dispatches nothing.
	txt.SetSelection(0, 0)
	if resets != 1 {
		t.Fatalf("resets after selection change without insertion style: got: %d, want: 1", resets)
	}

	// Discarding a set insertion style on a selection change dispatches once.
	txt.SetInsertionStyle(textstyle.Style{}.WithColor(red))
	txt.SetSelection(2, 2)
	if resets != 2 {
		t.Fatalf("resets after selection change: got: %d, want: 2", resets)
	}
}

func TestTextInsertionStyleUndoRedo(t *testing.T) {
	var txt textwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")
	var runs textstyle.Runs
	runs.SetVariation(0, 5, font.TagWght, float32(text.WeightBold))
	txt.CopyStyleRunsFrom(&runs)
	txt.SetSelection(5, 5)
	txt.SetInsertionStyle(textstyle.Style{}.WithVariation(font.TagWght, float32(text.WeightMedium)))
	txt.CommitTextByIME("x")

	if !txt.Undo() {
		t.Fatal("Undo() = false, want true")
	}
	if got := txt.Value(); got != "hello" {
		t.Fatalf("Value() after undo: got: %q, want: %q", got, "hello")
	}
	var wantRuns textstyle.Runs
	wantRuns.SetVariation(0, 5, font.TagWght, float32(text.WeightBold))
	if got := txt.StyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
		t.Errorf("after undo: got: %+v, want: %+v", got, runsSlice(&wantRuns))
	}

	if !txt.Redo() {
		t.Fatal("Redo() = false, want true")
	}
	if got := txt.Value(); got != "hellox" {
		t.Fatalf("Value() after redo: got: %q, want: %q", got, "hellox")
	}
	wantRuns.Clear()
	wantRuns.SetVariation(0, 5, font.TagWght, float32(text.WeightBold))
	wantRuns.SetVariation(5, 6, font.TagWght, float32(text.WeightMedium))
	if got := txt.StyleRuns(); !equalStyleRuns(got, runsSlice(&wantRuns)) {
		t.Errorf("after redo: got: %+v, want: %+v", got, runsSlice(&wantRuns))
	}
}
