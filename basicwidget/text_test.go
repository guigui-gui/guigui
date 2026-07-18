// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget_test

import (
	"image"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget"
)

func TestMultilineTextInputHorizontalOverflowKeepsWrapWidth(t *testing.T) {
	const (
		containerWidth = 200
		paddingStart   = 10
		paddingEnd     = 10
		overflowWidth  = 500
	)

	var txt basicwidget.TextInputText
	txt.ConfigureHorizontalLayout(basicwidget.WrapModeNormal, containerWidth, paddingStart, paddingEnd, overflowWidth)
	if got, want := txt.ContentWidth(), overflowWidth; got != want {
		t.Errorf("ContentWidth() = %d, want %d", got, want)
	}
	if got, want := txt.LayoutWidth(image.Rect(0, 0, overflowWidth, 100)), containerWidth-paddingStart-paddingEnd; got != want {
		t.Errorf("LayoutWidth() = %d, want %d", got, want)
	}
}
