// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textstyle

import (
	"html"
	"image/color"
	"strconv"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// AppendHTML appends an HTML fragment representing text with runs' style
// overrides applied and returns the extended slice. Hard line breaks become
// br elements. Style properties without a clean HTML mapping (font family,
// language, OpenType features, and variations other than weight) are
// dropped.
func AppendHTML(dst []byte, text string, runs *Runs) []byte {
	// pos is the start of the part of text not yet emitted.
	var pos int
	for _, run := range runs.runs {
		start := min(max(run.Start, 0), len(text))
		end := min(run.End, len(text))
		if end <= start {
			continue
		}
		if pos < start {
			dst = appendEscapedHTML(dst, text[pos:start])
		}
		dst = appendStyledHTML(dst, text[start:end], run.Style)
		pos = end
	}
	if pos < len(text) {
		dst = appendEscapedHTML(dst, text[pos:])
	}
	return dst
}

// appendStyledHTML appends text escaped for HTML, wrapped in a span carrying
// style's mappable properties as inline CSS. A style with no mappable
// properties appends the escaped text alone.
func appendStyledHTML(dst []byte, text string, style Style) []byte {
	css := appendStyleCSS(nil, style)
	if len(css) == 0 {
		return appendEscapedHTML(dst, text)
	}
	dst = append(dst, `<span style="`...)
	dst = append(dst, css...)
	dst = append(dst, `">`...)
	dst = appendEscapedHTML(dst, text)
	dst = append(dst, `</span>`...)
	return dst
}

// appendStyleCSS appends style's mappable properties as CSS declarations
// separated by "; ".
func appendStyleCSS(dst []byte, style Style) []byte {
	appendDecl := func(property, value string) {
		if len(dst) > 0 {
			dst = append(dst, `; `...)
		}
		dst = append(dst, property...)
		dst = append(dst, ':')
		dst = append(dst, value...)
	}
	if italic, ok := style.Italic(); ok {
		if italic {
			appendDecl("font-style", "italic")
		} else {
			appendDecl("font-style", "normal")
		}
	}
	for _, v := range style.Variations() {
		if v.Tag == font.TagWght {
			appendDecl("font-weight", strconv.FormatFloat(float64(v.Value), 'g', -1, 32))
		}
	}
	if scale, ok := style.Scale(); ok {
		appendDecl("font-size", strconv.FormatFloat(scale, 'g', -1, 64)+"em")
	}
	underline, underlineSet := style.Underline()
	strikethrough, strikethroughSet := style.Strikethrough()
	switch {
	case underlineSet && underline && strikethroughSet && strikethrough:
		appendDecl("text-decoration", "underline line-through")
	case underlineSet && underline:
		appendDecl("text-decoration", "underline")
	case strikethroughSet && strikethrough:
		appendDecl("text-decoration", "line-through")
	case underlineSet || strikethroughSet:
		appendDecl("text-decoration", "none")
	}
	if clr, ok := style.Color(); ok {
		appendDecl("color", cssColor(clr))
	}
	if clr, ok := style.BackgroundColor(); ok {
		appendDecl("background-color", cssColor(clr))
	}
	return dst
}

// cssColor returns clr as a CSS hex color value, with an alpha component
// only when clr is not opaque.
func cssColor(clr color.Color) string {
	n := color.NRGBAModel.Convert(clr).(color.NRGBA)
	const hex = "0123456789abcdef"
	b := []byte{
		'#',
		hex[n.R>>4], hex[n.R&0xf],
		hex[n.G>>4], hex[n.G&0xf],
		hex[n.B>>4], hex[n.B&0xf],
		hex[n.A>>4], hex[n.A&0xf],
	}
	if n.A == 0xff {
		b = b[:7]
	}
	return string(b)
}

// appendEscapedHTML appends text escaped for HTML, with hard line breaks
// replaced by br elements.
func appendEscapedHTML(dst []byte, text string) []byte {
	for len(text) > 0 {
		pos, length := textutil.FirstLineBreakPositionAndLen(text)
		if length == 0 {
			dst = append(dst, html.EscapeString(text)...)
			break
		}
		dst = append(dst, html.EscapeString(text[:pos])...)
		dst = append(dst, `<br>`...)
		text = text[pos+length:]
	}
	return dst
}
