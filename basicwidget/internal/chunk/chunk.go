// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

// Package chunk places cache-friendly boundaries within a single logical
// line of text. Callers use each chunk's bytes as the key of a per-chunk
// segmentation cache: an edit confined to one chunk re-segments only that
// chunk rather than the whole line.
//
// Every boundary is at once a UAX #29 grapheme boundary, a UAX #29 word
// boundary, and a UAX #14 line-break opportunity. Cuts fall only at breakable
// whitespace or sentence terminators, are snapped past trailing combining
// marks and format characters, and are placed only where the next chunk
// begins with letter or number content. That keeps per-chunk segmentation
// equal to whole-line segmentation for all three boundary kinds — a cut
// landing mid-segment would inject a boundary the whole-line pass would not
// have. Boundary placement is a pure function of the input; it is independent
// of width, wrap mode, and any layout state.
package chunk

import (
	"unicode"
	"unicode/utf8"
)

// fallbackBytes is the soft upper bound on chunk size. Once a chunk reaches
// this many bytes without hitting a sentence terminator, the chunker cuts
// at the next breakable space so a long terminator-free span (and the edits
// inside it) re-segments only a portion rather than the whole span.
const fallbackBytes = 4096

// AppendBoundaries appends to dst the exclusive end byte offset of each
// chunk of s and returns the extended slice. The chunks partition s: chunk
// i spans s[start:b[i]] where b is the appended boundaries, start is the
// previous boundary, and 0 for the first chunk. The last appended boundary
// is always len(s). An empty s yields a single boundary of 0 (one empty
// chunk).
func AppendBoundaries(dst []int, s string) []int {
	if len(s) == 0 {
		return append(dst, 0)
	}

	var chunkStart int

	for i := 0; i < len(s); {
		r, w := utf8.DecodeRuneInString(s[i:])

		// Sentence terminator: cut after the terminator cluster.
		if isATerm(r) || isSTerm(r) {
			clusterEnd, hasSTerm := extendTerminatorCluster(s, i)
			cut := -1
			if ws := skipBreakableSpaces(s, clusterEnd); ws > clusterEnd {
				// Followed by whitespace: cut after it, so the space stays
				// with the preceding chunk and the seam lands on a
				// breakable position.
				cut = ws
			} else if hasSTerm {
				// A strong terminator (! ? 。 ！ ？ …) with no trailing
				// space is still a sentence end; cut right after it. An
				// ATerm-only cluster (.) is not, so "3.14" stays whole.
				cut = clusterEnd
			}
			if cut >= 0 {
				if cut = snapPastMarks(s, cut); cut > chunkStart && beginsWordContent(s, cut) {
					dst = append(dst, cut)
					chunkStart = cut
					i = cut
					continue
				}
			}
			i = clusterEnd
			continue
		}

		// Breakable-whitespace size fallback.
		if isBreakableSpace(r) && i+w-chunkStart >= fallbackBytes {
			if cut := snapPastMarks(s, skipBreakableSpaces(s, i+w)); cut > chunkStart && beginsWordContent(s, cut) {
				dst = append(dst, cut)
				chunkStart = cut
				i = cut
				continue
			}
		}

		i += w
	}

	if chunkStart < len(s) {
		dst = append(dst, len(s))
	}
	return dst
}

// extendTerminatorCluster returns the end offset of the run of consecutive
// ATerm/STerm runes starting at i, and whether that run contains an STerm.
// Merging the run keeps clusters like "?!" or "..." together.
func extendTerminatorCluster(s string, i int) (end int, hasSTerm bool) {
	end = i
	for end < len(s) {
		r, w := utf8.DecodeRuneInString(s[end:])
		switch {
		case isSTerm(r):
			hasSTerm = true
			end += w
		case isATerm(r):
			end += w
		default:
			return end, hasSTerm
		}
	}
	return end, hasSTerm
}

// skipBreakableSpaces returns the smallest offset >= pos at which the next
// rune is not a breakable space.
func skipBreakableSpaces(s string, pos int) int {
	for pos < len(s) {
		r, w := utf8.DecodeRuneInString(s[pos:])
		if !isBreakableSpace(r) {
			return pos
		}
		pos += w
	}
	return pos
}

// beginsWordContent reports whether s[pos:] starts with a letter or number.
// A cut is placed only here so the boundary is a UAX #14 line-break
// opportunity, not merely a word boundary: after whitespace or a sentence
// terminator a break before letter/number content is always allowed, whereas
// a break before closing punctuation (as in "!)" or "。」") is not.
func beginsWordContent(s string, pos int) bool {
	if pos >= len(s) {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s[pos:])
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

// isBreakableSpace reports whether r is a space the chunker may cut at. It
// deliberately excludes the non-breaking spaces handled as glue at wrap
// time, and the line-break codepoints (a logical line carries at most a
// trailing one, which stays with the final chunk).
func isBreakableSpace(r rune) bool {
	switch r {
	case ' ', '\t', 0x3000: // SPACE, TAB, IDEOGRAPHIC SPACE
		return true
	}
	return false
}

// snapPastMarks returns the smallest offset >= pos that starts a new
// grapheme cluster, advancing past any trailing extending characters so a
// cut there never splits a cluster.
func snapPastMarks(s string, pos int) int {
	for pos < len(s) {
		if s[pos] < utf8.RuneSelf {
			// ASCII never extends a cluster.
			return pos
		}
		r, w := utf8.DecodeRuneInString(s[pos:])
		// Over-broad superset of WB4's trailing Extend | Format | ZWJ (Mark +
		// Cf, plus the few Extend codepoints outside both). Over-broad is
		// safe; stopping short of an Extend would split a grapheme.
		extends := unicode.IsMark(r) || unicode.Is(unicode.Cf, r) ||
			(0x1F3FB <= r && r <= 0x1F3FF) || // emoji modifiers
			r == 0xFF9E || r == 0xFF9F // halfwidth katakana sound marks
		if !extends {
			return pos
		}
		pos += w
	}
	return pos
}

// isATerm reports whether r is in UAX #29's Sentence_Break = ATerm class.
func isATerm(r rune) bool {
	switch r {
	case 0x002E, // FULL STOP
		0x2024, // ONE DOT LEADER
		0xFE52, // SMALL FULL STOP
		0xFF0E: // FULLWIDTH FULL STOP
		return true
	}
	return false
}

// isSTerm reports whether r is a sentence-final terminator (UAX #29
// Sentence_Break = STerm). Only the common BMP terminators are listed, not
// the full class; an omitted one just yields a coarser chunk.
func isSTerm(r rune) bool {
	switch r {
	case 0x0021, // EXCLAMATION MARK
		0x003F, // QUESTION MARK
		0x0589, // ARMENIAN FULL STOP
		0x061E, // ARABIC TRIPLE DOT PUNCTUATION MARK
		0x061F, // ARABIC QUESTION MARK
		0x06D4, // ARABIC FULL STOP
		0x07F9, // NKO EXCLAMATION MARK
		0x0964, // DEVANAGARI DANDA
		0x0965, // DEVANAGARI DOUBLE DANDA
		0x1362, // ETHIOPIC FULL STOP
		0x203C, // DOUBLE EXCLAMATION MARK
		0x203D, // INTERROBANG
		0x2047, // DOUBLE QUESTION MARK
		0x2048, // QUESTION EXCLAMATION MARK
		0x2049, // EXCLAMATION QUESTION MARK
		0x3002, // IDEOGRAPHIC FULL STOP
		0xFE15, // PRESENTATION FORM FOR VERTICAL EXCLAMATION MARK
		0xFE16, // PRESENTATION FORM FOR VERTICAL QUESTION MARK
		0xFE56, // SMALL QUESTION MARK
		0xFE57, // SMALL EXCLAMATION MARK
		0xFF01, // FULLWIDTH EXCLAMATION MARK
		0xFF1F, // FULLWIDTH QUESTION MARK
		0xFF61: // HALFWIDTH IDEOGRAPHIC FULL STOP
		return true
	}
	return false
}
