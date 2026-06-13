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
	testCases := []struct {
		name      string
		start     int
		end       int
		shiftSide basicwidget.SelectionSide
		idx       int
		want      int
	}{
		{
			name:      "caret extends to the right",
			start:     5,
			end:       5,
			shiftSide: basicwidget.SelectionSideNone,
			idx:       9,
			want:      5,
		},
		{
			name:      "caret extends to the left",
			start:     5,
			end:       5,
			shiftSide: basicwidget.SelectionSideNone,
			idx:       2,
			want:      5,
		},
		{
			name:      "moving end at the end keeps the start anchored",
			start:     5,
			end:       10,
			shiftSide: basicwidget.SelectionSideEnd,
			idx:       2,
			want:      5,
		},
		{
			name:      "moving end at the start keeps the end anchored",
			start:     5,
			end:       10,
			shiftSide: basicwidget.SelectionSideStart,
			idx:       20,
			want:      10,
		},
		{
			name:      "untracked selection: click to the right keeps the start",
			start:     5,
			end:       10,
			shiftSide: basicwidget.SelectionSideNone,
			idx:       20,
			want:      5,
		},
		{
			name:      "untracked selection: click to the left keeps the end",
			start:     5,
			end:       10,
			shiftSide: basicwidget.SelectionSideNone,
			idx:       2,
			want:      10,
		},
		{
			name:      "untracked selection: click inside nearer the end keeps the start",
			start:     0,
			end:       10,
			shiftSide: basicwidget.SelectionSideNone,
			idx:       7,
			want:      0,
		},
		{
			name:      "untracked selection: click inside nearer the start keeps the end",
			start:     0,
			end:       10,
			shiftSide: basicwidget.SelectionSideNone,
			idx:       3,
			want:      10,
		},
		{
			name:      "untracked selection: equidistant click keeps the start",
			start:     0,
			end:       10,
			shiftSide: basicwidget.SelectionSideNone,
			idx:       5,
			want:      0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := basicwidget.ShiftClickAnchor(tc.start, tc.end, tc.shiftSide, tc.idx); got != tc.want {
				t.Errorf("ShiftClickAnchor(%d, %d, %v, %d) = %d, want %d", tc.start, tc.end, tc.shiftSide, tc.idx, got, tc.want)
			}
		})
	}
}

func newMultilineText(value string) *basicwidget.Text {
	var txt basicwidget.Text
	txt.SetMultiline(true)
	txt.SetValue(value)
	return &txt
}

func TestTextWordNavigation(t *testing.T) {
	// "foo bar\nbaz qux": words "foo" [0,3) "bar" [4,7) "baz" [8,11)
	// "qux" [12,15); the logical line break is at byte 7.
	txt := newMultilineText("foo bar\nbaz qux")

	nextCases := []struct{ from, want int }{
		{0, 3}, {1, 3}, {3, 7}, {7, 11}, {11, 15}, {15, 15},
	}
	for _, tc := range nextCases {
		if got := txt.NextWordEnd(tc.from); got != tc.want {
			t.Errorf("NextWordEnd(%d) = %d, want %d", tc.from, got, tc.want)
		}
	}

	prevCases := []struct{ from, want int }{
		{15, 12}, {12, 8}, {8, 4}, {4, 0}, {0, 0},
	}
	for _, tc := range prevCases {
		if got := txt.PrevWordStart(tc.from); got != tc.want {
			t.Errorf("PrevWordStart(%d) = %d, want %d", tc.from, got, tc.want)
		}
	}

	// nextWordStart lands on the beginning of the next word (the Windows
	// convention), crossing the line break at byte 7.
	nextStartCases := []struct{ from, want int }{
		{0, 4}, {1, 4}, {3, 4}, {4, 8}, {7, 8}, {8, 12}, {12, 15}, {15, 15},
	}
	for _, tc := range nextStartCases {
		if got := txt.NextWordStart(tc.from); got != tc.want {
			t.Errorf("NextWordStart(%d) = %d, want %d", tc.from, got, tc.want)
		}
	}
}

func TestTextParagraphNavigation(t *testing.T) {
	// "foo bar\nbaz qux": line 0 [0,8) (content [0,7)), line 1 [8,15).
	txt := newMultilineText("foo bar\nbaz qux")

	startCases := []struct{ from, want int }{
		{5, 0}, {7, 0}, {0, 0}, {10, 8}, {8, 0}, {15, 8},
	}
	for _, tc := range startCases {
		if got := txt.ParagraphStart(tc.from); got != tc.want {
			t.Errorf("ParagraphStart(%d) = %d, want %d", tc.from, got, tc.want)
		}
	}

	endCases := []struct{ from, want int }{
		{5, 7}, {0, 7}, {7, 15}, {10, 15}, {15, 15},
	}
	for _, tc := range endCases {
		if got := txt.ParagraphEnd(tc.from); got != tc.want {
			t.Errorf("ParagraphEnd(%d) = %d, want %d", tc.from, got, tc.want)
		}
	}
}
