// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package highlight_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/guigui-gui/guigui/example/codeeditor/internal/highlight"
)

func TestAppendRanges(t *testing.T) {
	// A sub locates one expected highlighted range in the source: text is
	// the highlighted text, and in, when non-empty, is a longer unique
	// substring starting with text that disambiguates the occurrence.
	type sub struct {
		text string
		in   string
		kind highlight.Kind
	}
	testCases := []struct {
		name string
		src  string
		want []sub
	}{
		{
			name: "empty",
			src:  "",
			want: nil,
		},
		{
			name: "hello world",
			src: `// Greeting.
package main

import "fmt"

func main() {
	fmt.Println("Hello", 42)
}
`,
			want: []sub{
				{text: "// Greeting.", kind: highlight.KindComment},
				{text: "package", kind: highlight.KindKeyword},
				{text: "import", kind: highlight.KindKeyword},
				{text: `"fmt"`, kind: highlight.KindString},
				{text: "func", kind: highlight.KindKeyword},
				{text: "main", in: "main() {", kind: highlight.KindDeclName},
				{text: `"Hello"`, kind: highlight.KindString},
				{text: "42", kind: highlight.KindNumber},
			},
		},
		{
			name: "literals",
			src:  "x := 1 + 2.5 + 3i + 'a' + `raw`",
			want: []sub{
				{text: "1", kind: highlight.KindNumber},
				{text: "2.5", kind: highlight.KindNumber},
				{text: "3i", kind: highlight.KindNumber},
				{text: "'a'", kind: highlight.KindString},
				{text: "`raw`", kind: highlight.KindString},
			},
		},
		{
			name: "type declaration",
			src:  "type Point struct { x int }",
			want: []sub{
				{text: "type", kind: highlight.KindKeyword},
				{text: "Point", kind: highlight.KindDeclName},
				{text: "struct", kind: highlight.KindKeyword},
			},
		},
		{
			name: "comment between func and name",
			src:  "func /* c */ Foo() {}",
			want: []sub{
				{text: "func", kind: highlight.KindKeyword},
				{text: "/* c */", kind: highlight.KindComment},
				{text: "Foo", kind: highlight.KindDeclName},
			},
		},
		{
			name: "unterminated string",
			src:  `s := "abc`,
			want: []sub{
				{text: `"abc`, kind: highlight.KindString},
			},
		},
		{
			name: "unterminated general comment",
			src:  "x := 1\n/* dangling",
			want: []sub{
				{text: "1", kind: highlight.KindNumber},
				{text: "/* dangling", kind: highlight.KindComment},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var want []highlight.Range
			for _, s := range tc.want {
				locator := s.in
				if locator == "" {
					locator = s.text
				}
				idx := strings.Index(tc.src, locator)
				if idx < 0 {
					t.Fatalf("substring %q not found in %q", locator, tc.src)
				}
				want = append(want, highlight.Range{
					StartInBytes: idx,
					EndInBytes:   idx + len(s.text),
					Kind:         s.kind,
				})
			}
			got := highlight.AppendRanges(nil, tc.src)
			if !slices.Equal(got, want) {
				t.Errorf("AppendRanges(nil, %q) = %v; want %v", tc.src, got, want)
			}
		})
	}
}

func TestAppendRangesAppends(t *testing.T) {
	seed := []highlight.Range{
		{StartInBytes: 0, EndInBytes: 1, Kind: highlight.KindComment},
	}
	got := highlight.AppendRanges(seed, "return")
	want := []highlight.Range{
		{StartInBytes: 0, EndInBytes: 1, Kind: highlight.KindComment},
		{StartInBytes: 0, EndInBytes: 6, Kind: highlight.KindKeyword},
	}
	if !slices.Equal(got, want) {
		t.Errorf("AppendRanges(%v, %q) = %v; want %v", seed, "return", got, want)
	}
}

func TestAppendRangesOrderedAndDisjoint(t *testing.T) {
	src := strings.Repeat(`// A comment.
func f(a int) string {
	const n = 123
	return fmt.Sprintf("%d", n*2.5)
}
`, 100)
	ranges := highlight.AppendRanges(nil, src)
	if len(ranges) == 0 {
		t.Fatal("AppendRanges returned no ranges")
	}
	for i, r := range ranges {
		if r.StartInBytes < 0 || r.EndInBytes > len(src) {
			t.Errorf("range %v is out of bounds [0, %d)", r, len(src))
		}
		if r.StartInBytes >= r.EndInBytes {
			t.Errorf("range %v is empty or inverted", r)
		}
		if i > 0 && ranges[i-1].EndInBytes > r.StartInBytes {
			t.Errorf("range %v overlaps the previous range %v", r, ranges[i-1])
		}
	}
}

func BenchmarkAppendRanges(b *testing.B) {
	// Roughly a few hundred lines of representative code.
	src := strings.Repeat(`// A comment describing the function below.
func process(items []string, limit int) (int, error) {
	total := 0
	for i, item := range items {
		if i >= limit {
			break
		}
		total += len(item) * 2
	}
	return total, nil
}
`, 50)
	var ranges []highlight.Range
	b.ResetTimer()
	for range b.N {
		ranges = highlight.AppendRanges(ranges[:0], src)
	}
}
