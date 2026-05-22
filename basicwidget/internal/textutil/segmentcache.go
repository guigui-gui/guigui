// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"iter"
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui/basicwidget/internal/chunk"
)

// chunkSegments holds one chunk's interior segmentation as boundary offsets
// relative to the chunk start; the chunk's own start and end are omitted,
// being boundaries already. Each list partitions the chunk contiguously.
type chunkSegments struct {
	// graphemes are the interior grapheme-cluster boundaries.
	graphemes []int32
	// softLineBreaks are the interior UAX #14 line-break opportunities.
	softLineBreaks []int32
}

// segmentChunk segments a single chunk into its interior grapheme-cluster and
// line-break boundaries. chunkStr must be valid UTF-8 (a chunk of a logical
// line that has already been sanitized).
func segmentChunk(chunkStr string) chunkSegments {
	seg := pushSegmenter()
	defer popSegmenter()
	initSegmenterWithString(seg, chunkStr)

	var s chunkSegments
	// A chunk has at most as many grapheme clusters as runes, so preallocating
	// to the rune count lets the append loop fill without reallocating.
	s.graphemes = make([]int32, 0, utf8.RuneCountInString(chunkStr))
	for it := seg.GraphemeIterator(); it.Next(); {
		g := it.Grapheme()
		if end := g.OffsetInBytes + g.LengthInBytes; end < len(chunkStr) {
			s.graphemes = append(s.graphemes, int32(end))
		}
	}
	for it := seg.LineIterator(); it.Next(); {
		l := it.Line()
		if end := l.OffsetInBytes + l.LengthInBytes; end < len(chunkStr) {
			s.softLineBreaks = append(s.softLineBreaks, int32(end))
		}
	}
	return s
}

// segmentCacheSoftLimitEntries is the entry count above which idle entries are
// swept. Entries used within [entryAliveTicks] are kept even past it, so the
// cache may grow beyond this to hold the active working set.
const segmentCacheSoftLimitEntries = 512

// entryAliveTicks is how long after its last use a cache entry is kept from
// eviction — about a second at 60 ticks per second.
const entryAliveTicks = 60

// theSegmentCache is the process-wide cache shared by all text widgets. The UI
// is single-threaded, so it needs no locking.
var theSegmentCache = segmentCache{softLimit: segmentCacheSoftLimitEntries}

// sanitizedForCache returns s unchanged if it is valid UTF-8, otherwise a
// sanitized copy. The cache requires valid UTF-8 so that chunk byte offsets
// align with the string the caller slices.
func sanitizedForCache(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return sanitizeUTF8(s)
}

// segmentCache memoizes per-chunk segmentation keyed by chunk content. It is
// a pure function of the bytes — no width, wrap mode, or layout state — so an
// edit confined to one chunk only invalidates that chunk's entry. Entries idle
// for longer than [entryAliveTicks] are evicted once the cache holds more than
// its soft limit; entries used more recently are kept regardless. Not safe for
// concurrent use; the UI is single-threaded.
type segmentCache struct {
	// softLimit is the entry count above which idle entries are swept.
	softLimit int
	entries   map[string]*segmentCacheEntry
	// lastEvictTick is the tick of the most recent eviction sweep, so the O(n)
	// sweep runs at most once per tick even while the cache is over its limit.
	lastEvictTick int64
	// now reports the current tick; if nil, [ebiten.Tick] is used. Tests inject
	// a controllable clock.
	now func() int64
}

func (c *segmentCache) nowTick() int64 {
	if c.now == nil {
		return ebiten.Tick()
	}
	return c.now()
}

type segmentCacheEntry struct {
	segs chunkSegments
	// lastTick is the tick the entry was last created or read.
	lastTick int64
}

// graphemeBoundaries yields the byte offset just past each grapheme cluster of
// line, in order, ending at len(line). line must be valid UTF-8. The clusters
// are contiguous and cover the whole line, so each offset is also the start of
// the next cluster. Each chunk is a cache hit (replaying stored offsets) or a
// miss (segmenting that one chunk); the whole-line Unicode pass is never
// repeated for unchanged chunks.
func (c *segmentCache) graphemeBoundaries(line string) iter.Seq[int] {
	return c.boundaries(line, func(s *chunkSegments) []int32 { return s.graphemes })
}

// softLineBreakBoundaries yields the byte offset just past each segment of line
// delimited by UAX #14 line-break opportunities, in order, ending at len(line).
// line must be valid UTF-8. The segments are contiguous and cover the whole
// line; a soft wrap may be placed at any of these offsets.
func (c *segmentCache) softLineBreakBoundaries(line string) iter.Seq[int] {
	return c.boundaries(line, func(s *chunkSegments) []int32 { return s.softLineBreaks })
}

// boundaries yields the end offset of each segment of line described by the
// interior boundary offsets pick selects (graphemes or line breaks), which
// partition each chunk. The yielded offsets are increasing and end at len(line).
func (c *segmentCache) boundaries(line string, pick func(*chunkSegments) []int32) iter.Seq[int] {
	return func(yield func(int) bool) {
		var start int
		for _, end := range chunk.AppendBoundaries(nil, line) {
			if end == start {
				// The empty-line case yields a single zero-width chunk.
				continue
			}
			segs := c.segments(line[start:end])
			for _, o := range pick(&segs) {
				if !yield(start + int(o)) {
					return
				}
			}
			if !yield(end) {
				return
			}
			start = end
		}
	}
}

// segments returns chunkStr's segmentation, from the cache or by segmenting it
// on a miss (and caching the result). The value is returned by copy, so a
// later eviction does not disturb a caller still reading it.
func (c *segmentCache) segments(chunkStr string) chunkSegments {
	if e, ok := c.entries[chunkStr]; ok {
		e.lastTick = c.nowTick()
		return e.segs
	}
	segs := segmentChunk(chunkStr)
	c.add(chunkStr, segs)
	return segs
}

func (c *segmentCache) add(chunkStr string, segs chunkSegments) {
	if c.entries == nil {
		c.entries = map[string]*segmentCacheEntry{}
	}
	cur := c.nowTick()
	// Clone the key so a cached chunk does not pin the whole logical line's
	// backing array, of which chunkStr is a substring.
	c.entries[strings.Clone(chunkStr)] = &segmentCacheEntry{segs: segs, lastTick: cur}
	c.evictStaleIfNeeded(cur)
}

// evictStaleIfNeeded drops entries idle for more than [entryAliveTicks] when
// the cache is over its soft limit. Entries within the alive window are kept
// even if that leaves it over the limit.
func (c *segmentCache) evictStaleIfNeeded(cur int64) {
	// Nothing ages out within a tick, so sweep at most once per tick.
	if len(c.entries) <= c.softLimit || cur == c.lastEvictTick {
		return
	}
	c.lastEvictTick = cur
	for key, e := range c.entries {
		if cur-e.lastTick > entryAliveTicks {
			delete(c.entries, key)
		}
	}
}
