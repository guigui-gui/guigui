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

func TestLineByteOffsetsRebuildFromString(t *testing.T) {
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
			l.RebuildFromString(tc.input)
			got := make([]int, l.LineCount())
			for i := range got {
				got[i] = l.ByteOffsetByLineIndex(i)
			}
			if !slices.Equal(got, tc.starts) {
				t.Errorf("RebuildFromString(%q): starts = %v, want %v", tc.input, got, tc.starts)
			}
		})
	}
}

func TestLineByteOffsetsRebuildIsIdempotent(t *testing.T) {
	var l textutil.LineByteOffsets
	l.RebuildFromString("abc\ndef\nghi")
	l.RebuildFromString("abc\ndef\nghi")
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
	l.RebuildFromString("abc\ndef\nghi")
	l.RebuildFromString("xyz")
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
	l.RebuildFromString("abc\ndef\nghi")

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
	l.RebuildFromString("abc\n")
	if got, want := l.LineCount(), 2; got != want {
		t.Fatalf("LineCount = %d, want %d", got, want)
	}
	if got, want := l.LineIndexForByteOffset(4), 1; got != want {
		t.Errorf("LineIndexForByteOffset(4) = %d, want %d", got, want)
	}
}

func TestLineByteOffsetsReset(t *testing.T) {
	var l textutil.LineByteOffsets
	l.RebuildFromString("abc\ndef")
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
			ref.RebuildFromString(s)
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
	l.RebuildFromString(s)

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
