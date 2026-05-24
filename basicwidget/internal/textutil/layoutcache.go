// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
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
	text             string
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
}

type layoutCacheEntry struct {
	vlStarts []int
	// lastTick is the tick the entry was last created or read.
	lastTick int64
}

func (c *layoutCache) nowTick() int64 {
	if c.now == nil {
		return ebiten.Tick()
	}
	return c.now()
}

// get returns the cached visual-line starts for k, or ok=false on a miss. A
// zero faceID is always a miss: callers without a resolved face do not cache.
func (c *layoutCache) get(k layoutKey) ([]int, bool) {
	if k.faceID == 0 {
		return nil, false
	}
	if e, ok := c.entries[k]; ok {
		e.lastTick = c.nowTick()
		return e.vlStarts, true
	}
	return nil, false
}

// put stores vlStarts for k. A zero faceID is not stored. The returned slice
// must not be mutated by callers; get hands back the same slice.
func (c *layoutCache) put(k layoutKey, vlStarts []int) {
	if k.faceID == 0 {
		return
	}
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
}
