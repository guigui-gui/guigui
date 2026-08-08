// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget_test

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

func TestTextStyleProperties(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var s basicwidget.TextStyle
	if _, ok := s.Italic(); ok {
		t.Error("Italic() must be unset on the zero value")
	}

	s.SetItalic(true)
	if italic, ok := s.Italic(); !ok || !italic {
		t.Errorf("Italic() = %t, %t; want true, true", italic, ok)
	}
	s.UnsetItalic()
	if _, ok := s.Italic(); ok {
		t.Error("Italic() must be unset after UnsetItalic")
	}

	s.SetColor(red)
	if clr, ok := s.Color(); !ok || clr != color.Color(red) {
		t.Errorf("Color() = %v, %t; want %v, true", clr, ok, red)
	}

	s.SetScale(1.5)
	if scale, ok := s.Scale(); !ok || scale != 1.5 {
		t.Errorf("Scale() = %v, %t; want 1.5, true", scale, ok)
	}

	s.SetUnderline(true)
	if underline, ok := s.Underline(); !ok || !underline {
		t.Errorf("Underline() = %t, %t; want true, true", underline, ok)
	}

	s.SetStrikethrough(true)
	if strikethrough, ok := s.Strikethrough(); !ok || !strikethrough {
		t.Errorf("Strikethrough() = %t, %t; want true, true", strikethrough, ok)
	}
}

func TestTextStyleWeight(t *testing.T) {
	tagWght := text.MustParseTag("wght")

	var s basicwidget.TextStyle
	s.SetWeight(text.WeightBold)
	if w, ok := s.Weight(); !ok || w != text.WeightBold {
		t.Errorf("Weight() = %v, %t; want %v, true", w, ok, text.WeightBold)
	}
	// SetWeight sets the wght variation axis.
	if v, ok := s.Variation(tagWght); !ok || v != float32(text.WeightBold) {
		t.Errorf("Variation(wght) = %v, %t; want %v, true", v, ok, float32(text.WeightBold))
	}
	s.UnsetWeight()
	if _, ok := s.Weight(); ok {
		t.Error("Weight() must be unset after UnsetWeight")
	}
}

func TestTextOverrideStylesRoundTrip(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt basicwidget.Text
	txt.SetValue("hello world")

	var styles basicwidget.TextStyles
	styles.SetColorInRange(0, 5, red)
	styles.SetUnderlineInRange(6, 11, true)
	txt.SetOverrideStyles(&styles, false)

	var got basicwidget.TextStyles
	txt.ReadOverrideStyles(&got)
	if clr, uniform := got.ColorInRange(0, 5, nil); !uniform || clr != color.Color(red) {
		t.Errorf("ColorInRange(0, 5) = %v, %t; want %v, true", clr, uniform, red)
	}
	if underline, uniform := got.UnderlineInRange(6, 11, false); !uniform || !underline {
		t.Errorf("UnderlineInRange(6, 11) = %t, %t; want true, true", underline, uniform)
	}
}

func TestTextOverrideStylesInRange(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}

	var txt basicwidget.Text
	txt.SetValue("hello world")

	var styles basicwidget.TextStyles
	styles.SetColorInRange(0, 11, red)
	txt.SetOverrideStyles(&styles, false)

	// Replace [2, 6) with a blue override over its first 3 bytes.
	var repl basicwidget.TextStyles
	repl.SetColorInRange(0, 3, blue)
	txt.SetOverrideStylesInRange(&repl, 2, 6, false)

	// Read [0, 6) back; the result is rebased to 0.
	var got basicwidget.TextStyles
	txt.ReadOverrideStylesInRange(&got, 0, 6)
	if clr, uniform := got.ColorInRange(0, 2, nil); !uniform || clr != color.Color(red) {
		t.Errorf("ColorInRange(0, 2) = %v, %t; want %v, true", clr, uniform, red)
	}
	if clr, uniform := got.ColorInRange(2, 5, nil); !uniform || clr != color.Color(blue) {
		t.Errorf("ColorInRange(2, 5) = %v, %t; want %v, true", clr, uniform, blue)
	}
	if clr, uniform := got.ColorInRange(5, 6, nil); !uniform || clr != nil {
		t.Errorf("ColorInRange(5, 6) = %v, %t; want nil, true", clr, uniform)
	}
}

func TestTextReadEffectiveStyles(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt basicwidget.Text
	txt.SetValue("hello world")

	var base basicwidget.TextStyle
	base.SetItalic(true)
	base.SetWeight(text.WeightBold)
	txt.SetBaseStyle(&base)

	var styles basicwidget.TextStyles
	styles.SetColorInRange(3, 5, red)
	txt.SetOverrideStyles(&styles, false)

	// The effective styles resolve the base style, the rendering defaults,
	// and the overrides; every property is set.
	var effective basicwidget.TextStyles
	txt.ReadEffectiveStylesInRange(&effective, 2, 6)
	if italic, uniform := effective.ItalicInRange(0, 4, false); !uniform || !italic {
		t.Errorf("ItalicInRange(0, 4) = %t, %t; want true, true", italic, uniform)
	}
	if w, uniform := effective.WeightInRange(0, 4, text.WeightNormal); !uniform || w != text.WeightBold {
		t.Errorf("WeightInRange(0, 4) = %v, %t; want %v, true", w, uniform, text.WeightBold)
	}
	if scale, uniform := effective.ScaleInRange(0, 4, 0); !uniform || scale != 1 {
		t.Errorf("ScaleInRange(0, 4) = %v, %t; want 1, true", scale, uniform)
	}
	if clr, uniform := effective.ColorInRange(1, 3, nil); !uniform || clr != color.Color(red) {
		t.Errorf("ColorInRange(1, 3) = %v, %t; want %v, true", clr, uniform, red)
	}

	// The whole-value read covers every byte of the value.
	var whole basicwidget.TextStyles
	txt.ReadEffectiveStyles(&whole)
	if italic, uniform := whole.ItalicInRange(0, 11, false); !uniform || !italic {
		t.Errorf("ItalicInRange(0, 11) = %t, %t; want true, true", italic, uniform)
	}
	if _, uniform := whole.ColorInRange(0, 11, nil); uniform {
		t.Errorf("ColorInRange(0, 11) uniform = true; want false")
	}
	if clr, uniform := whole.ColorInRange(3, 5, nil); !uniform || clr != color.Color(red) {
		t.Errorf("ColorInRange(3, 5) = %v, %t; want %v, true", clr, uniform, red)
	}

	// The effective style at an index is the style that text typed there
	// adopts.
	var at basicwidget.TextStyle
	txt.ReadEffectiveStyleAt(&at, 4)
	if clr, ok := at.Color(); !ok || clr != color.Color(red) {
		t.Errorf("Color() at 4 = %v, %t; want %v, true", clr, ok, red)
	}
	txt.ReadEffectiveStyleAt(&at, 0)
	if w, ok := at.Weight(); !ok || w != text.WeightBold {
		t.Errorf("Weight() at 0 = %v, %t; want %v, true", w, ok, text.WeightBold)
	}
}

func TestTextBaseStyleRoundTrip(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt basicwidget.Text

	var base basicwidget.TextStyle
	base.SetItalic(true)
	base.SetWeight(text.WeightBold)
	base.SetColor(red)
	txt.SetBaseStyle(&base)

	var got basicwidget.TextStyle
	txt.ReadBaseStyle(&got)
	if italic, ok := got.Italic(); !ok || !italic {
		t.Errorf("Italic() = %t, %t; want true, true", italic, ok)
	}
	if w, ok := got.Weight(); !ok || w != text.WeightBold {
		t.Errorf("Weight() = %v, %t; want %v, true", w, ok, text.WeightBold)
	}
	if clr, ok := got.Color(); !ok || clr != color.Color(red) {
		t.Errorf("Color() = %v, %t; want %v, true", clr, ok, red)
	}
	if _, ok := got.FontFamily(); ok {
		t.Error("FontFamily() must be unset without a font family")
	}

	// Setting a base style with fewer properties unsets the rest.
	var smaller basicwidget.TextStyle
	smaller.SetItalic(false)
	txt.SetBaseStyle(&smaller)
	txt.ReadBaseStyle(&got)
	if _, ok := got.Weight(); ok {
		t.Error("Weight() must be unset after setting a base style without it")
	}
	if _, ok := got.Color(); ok {
		t.Error("Color() must be unset after setting a base style without it")
	}
}

func TestTextInsertionStyle(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt basicwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello")
	txt.SetSelection(5, 5)

	var insertion basicwidget.TextStyle
	insertion.SetColor(red)
	txt.SetInsertionStyle(&insertion)

	// Inserting at the caret materializes the insertion style over the
	// inserted span and resets it, dispatching the reset event.
	var resets int
	txt.OnInsertionStyleReset(func(context *guigui.Context) {
		resets++
	})
	txt.ReplaceValueAtSelection("ab")
	var styles basicwidget.TextStyles
	txt.ReadOverrideStyles(&styles)
	if clr, uniform := styles.ColorInRange(5, 7, nil); !uniform || clr != color.Color(red) {
		t.Errorf("ColorInRange(5, 7) = %v, %t; want %v, true", clr, uniform, red)
	}
	if clr, uniform := styles.ColorInRange(0, 5, nil); !uniform || clr != nil {
		t.Errorf("ColorInRange(0, 5) = %v, %t; want nil, true", clr, uniform)
	}
	if resets != 1 {
		t.Errorf("resets: got: %d, want: 1", resets)
	}
}

func TestTextOverrideStylesUndoRedo(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt basicwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello world")

	var valueChanges int
	txt.OnValueChangedWithoutText(func(context *guigui.Context, committed bool) {
		valueChanges++
	})

	// A ranged replacement records an undo entry; the write itself does not
	// dispatch a value change.
	var styles basicwidget.TextStyles
	styles.SetColorInRange(0, 5, red)
	txt.SetOverrideStylesInRange(&styles, 0, 5, true)
	if !txt.CanUndo() {
		t.Fatal("CanUndo must return true after the style change")
	}
	if valueChanges != 0 {
		t.Errorf("valueChanges: got: %d, want: 0", valueChanges)
	}

	// Undo restores the previous overrides, keeps the value, and dispatches
	// a value change so apps can read the overrides back.
	if !txt.Undo() {
		t.Fatal("Undo must return true")
	}
	if got, want := txt.Value(), "hello world"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
	var got basicwidget.TextStyles
	txt.ReadOverrideStyles(&got)
	if clr, uniform := got.ColorInRange(0, 5, nil); !uniform || clr != nil {
		t.Errorf("ColorInRange(0, 5) = %v, %t; want nil, true", clr, uniform)
	}
	if valueChanges != 1 {
		t.Errorf("valueChanges: got: %d, want: 1", valueChanges)
	}

	if !txt.Redo() {
		t.Fatal("Redo must return true")
	}
	txt.ReadOverrideStyles(&got)
	if clr, uniform := got.ColorInRange(0, 5, nil); !uniform || clr != color.Color(red) {
		t.Errorf("ColorInRange(0, 5) = %v, %t; want %v, true", clr, uniform, red)
	}
	if valueChanges != 2 {
		t.Errorf("valueChanges: got: %d, want: 2", valueChanges)
	}
}

func TestTextSetOverrideStylesRecordsNoHistory(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	var txt basicwidget.Text
	txt.SetEditable(true)
	txt.ForceSetValue("hello world")

	// Unrecorded whole-value replacements restore model state on every build
	// and must not grow the undo history.
	var styles basicwidget.TextStyles
	styles.SetColorInRange(0, 5, red)
	for range 3 {
		txt.SetOverrideStyles(&styles, false)
	}
	if txt.CanUndo() {
		t.Error("CanUndo must return false after whole-value replacements")
	}

	// A ranged replacement equal to the current overrides records nothing
	// either; only a changing one does.
	txt.SetOverrideStylesInRange(&styles, 0, 11, true)
	if txt.CanUndo() {
		t.Fatal("CanUndo must return false after an unchanged ranged replacement")
	}
	var underline basicwidget.TextStyles
	underline.SetUnderlineInRange(0, 5, true)
	txt.SetOverrideStylesInRange(&underline, 6, 11, true)
	if !txt.CanUndo() {
		t.Fatal("CanUndo must return true after the style change")
	}
	txt.SetOverrideStylesInRange(&underline, 6, 11, true)
	if !txt.Undo() {
		t.Fatal("Undo must return true")
	}
	if txt.CanUndo() {
		t.Error("CanUndo must return false after undoing the only style change")
	}
}
