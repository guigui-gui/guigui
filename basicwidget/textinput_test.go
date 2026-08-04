// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget_test

import (
	"testing"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

func TestSupportTextWidth(t *testing.T) {
	const defaultWidth = 240

	testCases := []struct {
		name        string
		constraints guigui.Constraints
		want        int
	}{
		{
			// The support text spans the whole widget width, which is the
			// fixed width and not what the text input area measures to.
			name:        "fixed width",
			constraints: guigui.FixedWidthConstraints(120),
			want:        120,
		},
		{
			name:        "no constraints",
			constraints: guigui.Constraints{},
			want:        defaultWidth,
		},
		{
			name:        "fixed height",
			constraints: guigui.FixedHeightConstraints(80),
			want:        defaultWidth,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := basicwidget.SupportTextWidth(tc.constraints, defaultWidth); got != tc.want {
				t.Errorf("SupportTextWidth() = %d, want %d", got, tc.want)
			}
		})
	}
}
