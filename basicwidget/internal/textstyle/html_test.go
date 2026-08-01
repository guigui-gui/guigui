// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textstyle_test

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
)

func TestAppendHTML(t *testing.T) {
	red := color.RGBA{R: 0xff, A: 0xff}
	translucent := color.RGBA{R: 0x80, A: 0x80}

	tests := []struct {
		name string
		text string
		ops  func(runs *textstyle.Runs)
		want string
	}{
		{
			name: "no overrides",
			text: "plain text",
			ops:  func(runs *textstyle.Runs) {},
			want: "plain text",
		},
		{
			name: "text is escaped",
			text: "a < b & \"c\"",
			ops:  func(runs *textstyle.Runs) {},
			want: "a &lt; b &amp; &#34;c&#34;",
		},
		{
			name: "line breaks become br",
			text: "one\ntwo\r\nthree",
			ops:  func(runs *textstyle.Runs) {},
			want: "one<br>two<br>three",
		},
		{
			name: "color",
			text: "abcdef",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(1, 4, red)
			},
			want: `a<span style="color:#ff0000">bcd</span>ef`,
		},
		{
			name: "translucent color",
			text: "ab",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 2, translucent)
			},
			want: `<span style="color:#ff000080">ab</span>`,
		},
		{
			name: "background color",
			text: "ab",
			ops: func(runs *textstyle.Runs) {
				runs.SetBackgroundColor(0, 2, red)
			},
			want: `<span style="background-color:#ff0000">ab</span>`,
		},
		{
			name: "weight from the wght variation",
			text: "bold",
			ops: func(runs *textstyle.Runs) {
				runs.SetVariation(0, 4, font.TagWght, float32(text.WeightBold))
			},
			want: `<span style="font-weight:700">bold</span>`,
		},
		{
			name: "italic",
			text: "ab",
			ops: func(runs *textstyle.Runs) {
				runs.SetItalic(0, 2, true)
			},
			want: `<span style="font-style:italic">ab</span>`,
		},
		{
			name: "explicit non-italic",
			text: "ab",
			ops: func(runs *textstyle.Runs) {
				runs.SetItalic(0, 2, false)
			},
			want: `<span style="font-style:normal">ab</span>`,
		},
		{
			name: "underline and strikethrough combine",
			text: "abcd",
			ops: func(runs *textstyle.Runs) {
				runs.SetUnderline(0, 4, true)
				runs.SetStrikethrough(2, 4, true)
			},
			want: `<span style="text-decoration:underline">ab</span>` +
				`<span style="text-decoration:underline line-through">cd</span>`,
		},
		{
			name: "explicit false decorations map to none",
			text: "ab",
			ops: func(runs *textstyle.Runs) {
				runs.SetUnderline(0, 2, false)
			},
			want: `<span style="text-decoration:none">ab</span>`,
		},
		{
			name: "scale becomes font-size in em",
			text: "big",
			ops: func(runs *textstyle.Runs) {
				runs.SetScale(0, 3, 1.5)
			},
			want: `<span style="font-size:1.5em">big</span>`,
		},
		{
			name: "multiple properties in one span",
			text: "ab",
			ops: func(runs *textstyle.Runs) {
				runs.SetItalic(0, 2, true)
				runs.SetColor(0, 2, red)
				runs.SetUnderline(0, 2, true)
			},
			want: `<span style="font-style:italic; text-decoration:underline; color:#ff0000">ab</span>`,
		},
		{
			name: "unmappable properties are dropped",
			text: "abcd",
			ops: func(runs *textstyle.Runs) {
				runs.SetLang(0, 2, language.Japanese)
				runs.SetFeature(0, 2, font.TagLiga, 0)
				runs.SetColor(2, 4, red)
			},
			want: `ab<span style="color:#ff0000">cd</span>`,
		},
		{
			name: "styled range spanning a line break",
			text: "one\ntwo",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 7, red)
			},
			want: `<span style="color:#ff0000">one<br>two</span>`,
		},
		{
			name: "run extending past the text is clipped",
			text: "ab",
			ops: func(runs *textstyle.Runs) {
				runs.SetColor(0, 100, red)
			},
			want: `<span style="color:#ff0000">ab</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var runs textstyle.Runs
			tt.ops(&runs)
			if got := string(textstyle.AppendHTML(nil, tt.text, &runs)); got != tt.want {
				t.Errorf("got: %q, want: %q", got, tt.want)
			}
		})
	}
}
