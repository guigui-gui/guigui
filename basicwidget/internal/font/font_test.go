// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package font_test

import (
	"testing"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

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
