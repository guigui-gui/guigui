// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package textutil

import (
	"image"
	"image/color"
	"iter"
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
// position. start, end, and each visualLine's pos are relative to vls' first
// byte; vlsStartInBytes is the whole-text byte offset of that byte. ok is
// false when either endpoint cannot be resolved.
func rangePositionsInVisualLines(layoutWidth int, vls []visualLine, vlsStartInBytes, start, end int, style *Style) (posStart, posEnd TextPosition, ok bool) {
	posStart0, posStart1, countStart := textPositionFromIndexInVisualLines(layoutWidth, slices.Values(vls), vlsStartInBytes, start, style)
	posEnd0, _, countEnd := textPositionFromIndexInVisualLines(layoutWidth, slices.Values(vls), vlsStartInBytes, end, style)
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
// preceding line), including the trailing empty line after a final break. A
// line intersecting a face run wraps through the run-aware path instead of
// the single-face cache. ok is false (dst left unchanged) when a line's
// starts are unavailable, so the caller falls back to shaping.
func appendVisualLinesFromCachedStarts(dst []visualLine, str string, width int, wrapMode WrapMode, face font.Face, faceRuns []FaceRun, tabWidth float64, keepTailingSpace bool) (lines []visualLine, ok bool) {
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
		if faceRunsIntersect(faceRuns, pos, lineEnd) {
			for vl := range visualLinesFromLogicalLine(width, line, wrapMode, face, faceRuns, pos, tabWidth, keepTailingSpace) {
				dst = append(dst, visualLine{pos: pos + vl.pos, str: vl.str})
			}
		} else {
			s, sok := cachedVisualLineStarts(width, line, wrapMode, face, tabWidth, keepTailingSpace)
			if !sok {
				return slices.Delete(dst, base, len(dst)), false
			}
			for i := range s {
				rs := pos + s[i]
				re := lineEnd
				if i+1 < len(s) {
					re = pos + s[i+1]
				}
				dst = append(dst, visualLine{pos: rs, str: str[rs:re]})
			}
		}
		if last {
			break
		}
		pos = lineEnd
	}
	return dst, true
}

// appendDrawnVisualLines appends the visual lines str is drawn as, laid out at
// layoutWidth with options, to dst.
func appendDrawnVisualLines(dst []visualLine, str string, layoutWidth int, options *DrawOptions) []visualLine {
	if options.WrapMode != WrapModeNone {
		if vls, ok := appendVisualLinesFromCachedStarts(dst, str, layoutWidth, options.WrapMode, options.Face, options.FaceRuns, options.TabWidth, options.KeepTailingSpace); ok {
			return vls
		}
	}
	for vl := range visualLines(layoutWidth, str, options.WrapMode, func(str string, strStartInBytes, endIndexInBytes int) float64 {
		return advanceWithFaces(str, strStartInBytes, endIndexInBytes, options.Face, options.FaceRuns, options.TabWidth, options.KeepTailingSpace)
	}) {
		dst = append(dst, vl)
	}
	return dst
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
	theVisualLinesBuffer = appendDrawnVisualLines(theVisualLinesBuffer, str, layoutWidth, options)

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
			if posStart, posEnd, ok := rangePositionsInVisualLines(layoutWidth, theVisualLinesBuffer, 0, runStart, runEnd, &options.Style); ok {
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
					if posStart, posEnd, ok := rangePositionsInVisualLines(layoutWidth, theVisualLinesBuffer, 0, start, end, &options.Style); ok {
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
					if posStart, posEnd, ok := rangePositionsInVisualLines(layoutWidth, theVisualLinesBuffer, 0, start, end, &options.Style); ok {
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
					if posStart, posEnd, ok := rangePositionsInVisualLines(layoutWidth, theVisualLinesBuffer, 0, start, end, &options.Style); ok {
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
		if options.EllipsisString != "" && advanceWithFaces(vlStr, start, len(vlStr), options.Face, options.FaceRuns, options.TabWidth, options.KeepTailingSpace) > float64(layoutWidth) {
			vlStr = truncateWithEllipsis(vlStr, start, options.EllipsisString, float64(layoutWidth), options.Face, options.FaceRuns, options.TabWidth)
			contentLen = len(vlStr) - len(options.EllipsisString)
		}
		mixedFaces := faceRunsIntersect(options.FaceRuns, start, start+contentLen)
		styled := mixedFaces
		if !styled {
			for _, run := range lineRuns {
				if run.Color != nil && run.Start < start+contentLen {
					styled = true
					break
				}
			}
		}
		switch {
		case styled:
			op.PrimaryAlign = text.AlignStart
			x := oneLineLeft(layoutWidth, vlStr, start, &options.Style)
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
			x := oneLineLeft(layoutWidth, vlStr, start, &options.Style)
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
			for d := range visualLineDecorations(layoutWidth, theVisualLinesBuffer, start, start+contentLen, lineRuns, options) {
				x := float32(d.X) + float32(bounds.Min.X)
				y := float32(d.Y) + float32(bounds.Min.Y)
				vector.FillRect(dst, x, y, float32(d.Width), float32(d.Thickness), d.Color, false)
			}
		}

		op.GeoM.Translate(0, options.LineHeight)
	}
}

// The ratios approximate the underline and strikeout metrics that the default
// face Inter declares in its post and OS/2 tables:
// thickness/(ascender+descender) = 0.067, underline center below the
// baseline / descender = 0.46, and strikeout center above the baseline /
// ascender = 0.32. The rendering face's own table values are not available
// through [text.Metrics].
//
// TODO: Use the rendering font's own underline and strikeout metrics once the
// text API exposes them.
const (
	// decorationThicknessRatio is the thickness of underline and
	// strikethrough lines as a ratio of the face's ascent+descent height.
	decorationThicknessRatio = 1.0 / 14

	// underlineOffsetRatio is the offset of the underline's center below the
	// baseline as a ratio of the face's descent.
	underlineOffsetRatio = 0.4

	// strikethroughOffsetRatio is the offset of the strikethrough's center
	// above the baseline as a ratio of the face's ascent, putting the line
	// roughly at the middle of lowercase letters.
	strikethroughOffsetRatio = 0.3
)

// decorationKind is the kind of line a [StyleRun] draws over its range.
type decorationKind int

const (
	decorationKindUnderline decorationKind = iota
	decorationKindStrikethrough
)

// enabled reports whether run carries the decoration.
func (k decorationKind) enabled(run StyleRun) bool {
	switch k {
	case decorationKindUnderline:
		return run.Underline
	case decorationKindStrikethrough:
		return run.Strikethrough
	}
	return false
}

// offsetFromBaseline returns the offset of the decoration's center from the
// baseline, positive downwards, for a face with the metrics m.
func (k decorationKind) offsetFromBaseline(m text.Metrics) float64 {
	switch k {
	case decorationKindUnderline:
		return underlineOffsetRatio * m.HDescent
	case decorationKindStrikethrough:
		return -strikethroughOffsetRatio * m.HAscent
	}
	return 0
}

// decoration is one underline or strikethrough line, in the coordinates
// [TextPosition] uses.
type decoration struct {
	X         float64
	Y         float64
	Width     float64
	Thickness float64
	Color     color.Color
}

// visualLineDecorations iterates the underline and strikethrough lines to draw
// for the visual line whose drawn content is [lineStart, contentEnd). runs are
// the style runs intersecting the line, sorted and disjoint, and vls the
// visual lines the positions are resolved in, starting at byte 0 of the drawn
// string.
func visualLineDecorations(layoutWidth int, vls []visualLine, lineStart, contentEnd int, runs []StyleRun, options *DrawOptions) iter.Seq[decoration] {
	return func(yield func(decoration) bool) {
		// Every face on a visual line is drawn on the base face's baseline
		// (see drawStyledVisualLine), so the baseline does not depend on the
		// faces the decorated range uses.
		baseAscent := options.Face.TextFace().Metrics().HAscent
		for _, kind := range [...]decorationKind{decorationKindUnderline, decorationKindStrikethrough} {
			for i := 0; i < len(runs); {
				if !kind.enabled(runs[i]) {
					i++
					continue
				}
				// A decorated span is the maximal contiguous byte range whose
				// runs all carry the decoration. The runs in it can still
				// differ in color, in which case the span is drawn as several
				// rectangles sharing one position and thickness.
				j := i + 1
				for j < len(runs) && kind.enabled(runs[j]) && runs[j].Start == runs[j-1].End {
					j++
				}
				span := runs[i:j]
				i = j

				spanStart := max(lineStart, span[0].Start)
				spanEnd := min(contentEnd, span[len(span)-1].End)
				if spanStart >= spanEnd {
					continue
				}
				// The part of the span on this visual line takes its metrics
				// from the smallest face it is drawn with: that face's line
				// crosses the lower half of the taller faces' glyphs, while a
				// taller face's line can miss the smaller ones entirely.
				m := smallestFaceInRange(options.FaceRuns, options.Face, spanStart, spanEnd).TextFace().Metrics()
				thickness := max(1, (m.HAscent+m.HDescent)*decorationThicknessRatio)
				offset := kind.offsetFromBaseline(m)

				for _, run := range span {
					runStart := max(lineStart, run.Start)
					runEnd := min(contentEnd, run.End)
					if runStart >= runEnd {
						continue
					}
					posStart, posEnd, ok := rangePositionsInVisualLines(layoutWidth, vls, 0, runStart, runEnd, &options.Style)
					if !ok {
						continue
					}
					clr := run.Color
					if clr == nil {
						clr = options.TextColor
					}
					if !yield(decoration{
						X:         posStart.X,
						Y:         posStart.Top + baseAscent + offset - thickness/2,
						Width:     posEnd.X - posStart.X,
						Thickness: thickness,
						Color:     clr,
					}) {
						return
					}
				}
			}
		}
	}
}

// styleRunColorAt returns the effective color at textIndexInBytes, a byte
// offset in the drawn string, and the exclusive offset at which the color
// may next change ([math.MaxInt] when it no longer changes). runs must be
// sorted by Start and disjoint. Bytes at or past contentEnd (an appended
// ellipsis) use textColor.
func styleRunColorAt(runs []StyleRun, textColor color.Color, contentEnd, textIndexInBytes int) (color.Color, int) {
	if textIndexInBytes >= contentEnd {
		return textColor, math.MaxInt
	}
	i, ok := slices.BinarySearchFunc(runs, textIndexInBytes, func(run StyleRun, textIndexInBytes int) int {
		switch {
		case run.End <= textIndexInBytes:
			return -1
		case run.Start > textIndexInBytes:
			return 1
		default:
			return 0
		}
	})
	if ok {
		clr := runs[i].Color
		if clr == nil {
			clr = textColor
		}
		return clr, min(runs[i].End, contentEnd)
	}
	if i < len(runs) {
		return textColor, min(runs[i].Start, contentEnd)
	}
	return textColor, math.MaxInt
}

// faceRunsIntersect reports whether any face run intersects [start, end).
// faceRuns must be sorted by Start and disjoint.
func faceRunsIntersect(faceRuns []FaceRun, start, end int) bool {
	lo, _ := slices.BinarySearchFunc(faceRuns, start, func(run FaceRun, start int) int {
		if run.End <= start {
			return -1
		}
		return 1
	})
	return lo < len(faceRuns) && faceRuns[lo].Start < end
}

// drawStyledVisualLine draws vlStr, whose byte i sits at offset lineStart+i
// of the drawn string, with runs holding the style runs intersecting the
// line, sorted and disjoint. Bytes at or past contentEnd (an appended
// ellipsis) use the base text color. op must be positioned at the line's
// left origin, with [text.AlignStart].
func drawStyledVisualLine(dst *ebiten.Image, vlStr string, lineStart, contentEnd int, runs []StyleRun, options *DrawOptions, op *text.DrawOptions) {
	origGeoM := op.GeoM
	origColorScale := op.ColorScale
	baseAscent := options.Face.TextFace().Metrics().HAscent
	var x float64
	var i int
	for i < len(vlStr) {
		if options.TabWidth != 0 && vlStr[i] == '\t' {
			x = nextIndentPosition(x, options.TabWidth)
			i++
			continue
		}
		// A tab- and face-delimited segment is measured standalone and placed
		// at the cumulative x of the preceding segments, matching
		// [advanceWithFaces]. Within it, color chunks are placed at
		// prefix-advance offsets so the shaping context is preserved across
		// color boundaries.
		segFace, faceChange := faceAt(options.FaceRuns, options.Face, lineStart+i)
		segEnd := min(faceChange-lineStart, len(vlStr))
		if options.TabWidth != 0 {
			if tabIdx := strings.IndexByte(vlStr[i:segEnd], '\t'); tabIdx >= 0 {
				segEnd = i + tabIdx
			}
		}
		seg := vlStr[i:segEnd]
		face := segFace.TextFace()
		// text.Draw positions text by its top, so a face whose ascent
		// differs from the base face's shifts by the difference to keep the
		// baselines aligned.
		dy := baseAscent - face.Metrics().HAscent
		var chunkStart int
		for chunkStart < len(seg) {
			clr, colorChange := styleRunColorAt(runs, options.TextColor, contentEnd, lineStart+i+chunkStart)
			chunkEnd := min(colorChange-lineStart-i, len(seg))
			geoM := op.GeoM
			op.GeoM.Translate(x+text.AdvanceAt(seg, chunkStart, face), dy)
			op.ColorScale.Reset()
			op.ColorScale.ScaleWithColor(clr)
			text.Draw(dst, seg[chunkStart:chunkEnd], face, op)
			op.GeoM = geoM
			chunkStart = chunkEnd
		}
		x += text.AdvanceAt(seg, len(seg), face)
		i = segEnd
	}
	op.GeoM = origGeoM
	op.ColorScale = origColorScale
}
