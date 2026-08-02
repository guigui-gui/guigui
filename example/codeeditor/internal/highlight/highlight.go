// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

// Package highlight maps Go source code to syntax-highlighted byte ranges.
package highlight

import (
	"go/scanner"
	"go/token"
)

// Kind identifies the syntax class of a highlighted range.
type Kind int

const (
	// KindKeyword is a Go keyword.
	KindKeyword Kind = iota

	// KindString is a string or character literal.
	KindString

	// KindNumber is an integer, floating-point, or imaginary literal.
	KindNumber

	// KindComment is a line or general comment.
	KindComment

	// KindDeclName is the identifier immediately following the func or type
	// keyword.
	KindDeclName
)

// Range is a byte range of source code with a syntax class.
type Range struct {
	// StartInBytes is the inclusive start byte offset of the range.
	StartInBytes int

	// EndInBytes is the exclusive end byte offset of the range.
	EndInBytes int

	// Kind is the syntax class of the range.
	Kind Kind
}

// AppendRanges appends the highlighted ranges of the Go source code src to
// ranges and returns the extended slice. The appended ranges are in
// ascending order of StartInBytes and do not overlap. Scan errors are
// ignored so that incomplete code still yields ranges.
func AppendRanges(ranges []Range, src string) []Range {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var s scanner.Scanner
	s.Init(file, []byte(src), nil, scanner.ScanComments)

	var prev token.Token
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		var kind Kind
		known := true
		switch {
		case tok.IsKeyword():
			kind = KindKeyword
		case tok == token.STRING || tok == token.CHAR:
			kind = KindString
		case tok == token.INT || tok == token.FLOAT || tok == token.IMAG:
			kind = KindNumber
		case tok == token.COMMENT:
			kind = KindComment
		case tok == token.IDENT && (prev == token.FUNC || prev == token.TYPE):
			kind = KindDeclName
		default:
			known = false
		}
		// Skip comments when tracking the previous token so that a comment
		// between func or type and the following identifier does not hide
		// the declaration name.
		if tok != token.COMMENT {
			prev = tok
		}
		if !known {
			continue
		}
		start := file.Offset(pos)
		end := start + len(lit)
		if lit == "" {
			end = start + len(tok.String())
		}
		end = min(end, len(src))
		ranges = append(ranges, Range{
			StartInBytes: start,
			EndInBytes:   end,
			Kind:         kind,
		})
	}
	return ranges
}
