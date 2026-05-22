// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/go-text/typesetting/segmenter"

	"github.com/guigui-gui/guigui/basicwidget/internal/chunk"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// Codepoints written as escapes so the source stays ASCII.
const (
	combiningAcute = "\u0301"     // COMBINING ACUTE ACCENT
	zwj            = "\u200d"     // ZERO WIDTH JOINER
	emojiMan       = "\U0001f468" // MAN
	emojiWoman     = "\U0001f469" // WOMAN
	skinTone       = "\U0001f3fb" // EMOJI MODIFIER FITZPATRICK TYPE-1-2
)

// segmenterGraphemeBoundaries returns the end offset of each grapheme cluster
// of s reported by the segmenter: the reference the cache reproduces.
func segmenterGraphemeBoundaries(t *testing.T, s string) []int {
	t.Helper()
	var seg segmenter.Segmenter
	if err := seg.InitWithString(s); err != nil {
		t.Fatalf("InitWithString(%q): %v", s, err)
	}
	var out []int
	for it := seg.GraphemeIterator(); it.Next(); {
		g := it.Grapheme()
		out = append(out, g.OffsetInBytes+g.LengthInBytes)
	}
	return out
}

// segmenterSoftLineBreakBoundaries returns the end offset of each line segment of s
// reported by the segmenter: the reference the cache reproduces.
func segmenterSoftLineBreakBoundaries(t *testing.T, s string) []int {
	t.Helper()
	var seg segmenter.Segmenter
	if err := seg.InitWithString(s); err != nil {
		t.Fatalf("InitWithString(%q): %v", s, err)
	}
	var out []int
	for it := seg.LineIterator(); it.Next(); {
		l := it.Line()
		out = append(out, l.OffsetInBytes+l.LengthInBytes)
	}
	return out
}

// TestSegmentCacheMatchesWholeLine checks that the grapheme boundaries and
// line segments assembled from the per-chunk cache equal whole-line
// segmentation, and that a second pass (now served from the cache) is
// identical.
func TestSegmentCacheMatchesWholeLine(t *testing.T) {
	cases := []string{
		"",
		"a",
		"hello world",
		"Hello. World! How are you?",
		"こんにちは。世界",
		"abcשלום",      // mixed direction, a single word per WB5
		"abc שלום def", // separated
		"3.14 and 1,000 are numbers.",
		"cafe" + combiningAcute,       // base + combining mark: one grapheme
		emojiMan + zwj + emojiWoman,   // ZWJ sequence: one grapheme and one word
		emojiMan + skinTone,           // base + emoji modifier: one grapheme
		strings.Repeat("word ", 1200), // spans several chunks via the size fallback
		strings.Repeat("あ", 2000),     // long spaceless CJK in a single chunk
	}
	cg := textutil.NewSegmentCacheForTest(1<<20, nil)
	cl := textutil.NewSegmentCacheForTest(1<<20, nil)
	for pass := range 2 {
		for _, s := range cases {
			if got, want := cg.GraphemeBoundaries(s), segmenterGraphemeBoundaries(t, s); !slices.Equal(got, want) {
				t.Errorf("grapheme pass=%d %q:\n got=%v\nwant=%v", pass, s, got, want)
			}
			if got, want := cl.SoftLineBreakBoundaries(s), segmenterSoftLineBreakBoundaries(t, s); !slices.Equal(got, want) {
				t.Errorf("line pass=%d %q:\n got=%v\nwant=%v", pass, s, got, want)
			}
		}
	}
}

// TestSegmentCacheKeepsActiveEntries checks that entries used within the alive
// window are never evicted, even far over the soft limit — so wrapping one long
// line never evicts the chunks it is still using.
func TestSegmentCacheKeepsActiveEntries(t *testing.T) {
	c := textutil.NewSegmentCacheForTest(8, func() int64 { return 1000 })
	for i := range 50 {
		c.Add(fmt.Sprintf("chunk%02d", i))
	}
	if c.Len() != 50 {
		t.Errorf("active entries were evicted: have %d, want 50", c.Len())
	}
}

// TestSegmentCacheEvictsStaleEntries checks that once over the soft limit,
// entries idle past the alive window are dropped while recent ones stay.
func TestSegmentCacheEvictsStaleEntries(t *testing.T) {
	var tick int64
	c := textutil.NewSegmentCacheForTest(8, func() int64 { return tick })
	for i := range 50 {
		c.Add(fmt.Sprintf("old%02d", i))
	}
	tick = textutil.EntryAliveTicks + 1
	c.Add("fresh")

	if c.Has("old00") {
		t.Errorf("stale entry old00 was not evicted")
	}
	if !c.Has("fresh") {
		t.Errorf("fresh entry is missing")
	}
	if c.Len() != 1 {
		t.Errorf("expected only the fresh entry to remain, have %d", c.Len())
	}
}

// wholeLineWord returns the word span containing idx by scanning the entire
// line's words: the reference the chunk-bounded FindWordBoundaries reproduces.
func wholeLineWord(t *testing.T, s string, idx int) (start, end int) {
	t.Helper()
	var seg segmenter.Segmenter
	if err := seg.InitWithString(s); err != nil {
		t.Fatalf("InitWithString(%q): %v", s, err)
	}
	for it := seg.WordIterator(); it.Next(); {
		w := it.Word()
		if ws, we := w.OffsetInBytes, w.OffsetInBytes+w.LengthInBytes; ws <= idx && idx <= we {
			return ws, we
		}
	}
	return idx, idx
}

// TestFindWordBoundariesMatchesWholeLine verifies that searching for the word
// containing idx within only its chunk yields the same span as scanning the
// whole line, for every byte index across multi-chunk inputs.
func TestFindWordBoundariesMatchesWholeLine(t *testing.T) {
	inputs := []string{
		"Hello there. How are you? I am fine, thanks.",
		"こんにちは。世界は広い。今日はいい天気ですね。",
		"3.14 and 1,000 are numbers. So is 42!",
	}
	for _, s := range inputs {
		if n := len(chunk.AppendBoundaries(nil, s)); n < 2 {
			t.Fatalf("input produced %d chunk(s); want multiple", n)
		}
		for idx := 0; idx <= len(s); idx++ {
			wantStart, wantEnd := wholeLineWord(t, s, idx)
			gotStart, gotEnd := textutil.FindWordBoundaries(s, idx)
			if gotStart != wantStart || gotEnd != wantEnd {
				t.Fatalf("idx=%d: got (%d,%d), want (%d,%d)", idx, gotStart, gotEnd, wantStart, wantEnd)
			}
		}
	}
}
