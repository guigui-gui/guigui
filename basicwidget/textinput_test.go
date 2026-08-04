// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget_test

import (
	"image"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget"
)

func TestSupportTextMinX(t *testing.T) {
	inputBounds := image.Rect(100, 0, 300, 20)

	testCases := []struct {
		name   string
		halign basicwidget.HorizontalAlign
		width  int
		want   int
	}{
		{
			name:   "start",
			halign: basicwidget.HorizontalAlignStart,
			width:  500,
			want:   100,
		},
		{
			name:   "center",
			halign: basicwidget.HorizontalAlignCenter,
			width:  500,
			want:   -50,
		},
		{
			name:   "end",
			halign: basicwidget.HorizontalAlignEnd,
			width:  500,
			want:   -200,
		},
		{
			// A support text no wider than the text input area keeps the
			// area's bounds whatever the alignment is.
			name:   "start, fits",
			halign: basicwidget.HorizontalAlignStart,
			width:  200,
			want:   100,
		},
		{
			name:   "center, fits",
			halign: basicwidget.HorizontalAlignCenter,
			width:  200,
			want:   100,
		},
		{
			name:   "end, fits",
			halign: basicwidget.HorizontalAlignEnd,
			width:  200,
			want:   100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := basicwidget.SupportTextMinX(tc.halign, inputBounds, tc.width); got != tc.want {
				t.Errorf("SupportTextMinX() = %d, want %d", got, tc.want)
			}
		})
	}
}
