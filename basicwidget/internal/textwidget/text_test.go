// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget_test

import (
	"testing"

	"github.com/guigui-gui/guigui/basicwidget/internal/textwidget"
)

func newMultilineText(value string) *textwidget.Text {
	var txt textwidget.Text
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

// TestTextDoubleClickWordBoundaries covers the double click at the end of a
// line: the click resolves to the line's end wherever it lands in the space
// after the line, and takes the line break there instead of the word before
// it.
func TestTextDoubleClickWordBoundaries(t *testing.T) {
	cases := []struct {
		name  string
		value string
		idx   int
		start int
		end   int
	}{
		// "foo bar\nbaz qux": words "foo" [0,3) "bar" [4,7) "baz" [8,11)
		// "qux" [12,15); the line break is at byte 7.
		{
			name:  "inside a word",
			value: "foo bar\nbaz qux",
			idx:   5,
			start: 4,
			end:   7,
		},
		{
			name:  "at a word start",
			value: "foo bar\nbaz qux",
			idx:   4,
			start: 4,
			end:   7,
		},
		{
			name:  "at a line end",
			value: "foo bar\nbaz qux",
			idx:   7,
			start: 7,
			end:   8,
		},
		{
			name:  "at a line start",
			value: "foo bar\nbaz qux",
			idx:   8,
			start: 8,
			end:   11,
		},
		{
			name:  "at the end of the last line",
			value: "foo bar\nbaz qux",
			idx:   15,
			start: 12,
			end:   15,
		},
		// "a\r\n\nb": the CRLF is [1,3), the empty line's LF is [3,4).
		{
			name:  "at a CRLF line end",
			value: "a\r\n\nb",
			idx:   1,
			start: 1,
			end:   3,
		},
		{
			name:  "on an empty line",
			value: "a\r\n\nb",
			idx:   3,
			start: 3,
			end:   4,
		},
		{
			name:  "on the last line",
			value: "a\r\n\nb",
			idx:   4,
			start: 4,
			end:   5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			txt := newMultilineText(tc.value)
			start, end := txt.FindWordBoundariesForDoubleClick(tc.idx)
			if start != tc.start || end != tc.end {
				t.Errorf("FindWordBoundariesForDoubleClick(%d) = (%d, %d), want (%d, %d)", tc.idx, start, end, tc.start, tc.end)
			}
		})
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
