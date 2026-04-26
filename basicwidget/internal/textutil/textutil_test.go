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

func TestNoWrapLines(t *testing.T) {
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
			for l := range textutil.Lines(0, tc.str, false, nil) {
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

func TestTextPositionFromIndex(t *testing.T) {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	face := &text.GoTextFace{Source: source, Size: 16}
	const lineHeight = 24.0
	op := &textutil.Options{
		Face:       face,
		LineHeight: lineHeight,
	}

	// Baseline: position at index 0 of a single-line string sits on visual
	// line 0. Use it to derive line N's Top without hard-coding the face's
	// vertical padding.
	baseline, _, _ := textutil.TextPositionFromIndex(1000, "a", 0, op)
	topOfLine := func(n int) float64 {
		return baseline.Top + float64(n)*lineHeight
	}

	testCases := []struct {
		name      string
		text      string
		index     int
		wantCount int
		// Visual line index for each returned position (0 = first line).
		// -1 means "don't check".
		wantLine0 int
		wantLine1 int
		// Whether pos0.X / pos1.X must be 0 (line start).
		wantPos0XZero bool
		wantPos1XZero bool
	}{
		{
			name:          "single-line/start",
			text:          "abc",
			index:         0,
			wantCount:     1,
			wantLine0:     0,
			wantLine1:     -1,
			wantPos0XZero: true,
		},
		{
			name:      "single-line/end",
			text:      "abc",
			index:     3,
			wantCount: 1,
			wantLine0: 0,
			wantLine1: -1,
		},
		{
			name:          "trailing-newline/end",
			text:          "a\n",
			index:         2,
			wantCount:     2,
			wantLine0:     0, // tail of "a\n" — must be on line 0 with X > 0
			wantLine1:     1, // head of empty line — at start of line 1
			wantPos1XZero: true,
		},
		{
			name:          "trailing-newline/start",
			text:          "a\n",
			index:         0,
			wantCount:     1,
			wantLine0:     0,
			wantLine1:     -1,
			wantPos0XZero: true,
		},
		{
			name:          "mid-newline-boundary",
			text:          "a\nb",
			index:         2,
			wantCount:     2,
			wantLine0:     0,
			wantLine1:     1,
			wantPos1XZero: true,
		},
		{
			name:          "empty",
			text:          "",
			index:         0,
			wantCount:     1,
			wantLine0:     0,
			wantLine1:     -1,
			wantPos0XZero: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pos0, pos1, count := textutil.TextPositionFromIndex(1000, tc.text, tc.index, op)
			if count != tc.wantCount {
				t.Fatalf("count: got %d, want %d", count, tc.wantCount)
			}
			if tc.wantLine0 >= 0 {
				if want := topOfLine(tc.wantLine0); pos0.Top != want {
					t.Errorf("pos0.Top: got %v, want %v (line %d)", pos0.Top, want, tc.wantLine0)
				}
			}
			if tc.wantLine1 >= 0 && count == 2 {
				if want := topOfLine(tc.wantLine1); pos1.Top != want {
					t.Errorf("pos1.Top: got %v, want %v (line %d)", pos1.Top, want, tc.wantLine1)
				}
			}
			if tc.wantPos0XZero && pos0.X != 0 {
				t.Errorf("pos0.X: got %v, want 0", pos0.X)
			}
			if tc.wantPos1XZero && count == 2 && pos1.X != 0 {
				t.Errorf("pos1.X: got %v, want 0", pos1.X)
			}
			// For "trailing-newline/end" specifically, the tail (pos0) must have
			// a non-zero X (after "a"), otherwise the selection rendering would
			// draw width=0 — the bug this regression test is guarding.
			if tc.name == "trailing-newline/end" && pos0.X <= 0 {
				t.Errorf("pos0.X for trailing newline tail: got %v, want > 0", pos0.X)
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
