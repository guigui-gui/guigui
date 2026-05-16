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

	// VisibleBounds restricts drawing to lines and glyphs that intersect this
	// rectangle. Lines fully above or below are skipped without shaping, and
	// glyphs whose drawn rectangle falls entirely outside are not submitted to
	// [(*ebiten.Image).DrawImage]. An empty rectangle draws nothing.
	VisibleBounds image.Rectangle
}

var theCachedVisualLines []visualLine

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

	yOffset := textPositionYOffset(bounds.Size(), str, &options.Options)
	op.GeoM.Translate(0, yOffset)

	theCachedVisualLines = theCachedVisualLines[:0]
	for vl := range visualLines(bounds.Dx(), str, options.WrapMode, func(str string, indexInBytes int) float64 {
		return advance(str, indexInBytes, options.Face, options.TabWidth, options.KeepTailingSpace)
	}) {
		theCachedVisualLines = append(theCachedVisualLines, vl)
	}

	for _, vl := range theCachedVisualLines {
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
					posStart0, posStart1, countStart := textPositionFromIndex(bounds.Dx(), str, slices.Values(theCachedVisualLines), start, &options.Options)
					posEnd0, _, countEnd := textPositionFromIndex(bounds.Dx(), str, slices.Values(theCachedVisualLines), end, &options.Options)
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
					posStart0, posStart1, countStart := textPositionFromIndex(bounds.Dx(), str, slices.Values(theCachedVisualLines), start, &options.Options)
					posEnd0, _, countEnd := textPositionFromIndex(bounds.Dx(), str, slices.Values(theCachedVisualLines), end, &options.Options)
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
					posStart0, posStart1, countStart := textPositionFromIndex(bounds.Dx(), str, slices.Values(theCachedVisualLines), start, &options.Options)
					posEnd0, _, countEnd := textPositionFromIndex(bounds.Dx(), str, slices.Values(theCachedVisualLines), end, &options.Options)
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
		if options.EllipsisString != "" && advance(vlStr, len(vlStr), options.Face, options.TabWidth, options.KeepTailingSpace) > float64(bounds.Dx()) {
			vlStr = truncateWithEllipsis(vlStr, options.EllipsisString, float64(bounds.Dx()), options.Face, options.TabWidth)
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
			text.Draw(dst, vlStr, options.Face, op)
		} else {
			op.PrimaryAlign = text.AlignStart
			x := oneLineLeft(bounds.Dx(), vlStr, options.Face, options.HorizontalAlign, options.TabWidth, options.KeepTailingSpace)
			op.GeoM.Translate(x, 0)
			var origX float64
			for {
				head, tail, ok := strings.Cut(vlStr, "\t")
				text.Draw(dst, head, options.Face, op)
				if !ok {
					break
				}
				x := origX + text.AdvanceAt(head, len(head), options.Face)
				nextX := nextIndentPosition(x, options.TabWidth)
				op.GeoM.Translate(nextX-origX, 0)
				origX = nextX
				vlStr = tail
			}
		}
		op.GeoM = origGeoM
		op.GeoM.Translate(0, options.LineHeight)
	}
}
