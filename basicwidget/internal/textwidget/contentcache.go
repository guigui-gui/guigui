// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"bytes"
	"hash"
	"hash/fnv"
	"io"

	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

type stringCacheEntry struct {
	valid        bool
	forRendering bool
	generation   int64
	start        int
	end          int
	str          string
}

// textContentCache memoizes data derived from a [textStore]'s content —
// substrings, the content hash, logical-line byte offsets, and the mask
// mapping — each keyed by [textStore.Generation] and rebuilt lazily after an
// edit.
type textContentCache struct {
	// valueBuilder is a reusable buffer for materializing store content.
	valueBuilder bytes.Buffer

	// valueEqualChecker is a reusable writer for comparing store content
	// against a string without materializing it.
	valueEqualChecker stringEqualChecker

	// stringCache memoizes substrings keyed by content generation and range
	// with round-robin replacement, reused until the next edit bumps the
	// generation.
	stringCache     [4]stringCacheEntry
	stringCacheNext int

	// hasher is a reusable streaming hasher used by contentHash to
	// fingerprint the current content.
	hasher hash.Hash

	// hashCache memoizes the most recently computed hash, keyed by
	// [textStore.Generation]. While the store has not been mutated, repeated
	// contentHash calls return the cached value without re-hashing.
	hashCache      [16]byte
	hashGeneration int64

	// lineByteOffsets holds the byte offset of each logical line start in the
	// committed text, refreshed lazily by ensureLineByteOffsets when
	// [textStore.Generation] advances past lineByteOffsetsGeneration.
	lineByteOffsets           textutil.LineByteOffsets
	lineByteOffsetsGeneration int64

	// maskMapping memoizes the masked rendering of the content, keyed by
	// [textStore.Generation], the mask rune, and whether the active IME
	// composition is included. A zero runeLen means it has not been built yet.
	maskMapping                maskMapping
	maskMappingGeneration      int64
	maskMappingRune            rune
	maskMappingWithComposition bool
}

// text returns the store's committed text.
func (c *textContentCache) text(store *textStore) string {
	c.valueBuilder.Reset()
	_, _ = store.WriteTextTo(&c.valueBuilder)
	return c.valueBuilder.String()
}

// textForRendering returns the store's rendering text: the committed text
// with the active IME composition spliced in.
func (c *textContentCache) textForRendering(store *textStore) string {
	c.valueBuilder.Reset()
	_, _ = store.WriteTextForRenderingTo(&c.valueBuilder)
	return c.valueBuilder.String()
}

// isEqualToText reports whether the store's committed text equals text.
func (c *textContentCache) isEqualToText(store *textStore, text string) bool {
	c.valueEqualChecker.Reset(text)
	_, _ = store.WriteTextTo(&c.valueEqualChecker)
	return c.valueEqualChecker.Result()
}

// stringWithRange returns the store substring [start, end) — rendering text
// when forRendering, else committed — reusing a cached copy until the next
// edit. A negative end means the text length.
func (c *textContentCache) stringWithRange(store *textStore, start, end int, forRendering bool) string {
	if end < 0 {
		end = store.TextLengthInBytes()
	}
	gen := store.Generation()
	for i := range c.stringCache {
		if e := &c.stringCache[i]; e.valid && e.generation == gen && e.forRendering == forRendering && e.start == start && e.end == end {
			return e.str
		}
	}
	c.valueBuilder.Reset()
	if forRendering {
		_, _ = store.WriteTextForRenderingRangeTo(&c.valueBuilder, start, end)
	} else {
		_, _ = store.WriteTextRangeTo(&c.valueBuilder, start, end)
	}
	str := c.valueBuilder.String()
	c.stringCache[c.stringCacheNext] = stringCacheEntry{
		valid:        true,
		forRendering: forRendering,
		generation:   gen,
		start:        start,
		end:          end,
		str:          str,
	}
	c.stringCacheNext = (c.stringCacheNext + 1) % len(c.stringCache)
	return str
}

// contentHash returns a 128-bit fingerprint of the store's rendering text
// (matching what drawing and measuring see).
func (c *textContentCache) contentHash(store *textStore) [16]byte {
	generation := store.Generation()
	if generation == c.hashGeneration {
		return c.hashCache
	}
	if c.hasher == nil {
		c.hasher = fnv.New128a()
	}
	c.hasher.Reset()
	_, _ = store.WriteTextForRenderingTo(c.hasher)
	var ch [16]byte
	c.hasher.Sum(ch[:0])
	c.hashCache = ch
	c.hashGeneration = generation
	return c.hashCache
}

// ensureLineByteOffsets refreshes lineByteOffsets if the store has been
// mutated since the last call. The offsets are built from the committed text
// only (no IME composition), matching what [textStore.WriteTextTo] returns.
func (c *textContentCache) ensureLineByteOffsets(store *textStore) {
	generation := store.Generation()
	if c.lineByteOffsets.LineCount() > 0 && generation == c.lineByteOffsetsGeneration {
		return
	}
	_ = c.lineByteOffsets.Rebuild(func(w io.Writer) error {
		_, err := store.WriteTextTo(w)
		return err
	})
	c.lineByteOffsetsGeneration = generation
}

// applyReplaceToLineByteOffsets folds an edit already applied to the store —
// text replacing [start, end) of the previous content — into lineByteOffsets
// incrementally. It is a no-op when the offsets have not been built yet.
func (c *textContentCache) applyReplaceToLineByteOffsets(store *textStore, text string, start, end int) {
	if c.lineByteOffsets.LineCount() == 0 {
		return
	}
	startCtx := c.stringWithRange(store, max(0, start-2), start, false)
	endCtxStart := start + len(text)
	endCtxEnd := endCtxStart + 3
	endCtx := c.stringWithRange(store, endCtxStart, endCtxEnd, false)
	atEOT := endCtxEnd >= store.TextLengthInBytes()
	c.lineByteOffsets.Replace(text, start, end, startCtx, endCtx, atEOT)
	c.lineByteOffsetsGeneration = store.Generation()
}

// maskMappingForRendering returns the [maskMapping] of the store's content
// masked with maskRune; withComposition selects whether the active IME
// composition is included. The returned pointer is owned by the cache and is
// invalidated by the next edit, so callers must not retain it.
func (c *textContentCache) maskMappingForRendering(store *textStore, maskRune rune, withComposition bool) *maskMapping {
	gen := store.Generation()
	if c.maskMapping.runeLen != 0 &&
		c.maskMappingGeneration == gen &&
		c.maskMappingRune == maskRune &&
		c.maskMappingWithComposition == withComposition {
		return &c.maskMapping
	}
	var src string
	if withComposition {
		src = c.textForRendering(store)
	} else {
		src = c.text(store)
	}
	c.maskMapping.reset(src, maskRune)
	c.maskMappingGeneration = gen
	c.maskMappingRune = maskRune
	c.maskMappingWithComposition = withComposition
	return &c.maskMapping
}

// stringEqualChecker is an [io.Writer] that compares the written bytes
// against a fixed string.
type stringEqualChecker struct {
	str    string
	pos    int
	result bool
}

func (s *stringEqualChecker) Reset(str string) {
	s.str = str
	s.pos = 0
	s.result = true
}

func (s *stringEqualChecker) Result() bool {
	if s.pos != len(s.str) {
		return false
	}
	return s.result
}

func (s *stringEqualChecker) Write(b []byte) (int, error) {
	if s.pos+len(b) > len(s.str) {
		s.result = false
		return 0, io.EOF
	}
	if !bytes.Equal([]byte(s.str[s.pos:s.pos+len(b)]), b) {
		s.result = false
		return 0, io.EOF
	}
	s.pos += len(b)
	return len(b), nil
}
