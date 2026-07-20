// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package font_test

import (
	"bytes"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

func TestFaceForStyleFamily(t *testing.T) {
	sources := make(map[string]*text.GoTextFaceSource)
	for name, ttf := range map[string][]byte{
		"regular":    goregular.TTF,
		"bold":       gobold.TTF,
		"italic":     goitalic.TTF,
		"bolditalic": gobolditalic.TTF,
	} {
		s, err := text.NewGoTextFaceSource(bytes.NewReader(ttf))
		if err != nil {
			t.Fatal(err)
		}
		sources[name] = s
	}
	family := font.NewFamily([]font.FaceSourceEntry{
		{FaceSource: sources["regular"]},
		{FaceSource: sources["bold"]},
		{FaceSource: sources["italic"]},
		{FaceSource: sources["bolditalic"]},
	}, &font.FamilyOptions{DisableFallback: true})

	const sample = "Styled sample text"
	sourceAdvance := func(source *text.GoTextFaceSource) float64 {
		return text.Advance(sample, &text.GoTextFace{Source: source, Size: 16})
	}
	seen := map[float64]string{}
	for name, source := range sources {
		a := sourceAdvance(source)
		if other, ok := seen[a]; ok {
			t.Fatalf("the %s and %s test fonts should have distinguishable advances", name, other)
		}
		seen[a] = name
	}

	var context guigui.Context
	tests := []struct {
		name   string
		italic bool
		weight text.Weight
		want   string
	}{
		{
			name:   "normal",
			italic: false,
			weight: text.WeightNormal,
			want:   "regular",
		},
		{
			name:   "bold",
			italic: false,
			weight: text.WeightBold,
			want:   "bold",
		},
		{
			name:   "italic",
			italic: true,
			weight: text.WeightNormal,
			want:   "italic",
		},
		{
			name:   "bold italic",
			italic: true,
			weight: text.WeightBold,
			want:   "bolditalic",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := font.Attributes{Size: 16, Italic: tt.italic}.WithVariation(font.TagWght, float32(tt.weight))
			f := font.NewFace(&context, family, a)
			if got, want := text.Advance(sample, f.TextFace()), sourceAdvance(sources[tt.want]); got != want {
				t.Errorf("advance: got: %v (%s), want: %v (%s)", got, seen[got], want, tt.want)
			}
		})
	}
}

func TestFaceForStaticWeightFamily(t *testing.T) {
	regular, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	bold, err := text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
	if err != nil {
		t.Fatal(err)
	}
	family := font.NewFamily([]font.FaceSourceEntry{
		{FaceSource: regular},
		{FaceSource: bold},
	}, &font.FamilyOptions{DisableFallback: true})

	const sample = "Static weights"
	sourceAdvance := func(source *text.GoTextFaceSource) float64 {
		return text.Advance(sample, &text.GoTextFace{Source: source, Size: 16})
	}
	if sourceAdvance(regular) == sourceAdvance(bold) {
		t.Fatal("the test fonts should have distinguishable advances")
	}

	var context guigui.Context
	tests := []struct {
		name   string
		weight text.Weight
		want   *text.GoTextFaceSource
	}{
		{
			name:   "normal picks the regular source",
			weight: text.WeightNormal,
			want:   regular,
		},
		{
			name:   "medium ties and the earlier source wins",
			weight: text.WeightMedium,
			want:   regular,
		},
		{
			name:   "bold picks the bold source",
			weight: text.WeightBold,
			want:   bold,
		},
		{
			name:   "black is closer to the bold source",
			weight: text.WeightBlack,
			want:   bold,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := font.Attributes{Size: 16}.WithVariation(font.TagWght, float32(tt.weight))
			f := font.NewFace(&context, family, a)
			if got, want := text.Advance(sample, f.TextFace()), sourceAdvance(tt.want); got != want {
				t.Errorf("advance for weight %v: got: %v, want: %v", tt.weight, got, want)
			}
		})
	}
}

func TestFaceForMixedStaticVariableFamily(t *testing.T) {
	bold, err := text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
	if err != nil {
		t.Fatal(err)
	}
	variable := font.DefaultFaceSourceEntry().FaceSource
	family := font.NewFamily([]font.FaceSourceEntry{
		{FaceSource: bold},
		{FaceSource: variable},
	}, &font.FamilyOptions{DisableFallback: true})

	const sample = "Mixed sources"
	variableAdvance := func(weight text.Weight) float64 {
		gtf := &text.GoTextFace{Source: variable, Size: 16}
		gtf.SetVariation(font.TagWght, float32(weight))
		return text.Advance(sample, gtf)
	}
	boldAdvance := text.Advance(sample, &text.GoTextFace{Source: bold, Size: 16})
	if boldAdvance == variableAdvance(text.WeightSemibold) {
		t.Fatal("the test fonts should have distinguishable advances")
	}

	var context guigui.Context

	// The variable font's wght axis covers the normal weight exactly, so it
	// wins over the static bold source listed before it.
	normal := font.NewFace(&context, family, font.Attributes{Size: 16}.WithVariation(font.TagWght, float32(text.WeightNormal)))
	if got, want := text.Advance(sample, normal.TextFace()), variableAdvance(text.WeightNormal); got != want {
		t.Errorf("advance for the normal weight: got: %v, want: %v", got, want)
	}

	// Both sources match the semibold weight exactly (the static bold
	// font's metadata weight is 600); the earlier entry wins.
	semibold := font.NewFace(&context, family, font.Attributes{Size: 16}.WithVariation(font.TagWght, float32(text.WeightSemibold)))
	if got, want := text.Advance(sample, semibold.TextFace()), boldAdvance; got != want {
		t.Errorf("advance for the semibold weight: got: %v, want: %v", got, want)
	}
}

func TestAttributesCanonical(t *testing.T) {
	// The setting order does not affect equality.
	a1 := font.Attributes{Size: 16}.WithVariation(font.TagWght, 700).WithFeature(font.TagLiga, 1).WithFeature(font.TagTnum, 0)
	a2 := font.Attributes{Size: 16}.WithFeature(font.TagTnum, 0).WithFeature(font.TagLiga, 1).WithVariation(font.TagWght, 700)
	if a1 != a2 {
		t.Errorf("attributes with the same settings should be equal: %v != %v", a1, a2)
	}

	// Setting a tag again overwrites the previous value.
	a3 := font.Attributes{Size: 16}.WithVariation(font.TagWght, 400).WithVariation(font.TagWght, 700)
	a4 := font.Attributes{Size: 16}.WithVariation(font.TagWght, 700)
	if a3 != a4 {
		t.Errorf("overwriting a tag should be equal to setting it once: %v != %v", a3, a4)
	}

	// Different values are not equal.
	a5 := font.Attributes{Size: 16}.WithVariation(font.TagWght, 400)
	if a4 == a5 {
		t.Errorf("attributes with different settings should not be equal: %v == %v", a4, a5)
	}
}

func TestFaceIDStableForSameRecipe(t *testing.T) {
	var context guigui.Context

	f1 := font.NewFace(&context, nil, font.Attributes{Size: 16})
	if f1.ID() == 0 {
		t.Fatal("a resolved face should have a nonzero id")
	}

	// The same recipe resolves to the same cached face, so the id is stable.
	f2 := font.NewFace(&context, nil, font.Attributes{Size: 16})
	if f1.ID() != f2.ID() {
		t.Errorf("same recipe should share an id: %d != %d", f1.ID(), f2.ID())
	}

	// A different recipe resolves to a different face, so the id differs.
	f3 := font.NewFace(&context, nil, font.Attributes{Size: 24})
	if f3.ID() == f1.ID() {
		t.Errorf("different recipe should have a different id, both %d", f1.ID())
	}
}
