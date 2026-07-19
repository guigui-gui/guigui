// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package font_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

func TestAttributesCanonical(t *testing.T) {
	tagWght := text.MustParseTag("wght")
	tagLiga := text.MustParseTag("liga")
	tagTnum := text.MustParseTag("tnum")

	// The setting order does not affect equality.
	a1 := font.Attributes{Size: 16}.WithVariation(tagWght, 700).WithFeature(tagLiga, 1).WithFeature(tagTnum, 0)
	a2 := font.Attributes{Size: 16}.WithFeature(tagTnum, 0).WithFeature(tagLiga, 1).WithVariation(tagWght, 700)
	if a1 != a2 {
		t.Errorf("attributes with the same settings should be equal: %v != %v", a1, a2)
	}

	// Setting a tag again overwrites the previous value.
	a3 := font.Attributes{Size: 16}.WithVariation(tagWght, 400).WithVariation(tagWght, 700)
	a4 := font.Attributes{Size: 16}.WithVariation(tagWght, 700)
	if a3 != a4 {
		t.Errorf("overwriting a tag should be equal to setting it once: %v != %v", a3, a4)
	}

	// Different values are not equal.
	a5 := font.Attributes{Size: 16}.WithVariation(tagWght, 400)
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
