// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// rebuildFromString is a test helper that rescans s for logical-line starts
// via [textutil.LineByteOffsets.Rebuild].
func rebuildFromString(l *textutil.LineByteOffsets, s string) {
	_ = l.Rebuild(func(w io.Writer) error {
		_, err := w.Write([]byte(s))
		return err
	})
}

func TestLineByteOffsetsRebuild(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		starts []int
	}{
		{
			name:   "empty",
			input:  "",
			starts: []int{0},
		},
		{
			name:   "single line no break",
			input:  "abc",
			starts: []int{0},
		},
		{
			name:   "trailing LF",
			input:  "abc\n",
			starts: []int{0, 4},
		},
		{
			name:   "two lines",
			input:  "abc\ndef",
			starts: []int{0, 4},
		},
		{
			name:   "two lines with trailing break",
			input:  "abc\ndef\n",
			starts: []int{0, 4, 8},
		},
		{
			name:   "lone LF",
			input:  "\n",
			starts: []int{0, 1},
		},
		{
			name:   "consecutive breaks",
			input:  "\n\n\n",
			starts: []int{0, 1, 2, 3},
		},
		{
			name:   "CRLF",
			input:  "abc\r\ndef",
			starts: []int{0, 5},
		},
		{
			name:   "CR alone",
			input:  "abc\rdef",
			starts: []int{0, 4},
		},
		{
			name:   "U+2028 line separator",
			input:  "abc\u2028def",
			starts: []int{0, 6},
		},
		{
			name:   "U+0085 NEL",
			input:  "abc\u0085def",
			starts: []int{0, 5},
		},
		{
			name:   "VT and FF",
			input:  "a\vb\fc",
			starts: []int{0, 2, 4},
		},
		{
			name:   "multibyte runes between breaks",
			input:  "一\n二",
			starts: []int{0, 4},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var l textutil.LineByteOffsets
			rebuildFromString(&l, tc.input)
			got := make([]int, l.LineCount())
			for i := range got {
				got[i] = l.ByteOffsetByLineIndex(i)
			}
			if !slices.Equal(got, tc.starts) {
				t.Errorf("Rebuild(%q): starts = %v, want %v", tc.input, got, tc.starts)
			}
		})
	}
}

func TestLineByteOffsetsRebuildIsIdempotent(t *testing.T) {
	var l textutil.LineByteOffsets
	rebuildFromString(&l, "abc\ndef\nghi")
	rebuildFromString(&l, "abc\ndef\nghi")
	want := []int{0, 4, 8}
	got := make([]int, l.LineCount())
	for i := range got {
		got[i] = l.ByteOffsetByLineIndex(i)
	}
	if !slices.Equal(got, want) {
		t.Errorf("after double rebuild: starts = %v, want %v", got, want)
	}
}

func TestLineByteOffsetsRebuildAfterShrink(t *testing.T) {
	var l textutil.LineByteOffsets
	rebuildFromString(&l, "abc\ndef\nghi")
	rebuildFromString(&l, "xyz")
	want := []int{0}
	got := make([]int, l.LineCount())
	for i := range got {
		got[i] = l.ByteOffsetByLineIndex(i)
	}
	if !slices.Equal(got, want) {
		t.Errorf("after shrink: starts = %v, want %v", got, want)
	}
}

func TestLineByteOffsetsLineIndexForByteOffset(t *testing.T) {
	var l textutil.LineByteOffsets
	// "abc\ndef\nghi" — line 0 covers bytes 0..3 (incl. '\n' at 3),
	// line 1 covers bytes 4..7 (incl. '\n' at 7), line 2 covers bytes 8..10.
	rebuildFromString(&l, "abc\ndef\nghi")

	testCases := []struct {
		offset   int
		wantLine int
	}{
		{-5, 0},
		{0, 0},
		{1, 0},
		{3, 0}, // the '\n' belongs to its own line
		{4, 1}, // first byte of "def"
		{7, 1},
		{8, 2},
		{10, 2},
		{100, 2}, // past end clamps to last line
	}
	for _, tc := range testCases {
		got := l.LineIndexForByteOffset(tc.offset)
		if got != tc.wantLine {
			t.Errorf("LineIndexForByteOffset(%d) = %d, want %d", tc.offset, got, tc.wantLine)
		}
	}
}

func TestLineByteOffsetsLineIndexForByteOffsetTrailingBreak(t *testing.T) {
	var l textutil.LineByteOffsets
	// "abc\n" has two logical lines; the second is empty and starts at byte 4.
	rebuildFromString(&l, "abc\n")
	if got, want := l.LineCount(), 2; got != want {
		t.Fatalf("LineCount = %d, want %d", got, want)
	}
	if got, want := l.LineIndexForByteOffset(4), 1; got != want {
		t.Errorf("LineIndexForByteOffset(4) = %d, want %d", got, want)
	}
}

func TestLineByteOffsetsReset(t *testing.T) {
	var l textutil.LineByteOffsets
	rebuildFromString(&l, "abc\ndef")
	l.Reset()
	if got := l.LineCount(); got != 0 {
		t.Errorf("after Reset: LineCount = %d, want 0", got)
	}
}

func TestLineByteOffsetsStreamingMatchesRebuild(t *testing.T) {
	// Inputs that exercise every line-break shape, including multi-byte
	// sequences that may straddle chunk boundaries.
	inputs := []string{
		"",
		"abc",
		"abc\n",
		"abc\ndef",
		"abc\ndef\n",
		"\n",
		"\n\n\n",
		"abc\r\ndef",
		"abc\rdef",
		"abc\r",
		"\r\n",
		"\r\r\n",
		"abc\u2028def",
		"abc\u2029def",
		"abc\u0085def",
		"a\vb\fc",
		"一\n二",
		"\u2028\u2028\u2028",
		"a\u2028b\u2029c\u0085d",
		// 0xC2 / 0xE2 followed by non-break continuations to exercise
		// the false-positive paths.
		"a\u00a0b", // NBSP starts with 0xC2 0xA0.
		"a‰b‱c",    // PER-MILLE / PER-TEN-THOUSAND start with 0xE2 0x80 0xB0/0xB1.
	}
	for _, s := range inputs {
		t.Run(s, func(t *testing.T) {
			var ref textutil.LineByteOffsets
			rebuildFromString(&ref, s)
			want := make([]int, ref.LineCount())
			for i := range want {
				want[i] = ref.ByteOffsetByLineIndex(i)
			}

			// Try every possible single chunk split.
			for split := 0; split <= len(s); split++ {
				var l textutil.LineByteOffsets
				err := l.Rebuild(func(w io.Writer) error {
					if _, err := w.Write([]byte(s[:split])); err != nil {
						return err
					}
					_, err := w.Write([]byte(s[split:]))
					return err
				})
				if err != nil {
					t.Fatalf("Rebuild(%q split %d): %v", s, split, err)
				}
				got := make([]int, l.LineCount())
				for i := range got {
					got[i] = l.ByteOffsetByLineIndex(i)
				}
				if !slices.Equal(got, want) {
					t.Errorf("streaming %q split at %d: got %v, want %v", s, split, got, want)
				}
			}

			// One-byte-at-a-time streaming: the most fragmented split.
			var l textutil.LineByteOffsets
			err := l.Rebuild(func(w io.Writer) error {
				for j := range len(s) {
					if _, err := w.Write([]byte{s[j]}); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				t.Fatalf("Rebuild(%q byte-by-byte): %v", s, err)
			}
			got := make([]int, l.LineCount())
			for i := range got {
				got[i] = l.ByteOffsetByLineIndex(i)
			}
			if !slices.Equal(got, want) {
				t.Errorf("streaming %q byte-by-byte: got %v, want %v", s, got, want)
			}
		})
	}
}

func TestLineByteOffsetsReplaceMatchesRebuild(t *testing.T) {
	// Inputs span the cases that exercise every line-break shape, including
	// breaks that may sit at or straddle the splice's boundaries.
	bases := []string{
		"",
		"abc",
		"abc\n",
		"abc\ndef",
		"abc\ndef\n",
		"\n\n\n",
		"abc\r\ndef",
		"abc\rdef",
		"abc\r",
		"\r\n",
		"\r\r\n",
		"a\u2028b\u2028c",
		"a\u0085b\u0085c",
		"a\u2029b\u2029c",
		"a\vb\fc",
		"一\n二\n三",
		"a‰b‱c\nd",
		// Adjacent multi-byte breaks of mixed kinds.
		"\u0085\u2028\u2029",
		"a\u0085\u2028b",
	}
	patches := []string{
		"",
		"X",
		"\n",
		"\r",
		"\r\n",
		"\u2028",
		"\u0085",
		"\u2029",
		"X\nY",
		"\nX\n",
		"\n\n",
		// Partial / orphan break-lead bytes that may combine with
		// surrounding base bytes to form (or fail to form) a break.
		"\xc2",
		"\x85",
		"\xe2",
		"\xe2\x80",
		"\xa8",
	}
	for _, base := range bases {
		for _, patch := range patches {
			for start := 0; start <= len(base); start++ {
				for end := start; end <= len(base); end++ {
					post := base[:start] + patch + base[end:]

					var l textutil.LineByteOffsets
					rebuildFromString(&l, base)
					startCtx := base[max(0, start-2):start]
					endCtxStart := start + len(patch)
					endCtxEnd := min(endCtxStart+3, len(post))
					endCtx := post[endCtxStart:endCtxEnd]
					atEOT := endCtxStart+3 >= len(post)
					l.Replace(patch, start, end, startCtx, endCtx, atEOT)
					gotReplace := make([]int, l.LineCount())
					for i := range gotReplace {
						gotReplace[i] = l.ByteOffsetByLineIndex(i)
					}

					var ref textutil.LineByteOffsets
					rebuildFromString(&ref, post)
					want := make([]int, ref.LineCount())
					for i := range want {
						want[i] = ref.ByteOffsetByLineIndex(i)
					}

					if !slices.Equal(gotReplace, want) {
						t.Errorf("base=%q splice [%d,%d)=%q -> post=%q: Replace=%v, want %v",
							base, start, end, patch, post, gotReplace, want)
					}
				}
			}
		}
	}
}

func TestLineByteOffsetsReplaceSequence(t *testing.T) {
	// Apply a sequence of edits and verify the offsets stay in sync with a
	// fresh rebuild of the cumulative result, exercising suffix shifts.
	cur := "alpha\nbeta\ngamma\n"
	var l textutil.LineByteOffsets
	rebuildFromString(&l, cur)

	steps := []struct {
		start, end int
		patch      string
	}{
		{
			start: 6,
			end:   6,
			patch: "BB",
		},
		{
			start: 0,
			end:   5,
			patch: "ALPHA",
		},
		{
			start: 8,
			end:   8,
			patch: "Q\n",
		},
		{
			start: 2,
			end:   8,
			patch: "",
		},
		{
			start: 0,
			end:   0,
			patch: "\r\n",
		},
		{
			start: 0,
			end:   1,
			patch: "",
		},
	}
	for i, s := range steps {
		next := cur[:s.start] + s.patch + cur[s.end:]
		startCtx := cur[max(0, s.start-2):s.start]
		endCtxStart := s.start + len(s.patch)
		endCtxEnd := min(endCtxStart+3, len(next))
		endCtx := next[endCtxStart:endCtxEnd]
		atEOT := endCtxStart+3 >= len(next)
		l.Replace(s.patch, s.start, s.end, startCtx, endCtx, atEOT)
		got := make([]int, l.LineCount())
		for j := range got {
			got[j] = l.ByteOffsetByLineIndex(j)
		}
		var ref textutil.LineByteOffsets
		rebuildFromString(&ref, next)
		want := make([]int, ref.LineCount())
		for j := range want {
			want[j] = ref.ByteOffsetByLineIndex(j)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("step %d (splice [%d,%d)=%q on %q): got %v, want %v",
				i, s.start, s.end, s.patch, cur, got, want)
		}
		cur = next
	}
}

func TestLineByteOffsetsLargeBuffer(t *testing.T) {
	// Sanity-check correctness on a multi-thousand-line buffer that exercises
	// the binary-search path without being slow.
	const n = 5000
	var b strings.Builder
	for range n {
		b.WriteString("line")
		b.WriteByte('\n')
	}
	s := b.String()

	var l textutil.LineByteOffsets
	rebuildFromString(&l, s)

	// "line\n" is 5 bytes, so line i starts at byte 5*i.
	if got, want := l.LineCount(), n+1; got != want {
		t.Fatalf("LineCount = %d, want %d", got, want)
	}
	for i := range n {
		if got, want := l.ByteOffsetByLineIndex(i), 5*i; got != want {
			t.Fatalf("ByteOffsetByLineIndex(%d) = %d, want %d", i, got, want)
		}
	}
	// Offset in the middle of line 1234 ("line\n" mid-content).
	if got, want := l.LineIndexForByteOffset(5*1234+2), 1234; got != want {
		t.Errorf("LineIndexForByteOffset = %d, want %d", got, want)
	}
}
