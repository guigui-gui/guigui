// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package textutil

import (
	"fmt"
	"image"
	"iter"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/rivo/uniseg"
)

func nextIndentPosition(position float64, indentWidth float64) float64 {
	if indentWidth == 0 {
		return position
	}
	// TODO: The calculation should consider the center and right alignment (#162).
	return float64(int(position/indentWidth)+1) * indentWidth
}

func advance(str string, face text.Face, tabWidth float64, keepTailingSpace bool) float64 {
	var hasLineBreak bool
	if !keepTailingSpace {
		str = strings.TrimRightFunc(str, unicode.IsSpace)
	} else if l := tailingLineBreakLen(str); l > 0 {
		str = str[:len(str)-l]
		hasLineBreak = true
	}
	if tabWidth == 0 {
		return text.Advance(str, face)
	}
	var width float64
	for {
		head, tail, ok := strings.Cut(str, "\t")
		width += text.Advance(head, face)
		if !ok {
			break
		}
		width = nextIndentPosition(width, tabWidth)
		str = tail
	}
	if hasLineBreak {
		// Always add the advance of a space for the line break for a consistent behavior.
		width += text.Advance(" ", face)
	}
	return width
}

type Options struct {
	AutoWrap         bool
	Face             text.Face
	LineHeight       float64
	HorizontalAlign  HorizontalAlign
	VerticalAlign    VerticalAlign
	TabWidth         float64
	KeepTailingSpace bool
}

type HorizontalAlign int

const (
	HorizontalAlignStart HorizontalAlign = iota
	HorizontalAlignCenter
	HorizontalAlignEnd
	HorizontalAlignLeft
	HorizontalAlignRight
)

type VerticalAlign int

const (
	VerticalAlignTop VerticalAlign = iota
	VerticalAlignMiddle
	VerticalAlignBottom
)

func visibleCulsters(str string, face text.Face) []text.Glyph {
	return text.AppendGlyphs(nil, str, face, nil)
}

type line struct {
	pos int
	str string
}

func lines(width int, str string, autoWrap bool, advance func(str string) float64) iter.Seq[line] {
	return func(yield func(line) bool) {
		origStr := str

		if !autoWrap {
			var pos int
			for pos < len(str) {
				p, l := FirstLineBreakPositionAndLen(str[pos:])
				if p == -1 {
					if !yield(line{
						pos: pos,
						str: str[pos:],
					}) {
						return
					}
					break
				}
				if !yield(line{
					pos: pos,
					str: str[pos : pos+p+l],
				}) {
					return
				}
				pos += p + l
			}
		} else {
			var lineStart int
			var lineEnd int
			var pos int
			state := -1
			for len(str) > 0 {
				segment, nextStr, mustBreak, nextState := uniseg.FirstLineSegmentInString(str, state)
				if lineEnd-lineStart > 0 {
					l := origStr[lineStart : lineEnd+len(segment)]
					// TODO: Consider a line alignment and/or editable/selectable states when calculating the width.
					if advance(l[:len(l)-tailingLineBreakLen(l)]) > float64(width) {
						if !yield(line{
							pos: pos,
							str: origStr[lineStart:lineEnd],
						}) {
							return
						}
						pos += lineEnd - lineStart
						lineStart = lineEnd
					}
				}
				lineEnd += len(segment)
				if mustBreak {
					if !yield(line{
						pos: pos,
						str: origStr[lineStart:lineEnd],
					}) {
						return
					}
					pos += lineEnd - lineStart
					lineStart = lineEnd
				}
				str = nextStr
				state = nextState
			}

			if lineEnd-lineStart > 0 {
				if !yield(line{
					pos: pos,
					str: origStr[lineStart:lineEnd],
				}) {
					return
				}
				pos += lineEnd - lineStart
				lineStart = lineEnd
			}
		}

		// If the string ends with a line break, or an empty line, add an extra empty line.
		if tailingLineBreakLen(origStr) > 0 || origStr == "" {
			if !yield(line{
				pos: len(origStr),
			}) {
				return
			}
		}
	}
}

func oneLineLeft(width int, line string, face text.Face, hAlign HorizontalAlign, tabWidth float64, keepTailingSpace bool) float64 {
	w := advance(line[:len(line)-tailingLineBreakLen(line)], face, tabWidth, keepTailingSpace)
	switch hAlign {
	case HorizontalAlignStart, HorizontalAlignLeft:
		// For RTL languages, HorizontalAlignStart should be the same as HorizontalAlignRight.
		return 0
	case HorizontalAlignCenter:
		return (float64(width) - w) / 2
	case HorizontalAlignEnd, HorizontalAlignRight:
		// For RTL languages, HorizontalAlignEnd should be the same as HorizontalAlignLeft.
		return float64(width) - w
	default:
		panic(fmt.Sprintf("textutil: invalid HorizontalAlign: %d", hAlign))
	}
}

func TextIndexFromPosition(width int, position image.Point, str string, options *Options) int {
	// Determine the line first.
	padding := textPadding(options.Face, options.LineHeight)
	n := int((float64(position.Y) + padding) / options.LineHeight)

	var pos int
	var line string
	var lineIndex int
	for l := range lines(width, str, options.AutoWrap, func(str string) float64 {
		return advance(str, options.Face, options.TabWidth, options.KeepTailingSpace)
	}) {
		line = l.str
		pos = l.pos
		if lineIndex >= n {
			break
		}
		lineIndex++
	}

	// Deterine the line index.
	left := oneLineLeft(width, line, options.Face, options.HorizontalAlign, options.TabWidth, options.KeepTailingSpace)
	var prevA float64
	var clusterFound bool
	for _, c := range visibleCulsters(line, options.Face) {
		a := advance(line[:c.EndIndexInBytes], options.Face, options.TabWidth, true)
		if (float64(position.X) - left) < (prevA + (a-prevA)/2) {
			pos += c.StartIndexInBytes
			clusterFound = true
			break
		}
		prevA = a
	}
	if !clusterFound {
		pos += len(line)
		pos -= tailingLineBreakLen(line)
	}

	return pos
}

type TextPosition struct {
	X      float64
	Top    float64
	Bottom float64
}

func TextPositionFromIndex(width int, str string, index int, options *Options) (position0, position1 TextPosition, count int) {
	if index < 0 || index > len(str) {
		return TextPosition{}, TextPosition{}, 0
	}
	return textPositionFromIndex(width, str, lines(width, str, options.AutoWrap, func(str string) float64 {
		return advance(str, options.Face, options.TabWidth, options.KeepTailingSpace)
	}), index, options)
}

func textPositionFromIndex(width int, str string, lines iter.Seq[line], index int, options *Options) (position0, position1 TextPosition, count int) {
	if index < 0 || index > len(str) {
		return TextPosition{}, TextPosition{}, 0
	}

	var y, y0, y1 float64
	var indexInLine0, indexInLine1 int
	var line0, line1 string
	var found0, found1 bool
	for l := range lines {
		// When auto wrap is on or the string ends with a line break, there can be two positions:
		// one in the tail of the previous line and one in the head of the next line.
		if index == l.pos+len(l.str) {
			found0 = true
			line0 = l.str
			indexInLine0 = index - l.pos
			y0 = y
		} else if l.pos <= index && index < l.pos+len(l.str) {
			found1 = true
			line1 = l.str
			indexInLine1 = index - l.pos
			y1 = y
			break
		}
		y += options.LineHeight
	}

	if !found0 && !found1 {
		return TextPosition{}, TextPosition{}, 0
	}

	paddingY := textPadding(options.Face, options.LineHeight)

	var pos0, pos1 TextPosition
	if found0 {
		x0 := oneLineLeft(width, line0, options.Face, options.HorizontalAlign, options.TabWidth, options.KeepTailingSpace)
		x0 += advance(line0[:indexInLine0], options.Face, options.TabWidth, true)
		pos0 = TextPosition{
			X:      x0,
			Top:    y0 + paddingY,
			Bottom: y0 + options.LineHeight - paddingY,
		}
	}
	if found1 {
		x1 := oneLineLeft(width, line1, options.Face, options.HorizontalAlign, options.TabWidth, options.KeepTailingSpace)
		x1 += advance(line1[:indexInLine1], options.Face, options.TabWidth, true)
		pos1 = TextPosition{
			X:      x1,
			Top:    y1 + paddingY,
			Bottom: y1 + options.LineHeight - paddingY,
		}
	}
	if found0 && !found1 {
		return pos0, TextPosition{}, 1
	}
	if found1 && !found0 {
		return pos1, TextPosition{}, 1
	}
	return pos0, pos1, 2
}

func FirstLineBreakPositionAndLen(str string) (pos, length int) {
	for i, r := range str {
		if r == 0x000a || r == 0x000b || r == 0x000c {
			return i, 1
		}
		if r == 0x0085 {
			return i, 2
		}
		if r == 0x2028 || r == 0x2029 {
			return i, 3
		}
		if r == 0x000d {
			// \r\n
			if len(str[i:]) > 0 && str[i+1] == 0x000a {
				return i, 2
			}
			return i, 1
		}
	}
	return -1, 0
}

func tailingLineBreakLen(str string) int {
	// uniseg.HasTrailingLineBreakInString is slow and doesn't check \r\n.
	// Hard-code the check here.
	// See also: https://en.wikipedia.org/wiki/Newline#Unicode
	if r, s := utf8.DecodeLastRuneInString(str); s > 0 {
		if r == 0x000b || r == 0x000c || r == 0x000d || r == 0x0085 || r == 0x2028 || r == 0x2029 {
			return s
		}
		if r == 0x000a {
			// \r\n
			if r, s := utf8.DecodeLastRuneInString(str[:len(str)-s]); s > 0 && r == 0x000d {
				return 2
			}
			return 1
		}
	}
	return 0
}

func trimTailingLineBreak(str string) string {
	for {
		c := tailingLineBreakLen(str)
		if c == 0 {
			break
		}
		str = str[:len(str)-c]
	}
	return str
}

func lineCount(width int, str string, autoWrap bool, face text.Face, tabWidth float64, keepTailingSpace bool) int {
	var count int
	for range lines(width, str, autoWrap, func(str string) float64 {
		return advance(str, face, tabWidth, keepTailingSpace)
	}) {
		count++
	}
	return count
}

func Measure(width int, str string, autoWrap bool, face text.Face, lineHeight float64, tabWidth float64, keepTailingSpace bool) (float64, float64) {
	var maxWidth, height float64
	for l := range lines(width, str, autoWrap, func(str string) float64 {
		return advance(str, face, tabWidth, keepTailingSpace)
	}) {
		line := l.str
		if !keepTailingSpace {
			line = trimTailingLineBreak(line)
		}
		maxWidth = max(maxWidth, advance(line, face, tabWidth, keepTailingSpace))
		// The text is already shifted by (lineHeight - (m.HAscent + m.Descent)) / 2.
		// Thus, just counting the line number is enough.
		height += lineHeight
	}
	return maxWidth, height
}

func textPadding(face text.Face, lineHeight float64) float64 {
	m := face.Metrics()
	padding := (lineHeight - (m.HAscent + m.HDescent)) / 2
	return padding
}

func textPositionYOffset(size image.Point, str string, options *Options) float64 {
	c := lineCount(size.X, str, options.AutoWrap, options.Face, options.TabWidth, options.KeepTailingSpace)
	textHeight := options.LineHeight * float64(c)
	yOffset := textPadding(options.Face, options.LineHeight)
	switch options.VerticalAlign {
	case VerticalAlignTop:
	case VerticalAlignMiddle:
		yOffset += (float64(size.Y) - textHeight) / 2
	case VerticalAlignBottom:
		yOffset += float64(size.Y) - textHeight
	}
	return yOffset
}
