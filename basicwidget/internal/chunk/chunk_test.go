// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package chunk_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/go-text/typesetting/segmenter"
	"github.com/guigui-gui/guigui/basicwidget/internal/chunk"
)

// Invisible and format codepoints, written as escapes so the source stays
// ASCII; the names keep the test cases readable.
const (
	combiningAcute  = "\u0301"     // COMBINING ACUTE ACCENT (a mark)
	zeroWidthJoiner = "\u200d"     // ZERO WIDTH JOINER (format)
	nonBreakSpace   = "\u00a0"     // NO-BREAK SPACE (glue)
	emojiModifier   = "\U0001f3fb" // EMOJI MODIFIER FITZPATRICK TYPE-1-2 (Extend, not Mark/Cf)
	kanaVoicedMark  = "\uff9e"     // HALFWIDTH KATAKANA VOICED SOUND MARK (Extend, not Mark/Cf)
)

func TestAppendBoundaries(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []int
	}{
		{
			name: "empty",
			in:   "",
			want: []int{0},
		},
		{
			name: "plain",
			in:   "hello",
			want: []int{5},
		},
		{
			name: "ATerm followed by space",
			in:   "Hello. World",
			want: []int{7, 12},
		},
		{
			name: "STerm followed by space",
			in:   "Hello! World",
			want: []int{7, 12},
		},
		{
			name: "STerm without space",
			in:   "Hello!World",
			want: []int{6, 11},
		},
		{
			name: "ATerm inside a number stays whole",
			in:   "3.14",
			want: []int{4},
		},
		{
			name: "terminator cluster",
			in:   "Eh?! No",
			want: []int{5, 7},
		},
		{
			name: "CJK ideographic full stop",
			in:   "こんにちは。世界",
			want: []int{18, 24},
		},
		{
			// One word per UAX #29 WB5, so it must not be cut mid-word.
			name: "mixed direction without a separator stays whole",
			in:   "abcשלום",
			want: []int{11},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chunk.AppendBoundaries(nil, tt.in)
			if !slices.Equal(got, tt.want) {
				t.Errorf("AppendBoundaries(nil, %q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestAppendBoundariesPartition(t *testing.T) {
	// Whatever the cuts are, the boundaries must be strictly increasing and
	// end at len(s) so the chunks exactly partition s. A combining mark and
	// a zero-width joiner are included so cuts near them exercise the
	// mark/format snapping.
	for _, s := range []string{
		"",
		"a",
		"Hello. World! How are you? 你好。",
		"abcשלום123! test" + combiningAcute + " ok" + zeroWidthJoiner + " done.",
	} {
		b := chunk.AppendBoundaries(nil, s)
		if len(b) == 0 {
			t.Fatalf("%q: no boundaries", s)
		}
		if b[len(b)-1] != len(s) {
			t.Errorf("%q: last boundary = %d, want %d", s, b[len(b)-1], len(s))
		}
		prev := 0
		for i, e := range b {
			if e < prev || (i > 0 && e == prev) {
				t.Errorf("%q: boundaries not strictly increasing: %v", s, b)
				break
			}
			prev = e
		}
	}
}

func TestAppendBoundariesWhitespaceFallback(t *testing.T) {
	const word = "abcde " // 6 bytes including the trailing space
	n := chunk.FallbackBytes/len(word) + 50
	s := strings.Repeat(word, n)

	b := chunk.AppendBoundaries(nil, s)
	if len(b) < 2 {
		t.Fatalf("expected the size fallback to split into multiple chunks, got %d", len(b))
	}
	if b[len(b)-1] != len(s) {
		t.Errorf("last boundary = %d, want %d", b[len(b)-1], len(s))
	}
	var prev int
	for _, e := range b {
		if size := e - prev; size > chunk.FallbackBytes+len(word) {
			t.Errorf("chunk [%d,%d) size %d exceeds fallback bound", prev, e, size)
		}
		prev = e
	}
	// Interior cuts must land right after a breakable space.
	for _, e := range b[:len(b)-1] {
		if s[e-1] != ' ' {
			t.Errorf("interior boundary %d does not follow a space (got %q)", e, s[e-1])
		}
	}
}

func TestAppendBoundariesNonBreakingSpaceIsNotACut(t *testing.T) {
	// A long run joined only by non-breaking spaces has no breakable space
	// and no terminator, so it must stay a single chunk: the fallback never
	// fires at glue.
	s := strings.Repeat("a"+nonBreakSpace, chunk.FallbackBytes)
	b := chunk.AppendBoundaries(nil, s)
	if want := []int{len(s)}; !slices.Equal(b, want) {
		t.Errorf("got %v, want single chunk %v", b, want)
	}
}

func TestSnapPastMarks(t *testing.T) {
	tests := []struct {
		name string
		s    string
		pos  int
		want int
	}{
		{
			name: "ascii base",
			s:    "abc",
			pos:  1,
			want: 1,
		},
		{
			// Skip U+0301 COMBINING ACUTE ACCENT (2 bytes).
			name: "combining mark",
			s:    "a" + combiningAcute + "b",
			pos:  1,
			want: 3,
		},
		{
			// Skip U+200D ZERO WIDTH JOINER (3 bytes, format).
			name: "zero width joiner",
			s:    "a" + zeroWidthJoiner + "b",
			pos:  1,
			want: 4,
		},
		{
			// Skip U+1F3FB EMOJI MODIFIER (4 bytes): an Extend outside Mark/Cf.
			name: "emoji modifier",
			s:    "a" + emojiModifier + "b",
			pos:  1,
			want: 5,
		},
		{
			// Skip U+FF9E katakana voiced sound mark (3 bytes): an Extend Lm.
			name: "halfwidth katakana sound mark",
			s:    "a" + kanaVoicedMark + "b",
			pos:  1,
			want: 4,
		},
		{
			name: "trailing mark at end",
			s:    "a" + combiningAcute,
			pos:  1,
			want: 3,
		},
		{
			name: "already on a base codepoint",
			s:    "a" + combiningAcute + "b",
			pos:  0,
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chunk.SnapPastMarks(tt.s, tt.pos); got != tt.want {
				t.Errorf("SnapPastMarks(%q, %d) = %d, want %d", tt.s, tt.pos, got, tt.want)
			}
		})
	}
}

// TestChunkBoundariesAreLineBreakableWordBoundaries asserts the invariant the
// cache relies on: every chunk boundary is both a UAX #29 word boundary and a
// UAX #14 line-break opportunity (hence a grapheme boundary too), so per-chunk
// segmentation reproduces whole-line segmentation exactly.
func TestChunkBoundariesAreLineBreakableWordBoundaries(t *testing.T) {
	for _, s := range []string{
		"abcשלום",                // LTR/RTL letters with no separator: one word per WB5
		"abc שלום",               // separated by a space
		"Hello. World",           // ATerm + space
		"Hello!World",            // STerm, no space
		"Hello!)",                // terminator glued to closing punctuation: no cut
		"He said \"Hi!\" loudly", // terminator glued to a closing quote: no cut
		"こんにちは。世界",               // CJK ideographic full stop
		"「こんにちは。」と言った",           // terminator glued to a CJK closing bracket: no cut
		"3.14 is roughly pi",     // ATerm inside a number stays whole
		"a (b) c.",
		strings.Repeat("word ", 1200), // exercises the size fallback
	} {
		words := wordBoundarySet(t, s)
		opps := lineBreakOpportunitySet(t, s)
		for _, b := range chunk.AppendBoundaries(nil, s) {
			if isWord, isLineBreak := words[b], opps[b]; !isWord || !isLineBreak {
				t.Errorf("%q: chunk boundary %d is not a line-breakable word boundary (word boundary=%t, line-break opportunity=%t; word boundaries=%v, line breaks=%v)",
					s, b, isWord, isLineBreak, sortedKeys(words), sortedKeys(opps))
			}
		}
	}
}

// lineBreakOpportunitySet returns the set of byte offsets at UAX #14
// line-break opportunities of s, including 0 and len(s).
func lineBreakOpportunitySet(t *testing.T, s string) map[int]bool {
	t.Helper()
	var seg segmenter.Segmenter
	if err := seg.InitWithString(s); err != nil {
		t.Fatalf("InitWithString(%q): %v", s, err)
	}
	set := map[int]bool{0: true, len(s): true}
	for it := seg.LineIterator(); it.Next(); {
		l := it.Line()
		set[l.OffsetInBytes] = true
		set[l.OffsetInBytes+l.LengthInBytes] = true
	}
	return set
}

// wordBoundarySet returns the set of byte offsets at UAX #29 word
// boundaries of s, including 0 and len(s).
func wordBoundarySet(t *testing.T, s string) map[int]bool {
	t.Helper()
	var seg segmenter.Segmenter
	if err := seg.InitWithString(s); err != nil {
		t.Fatalf("InitWithString(%q): %v", s, err)
	}
	set := map[int]bool{0: true, len(s): true}
	it := seg.WordIterator()
	for it.Next() {
		w := it.Word()
		set[w.OffsetInBytes] = true
		set[w.OffsetInBytes+w.LengthInBytes] = true
	}
	return set
}

func sortedKeys(m map[int]bool) []int {
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	slices.Sort(ks)
	return ks
}
