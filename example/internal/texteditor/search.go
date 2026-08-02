// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package texteditor

import (
	"unicode"
	"unicode/utf8"
)

// matchRange is the byte range [start, end) of one occurrence reported by
// [substringSearcher]. The zero value reports false from found.
type matchRange struct {
	start int
	end   int
}

// found reports whether the range denotes an actual occurrence.
func (m matchRange) found() bool {
	return m.end > m.start
}

// substringSearcher is an [io.Writer] that reports every non-overlapping
// occurrence of query in the bytes written to it via onMatch. Matching is
// case-insensitive, so the reported range may differ in length from query.
// onMatch returns false to stop scanning; subsequent writes still consume
// bytes but do not produce more matches. An empty query has no occurrences.
//
// The matcher is a Knuth–Morris–Pratt state machine over case-folded runes.
type substringSearcher struct {
	// query is the case-folded form of the substring being searched for.
	query []rune
	// failure is the KMP failure function over query: failure[i] is the
	// length of the longest proper prefix of query[:i+1] that is also a
	// suffix of query[:i+1].
	failure []int
	// state is the length of the query prefix currently matched.
	state int
	// runeStarts is a ring buffer, indexed by runeCount, of the byte offset
	// at which each of the last len(query) consumed runes begins.
	runeStarts []int
	// runeCount is the number of runes consumed so far.
	runeCount int
	// abs is the number of bytes consumed as runes so far.
	abs int
	// pending holds the bytes of a rune that Write received only partially,
	// to be decoded once the rest arrives.
	pending []byte
	// onMatch is invoked at each non-overlapping occurrence of query;
	// returning false stops further matching.
	onMatch func(m matchRange) bool
	// stopped is true once onMatch has returned false.
	stopped bool
}

func newSubstringSearcher(query string, onMatch func(m matchRange) bool) *substringSearcher {
	q := make([]rune, 0, len(query))
	for _, r := range query {
		q = append(q, foldRune(r))
	}
	f := make([]int, len(q))
	for i := 1; i < len(q); i++ {
		j := f[i-1]
		for j > 0 && q[i] != q[j] {
			j = f[j-1]
		}
		if q[i] == q[j] {
			j++
		}
		f[i] = j
	}
	return &substringSearcher{
		query:      q,
		failure:    f,
		runeStarts: make([]int, len(q)),
		onMatch:    onMatch,
	}
}

func (s *substringSearcher) Write(p []byte) (int, error) {
	n := len(p)
	if s.stopped || len(s.query) == 0 {
		return n, nil
	}

	for {
		var r rune
		var size int
		switch {
		case len(s.pending) > 0:
			for len(p) > 0 && !utf8.FullRune(s.pending) {
				s.pending = append(s.pending, p[0])
				p = p[1:]
			}
			if !utf8.FullRune(s.pending) {
				return n, nil
			}
			r, size = utf8.DecodeRune(s.pending)
			// A rune that decodes shorter than the buffered bytes means the
			// encoding turned out to be invalid; keep the remainder for the
			// next iteration.
			s.pending = append(s.pending[:0], s.pending[size:]...)
		case len(p) == 0:
			return n, nil
		case !utf8.FullRune(p):
			s.pending = append(s.pending, p...)
			return n, nil
		default:
			r, size = utf8.DecodeRune(p)
			p = p[size:]
		}
		if !s.feedRune(r, size) {
			s.stopped = true
			return n, nil
		}
	}
}

// feedRune advances the state machine by one rune of the given byte width and
// returns false once no further matches are wanted.
func (s *substringSearcher) feedRune(r rune, size int) bool {
	s.runeStarts[s.runeCount%len(s.runeStarts)] = s.abs
	s.runeCount++
	s.abs += size

	r = foldRune(r)
	for s.state > 0 && r != s.query[s.state] {
		s.state = s.failure[s.state-1]
	}
	if r == s.query[s.state] {
		s.state++
	}
	if s.state < len(s.query) {
		return true
	}
	// The slot about to be overwritten holds the start of the rune len(query)
	// positions back, which is where the occurrence begins.
	m := matchRange{
		start: s.runeStarts[s.runeCount%len(s.runeStarts)],
		end:   s.abs,
	}
	s.state = 0
	return s.onMatch(m)
}

// foldRune returns the representative of the Unicode simple-fold orbit of r,
// so that runes equal apart from case map to the same value.
func foldRune(r rune) rune {
	f := r
	for c := unicode.SimpleFold(r); c != r; c = unicode.SimpleFold(c) {
		f = min(f, c)
	}
	return f
}
