// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package textutil

import (
	"iter"
	"slices"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

type VisualLine struct {
	Pos int
	Str string
}

func VisualLines(width int, str string, wrapMode WrapMode, advance func(str string, indexInBytes int) float64) iter.Seq[VisualLine] {
	return func(yield func(VisualLine) bool) {
		for l := range visualLines(width, str, wrapMode, advance) {
			if !yield(VisualLine{
				Pos: l.pos,
				Str: l.str,
			}) {
				return
			}
		}
	}
}

func NextIndentPosition(position float64, indentWidth float64) float64 {
	return nextIndentPosition(position, indentWidth)
}

// AdvanceForTest exposes the internal advance with tabWidth 0 and
// keepTailingSpace false, matching the closure VisualLines is fed in Draw.
func AdvanceForTest(str string, indexInBytes int, face font.Face) float64 {
	return advance(str, indexInBytes, face.TextFace(), 0, false)
}

// VisualLinesFromCachedStarts builds the visual lines via the cache-backed
// Draw path (appendVisualLinesFromCachedStarts), mirroring what Draw does, so
// tests can compare it against VisualLines.
func VisualLinesFromCachedStarts(width int, str string, wrapMode WrapMode, face font.Face, tabWidth float64, keepTailingSpace bool) ([]VisualLine, bool) {
	vls, ok := appendVisualLinesFromCachedStarts(nil, str, width, wrapMode, face, tabWidth, keepTailingSpace)
	out := make([]VisualLine, len(vls))
	for i, vl := range vls {
		out[i] = VisualLine{Pos: vl.pos, Str: vl.str}
	}
	return out, ok
}

// LayoutCacheForTest is a test handle over a fresh layoutCache with an
// injectable tick clock; now may be nil to use the default clock.
type LayoutCacheForTest struct {
	c layoutCache
}

func NewLayoutCacheForTest(softLimit int, now func() int64) *LayoutCacheForTest {
	return &LayoutCacheForTest{c: layoutCache{softLimit: softLimit, now: now}}
}

// testFaceID is a constant nonzero face id, so cache entries are distinguished
// by their text alone.
const testFaceID = 1

// Touch looks up text, inserting a one-element visual-line-starts slice on a miss.
func (l *LayoutCacheForTest) Touch(text string) {
	k := layoutKey{text: text, faceID: testFaceID}
	if _, ok := l.c.get(k); ok {
		return
	}
	l.c.put(k, []int{0})
}

func (l *LayoutCacheForTest) Len() int {
	return len(l.c.entries)
}

func (l *LayoutCacheForTest) Has(text string) bool {
	_, ok := l.c.entries[layoutKey{text: text, faceID: testFaceID}]
	return ok
}

// EntryAliveTicks exposes entryAliveTicks for the cache-eviction tests.
const EntryAliveTicks = entryAliveTicks

// SegmentCacheForTest is a test handle over a fresh segmentCache with an
// injectable tick clock; now may be nil to use the default clock.
type SegmentCacheForTest struct {
	c segmentCache
}

func NewSegmentCacheForTest(softLimit int, now func() int64) *SegmentCacheForTest {
	return &SegmentCacheForTest{
		c: segmentCache{softLimit: softLimit, now: now},
	}
}

// Add inserts an entry under key with empty segmentation; its content does not
// matter to the eviction tests.
func (s *SegmentCacheForTest) Add(key string) {
	s.c.add(key, chunkSegments{})
}

func (s *SegmentCacheForTest) Len() int {
	return len(s.c.entries)
}

func (s *SegmentCacheForTest) Has(key string) bool {
	_, ok := s.c.entries[key]
	return ok
}

func (s *SegmentCacheForTest) GraphemeBoundaries(line string) []int {
	return slices.Collect(s.c.graphemeBoundaries(line))
}

func (s *SegmentCacheForTest) SoftLineBreakBoundaries(line string) []int {
	return slices.Collect(s.c.softLineBreakBoundaries(line))
}
