// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package textutil

import (
	"iter"
	"slices"
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
