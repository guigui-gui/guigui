// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textstyle_test

import (
	"image/color"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
)

func TestStyleAffectsFaceSelection(t *testing.T) {
	tests := []struct {
		name  string
		style textstyle.Style
		want  bool
	}{
		{
			name:  "zero",
			style: textstyle.Style{},
			want:  false,
		},
		{
			name:  "color",
			style: textstyle.Style{}.WithColor(color.RGBA{R: 0xff, A: 0xff}),
			want:  false,
		},
		{
			name:  "underline",
			style: textstyle.Style{}.WithUnderline(true),
			want:  false,
		},
		{
			name:  "variation",
			style: textstyle.Style{}.WithVariation(font.TagWght, float32(text.WeightBold)),
			want:  true,
		},
		{
			name:  "feature",
			style: textstyle.Style{}.WithFeature(font.TagTnum, 1),
			want:  true,
		},
		{
			name:  "italic",
			style: textstyle.Style{}.WithItalic(true),
			want:  true,
		},
		{
			name:  "explicit nil family",
			style: textstyle.Style{}.WithFamily(nil),
			want:  true,
		},
		{
			name:  "lang",
			style: textstyle.Style{}.WithLang(language.Japanese),
			want:  true,
		},
		{
			name:  "scale",
			style: textstyle.Style{}.WithScale(1.5),
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.style.AffectsFaceSelection(); got != tt.want {
				t.Errorf("got: %t, want: %t", got, tt.want)
			}
		})
	}
}

func TestStyleIsZero(t *testing.T) {
	tests := []struct {
		name  string
		style textstyle.Style
		want  bool
	}{
		{
			name:  "zero",
			style: textstyle.Style{},
			want:  true,
		},
		{
			name:  "italic",
			style: textstyle.Style{}.WithItalic(true),
			want:  false,
		},
		{
			name:  "scale",
			style: textstyle.Style{}.WithScale(1.5),
			want:  false,
		},
		{
			name:  "color",
			style: textstyle.Style{}.WithColor(color.RGBA{R: 0xff, A: 0xff}),
			want:  false,
		},
		{
			name:  "nil color",
			style: textstyle.Style{}.WithColor(nil),
			want:  false,
		},
		{
			name:  "background color",
			style: textstyle.Style{}.WithBackgroundColor(color.RGBA{B: 0xff, A: 0xff}),
			want:  false,
		},
		{
			name:  "underline off",
			style: textstyle.Style{}.WithUnderline(false),
			want:  false,
		},
		{
			name:  "strikethrough",
			style: textstyle.Style{}.WithStrikethrough(true),
			want:  false,
		},
		{
			name:  "lang",
			style: textstyle.Style{}.WithLang(language.Japanese),
			want:  false,
		},
		{
			name:  "feature",
			style: textstyle.Style{}.WithFeature(font.TagTnum, 1),
			want:  false,
		},
		{
			name:  "variation",
			style: textstyle.Style{}.WithVariation(font.TagWght, float32(text.WeightBold)),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.style.IsZero(); got != tt.want {
				t.Errorf("got: %t, want: %t", got, tt.want)
			}
		})
	}
}

func TestStyleEqual(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}

	tests := []struct {
		name string
		a    textstyle.Style
		b    textstyle.Style
		want bool
	}{
		{
			name: "zero styles",
			a:    textstyle.Style{},
			b:    textstyle.Style{},
			want: true,
		},
		{
			name: "same properties",
			a:    textstyle.Style{}.WithColor(red).WithUnderline(true),
			b:    textstyle.Style{}.WithColor(red).WithUnderline(true),
			want: true,
		},
		{
			name: "different bools",
			a:    textstyle.Style{}.WithUnderline(true),
			b:    textstyle.Style{}.WithUnderline(false),
			want: false,
		},
		{
			name: "set false differs from unset",
			a:    textstyle.Style{}.WithUnderline(false),
			b:    textstyle.Style{},
			want: false,
		},
		{
			name: "explicit nil color differs from unset",
			a:    textstyle.Style{}.WithColor(nil),
			b:    textstyle.Style{},
			want: false,
		},
		{
			name: "same features",
			a:    textstyle.Style{}.WithFeature(font.TagLiga, 1),
			b:    textstyle.Style{}.WithFeature(font.TagLiga, 1),
			want: true,
		},
		{
			name: "different feature values",
			a:    textstyle.Style{}.WithFeature(font.TagLiga, 1),
			b:    textstyle.Style{}.WithFeature(font.TagLiga, 0),
			want: false,
		},
		{
			name: "feature build order does not matter",
			a:    textstyle.Style{}.WithFeature(font.TagLiga, 1).WithFeature(font.TagTnum, 1),
			b:    textstyle.Style{}.WithFeature(font.TagTnum, 1).WithFeature(font.TagLiga, 1),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equal(tt.b); got != tt.want {
				t.Errorf("got: %t, want: %t", got, tt.want)
			}
			if got := tt.b.Equal(tt.a); got != tt.want {
				t.Errorf("(reversed) got: %t, want: %t", got, tt.want)
			}
		})
	}
}

func TestStyleWithVariation(t *testing.T) {
	s := textstyle.Style{}.WithVariation(font.TagWght, 700)
	if v, ok := s.Variation(font.TagWght); !ok || v != 700 {
		t.Errorf("Variation(wght) = %v, %t; want 700, true", v, ok)
	}

	// Replacing a value does not mutate the original style.
	s2 := s.WithVariation(font.TagWght, 400)
	if v, ok := s2.Variation(font.TagWght); !ok || v != 400 {
		t.Errorf("Variation(wght) = %v, %t; want 400, true", v, ok)
	}
	if v, ok := s.Variation(font.TagWght); !ok || v != 700 {
		t.Errorf("original Variation(wght) = %v, %t; want 700, true", v, ok)
	}

	// Setting the same value keeps the style equal.
	if s3 := s.WithVariation(font.TagWght, 700); !s3.Equal(s) {
		t.Errorf("style changed by setting an identical variation")
	}

	// Adding another tag does not mutate the original style.
	s4 := s.WithVariation(font.TagTnum, 1)
	if _, ok := s4.Variation(font.TagTnum); !ok {
		t.Errorf("Variation(tnum) not set")
	}
	if _, ok := s.Variation(font.TagTnum); ok {
		t.Errorf("original style gained a tnum variation")
	}
}

func TestStyleWithoutVariation(t *testing.T) {
	s := textstyle.Style{}.WithVariation(font.TagWght, 700).WithVariation(font.TagTnum, 1)
	s2 := s.WithoutVariation(font.TagWght)
	if _, ok := s2.Variation(font.TagWght); ok {
		t.Errorf("Variation(wght) still set")
	}
	if _, ok := s2.Variation(font.TagTnum); !ok {
		t.Errorf("Variation(tnum) removed")
	}
	if _, ok := s.Variation(font.TagWght); !ok {
		t.Errorf("original style lost its wght variation")
	}

	// Removing an absent tag keeps the style equal.
	if s3 := s.WithoutVariation(font.TagLiga); !s3.Equal(s) {
		t.Errorf("style changed by removing an absent variation")
	}
}

func TestStyleWithFeature(t *testing.T) {
	s := textstyle.Style{}.WithFeature(font.TagLiga, 1)
	s2 := s.WithFeature(font.TagLiga, 0)
	if got, want := s2.Features(), []font.Feature{{Tag: font.TagLiga, Value: 0}}; !slices.Equal(got, want) {
		t.Errorf("Features() = %v; want %v", got, want)
	}
	if got, want := s.Features(), []font.Feature{{Tag: font.TagLiga, Value: 1}}; !slices.Equal(got, want) {
		t.Errorf("original Features() = %v; want %v", got, want)
	}
	if s3 := s.WithoutFeature(font.TagLiga); len(s3.Features()) != 0 {
		t.Errorf("Features() = %v; want none", s3.Features())
	}
}

func TestStyleMerge(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}

	base := textstyle.Style{}.
		WithItalic(false).
		WithColor(red).
		WithLang(language.Japanese).
		WithVariation(font.TagWght, 500)
	override := textstyle.Style{}.
		WithItalic(true).
		WithColor(blue).
		WithVariation(font.TagWght, 700).
		WithVariation(font.TagTnum, 1).
		WithUnderline(true)

	merged := base.Merge(override)
	if italic, ok := merged.Italic(); !ok || !italic {
		t.Errorf("Italic() = %t, %t; want true, true", italic, ok)
	}
	if clr, ok := merged.Color(); !ok || clr != color.Color(blue) {
		t.Errorf("Color() = %v, %t; want %v, true", clr, ok, blue)
	}
	// A property unset in the override passes the base value through.
	if lang, ok := merged.Lang(); !ok || lang != language.Japanese {
		t.Errorf("Lang() = %v, %t; want %v, true", lang, ok, language.Japanese)
	}
	// The override wins on a shared variation tag.
	if v, ok := merged.Variation(font.TagWght); !ok || v != 700 {
		t.Errorf("Variation(wght) = %v, %t; want 700, true", v, ok)
	}
	if v, ok := merged.Variation(font.TagTnum); !ok || v != 1 {
		t.Errorf("Variation(tnum) = %v, %t; want 1, true", v, ok)
	}
	// A property set only in the override is carried over.
	if underline, ok := merged.Underline(); !ok || !underline {
		t.Errorf("Underline() = %t, %t; want true, true", underline, ok)
	}
	// Merging the zero style changes nothing.
	if got := base.Merge(textstyle.Style{}); !got.Equal(base) {
		t.Errorf("Merge(zero) changed the style")
	}
}
