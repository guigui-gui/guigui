// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget_test

import (
	"testing"

	"github.com/guigui-gui/guigui/basicwidget/internal/textwidget"
)

func TestTextHotspotRangesFollowEdits(t *testing.T) {
	tests := []struct {
		name string
		edit func(txt *textwidget.Text)
		want []textwidget.TextRange
	}{
		{
			name: "insertion inside the range extends it",
			edit: func(txt *textwidget.Text) {
				txt.ReplaceTextAt("x", 8, 8)
			},
			want: []textwidget.TextRange{
				{StartInBytes: 6, EndInBytes: 12},
			},
		},
		{
			name: "insertion before the range shifts it",
			edit: func(txt *textwidget.Text) {
				txt.ReplaceTextAt("xy", 0, 0)
			},
			want: []textwidget.TextRange{
				{StartInBytes: 8, EndInBytes: 13},
			},
		},
		{
			name: "deletion overlapping the range's head truncates it",
			edit: func(txt *textwidget.Text) {
				txt.ReplaceTextAt("", 5, 8)
			},
			want: []textwidget.TextRange{
				{StartInBytes: 5, EndInBytes: 8},
			},
		},
		{
			name: "replacing the whole value removes the range",
			edit: func(txt *textwidget.Text) {
				txt.ForceSetValue("goodbye")
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var txt textwidget.Text
			txt.SetEditable(true)
			txt.ForceSetValue("hello world")
			txt.SetHotspotRanges([]textwidget.TextRange{
				{StartInBytes: 6, EndInBytes: 11},
			})
			tt.edit(&txt)
			if got := txt.AppendHotspotRanges(nil); !equalTextRanges(got, tt.want) {
				t.Errorf("got: %+v, want: %+v", got, tt.want)
			}
		})
	}
}

func equalTextRanges(a, b []textwidget.TextRange) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
