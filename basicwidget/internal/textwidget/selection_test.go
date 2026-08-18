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

func TestSetSelectionSnapsToRuneBoundaries(t *testing.T) {
	testCases := []struct {
		name      string
		value     string
		start     int
		end       int
		wantStart int
		wantEnd   int
	}{
		// "あい" is two 3-byte runes: "あ" [0, 3) and "い" [3, 6).
		{
			name:      "caret on a boundary",
			value:     "あい",
			start:     3,
			end:       3,
			wantStart: 3,
			wantEnd:   3,
		},
		{
			name:      "caret inside a rune moves to its start",
			value:     "あい",
			start:     2,
			end:       2,
			wantStart: 0,
			wantEnd:   0,
		},
		{
			name:      "range inside a rune expands to it",
			value:     "あい",
			start:     1,
			end:       2,
			wantStart: 0,
			wantEnd:   3,
		},
		{
			name:      "range expands over both ends",
			value:     "あい",
			start:     1,
			end:       4,
			wantStart: 0,
			wantEnd:   6,
		},
		{
			name:      "range on boundaries is kept",
			value:     "あい",
			start:     0,
			end:       3,
			wantStart: 0,
			wantEnd:   3,
		},
		{
			name:      "out-of-range offsets are clamped",
			value:     "あい",
			start:     -5,
			end:       100,
			wantStart: 0,
			wantEnd:   6,
		},
		{
			name:      "text ending inside a rune",
			value:     "あ"[:2],
			start:     1,
			end:       1,
			wantStart: 0,
			wantEnd:   0,
		},
		{
			name:      "ASCII neighborhood in invalid UTF-8 is untouched",
			value:     "caf\xe9 latte",
			start:     4,
			end:       5,
			wantStart: 4,
			wantEnd:   5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var txt textwidget.Text
			txt.SetEditable(true)
			txt.ForceSetValue(tc.value)
			txt.SetSelection(tc.start, tc.end)
			start, end := txt.Selection()
			if start != tc.wantStart || end != tc.wantEnd {
				t.Errorf("Selection() = (%d, %d), want (%d, %d)", start, end, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

func TestEditAtSnappedSelectionKeepsValidUTF8(t *testing.T) {
	testCases := []struct {
		name  string
		start int
		end   int
		edit  func(txt *textwidget.Text)
		want  string
	}{
		{
			name:  "insertion at a caret inside a rune",
			start: 1,
			end:   1,
			edit: func(txt *textwidget.Text) {
				txt.ReplaceValueAtSelection("x")
			},
			want: "xあい",
		},
		{
			name:  "insertion over a range inside a rune",
			start: 1,
			end:   2,
			edit: func(txt *textwidget.Text) {
				txt.ReplaceValueAtSelection("x")
			},
			want: "xい",
		},
		{
			name:  "deletion of a range inside a rune",
			start: 0,
			end:   1,
			edit: func(txt *textwidget.Text) {
				// Backspace and Delete remove the selected range.
				start, end := txt.Selection()
				txt.ReplaceTextAt("", start, end, nil)
			},
			want: "い",
		},
		{
			name:  "IME commit over a range inside a rune",
			start: 2,
			end:   4,
			edit: func(txt *textwidget.Text) {
				txt.CommitTextByIME("z")
			},
			want: "z",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var txt textwidget.Text
			txt.SetEditable(true)
			txt.ForceSetValue("あい")
			txt.SetSelection(tc.start, tc.end)
			tc.edit(&txt)
			if got := txt.Value(); got != tc.want {
				t.Errorf("Value() = %q, want %q", got, tc.want)
			}
		})
	}
}
