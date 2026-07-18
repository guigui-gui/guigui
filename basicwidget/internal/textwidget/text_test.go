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
