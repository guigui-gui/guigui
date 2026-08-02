// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget_test

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/guigui-gui/guigui/basicwidget"
)

func TestTextStylesItalicInRange(t *testing.T) {
	var s basicwidget.TextStyles
	s.SetItalicInRange(2, 5, true)

	// A fully covered range is uniform with the override's value.
	if italic, uniform := s.ItalicInRange(2, 5, false); !uniform || !italic {
		t.Errorf("ItalicInRange(2, 5, false) = %t, %t; want true, true", italic, uniform)
	}
	// A partially covered range is not uniform when the fallback disagrees
	// with the override.
	if _, uniform := s.ItalicInRange(0, 5, false); uniform {
		t.Errorf("ItalicInRange(0, 5, false) uniform = true; want false")
	}
	// A partially covered range is uniform when the fallback agrees with the
	// override.
	if italic, uniform := s.ItalicInRange(0, 5, true); !uniform || !italic {
		t.Errorf("ItalicInRange(0, 5, true) = %t, %t; want true, true", italic, uniform)
	}
	// An uncovered range is uniform with the fallback.
	if italic, uniform := s.ItalicInRange(5, 10, false); !uniform || italic {
		t.Errorf("ItalicInRange(5, 10, false) = %t, %t; want false, true", italic, uniform)
	}
	// An empty range is uniform with the fallback.
	if italic, uniform := s.ItalicInRange(3, 3, false); !uniform || italic {
		t.Errorf("ItalicInRange(3, 3, false) = %t, %t; want false, true", italic, uniform)
	}
	// A negative start behaves like 0.
	if _, uniform := s.ItalicInRange(-3, 5, false); uniform {
		t.Errorf("ItalicInRange(-3, 5, false) uniform = true; want false")
	}
}

func TestTextStylesItalicInRangeEmptySet(t *testing.T) {
	var s basicwidget.TextStyles
	if italic, uniform := s.ItalicInRange(0, 10, true); !uniform || !italic {
		t.Errorf("ItalicInRange(0, 10, true) = %t, %t; want true, true", italic, uniform)
	}
	if italic, uniform := s.ItalicInRange(0, 10, false); !uniform || italic {
		t.Errorf("ItalicInRange(0, 10, false) = %t, %t; want false, true", italic, uniform)
	}
}

func TestTextStylesWeightInRange(t *testing.T) {
	var s basicwidget.TextStyles
	s.SetWeightInRange(0, 4, text.WeightBold)

	if weight, uniform := s.WeightInRange(0, 4, text.WeightNormal); !uniform || weight != text.WeightBold {
		t.Errorf("WeightInRange(0, 4, WeightNormal) = %v, %t; want %v, true", weight, uniform, text.WeightBold)
	}
	if _, uniform := s.WeightInRange(0, 6, text.WeightNormal); uniform {
		t.Errorf("WeightInRange(0, 6, WeightNormal) uniform = true; want false")
	}
	if weight, uniform := s.WeightInRange(0, 6, text.WeightBold); !uniform || weight != text.WeightBold {
		t.Errorf("WeightInRange(0, 6, WeightBold) = %v, %t; want %v, true", weight, uniform, text.WeightBold)
	}
	if weight, uniform := s.WeightInRange(4, 6, text.WeightNormal); !uniform || weight != text.WeightNormal {
		t.Errorf("WeightInRange(4, 6, WeightNormal) = %v, %t; want %v, true", weight, uniform, text.WeightNormal)
	}
}

func TestTextStylesVariationInRange(t *testing.T) {
	tagWght := text.MustParseTag("wght")

	var s basicwidget.TextStyles
	s.SetVariationInRange(1, 3, tagWght, 700)

	if v, uniform := s.VariationInRange(1, 3, tagWght, 400); !uniform || v != 700 {
		t.Errorf("VariationInRange(1, 3, wght, 400) = %v, %t; want 700, true", v, uniform)
	}
	if _, uniform := s.VariationInRange(0, 3, tagWght, 400); uniform {
		t.Errorf("VariationInRange(0, 3, wght, 400) uniform = true; want false")
	}
	// SetWeightInRange sets the wght variation axis.
	var s2 basicwidget.TextStyles
	s2.SetWeightInRange(0, 2, text.WeightBold)
	if v, uniform := s2.VariationInRange(0, 2, tagWght, 400); !uniform || v != float32(text.WeightBold) {
		t.Errorf("VariationInRange(0, 2, wght, 400) = %v, %t; want %v, true", v, uniform, float32(text.WeightBold))
	}
}

func TestTextStylesScaleInRange(t *testing.T) {
	var s basicwidget.TextStyles
	s.SetScaleInRange(3, 6, 2)

	if scale, uniform := s.ScaleInRange(3, 6, 1); !uniform || scale != 2 {
		t.Errorf("ScaleInRange(3, 6, 1) = %v, %t; want 2, true", scale, uniform)
	}
	if _, uniform := s.ScaleInRange(0, 6, 1); uniform {
		t.Errorf("ScaleInRange(0, 6, 1) uniform = true; want false")
	}
	if scale, uniform := s.ScaleInRange(0, 6, 2); !uniform || scale != 2 {
		t.Errorf("ScaleInRange(0, 6, 2) = %v, %t; want 2, true", scale, uniform)
	}
	if scale, uniform := s.ScaleInRange(6, 9, 1); !uniform || scale != 1 {
		t.Errorf("ScaleInRange(6, 9, 1) = %v, %t; want 1, true", scale, uniform)
	}
}

func TestTextStylesColorInRange(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}

	var s basicwidget.TextStyles
	s.SetColorInRange(0, 3, red)
	s.SetColorInRange(3, 6, blue)

	if clr, uniform := s.ColorInRange(0, 3, nil); !uniform || clr != red {
		t.Errorf("ColorInRange(0, 3, nil) = %v, %t; want %v, true", clr, uniform, red)
	}
	if _, uniform := s.ColorInRange(0, 6, nil); uniform {
		t.Errorf("ColorInRange(0, 6, nil) uniform = true; want false")
	}
	if _, uniform := s.ColorInRange(0, 8, red); uniform {
		t.Errorf("ColorInRange(0, 8, red) uniform = true; want false")
	}
	// A nil fallback reports an uncovered range as uniform with nil.
	if clr, uniform := s.ColorInRange(6, 9, nil); !uniform || clr != nil {
		t.Errorf("ColorInRange(6, 9, nil) = %v, %t; want nil, true", clr, uniform)
	}
	if clr, uniform := s.ColorInRange(6, 9, red); !uniform || clr != red {
		t.Errorf("ColorInRange(6, 9, red) = %v, %t; want %v, true", clr, uniform, red)
	}
}

func TestTextStylesUnderlineInRange(t *testing.T) {
	var s basicwidget.TextStyles
	s.SetUnderlineInRange(1, 4, true)

	if underline, uniform := s.UnderlineInRange(1, 4, false); !uniform || !underline {
		t.Errorf("UnderlineInRange(1, 4, false) = %t, %t; want true, true", underline, uniform)
	}
	if _, uniform := s.UnderlineInRange(0, 4, false); uniform {
		t.Errorf("UnderlineInRange(0, 4, false) uniform = true; want false")
	}
	if underline, uniform := s.UnderlineInRange(4, 8, false); !uniform || underline {
		t.Errorf("UnderlineInRange(4, 8, false) = %t, %t; want false, true", underline, uniform)
	}
}

func TestTextStylesStrikethroughInRange(t *testing.T) {
	var s basicwidget.TextStyles
	s.SetStrikethroughInRange(1, 4, true)

	if strikethrough, uniform := s.StrikethroughInRange(1, 4, false); !uniform || !strikethrough {
		t.Errorf("StrikethroughInRange(1, 4, false) = %t, %t; want true, true", strikethrough, uniform)
	}
	if _, uniform := s.StrikethroughInRange(0, 4, false); uniform {
		t.Errorf("StrikethroughInRange(0, 4, false) uniform = true; want false")
	}
	if strikethrough, uniform := s.StrikethroughInRange(4, 8, false); !uniform || strikethrough {
		t.Errorf("StrikethroughInRange(4, 8, false) = %t, %t; want false, true", strikethrough, uniform)
	}
}

func TestTextStylesReadersAfterUnset(t *testing.T) {
	var s basicwidget.TextStyles
	s.SetItalicInRange(0, 10, true)
	s.UnsetItalicInRange(0, 10)

	if italic, uniform := s.ItalicInRange(0, 10, false); !uniform || italic {
		t.Errorf("ItalicInRange(0, 10, false) = %t, %t; want false, true", italic, uniform)
	}
}
