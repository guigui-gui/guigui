// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"math"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

// FaceRun overrides the face used to measure and draw a byte range of the
// laid-out text.
type FaceRun struct {
	// Start is the inclusive start of the range in bytes.
	Start int

	// End is the exclusive end of the range in bytes.
	End int

	// Face measures and draws the range.
	Face font.Face
}

// faceAt returns the face at offset and the exclusive offset at which the
// face may next change ([math.MaxInt] when it no longer changes). faceRuns
// must be sorted by Start and disjoint; offsets outside every run resolve to
// def.
func faceAt(faceRuns []FaceRun, def font.Face, offset int) (font.Face, int) {
	i, ok := slices.BinarySearchFunc(faceRuns, offset, func(run FaceRun, offset int) int {
		switch {
		case run.End <= offset:
			return -1
		case run.Start > offset:
			return 1
		default:
			return 0
		}
	})
	if ok {
		return faceRuns[i].Face, faceRuns[i].End
	}
	if i < len(faceRuns) {
		return def, faceRuns[i].Start
	}
	return def, math.MaxInt
}

// advanceWithFaces is [advance] with per-range face overrides: it returns the
// advance of str[:endIndexInBytes], splitting the prefix at tab positions
// and face boundaries and measuring each segment standalone with its face,
// matching how mixed-face text is drawn. strStartInBytes is the byte offset
// of str's start in the text that faceRuns' offsets index. An empty faceRuns
// behaves exactly like [advance].
func advanceWithFaces(str string, strStartInBytes, endIndexInBytes int, face font.Face, faceRuns []FaceRun, tabWidth float64, keepTailingSpace bool) float64 {
	if len(faceRuns) == 0 {
		return advance(str, endIndexInBytes, face.TextFace(), tabWidth, keepTailingSpace)
	}
	end, hasLineBreak := measuredPrefixEnd(str, endIndexInBytes, keepTailingSpace)
	var width float64
	var pos int
	for pos < end {
		if tabWidth != 0 && str[pos] == '\t' {
			width = nextIndentPosition(width, tabWidth)
			pos++
			continue
		}
		segFace, faceEnd := faceAt(faceRuns, face, strStartInBytes+pos)
		segEnd := min(end, faceEnd-strStartInBytes)
		if tabWidth != 0 {
			if i := strings.IndexByte(str[pos:segEnd], '\t'); i >= 0 {
				segEnd = pos + i
			}
		}
		width += text.AdvanceAt(str[pos:segEnd], segEnd-pos, segFace.TextFace())
		pos = segEnd
	}
	if hasLineBreak {
		lineBreakFace, _ := faceAt(faceRuns, face, strStartInBytes+end)
		// Always add the advance of a space for the line break for a
		// consistent behavior.
		width += text.AdvanceAt(" ", 1, lineBreakFace.TextFace())
	}
	return width
}
