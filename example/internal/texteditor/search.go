// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package texteditor

// substringSearcher is an [io.Writer] that reports the byte offset of every
// non-overlapping occurrence of query in the bytes written to it via
// onMatch. onMatch returns false to stop scanning; subsequent writes still
// consume bytes but do not produce more matches.
//
// The matcher is a Knuth–Morris–Pratt state machine.
type substringSearcher struct {
	// query is the substring being searched for.
	query []byte
	// failure is the KMP failure function over query: failure[i] is the
	// length of the longest proper prefix of query[:i+1] that is also a
	// suffix of query[:i+1].
	failure []int
	// state is the length of the query prefix currently matched.
	state int
	// abs is the number of bytes consumed by Write so far.
	abs int
	// onMatch is invoked at each non-overlapping occurrence of query;
	// returning false stops further matching.
	onMatch func(absPos int) bool
	// stopped is true once onMatch has returned false.
	stopped bool
}

func newSubstringSearcher(query []byte, onMatch func(absPos int) bool) *substringSearcher {
	f := make([]int, len(query))
	for i := 1; i < len(query); i++ {
		j := f[i-1]
		for j > 0 && query[i] != query[j] {
			j = f[j-1]
		}
		if query[i] == query[j] {
			j++
		}
		f[i] = j
	}
	return &substringSearcher{
		query:   query,
		failure: f,
		onMatch: onMatch,
	}
}

func (s *substringSearcher) Write(p []byte) (int, error) {
	if s.stopped {
		s.abs += len(p)
		return len(p), nil
	}
	for i, b := range p {
		for s.state > 0 && b != s.query[s.state] {
			s.state = s.failure[s.state-1]
		}
		if b == s.query[s.state] {
			s.state++
		}
		if s.state == len(s.query) {
			matchAbs := s.abs + i + 1 - len(s.query)
			if !s.onMatch(matchAbs) {
				s.stopped = true
				s.abs += len(p)
				return len(p), nil
			}
			s.state = 0
		}
	}
	s.abs += len(p)
	return len(p), nil
}
