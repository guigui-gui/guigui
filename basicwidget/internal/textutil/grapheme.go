// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package textutil

import (
	"iter"
)

func graphemes(str string) iter.Seq[string] {
	return func(yield func(s string) bool) {
		seg := pushSegmenter()
		defer popSegmenter()
		initSegmenterWithString(seg, str)
		it := seg.GraphemeIterator()
		for it.Next() {
			g := it.Grapheme()
			if !yield(string(g.Text)) {
				return
			}
		}
	}
}

func PrevPositionOnGraphemes(str string, position int) int {
	seg := pushSegmenter()
	defer popSegmenter()
	initSegmenterWithString(seg, str)
	it := seg.GraphemeIterator()
	var bytePos int
	for it.Next() {
		g := it.Grapheme()
		s := string(g.Text)
		startPos := bytePos
		endPos := bytePos + len(s)
		if position > endPos {
			bytePos = endPos
			continue
		}
		return startPos
	}
	return position
}

func NextPositionOnGraphemes(str string, position int) int {
	seg := pushSegmenter()
	defer popSegmenter()
	initSegmenterWithString(seg, str)
	it := seg.GraphemeIterator()
	var bytePos int
	for it.Next() {
		g := it.Grapheme()
		s := string(g.Text)
		startPos := bytePos
		endPos := bytePos + len(s)
		if position > startPos {
			bytePos = endPos
			continue
		}
		return endPos
	}
	return position
}
