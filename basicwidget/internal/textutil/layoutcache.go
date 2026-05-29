// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

// layoutCacheSoftLimitEntries is the entry count above which idle entries are
// swept. Entries used within [entryAliveTicks] are kept even past it, so the
// cache may grow beyond this to hold the active working set.
const layoutCacheSoftLimitEntries = 256

// layoutKey identifies a wrap layout by every input that determines it. The
// text is the logical line's bytes, so an edit yields a different key and a
// natural miss. faceID is the resolved face's identity, which changes when a
// face re-resolves (e.g. on a locale change), so an entry never outlives the
// exact face that produced it.
type layoutKey struct {
	layoutStyleKey
	text string
}

// layoutStyleKey is layoutKey without the text: the inputs an edit leaves
// unchanged. The last-measured line is patched (not rebuilt) only when these match.
type layoutStyleKey struct {
	faceID           uint64
	width            int
	wrapMode         WrapMode
	tabWidth         float64
	keepTailingSpace bool
}

// theLayoutCache is the process-wide layout cache shared by all text widgets.
// The UI is single-threaded, so it needs no locking.
var theLayoutCache = layoutCache{softLimit: layoutCacheSoftLimitEntries}

// layoutCache memoizes visual-line start offsets keyed by content and layout
// parameters. Entries idle for longer than [entryAliveTicks] are evicted once
// the cache holds more than its soft limit; entries used more recently are kept
// regardless. Not safe for concurrent use; the UI is single-threaded.
type layoutCache struct {
	softLimit int
	entries   map[layoutKey]*layoutCacheEntry
	// lastEvictTick is the tick of the most recent eviction sweep, so the O(n)
	// sweep runs at most once per tick even while the cache is over its limit.
	lastEvictTick int64
	// now reports the current tick; if nil, [ebiten.Tick] is used. Tests inject
	// a controllable clock.
	now func() int64
	// lastMeasured holds the most recently measured logical line so an edit of it
	// patches its advanceUpTo instead of reshaping the whole line.
	lastMeasured measuredLogicalLine
}

type layoutCacheEntry struct {
	vlStarts []int
	// lastTick is the tick the entry was last created or read.
	lastTick int64
}

// measuredLogicalLine is a logical line together with the parameters and face it
// was measured under and the resulting advanceUpTo widths. The layout cache keeps
// the most recently measured one (lastMeasured) so an edit of that line can patch
// advanceUpTo by reshaping only the changed span instead of the whole line. It
// holds no visual-line starts; those are recomputed by repacking advanceUpTo.
type measuredLogicalLine struct {
	valid       bool
	style       layoutStyleKey
	line        string
	face        font.Face
	advanceUpTo []float64
	// advanceUpToBuf is a reused backing array for the next layout's advanceUpTo
	// build, so steady-state editing allocates none.
	advanceUpToBuf []float64
	// vlStartsLen is this line's visual-line count, used to presize the next
	// layout's collect so it fills without regrowing.
	vlStartsLen int
	// lastTick is the tick this line was last measured, for idle eviction.
	lastTick int64
}

func (c *layoutCache) nowTick() int64 {
	if c.now == nil {
		return ebiten.Tick()
	}
	return c.now()
}

// get returns the cached visual-line starts for k, or ok=false on a miss.
func (c *layoutCache) get(k layoutKey) ([]int, bool) {
	if e, ok := c.entries[k]; ok {
		e.lastTick = c.nowTick()
		return e.vlStarts, true
	}
	return nil, false
}

// put stores vlStarts for k. The stored slice must not be mutated by callers;
// get hands back the same slice.
func (c *layoutCache) put(k layoutKey, vlStarts []int) {
	if c.entries == nil {
		c.entries = map[layoutKey]*layoutCacheEntry{}
	}
	cur := c.nowTick()
	// Clone the key text so a cached entry does not pin the whole document
	// buffer, of which the line is a substring.
	k.text = strings.Clone(k.text)
	c.entries[k] = &layoutCacheEntry{vlStarts: vlStarts, lastTick: cur}
	c.evictStaleIfNeeded(cur)
}

// evictStaleIfNeeded drops entries idle for more than [entryAliveTicks] when the
// cache is over its soft limit. Entries within the alive window are kept even if
// that leaves it over the limit.
func (c *layoutCache) evictStaleIfNeeded(cur int64) {
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
	// Drop the last-measured line once editing has gone idle. Reusing its
	// advanceUpTo/advanceUpToBuf capacity would be correct, but for a giant line
	// each is several MB and would stay pinned for the process lifetime; the
	// per-keystroke capacity reuse is already handled by the swap in relayout,
	// and the next layout after idle is a cold path where reallocation is cheap.
	if c.lastMeasured.valid && cur-c.lastMeasured.lastTick > entryAliveTicks {
		c.lastMeasured = measuredLogicalLine{}
	}
}

// relayout returns the visual-line starts for k.text, updating the last-measured
// line. Callers must have a non-zero faceID; face is the resolved face for
// k.faceID. k.text must be valid UTF-8.
func (c *layoutCache) relayout(k layoutKey, face font.Face) []int {
	line := k.text
	style := k.layoutStyleKey
	tf := face.TextFace()

	var work []float64
	// capHint presizes the visual-line-starts append to the previous layout's
	// count, which changes by about one per edit, avoiding regrowth.
	var capHint int
	// replaceLastMeasured reports whether the current line becomes the
	// tracked last-measured line. Which line to keep is a tunable performance
	// heuristic, not a correctness decision: a wrong guess only costs a full
	// reshape next tick, never a wrong layout.
	var replaceLastMeasured bool
	if c.lastMeasured.valid && c.lastMeasured.style == style {
		capHint = c.lastMeasured.vlStartsLen
		var reshapedLengthInBytes int
		work, reshapedLengthInBytes = patchAdvanceUpTo(c.lastMeasured.advanceUpTo, c.lastMeasured.advanceUpToBuf, c.lastMeasured.line, line, tf)
		// Replace the last-measured line when the new line is at least as long.
		// Otherwise keep the (longer) last-measured line unless the reshaped
		// window is small on both lines, so a short line laid out the same tick
		// (e.g. an empty trailing line, or a virtualized draw substring) cannot
		// force the next keystroke to reshape the giant line whole.
		replaceLastMeasured = len(line) >= len(c.lastMeasured.line)
		if !replaceLastMeasured {
			// The reshaped window spans the same logical range of both lines,
			// but is reshapedLengthInBytes bytes long on the new line and, since
			// the suffix shifted by delta, reshapedLengthInBytes minus delta
			// bytes long on the old line.
			delta := len(line) - len(c.lastMeasured.line)
			oldReshapedLengthInBytes := reshapedLengthInBytes - delta
			smallReshapeOnNewLine := 2*reshapedLengthInBytes <= len(line)
			smallReshapeOnOldLine := 2*oldReshapedLengthInBytes <= len(c.lastMeasured.line)
			replaceLastMeasured = smallReshapeOnNewLine && smallReshapeOnOldLine
		}
	} else {
		// A full rebuild always becomes the new last-measured line.
		work = appendAdvanceUpTo(c.lastMeasured.advanceUpToBuf[:0], line, tf)
		replaceLastMeasured = true
	}

	ra := newRangeAdvancer(line, face, k.tabWidth, k.keepTailingSpace)
	// work is line's advanceUpTo (length len(line)+1), so seeding it lets the packer
	// measure without reshaping line.
	ra.advanceUpTo = work
	vlStarts := appendVisualLineStarts(make([]int, 0, capHint), k.width, line, k.wrapMode, ra)

	if replaceLastMeasured {
		// Swap: the array just built becomes the live one; the previous live array
		// becomes the build buffer for the next layout.
		c.lastMeasured.advanceUpToBuf, c.lastMeasured.advanceUpTo = c.lastMeasured.advanceUpTo, work
		// line is a substring of the document buffer; clone it so retaining it
		// across ticks does not pin the whole buffer.
		c.lastMeasured.line = strings.Clone(line)
		c.lastMeasured.face = face
		c.lastMeasured.style = style
		c.lastMeasured.vlStartsLen = len(vlStarts)
		c.lastMeasured.valid = true
	} else {
		// Keep the existing (longer) line as lastMeasured for the next
		// keystroke's patch, recycling the just-built array as the scratch
		// buffer. work was grown from advanceUpToBuf, so this is a no-op unless
		// patchAdvanceUpTo reallocated it, in which case it keeps the larger
		// capacity.
		c.lastMeasured.advanceUpToBuf = work
	}
	c.lastMeasured.lastTick = c.nowTick()
	return vlStarts
}

// patchAdvanceUpTo returns newLine's advance-up-to array built from
// oldAdvanceUpTo (oldLine's array) by reshaping only the span that differs
// between oldLine and newLine. newAdvanceUpToBuf's backing array is reused
// when large enough. oldAdvanceUpTo and newAdvanceUpToBuf must not alias.
// reshapedLengthInBytes is the byte length of newLine that was reshaped; it
// is small for an edit of oldLine and approaches len(newLine) when newLine is
// unrelated to oldLine or has no internal line-break opportunities.
func patchAdvanceUpTo(oldAdvanceUpTo, newAdvanceUpToBuf []float64, oldLine, newLine string, face text.Face) (newAdvanceUpTo []float64, reshapedLengthInBytes int) {
	n := len(newLine) + 1
	newAdvanceUpTo = slices.Grow(newAdvanceUpToBuf[:0], n)[:n]
	delta := len(newLine) - len(oldLine)

	start, end := editSpan(oldLine, newLine)

	// Expand the window outward to the enclosing line-break opportunities in
	// newLine so any ligature, kerning, or cursive join touching the edit lies
	// inside the reshape: shaping context does not cross a line-break
	// opportunity, so once the window covers the nearest ones, the reused
	// prefix and suffix advances are independent of the edit. 0 is an implicit
	// break (start of line) used as the floor for windowStart.
	//
	// TODO: line-break opportunities are not perfectly safe shaping boundaries —
	// some fonts kern across spaces, and cursive scripts can join across longer
	// ranges. The fully rigorous fix is to use the shaper's safe-to-break
	// positions (HarfBuzz's HB_GLYPH_FLAG_UNSAFE_TO_BREAK), but Ebitengine's
	// text/v2 does not currently expose those.
	var windowStart int
	newWindowEnd := len(newLine)
	for b := range theSegmentCache.softLineBreakBoundaries(newLine) {
		if b <= start {
			windowStart = b
			continue
		}
		if b >= end {
			newWindowEnd = b
			break
		}
	}
	oldWindowEnd := newWindowEnd - delta

	// Prefix [0, windowStart]: clusters ending at or before windowStart are unchanged.
	copy(newAdvanceUpTo[:windowStart+1], oldAdvanceUpTo[:windowStart+1])

	// Window (windowStart, newWindowEnd]: reshape newLine[windowStart:newWindowEnd]
	// and accumulate from oldAdvanceUpTo[windowStart].
	for i := windowStart + 1; i <= newWindowEnd; i++ {
		newAdvanceUpTo[i] = 0
	}
	theLazyGlyphsBuffer = text.AppendLazyGlyphs(theLazyGlyphsBuffer[:0], newLine[windowStart:newWindowEnd], face, nil)
	for i := range theLazyGlyphsBuffer {
		g := &theLazyGlyphsBuffer[i]
		newAdvanceUpTo[windowStart+g.EndIndexInBytes] += g.AdvanceX
	}
	theLazyGlyphsBuffer = slices.Delete(theLazyGlyphsBuffer, 0, len(theLazyGlyphsBuffer))
	run := oldAdvanceUpTo[windowStart]
	for i := windowStart + 1; i <= newWindowEnd; i++ {
		run += newAdvanceUpTo[i]
		newAdvanceUpTo[i] = run
	}

	// Suffix (newWindowEnd, n): unchanged clusters, shifted by the window's width change.
	// newAdvanceUpTo[newWindowEnd]-oldAdvanceUpTo[oldWindowEnd] is that change (both measure from the same prefix).
	windowDelta := newAdvanceUpTo[newWindowEnd] - oldAdvanceUpTo[oldWindowEnd]
	for i := newWindowEnd + 1; i < n; i++ {
		newAdvanceUpTo[i] = oldAdvanceUpTo[i-delta] + windowDelta
	}
	return newAdvanceUpTo, newWindowEnd - windowStart
}

// editSpan returns the bounds in newLine of the change relative to oldLine:
// the changed window is newLine[start:end]. oldLine and newLine share a
// common prefix of length start and a non-overlapping common suffix.
// Identical strings give an empty window.
func editSpan(oldLine, newLine string) (start, end int) {
	m := min(len(oldLine), len(newLine))
	for start < m && oldLine[start] == newLine[start] {
		start++
	}
	oldEnd := len(oldLine)
	end = len(newLine)
	for oldEnd > start && end > start && oldLine[oldEnd-1] == newLine[end-1] {
		oldEnd--
		end--
	}
	return start, end
}
