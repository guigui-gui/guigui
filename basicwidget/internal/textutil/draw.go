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

	// LayoutWidth is the width used to wrap and align text. When zero, the
	// drawing bounds width is used.
	LayoutWidth int

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

	// StyleRuns are paint-only style overrides applied to byte ranges of the
	// drawn string, sorted by Start and disjoint. An empty slice draws the
	// whole string uniformly.
	StyleRuns []StyleRun
}

// StyleRun is a paint-only style override applied to a byte range of the
// drawn string.
type StyleRun struct {
	// Start is the inclusive start of the range in bytes.
	Start int

	// End is the exclusive end of the range in bytes.
	End int

	// Color overrides the text and decoration color. Nil inherits
	// [DrawOptions.TextColor].
	Color color.Color

	// BackgroundColor fills the range's background. Nil draws no background.
	BackgroundColor color.Color

	// Underline draws a line under the text.
	Underline bool

	// Strikethrough draws a line through the text.
	Strikethrough bool
}

// intersectingStyleRuns returns the subslice of runs that intersect
// [start, end). runs must be sorted by Start and disjoint.
func intersectingStyleRuns(runs []StyleRun, start, end int) []StyleRun {
	lo, _ := slices.BinarySearchFunc(runs, start, func(run StyleRun, start int) int {
		if run.End <= start {
			return -1
		}
		return 1
	})
	n, _ := slices.BinarySearchFunc(runs[lo:], end, func(run StyleRun, end int) int {
		if run.Start < end {
			return -1
		}
		return 1
	})
	return runs[lo : lo+n]
}

// rangePositionsInVisualLines returns the positions of [start, end) resolved
// within the visual lines: posStart is start's position (on the second line
// when start sits on a visual line boundary) and posEnd is end's first
// position. ok is false when either endpoint cannot be resolved.
func rangePositionsInVisualLines(layoutWidth int, vls []visualLine, start, end int, style *Style) (posStart, posEnd TextPosition, ok bool) {
	posStart0, posStart1, countStart := textPositionFromIndexInVisualLines(layoutWidth, slices.Values(vls), start, style)
	posEnd0, _, countEnd := textPositionFromIndexInVisualLines(layoutWidth, slices.Values(vls), end, style)
	if countStart == 0 || countEnd == 0 {
		return TextPosition{}, TextPosition{}, false
	}
	posStart = posStart0
	if countStart == 2 {
		posStart = posStart1
	}
	return posStart, posEnd0, true
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
	layoutWidth := options.LayoutWidth
	if layoutWidth <= 0 {
		layoutWidth = bounds.Dx()
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(bounds.Min.X), float64(bounds.Min.Y))
	op.ColorScale.ScaleWithColor(options.TextColor)
	if dst.Bounds() != clip {
		dst = dst.RecyclableSubImage(clip)
		defer dst.Recycle()
	}

	op.LineSpacing = options.LineHeight

	yOffset := textPositionYOffset(image.Pt(layoutWidth, bounds.Dy()), str, &options.Style)
	op.GeoM.Translate(0, yOffset)

	theVisualLinesBuffer = theVisualLinesBuffer[:0]
	defer func() {
		theVisualLinesBuffer = slices.Delete(theVisualLinesBuffer, 0, len(theVisualLinesBuffer))
	}()
	var built bool
	if options.WrapMode != WrapModeNone {
		if vls, ok := appendVisualLinesFromCachedStarts(theVisualLinesBuffer, str, layoutWidth, options.WrapMode, options.Face, options.TabWidth, options.KeepTailingSpace); ok {
			theVisualLinesBuffer = vls
			built = true
		}
	}
	if !built {
		theVisualLinesBuffer = theVisualLinesBuffer[:0]
		for vl := range visualLines(layoutWidth, str, options.WrapMode, func(str string, indexInBytes int) float64 {
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

		lineRuns := intersectingStyleRuns(options.StyleRuns, start, end)

		for _, run := range lineRuns {
			if run.BackgroundColor == nil {
				continue
			}
			runStart := max(start, run.Start)
			runEnd := min(end, run.End)
			if posStart, posEnd, ok := rangePositionsInVisualLines(layoutWidth, theVisualLinesBuffer, runStart, runEnd, &options.Style); ok {
				x := float32(posStart.X) + float32(bounds.Min.X)
				y := float32(posStart.Top) + float32(bounds.Min.Y)
				w := float32(posEnd.X - posStart.X)
				h := float32(posStart.Bottom - posStart.Top)
				vector.FillRect(dst, x, y, w, h, run.BackgroundColor, false)
			}
		}

		if options.DrawSelection {
			if start <= options.SelectionEnd && end >= options.SelectionStart {
				start := max(start, options.SelectionStart)
				end := min(end, options.SelectionEnd)
				if start != end {
					if posStart, posEnd, ok := rangePositionsInVisualLines(layoutWidth, theVisualLinesBuffer, start, end, &options.Style); ok {
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
					if posStart, posEnd, ok := rangePositionsInVisualLines(layoutWidth, theVisualLinesBuffer, start, end, &options.Style); ok {
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
					if posStart, posEnd, ok := rangePositionsInVisualLines(layoutWidth, theVisualLinesBuffer, start, end, &options.Style); ok {
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
		// contentLen is the length of the drawn prefix whose bytes keep their
		// original offsets; an appended ellipsis is excluded.
		contentLen := len(vlStr)
		if options.EllipsisString != "" && advance(vlStr, len(vlStr), options.Face.TextFace(), options.TabWidth, options.KeepTailingSpace) > float64(layoutWidth) {
			vlStr = truncateWithEllipsis(vlStr, options.EllipsisString, float64(layoutWidth), options.Face.TextFace(), options.TabWidth)
			contentLen = len(vlStr) - len(options.EllipsisString)
		}
		var styled bool
		for _, run := range lineRuns {
			if run.Color != nil && run.Start < start+contentLen {
				styled = true
				break
			}
		}
		switch {
		case styled:
			op.PrimaryAlign = text.AlignStart
			x := oneLineLeft(layoutWidth, vlStr, options.Face.TextFace(), options.HorizontalAlign, options.TabWidth, options.KeepTailingSpace)
			op.GeoM.Translate(x, 0)
			drawStyledVisualLine(dst, vlStr, start, start+contentLen, lineRuns, options, op)
		// Ebitengine's text.Draw does not handle tab characters, so lines
		// containing tabs must use manual alignment via oneLineLeft and GeoM.
		case !strings.Contains(vlStr, "\t"):
			// Use Ebitengine's PrimaryAlign for horizontal alignment so that the
			// text origin accounts for the alignment offset. This ensures that each
			// glyph's subpixel position is determined relative to the aligned origin,
			// producing consistent rendering when the text content changes
			// (e.g., right-aligned text gaining/losing characters).
			switch options.HorizontalAlign {
			case HorizontalAlignCenter:
				op.PrimaryAlign = text.AlignCenter
				op.GeoM.Translate(float64(layoutWidth)/2, 0)
			case HorizontalAlignEnd, HorizontalAlignRight:
				op.PrimaryAlign = text.AlignEnd
				op.GeoM.Translate(float64(layoutWidth), 0)
			default:
				op.PrimaryAlign = text.AlignStart
			}
			text.Draw(dst, vlStr, options.Face.TextFace(), op)
		default:
			op.PrimaryAlign = text.AlignStart
			x := oneLineLeft(layoutWidth, vlStr, options.Face.TextFace(), options.HorizontalAlign, options.TabWidth, options.KeepTailingSpace)
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

		if len(lineRuns) > 0 {
			// The ratios approximate the underline and strikeout metrics
			// that the default face Inter declares in its post and OS/2
			// tables: thickness/(ascender+descender) = 0.067, underline
			// center below the baseline / descender = 0.46, and strikeout
			// center above the baseline / ascender = 0.32. The rendering
			// face's own table values are not available through
			// [text.Metrics].
			//
			// TODO: Use the rendering font's own underline and strikeout
			// metrics once the text API exposes them, and decide how to
			// resolve them when a run is rendered with multiple fonts.
			const (
				// decorationThicknessRatio is the thickness of underline and
				// strikethrough lines as a ratio of the face's ascent+descent
				// height.
				decorationThicknessRatio = 1.0 / 14

				// underlineOffsetRatio is the offset of the underline's
				// center below the baseline as a ratio of the face's descent.
				underlineOffsetRatio = 0.4

				// strikethroughOffsetRatio is the offset of the
				// strikethrough's center above the baseline as a ratio of the
				// face's ascent, putting the line roughly at the middle of
				// lowercase letters.
				strikethroughOffsetRatio = 0.3
			)
			m := options.Face.TextFace().Metrics()
			baseline := textPadding(options.Face.TextFace(), options.LineHeight) + m.HAscent
			thickness := max(1, (m.HAscent+m.HDescent)*decorationThicknessRatio)
			for _, run := range lineRuns {
				if !run.Underline && !run.Strikethrough {
					continue
				}
				runStart := max(start, run.Start)
				runEnd := min(start+contentLen, run.End)
				if runStart >= runEnd {
					continue
				}
				posStart, posEnd, ok := rangePositionsInVisualLines(layoutWidth, theVisualLinesBuffer, runStart, runEnd, &options.Style)
				if !ok {
					continue
				}
				clr := run.Color
				if clr == nil {
					clr = options.TextColor
				}
				x := float32(posStart.X) + float32(bounds.Min.X)
				w := float32(posEnd.X - posStart.X)
				lineTop := posStart.Top + float64(bounds.Min.Y)
				if run.Underline {
					y := lineTop + baseline + underlineOffsetRatio*m.HDescent - thickness/2
					vector.FillRect(dst, x, float32(y), w, float32(thickness), clr, false)
				}
				if run.Strikethrough {
					y := lineTop + baseline - strikethroughOffsetRatio*m.HAscent - thickness/2
					vector.FillRect(dst, x, float32(y), w, float32(thickness), clr, false)
				}
			}
		}

		op.GeoM.Translate(0, options.LineHeight)
	}
}

// drawStyledVisualLine draws vlStr, whose byte i sits at offset lineStart+i
// of the drawn string, splitting at tab positions and at effective-color
// boundaries so each segment is drawn in its run's color. runs holds the
// style runs intersecting the line, sorted and disjoint. Bytes at or past
// contentEnd (an appended ellipsis) use the base text color. op must be
// positioned at the line's left origin, with [text.AlignStart].
func drawStyledVisualLine(dst *ebiten.Image, vlStr string, lineStart, contentEnd int, runs []StyleRun, options *DrawOptions, op *text.DrawOptions) {
	face := options.Face.TextFace()
	// colorAt returns the effective color at the absolute byte offset abs and
	// the absolute offset at which the effective color may next change. It
	// must be called with nondecreasing offsets: runIdx advances monotonically
	// over runs, so a whole line costs one pass regardless of segment count.
	var runIdx int
	colorAt := func(abs int) (color.Color, int) {
		if abs >= contentEnd {
			return options.TextColor, math.MaxInt
		}
		for runIdx < len(runs) && runs[runIdx].End <= abs {
			runIdx++
		}
		if runIdx >= len(runs) {
			return options.TextColor, math.MaxInt
		}
		run := runs[runIdx]
		if abs < run.Start {
			return options.TextColor, min(run.Start, contentEnd)
		}
		clr := run.Color
		if clr == nil {
			clr = options.TextColor
		}
		return clr, min(run.End, contentEnd)
	}

	origGeoM := op.GeoM
	origColorScale := op.ColorScale
	// origX and pos are the drawing origin set after the last tab: its x
	// offset from the line's left, and its byte position in vlStr.
	var origX float64
	var pos int
	var i int
	for i < len(vlStr) {
		if vlStr[i] == '\t' {
			x := origX + text.AdvanceAt(vlStr, i, face) - text.AdvanceAt(vlStr, pos, face)
			nextX := nextIndentPosition(x, options.TabWidth)
			op.GeoM.Translate(nextX-origX, 0)
			origX = nextX
			i++
			pos = i
			continue
		}
		clr, changeAt := colorAt(lineStart + i)
		next := min(changeAt-lineStart, len(vlStr))
		if tabIdx := strings.IndexByte(vlStr[i:], '\t'); tabIdx >= 0 {
			next = min(next, i+tabIdx)
		}
		geoM := op.GeoM
		op.GeoM.Translate(text.AdvanceAt(vlStr, i, face)-text.AdvanceAt(vlStr, pos, face), 0)
		op.ColorScale.Reset()
		op.ColorScale.ScaleWithColor(clr)
		text.Draw(dst, vlStr[i:next], face, op)
		op.GeoM = geoM
		i = next
	}
	op.GeoM = origGeoM
	op.ColorScale = origColorScale
}
