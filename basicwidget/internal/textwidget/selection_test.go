// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget_test

import (
	"testing"

	"github.com/guigui-gui/guigui/basicwidget/internal/textwidget"
)

func TestShiftClickAnchor(t *testing.T) {
	testCases := []struct {
		name      string
		start     int
		end       int
		shiftSide textwidget.SelectionSide
		idx       int
		want      int
	}{
		{
			name:      "caret extends to the right",
			start:     5,
			end:       5,
			shiftSide: textwidget.SelectionSideNone,
			idx:       9,
			want:      5,
		},
		{
			name:      "caret extends to the left",
			start:     5,
			end:       5,
			shiftSide: textwidget.SelectionSideNone,
			idx:       2,
			want:      5,
		},
		{
			name:      "moving end at the end keeps the start anchored",
			start:     5,
			end:       10,
			shiftSide: textwidget.SelectionSideEnd,
			idx:       2,
			want:      5,
		},
		{
			name:      "moving end at the start keeps the end anchored",
			start:     5,
			end:       10,
			shiftSide: textwidget.SelectionSideStart,
			idx:       20,
			want:      10,
		},
		{
			name:      "untracked selection: click to the right keeps the start",
			start:     5,
			end:       10,
			shiftSide: textwidget.SelectionSideNone,
			idx:       20,
			want:      5,
		},
		{
			name:      "untracked selection: click to the left keeps the end",
			start:     5,
			end:       10,
			shiftSide: textwidget.SelectionSideNone,
			idx:       2,
			want:      10,
		},
		{
			name:      "untracked selection: click inside nearer the end keeps the start",
			start:     0,
			end:       10,
			shiftSide: textwidget.SelectionSideNone,
			idx:       7,
			want:      0,
		},
		{
			name:      "untracked selection: click inside nearer the start keeps the end",
			start:     0,
			end:       10,
			shiftSide: textwidget.SelectionSideNone,
			idx:       3,
			want:      10,
		},
		{
			name:      "untracked selection: equidistant click keeps the start",
			start:     0,
			end:       10,
			shiftSide: textwidget.SelectionSideNone,
			idx:       5,
			want:      0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := textwidget.ShiftClickAnchor(tc.start, tc.end, tc.shiftSide, tc.idx); got != tc.want {
				t.Errorf("ShiftClickAnchor(%d, %d, %v, %d) = %d, want %d", tc.start, tc.end, tc.shiftSide, tc.idx, got, tc.want)
			}
		})
	}
}
