// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textstyle_test

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
)

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
