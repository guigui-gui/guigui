// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package textutil

import (
	"image"
	"image/color"
	"math"
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type DrawOptions struct {
	Options

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
}

var theCachedLines []line

func Draw(bounds image.Rectangle, dst *ebiten.Image, str string, options *DrawOptions) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(bounds.Min.X), float64(bounds.Min.Y))
	op.ColorScale.ScaleWithColor(options.TextColor)
	if dst.Bounds() != bounds {
		dst = dst.RecyclableSubImage(bounds)
		defer dst.Recycle()
	}

	op.LineSpacing = options.LineHeight

	yOffset := textPositionYOffset(bounds.Size(), str, &options.Options)
	op.GeoM.Translate(0, yOffset)

	theCachedLines = theCachedLines[:0]
	for line := range lines(bounds.Dx(), str, options.AutoWrap, func(str string) float64 {
		return advance(str, options.Face, options.TabWidth, options.KeepTailingSpace)
	}) {
		theCachedLines = append(theCachedLines, line)
	}

	for _, line := range theCachedLines {
		y := op.GeoM.Element(1, 2)
		if int(math.Ceil(y+options.LineHeight)) < bounds.Min.Y {
			continue
		}
		if int(math.Floor(y)) >= bounds.Max.Y {
			break
		}

		start := line.pos
		end := line.pos + len(line.str)

		if options.DrawSelection {
			if start <= options.SelectionEnd && end >= options.SelectionStart {
				start := max(start, options.SelectionStart)
				end := min(end, options.SelectionEnd)
				if start != end {
					posStart0, posStart1, countStart := textPositionFromIndex(bounds.Dx(), str, theCachedLines, start, &options.Options)
					posEnd0, _, countEnd := textPositionFromIndex(bounds.Dx(), str, theCachedLines, end, &options.Options)
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
					posStart0, posStart1, countStart := textPositionFromIndex(bounds.Dx(), str, theCachedLines, start, &options.Options)
					posEnd0, _, countEnd := textPositionFromIndex(bounds.Dx(), str, theCachedLines, end, &options.Options)
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
					posStart0, posStart1, countStart := textPositionFromIndex(bounds.Dx(), str, theCachedLines, start, &options.Options)
					posEnd0, _, countEnd := textPositionFromIndex(bounds.Dx(), str, theCachedLines, end, &options.Options)
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
		lineStr := line.str
		origGeoM := op.GeoM
		if !options.KeepTailingSpace {
			lineStr = strings.TrimRightFunc(lineStr, unicode.IsSpace)
		}
		if options.EllipsisString != "" && advance(lineStr, options.Face, options.TabWidth, options.KeepTailingSpace) > float64(bounds.Dx()) {
			lineStr = truncateWithEllipsis(lineStr, options.EllipsisString, float64(bounds.Dx()), options.Face, options.TabWidth)
		}
		// Ebitengine's text.Draw does not handle tab characters, so lines
		// containing tabs must use manual alignment via oneLineLeft and GeoM.
		if !strings.Contains(lineStr, "\t") {
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
			text.Draw(dst, lineStr, options.Face, op)
		} else {
			op.PrimaryAlign = text.AlignStart
			x := oneLineLeft(bounds.Dx(), lineStr, options.Face, options.HorizontalAlign, options.TabWidth, options.KeepTailingSpace)
			op.GeoM.Translate(x, 0)
			var origX float64
			for {
				head, tail, ok := strings.Cut(lineStr, "\t")
				text.Draw(dst, head, options.Face, op)
				if !ok {
					break
				}
				x := origX + text.Advance(head, options.Face)
				nextX := nextIndentPosition(x, options.TabWidth)
				op.GeoM.Translate(nextX-origX, 0)
				origX = nextX
				lineStr = tail
			}
		}
		op.GeoM = origGeoM
		op.GeoM.Translate(0, options.LineHeight)
	}
}
