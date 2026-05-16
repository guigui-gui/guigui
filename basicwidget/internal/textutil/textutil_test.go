// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package textutil_test

import (
	"bytes"
	"fmt"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

func TestNoWrapVisualLines(t *testing.T) {
	testCases := []struct {
		str       string
		positions []int
		lines     []string
	}{
		{
			str:       "Hello, World!",
			positions: []int{0},
			lines:     []string{"Hello, World!"},
		},
		{
			str:       "Hello,\nWorld!",
			positions: []int{0, 7},
			lines:     []string{"Hello,\n", "World!"},
		},
		{
			str:       "Hello,\nWorld!\n",
			positions: []int{0, 7, 14},
			lines:     []string{"Hello,\n", "World!\n", ""},
		},
		{
			str:       "Hello,\rWorld!",
			positions: []int{0, 7},
			lines:     []string{"Hello,\r", "World!"},
		},
		{
			str:       "Hello,\u0085World!",
			positions: []int{0, 8}, // U+0085 is 2 bytes in UTF-8.
			lines:     []string{"Hello,\u0085", "World!"},
		},
		{
			str:       "Hello,\n\nWorld!",
			positions: []int{0, 7, 8},
			lines:     []string{"Hello,\n", "\n", "World!"},
		},
		{
			str:       "Hello,\r\nWorld!",
			positions: []int{0, 8},
			lines:     []string{"Hello,\r\n", "World!"},
		},
		{
			str:       "Hello,\n\rWorld!",
			positions: []int{0, 7, 8},
			lines:     []string{"Hello,\n", "\r", "World!"},
		},
		{
			str:       "",
			positions: []int{0},
			lines:     []string{""},
		},
		{
			str:       "\n",
			positions: []int{0, 1},
			lines:     []string{"\n", ""},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.str, func(t *testing.T) {
			var gotPositions []int
			var gotLines []string
			for l := range textutil.VisualLines(0, tc.str, textutil.WrapModeNone, nil) {
				gotPositions = append(gotPositions, l.Pos)
				gotLines = append(gotLines, l.Str)
			}
			if !slices.Equal(gotPositions, tc.positions) {
				t.Errorf("got positions %v, want %v", gotPositions, tc.positions)
			}
			if !slices.Equal(gotLines, tc.lines) {
				t.Errorf("got lines %v, want %v", gotLines, tc.lines)
			}
		})
	}
}

// TestAutoWrapInvalidUTF8 guards against a panic when auto-wrapping a string
// that is not valid UTF-8. The segmenter sanitizes its input by replacing
// invalid bytes with U+FFFD (3 bytes), so its byte offsets/segment lengths
// would otherwise grow past the original string's length and cause an
// out-of-range slice when iterating lines.
func TestAutoWrapInvalidUTF8(t *testing.T) {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	face := &text.GoTextFace{Source: source, Size: 16}
	advance := func(s string, indexInBytes int) float64 {
		return text.AdvanceAt(s, indexInBytes, face)
	}

	cases := []string{
		"\xff",
		"abc\xffdef",
		"hello\xff\xfe\xfdworld",
		"\xff\xff\xff\xff\xff\xff\xff\xff",
	}
	for _, s := range cases {
		t.Run(fmt.Sprintf("%q", s), func(t *testing.T) {
			// Iterating must not panic. The exact line breakdown is not part
			// of the contract; this only verifies that Lines completes.
			for range textutil.VisualLines(100, s, textutil.WrapModeWord, advance) {
			}
		})
	}
}

func TestFindWordBoundaries(t *testing.T) {
	testCases := []struct {
		text      string
		idx       int
		wantStart int
		wantEnd   int
	}{
		// Basic word selection
		{
			text:      "hello",
			idx:       0,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			text:      "hello",
			idx:       2,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			text:      "hello",
			idx:       4,
			wantStart: 0,
			wantEnd:   5,
		},
		// Clicking at the end of a word should select that word
		{
			text:      "hello",
			idx:       5,
			wantStart: 0,
			wantEnd:   5,
		},
		// Words with spaces between them
		{
			text:      "hello world",
			idx:       0,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			text:      "hello world",
			idx:       3,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			text:      "hello world",
			idx:       4,
			wantStart: 0,
			wantEnd:   5,
		},
		// Clicking at the end of the first word (before space)
		{
			text:      "hello world",
			idx:       5,
			wantStart: 0,
			wantEnd:   5,
		},
		// Clicking at the start of the second word
		{
			text:      "hello world",
			idx:       6,
			wantStart: 6,
			wantEnd:   11,
		},
		{
			text:      "hello world",
			idx:       8,
			wantStart: 6,
			wantEnd:   11,
		},
		// Clicking at the end of the second word
		{
			text:      "hello world",
			idx:       11,
			wantStart: 6,
			wantEnd:   11,
		},
		// Japanese katakana: "テスト" is treated as a single word (9 bytes)
		{
			text:      "テスト",
			idx:       0,
			wantStart: 0,
			wantEnd:   9,
		},
		{
			text:      "テスト",
			idx:       3,
			wantStart: 0,
			wantEnd:   9,
		},
		{
			text:      "テスト",
			idx:       9,
			wantStart: 0,
			wantEnd:   9,
		},
		// Japanese with a space: the second word starts at byte 10.
		// This tests the bug where manual bytePos tracking skipped non-word bytes.
		// "日本語 テスト": "日" [0,3), "語" [6,9), " " [9,10), "テスト" [10,19)
		{
			text:      "日本語 テスト",
			idx:       10,
			wantStart: 10,
			wantEnd:   19,
		},
		{
			text:      "日本語 テスト",
			idx:       14,
			wantStart: 10,
			wantEnd:   19,
		},
		{
			text:      "日本語 テスト",
			idx:       19,
			wantStart: 10,
			wantEnd:   19,
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%q/%d", tc.text, tc.idx), func(t *testing.T) {
			gotStart, gotEnd := textutil.FindWordBoundaries(tc.text, tc.idx)
			if gotStart != tc.wantStart || gotEnd != tc.wantEnd {
				t.Errorf("got (%d, %d), want (%d, %d)", gotStart, gotEnd, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

func TestNextIndentPosition(t *testing.T) {
	testCases := []struct {
		position    float64
		indentWidth float64
		expected    float64
	}{
		{
			position:    0,
			indentWidth: 10.5,
			expected:    10.5,
		},
		{
			position:    104,
			indentWidth: 10.5,
			expected:    105,
		},
		{
			position:    104.9995,
			indentWidth: 10.5,
			expected:    105,
		},
		{
			position:    105,
			indentWidth: 10.5,
			expected:    115.5,
		},
		{
			position:    105.0001,
			indentWidth: 10.5,
			expected:    115.5,
		},
		{
			position:    106,
			indentWidth: 10.5,
			expected:    115.5,
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("position=%f indentWidth=%f", tc.position, tc.indentWidth), func(t *testing.T) {
			got := textutil.NextIndentPosition(tc.position, tc.indentWidth)
			if got != tc.expected {
				t.Errorf("got %f, want %f", got, tc.expected)
			}
		})
	}
}

func TestFirstLineBreakPositionAndLen(t *testing.T) {
	testCases := []struct {
		str        string
		wantPos    int
		wantLength int
	}{
		{"", -1, 0},
		{"abc", -1, 0},
		{"abc\ndef", 3, 1},
		{"abc\rdef", 3, 1},
		{"abc\r\ndef", 3, 2},
		{"\ndef", 0, 1},
		{"abc\vdef", 3, 1},
		{"abc\fdef", 3, 1},
		{"abc\u0085def", 3, 2},
		{"abc\u2028def", 3, 3},
		{"abc\u2029def", 3, 3},
		{"abc\ndef\nghi", 3, 1},
		// Trailing line breaks with no following byte: the \r\n
		// look-ahead must not read past the end of str.
		{"abc\r", 3, 1},
		{"abc\n", 3, 1},
		{"abc\r\n", 3, 2},
		{"\r", 0, 1},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%q", tc.str), func(t *testing.T) {
			gotPos, gotLen := textutil.FirstLineBreakPositionAndLen(tc.str)
			if gotPos != tc.wantPos || gotLen != tc.wantLength {
				t.Errorf("got (%d, %d), want (%d, %d)", gotPos, gotLen, tc.wantPos, tc.wantLength)
			}
		})
	}
}

func TestTrimPartialUTF8Prefix(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "",
		},
		{
			name: "all ASCII",
			in:   "abc",
			want: "abc",
		},
		{
			name: "valid multibyte at start",
			in:   "\xc3\xa9abc",
			want: "\xc3\xa9abc",
		},
		// "\xa9" alone is a continuation byte (10xxxxxx) that lost its lead.
		{
			name: "single stray continuation",
			in:   "\xa9abc",
			want: "abc",
		},
		// Two leading continuation bytes (3-byte sequence cut in half).
		{
			name: "two stray continuations",
			in:   "\xa8\xa9abc",
			want: "abc",
		},
		// Three leading continuation bytes (4-byte sequence cut after lead).
		{
			name: "three stray continuations",
			in:   "\xa8\xa9\xaaabc",
			want: "abc",
		},
		// All-continuation strings get fully trimmed.
		{
			name: "only continuations",
			in:   "\xa8\xa9",
			want: "",
		},
		// Lead byte at the start is kept (the function only strips
		// continuation bytes).
		{
			name: "lead at start",
			in:   "\xc3abc",
			want: "\xc3abc",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := textutil.TrimPartialUTF8Prefix(tc.in); got != tc.want {
				t.Errorf("TrimPartialUTF8Prefix(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestTrimPartialUTF8Suffix(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "",
		},
		{
			name: "all ASCII",
			in:   "abc",
			want: "abc",
		},
		// Complete sequences at the end stay.
		{
			name: "complete 2-byte",
			in:   "abc\xc3\xa9",
			want: "abc\xc3\xa9",
		},
		{
			name: "complete 3-byte",
			in:   "abc\xe2\x98\x83",
			want: "abc\xe2\x98\x83",
		},
		{
			name: "complete 4-byte",
			in:   "abc\xf0\x9f\x98\x80",
			want: "abc\xf0\x9f\x98\x80",
		},
		// Trailing lone lead byte: drop it.
		{
			name: "lone 2-byte lead",
			in:   "abc\xc3",
			want: "abc",
		},
		{
			name: "lone 3-byte lead",
			in:   "abc\xe2",
			want: "abc",
		},
		{
			name: "lone 4-byte lead",
			in:   "abc\xf0",
			want: "abc",
		},
		// 3-byte sequence with only one continuation byte: drop both.
		{
			name: "truncated 3-byte",
			in:   "abc\xe2\x98",
			want: "abc",
		},
		// 4-byte sequence with two of three continuation bytes: drop all three.
		{
			name: "truncated 4-byte",
			in:   "abc\xf0\x9f\x98",
			want: "abc",
		},
		// ASCII followed by stray continuation bytes: keep through ASCII.
		{
			name: "ASCII then stray cont",
			in:   "abc\x80",
			want: "abc",
		},
		{
			name: "ASCII then 2 stray cont",
			in:   "abc\x80\x80",
			want: "abc",
		},
		{
			name: "ASCII then 3 stray cont",
			in:   "abc\x80\x80\x80",
			want: "abc",
		},
		// Invalid lead byte (>= 0xF8) treated as a 4-byte lead and stripped
		// when its continuation bytes don't follow.
		{
			name: "invalid lead alone",
			in:   "abc\xff",
			want: "abc",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := textutil.TrimPartialUTF8Suffix(tc.in); got != tc.want {
				t.Errorf("TrimPartialUTF8Suffix(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestLastLineBreakPositionAndLen(t *testing.T) {
	testCases := []struct {
		str        string
		wantPos    int
		wantLength int
	}{
		{"", -1, 0},
		{"abc", -1, 0},
		{"abc\ndef", 3, 1},
		{"abc\rdef", 3, 1},
		{"abc\r\ndef", 3, 2},
		{"\ndef", 0, 1},
		{"abc\vdef", 3, 1},
		{"abc\fdef", 3, 1},
		{"abc\u0085def", 3, 2},
		{"abc\u2028def", 3, 3},
		{"abc\u2029def", 3, 3},
		{"abc\ndef\nghi", 7, 1},
		{"abc\ndef\r\nghi", 7, 2},
		{"abc\n", 3, 1},
		{"\n", 0, 1},
		{"\r\n", 0, 2},
		{"abc\ndef\n", 7, 1},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%q", tc.str), func(t *testing.T) {
			gotPos, gotLen := textutil.LastLineBreakPositionAndLen(tc.str)
			if gotPos != tc.wantPos || gotLen != tc.wantLength {
				t.Errorf("got (%d, %d), want (%d, %d)", gotPos, gotLen, tc.wantPos, tc.wantLength)
			}
		})
	}
}
