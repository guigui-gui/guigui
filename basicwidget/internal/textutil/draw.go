// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package textutil

import (
	"image"
	"image/color"
	"math"
	"slices"
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

type DrawOptions struct {
	Style

	TextColor color.Color

	DrawSelection  bool
	SelectionStart int
	SelectionEnd   int
	SelectionColor color.Color

	DrawComposition          bool
	CompositionStart         int
	CompositionEnd           int
	CompositionActiveStart   int
	CompositionActiveEnd     int
	InactiveCompositionColor color.Color
	ActiveCompositionColor   color.Color
	CompositionBorderWidth   float32

	// VisibleBounds restricts drawing to lines and glyphs that intersect this
	// rectangle. Lines fully above or below are skipped without shaping, and
	// glyphs whose drawn rectangle falls entirely outside are not submitted to
	// [(*ebiten.Image).DrawImage]. An empty rectangle draws nothing.
	VisibleBounds image.Rectangle
}

var theVisualLinesBuffer []visualLine

// appendVisualLinesFromCachedStarts reproduces visualLines for str by reading
// each logical line's wrap points from the layout cache (cachedVisualLineStarts)
// instead of shaping. str is split at hard breaks (the break stays with the
// preceding line), including the trailing empty line after a final break. ok is
// false (dst left unchanged) when a line's starts are unavailable, so the caller
// falls back to shaping.
func appendVisualLinesFromCachedStarts(dst []visualLine, str string, width int, wrapMode WrapMode, face font.Face, tabWidth float64, keepTailingSpace bool) (lines []visualLine, ok bool) {
	base := len(dst)
	var pos int
	for {
		p, l := FirstLineBreakPositionAndLen(str[pos:])
		last := p == -1
		lineEnd := len(str)
		if !last {
			lineEnd = pos + p + l
		}
		line := str[pos:lineEnd]
		s, sok := cachedVisualLineStarts(width, line, wrapMode, face, tabWidth, keepTailingSpace)
		if !sok {
			return dst[:base], false
		}
		for i := range s {
			rs := pos + s[i]
			re := lineEnd
			if i+1 < len(s) {
				re = pos + s[i+1]
			}
			dst = append(dst, visualLine{pos: rs, str: str[rs:re]})
		}
		if last {
			break
		}
		pos = lineEnd
	}
	return dst, true
}

func Draw(bounds image.Rectangle, dst *ebiten.Image, str string, options *DrawOptions) {
	clip := bounds.Intersect(options.VisibleBounds)
	if clip.Empty() {
		return
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(bounds.Min.X), float64(bounds.Min.Y))
	op.ColorScale.ScaleWithColor(options.TextColor)
	if dst.Bounds() != clip {
		dst = dst.RecyclableSubImage(clip)
		defer dst.Recycle()
	}

	op.LineSpacing = options.LineHeight

	yOffset := textPositionYOffset(bounds.Size(), str, &options.Style)
	op.GeoM.Translate(0, yOffset)

	theVisualLinesBuffer = theVisualLinesBuffer[:0]
	var built bool
	if options.WrapMode != WrapModeNone {
		if vls, ok := appendVisualLinesFromCachedStarts(theVisualLinesBuffer, str, bounds.Dx(), options.WrapMode, options.Face, options.TabWidth, options.KeepTailingSpace); ok {
			theVisualLinesBuffer = vls
			built = true
		}
	}
	if !built {
		theVisualLinesBuffer = theVisualLinesBuffer[:0]
		for vl := range visualLines(bounds.Dx(), str, options.WrapMode, func(str string, indexInBytes int) float64 {
			return advance(str, indexInBytes, options.Face.TextFace(), options.TabWidth, options.KeepTailingSpace)
		}) {
			theVisualLinesBuffer = append(theVisualLinesBuffer, vl)
		}
	}

	for _, vl := range theVisualLinesBuffer {
		y := op.GeoM.Element(1, 2)
		if int(math.Ceil(y+options.LineHeight)) < clip.Min.Y {
			// Advance to the next line so the loop terminates; the bottom-of-body
			// translation is skipped by [continue].
			op.GeoM.Translate(0, options.LineHeight)
			continue
		}
		if int(math.Floor(y)) >= clip.Max.Y {
			break
		}

		start := vl.pos
		end := vl.pos + len(vl.str)

		if options.DrawSelection {
			if start <= options.SelectionEnd && end >= options.SelectionStart {
				start := max(start, options.SelectionStart)
				end := min(end, options.SelectionEnd)
				if start != end {
					posStart0, posStart1, countStart := textPositionFromIndexInVisualLines(bounds.Dx(), slices.Values(theVisualLinesBuffer), start, &options.Style)
					posEnd0, _, countEnd := textPositionFromIndexInVisualLines(bounds.Dx(), slices.Values(theVisualLinesBuffer), end, &options.Style)
					if countStart > 0 && countEnd > 0 {
						posStart := posStart0
						if countStart == 2 {
							posStart = posStart1
						}
						posEnd := posEnd0
						x := float32(posStart.X) + float32(bounds.Min.X)
						y := float32(posStart.Top) + float32(bounds.Min.Y)
						width := float32(posEnd.X - posStart.X)
						height := float32(posStart.Bottom - posStart.Top)
						vector.FillRect(dst, x, y, width, height, options.SelectionColor, false)
					}
				}
			}
		}

		if options.DrawComposition {
			if start <= options.CompositionEnd && end >= options.CompositionStart {
				start := max(start, options.CompositionStart)
				end := min(end, options.CompositionEnd)
				if start != end {
					posStart0, posStart1, countStart := textPositionFromIndexInVisualLines(bounds.Dx(), slices.Values(theVisualLinesBuffer), start, &options.Style)
					posEnd0, _, countEnd := textPositionFromIndexInVisualLines(bounds.Dx(), slices.Values(theVisualLinesBuffer), end, &options.Style)
					if countStart > 0 && countEnd > 0 {
						posStart := posStart0
						if countStart == 2 {
							posStart = posStart1
						}
						posEnd := posEnd0
						x := float32(posStart.X) + float32(bounds.Min.X)
						y := float32(posStart.Bottom) + float32(bounds.Min.Y) - options.CompositionBorderWidth
						w := float32(posEnd.X - posStart.X)
						h := options.CompositionBorderWidth
						vector.FillRect(dst, x, y, w, h, options.InactiveCompositionColor, false)
					}
				}
			}
			if start <= options.CompositionActiveEnd && end >= options.CompositionActiveStart {
				start := max(start, options.CompositionActiveStart)
				end := min(end, options.CompositionActiveEnd)
				if start != end {
					posStart0, posStart1, countStart := textPositionFromIndexInVisualLines(bounds.Dx(), slices.Values(theVisualLinesBuffer), start, &options.Style)
					posEnd0, _, countEnd := textPositionFromIndexInVisualLines(bounds.Dx(), slices.Values(theVisualLinesBuffer), end, &options.Style)
					if countStart > 0 && countEnd > 0 {
						posStart := posStart0
						if countStart == 2 {
							posStart = posStart1
						}
						posEnd := posEnd0
						x := float32(posStart.X) + float32(bounds.Min.X)
						y := float32(posStart.Bottom) + float32(bounds.Min.Y) - options.CompositionBorderWidth
						w := float32(posEnd.X - posStart.X)
						h := options.CompositionBorderWidth
						vector.FillRect(dst, x, y, w, h, options.ActiveCompositionColor, false)
					}
				}
			}
		}

		// Draw the text.
		vlStr := vl.str
		origGeoM := op.GeoM
		if !options.KeepTailingSpace {
			vlStr = strings.TrimRightFunc(vlStr, unicode.IsSpace)
		}
		if options.EllipsisString != "" && advance(vlStr, len(vlStr), options.Face.TextFace(), options.TabWidth, options.KeepTailingSpace) > float64(bounds.Dx()) {
			vlStr = truncateWithEllipsis(vlStr, options.EllipsisString, float64(bounds.Dx()), options.Face.TextFace(), options.TabWidth)
		}
		// Ebitengine's text.Draw does not handle tab characters, so lines
		// containing tabs must use manual alignment via oneLineLeft and GeoM.
		if !strings.Contains(vlStr, "\t") {
			// Use Ebitengine's PrimaryAlign for horizontal alignment so that the
			// text origin accounts for the alignment offset. This ensures that each
			// glyph's subpixel position is determined relative to the aligned origin,
			// producing consistent rendering when the text content changes
			// (e.g., right-aligned text gaining/losing characters).
			switch options.HorizontalAlign {
			case HorizontalAlignCenter:
				op.PrimaryAlign = text.AlignCenter
				op.GeoM.Translate(float64(bounds.Dx())/2, 0)
			case HorizontalAlignEnd, HorizontalAlignRight:
				op.PrimaryAlign = text.AlignEnd
				op.GeoM.Translate(float64(bounds.Dx()), 0)
			default:
				op.PrimaryAlign = text.AlignStart
			}
			text.Draw(dst, vlStr, options.Face.TextFace(), op)
		} else {
			op.PrimaryAlign = text.AlignStart
			x := oneLineLeft(bounds.Dx(), vlStr, options.Face.TextFace(), options.HorizontalAlign, options.TabWidth, options.KeepTailingSpace)
			op.GeoM.Translate(x, 0)
			origVlStr := vlStr
			var origX float64
			var pos int
			for {
				head, tail, ok := strings.Cut(vlStr, "\t")
				text.Draw(dst, head, options.Face.TextFace(), op)
				if !ok {
					break
				}
				tabIdx := pos + len(head)
				x := origX + text.AdvanceAt(origVlStr, tabIdx, options.Face.TextFace()) - text.AdvanceAt(origVlStr, pos, options.Face.TextFace())
				nextX := nextIndentPosition(x, options.TabWidth)
				op.GeoM.Translate(nextX-origX, 0)
				origX = nextX
				pos = tabIdx + 1
				vlStr = tail
			}
		}
		op.GeoM = origGeoM
		op.GeoM.Translate(0, options.LineHeight)
	}
}
