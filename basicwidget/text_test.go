// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget_test

import (
	"fmt"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget"
)

func TestReplaceNewLineWithSpace(t *testing.T) {
	testCases := []struct {
		text     string
		start    int
		end      int
		outText  string
		outStart int
		outEnd   int
	}{
		{
			text:     "",
			start:    0,
			end:      0,
			outText:  "",
			outStart: 0,
			outEnd:   0,
		},
		{
			text:     "Hello,\nWorld!",
			start:    7,
			end:      13,
			outText:  "Hello, World!",
			outStart: 7,
			outEnd:   13,
		},
		{
			text:     "Hello,\nWorld!",
			start:    7,
			end:      13,
			outText:  "Hello, World!",
			outStart: 7,
			outEnd:   13,
		},
		{
			text:     "Hello,\r\nWorld!",
			start:    6,
			end:      6,
			outText:  "Hello, World!",
			outStart: 6,
			outEnd:   6,
		},
		{
			text:     "Hello,\r\nWorld!",
			start:    8,
			end:      14,
			outText:  "Hello, World!",
			outStart: 7,
			outEnd:   13,
		},
		{
			text:     "Hello,\u2028World!",
			start:    9,
			end:      15,
			outText:  "Hello, World!",
			outStart: 7,
			outEnd:   13,
		},
		{
			text:     "Hello,\r\nWorld!",
			start:    6,
			end:      7, // In between \r and \n
			outText:  "Hello, World!",
			outStart: 6,
			outEnd:   7,
		},
		{
			text:     "a\r\u2028\nb",
			start:    5,
			end:      7,
			outText:  "a   b",
			outStart: 3,
			outEnd:   5,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%q", tc.text), func(t *testing.T) {
			gotText, gotStart, gotEnd := basicwidget.ReplaceNewLinesWithSpace(tc.text, tc.start, tc.end)
			if gotText != tc.outText || gotStart != tc.outStart || gotEnd != tc.outEnd {
				t.Errorf("got (%q, %d, %d), want (%q, %d, %d)", gotText, gotStart, gotEnd, tc.outText, tc.outStart, tc.outEnd)
			}
		})
	}
}

func TestShiftClickAnchor(t *testing.T) {
	const noShiftIndex = -1
	testCases := []struct {
		name       string
		start      int
		end        int
		shiftIndex int
		idx        int
		want       int
	}{
		{
			name:       "caret extends to the right",
			start:      5,
			end:        5,
			shiftIndex: noShiftIndex,
			idx:        9,
			want:       5,
		},
		{
			name:       "caret extends to the left",
			start:      5,
			end:        5,
			shiftIndex: noShiftIndex,
			idx:        2,
			want:       5,
		},
		{
			name:       "moving end at the end keeps the start anchored",
			start:      5,
			end:        10,
			shiftIndex: 10,
			idx:        2,
			want:       5,
		},
		{
			name:       "moving end at the start keeps the end anchored",
			start:      5,
			end:        10,
			shiftIndex: 5,
			idx:        20,
			want:       10,
		},
		{
			name:       "untracked selection: click to the right keeps the start",
			start:      5,
			end:        10,
			shiftIndex: noShiftIndex,
			idx:        20,
			want:       5,
		},
		{
			name:       "untracked selection: click to the left keeps the end",
			start:      5,
			end:        10,
			shiftIndex: noShiftIndex,
			idx:        2,
			want:       10,
		},
		{
			name:       "untracked selection: click inside nearer the end keeps the start",
			start:      0,
			end:        10,
			shiftIndex: noShiftIndex,
			idx:        7,
			want:       0,
		},
		{
			name:       "untracked selection: click inside nearer the start keeps the end",
			start:      0,
			end:        10,
			shiftIndex: noShiftIndex,
			idx:        3,
			want:       10,
		},
		{
			name:       "untracked selection: equidistant click keeps the start",
			start:      0,
			end:        10,
			shiftIndex: noShiftIndex,
			idx:        5,
			want:       0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := basicwidget.ShiftClickAnchor(tc.start, tc.end, tc.shiftIndex, tc.idx); got != tc.want {
				t.Errorf("ShiftClickAnchor(%d, %d, %d, %d) = %d, want %d", tc.start, tc.end, tc.shiftIndex, tc.idx, got, tc.want)
			}
		})
	}
}
