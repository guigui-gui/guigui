// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

// Package textstyle provides the attributed-run model for ranged text
// styling: a set of style overrides laid over byte ranges of a single text
// value.
package textstyle

import (
	"cmp"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

// optional is a value that may be unset. The zero value is unset.
type optional[T any] struct {
	value T
	set   bool
}

// opt returns an [optional] set to value.
func opt[T any](value T) optional[T] {
	return optional[T]{
		value: value,
		set:   true,
	}
}

// Value returns the value and whether it is set.
func (o optional[T]) Value() (T, bool) {
	return o.value, o.set
}

// Style is a set of style overrides for a byte range of a text value. The
// zero value overrides nothing. Styles are produced by [Runs]; an unset
// property does not override anything.
type Style struct {
	family          optional[*font.Family]
	italic          optional[bool]
	scale           optional[float64]
	color           optional[color.Color]
	backgroundColor optional[color.Color]
	underline       optional[bool]
	strikethrough   optional[bool]
	lang            optional[language.Tag]

	// features and variations are sorted by tag with at most one entry per
	// tag.
	features   []font.Feature
	variations []font.Variation
}

// Family returns the font family override and whether it is set.
func (s Style) Family() (*font.Family, bool) {
	return s.family.Value()
}

// Italic returns the italic face selection override and whether it is set.
func (s Style) Italic() (bool, bool) {
	return s.italic.Value()
}

// Scale returns the base font size multiplier override and whether it is
// set.
func (s Style) Scale() (float64, bool) {
	return s.scale.Value()
}

// Color returns the text color override and whether it is set.
func (s Style) Color() (color.Color, bool) {
	return s.color.Value()
}

// BackgroundColor returns the background color override and whether it is
// set.
func (s Style) BackgroundColor() (color.Color, bool) {
	return s.backgroundColor.Value()
}

// Underline returns the underline override and whether it is set.
func (s Style) Underline() (bool, bool) {
	return s.underline.Value()
}

// Strikethrough returns the strikethrough override and whether it is set.
func (s Style) Strikethrough() (bool, bool) {
	return s.strikethrough.Value()
}

// Lang returns the language override and whether it is set.
func (s Style) Lang() (language.Tag, bool) {
	return s.lang.Value()
}

// Features returns the OpenType feature overrides, sorted by tag.
func (s Style) Features() []font.Feature {
	return s.features
}

// Variation returns the override value of the OpenType variation axis tag
// and whether it is set.
func (s Style) Variation(tag text.Tag) (float32, bool) {
	i, ok := slices.BinarySearchFunc(s.variations, tag, func(v font.Variation, tag text.Tag) int {
		return cmp.Compare(v.Tag, tag)
	})
	if !ok {
		return 0, false
	}
	return s.variations[i].Value, true
}

// Variations returns the OpenType variation axis overrides, sorted by tag.
func (s Style) Variations() []font.Variation {
	return s.variations
}

// IsZero reports whether the style overrides nothing.
func (s Style) IsZero() bool {
	return !s.family.set &&
		!s.italic.set &&
		!s.scale.set &&
		!s.color.set &&
		!s.backgroundColor.set &&
		!s.underline.set &&
		!s.strikethrough.set &&
		!s.lang.set &&
		len(s.features) == 0 &&
		len(s.variations) == 0
}

// Equal reports whether two styles are identical.
func (s Style) Equal(other Style) bool {
	return s.family == other.family &&
		s.italic == other.italic &&
		s.scale == other.scale &&
		s.color == other.color &&
		s.backgroundColor == other.backgroundColor &&
		s.underline == other.underline &&
		s.strikethrough == other.strikethrough &&
		s.lang == other.lang &&
		slices.Equal(s.features, other.features) &&
		slices.Equal(s.variations, other.variations)
}

// WithFamily returns s with the font family set to family.
func (s Style) WithFamily(family *font.Family) Style {
	s.family = opt(family)
	return s
}

// WithItalic returns s with the italic face selection set to italic.
func (s Style) WithItalic(italic bool) Style {
	s.italic = opt(italic)
	return s
}

// WithColor returns s with the text color set to clr.
func (s Style) WithColor(clr color.Color) Style {
	s.color = opt(clr)
	return s
}

// WithLang returns s with the language set to lang.
func (s Style) WithLang(lang language.Tag) Style {
	s.lang = opt(lang)
	return s
}

// WithVariation returns s with the OpenType variation axis tag set to value.
func (s Style) WithVariation(tag text.Tag, value float32) Style {
	s.variations = withTagged(s.variations, font.Variation{Tag: tag, Value: value}, func(v font.Variation) text.Tag {
		return v.Tag
	})
	return s
}

// WithoutVariation returns s with the OpenType variation axis tag removed.
func (s Style) WithoutVariation(tag text.Tag) Style {
	s.variations = removeTagged(s.variations, []text.Tag{tag}, func(v font.Variation) text.Tag {
		return v.Tag
	})
	return s
}

// WithFeature returns s with the OpenType feature tag set to value.
func (s Style) WithFeature(tag text.Tag, value uint32) Style {
	s.features = withTagged(s.features, font.Feature{Tag: tag, Value: value}, func(f font.Feature) text.Tag {
		return f.Tag
	})
	return s
}

// WithoutFeature returns s with the OpenType feature tag removed.
func (s Style) WithoutFeature(tag text.Tag) Style {
	s.features = removeTagged(s.features, []text.Tag{tag}, func(f font.Feature) text.Tag {
		return f.Tag
	})
	return s
}

// withTagged returns entries with entry set, keeping the tag order and at
// most one entry per tag. entries is returned as is when it already contains
// entry; the stored slice is never mutated.
func withTagged[T comparable](entries []T, entry T, tag func(T) text.Tag) []T {
	i, ok := slices.BinarySearchFunc(entries, tag(entry), func(e T, t text.Tag) int {
		return cmp.Compare(tag(e), t)
	})
	if ok && entries[i] == entry {
		return entries
	}
	result := make([]T, 0, len(entries)+1)
	result = append(result, entries[:i]...)
	result = append(result, entry)
	if ok {
		i++
	}
	result = append(result, entries[i:]...)
	return result
}

// Merge returns s with other's set properties applied on top. Features and
// variations are merged by tag, with other winning on a shared tag.
func (s Style) Merge(other Style) Style {
	if other.family.set {
		s.family = other.family
	}
	if other.italic.set {
		s.italic = other.italic
	}
	if other.scale.set {
		s.scale = other.scale
	}
	if other.color.set {
		s.color = other.color
	}
	if other.backgroundColor.set {
		s.backgroundColor = other.backgroundColor
	}
	if other.underline.set {
		s.underline = other.underline
	}
	if other.strikethrough.set {
		s.strikethrough = other.strikethrough
	}
	if other.lang.set {
		s.lang = other.lang
	}
	s.features = mergeTagged(s.features, other.features, func(f font.Feature) text.Tag {
		return f.Tag
	})
	s.variations = mergeTagged(s.variations, other.variations, func(v font.Variation) text.Tag {
		return v.Tag
	})
	return s
}

// styleMask selects style properties without carrying values.
type styleMask struct {
	family          bool
	italic          bool
	scale           bool
	color           bool
	backgroundColor bool
	underline       bool
	strikethrough   bool
	lang            bool
	featureTags     []text.Tag
	variationTags   []text.Tag
}

// isZero reports whether the mask selects nothing.
func (m styleMask) isZero() bool {
	return !m.family &&
		!m.italic &&
		!m.scale &&
		!m.color &&
		!m.backgroundColor &&
		!m.underline &&
		!m.strikethrough &&
		!m.lang &&
		len(m.featureTags) == 0 &&
		len(m.variationTags) == 0
}

// remove returns s without the properties selected by mask.
func (s Style) remove(mask styleMask) Style {
	if mask.family {
		s.family = optional[*font.Family]{}
	}
	if mask.italic {
		s.italic = optional[bool]{}
	}
	if mask.scale {
		s.scale = optional[float64]{}
	}
	if mask.color {
		s.color = optional[color.Color]{}
	}
	if mask.backgroundColor {
		s.backgroundColor = optional[color.Color]{}
	}
	if mask.underline {
		s.underline = optional[bool]{}
	}
	if mask.strikethrough {
		s.strikethrough = optional[bool]{}
	}
	if mask.lang {
		s.lang = optional[language.Tag]{}
	}
	s.features = removeTagged(s.features, mask.featureTags, func(f font.Feature) text.Tag {
		return f.Tag
	})
	s.variations = removeTagged(s.variations, mask.variationTags, func(v font.Variation) text.Tag {
		return v.Tag
	})
	return s
}

// removeTagged returns entries without the listed tags. entries is returned
// as is when no tag matches.
func removeTagged[T any](entries []T, tags []text.Tag, tag func(T) text.Tag) []T {
	if len(entries) == 0 || len(tags) == 0 {
		return entries
	}
	hasTag := func(e T) bool {
		return slices.Contains(tags, tag(e))
	}
	if !slices.ContainsFunc(entries, hasTag) {
		return entries
	}
	// entries may be shared with stored styles, so delete from a clone.
	result := slices.DeleteFunc(slices.Clone(entries), hasTag)
	if len(result) == 0 {
		return nil
	}
	return result
}

// canonicalized returns s with features and variations sorted by tag and
// deduplicated, keeping the last entry for each tag.
func (s Style) canonicalized() Style {
	s.features = canonicalTagged(s.features, func(f font.Feature) text.Tag {
		return f.Tag
	})
	s.variations = canonicalTagged(s.variations, func(v font.Variation) text.Tag {
		return v.Tag
	})
	return s
}

// canonicalTagged returns entries sorted by tag with at most one entry per
// tag, keeping the last entry for each tag. A nil slice is returned for an
// empty input; an already canonical entries is returned as is.
func canonicalTagged[T any](entries []T, tag func(T) text.Tag) []T {
	if len(entries) == 0 {
		return nil
	}
	if isCanonicalTagged(entries, tag) {
		return entries
	}
	es := slices.Clone(entries)
	slices.SortStableFunc(es, func(a, b T) int {
		return cmp.Compare(tag(a), tag(b))
	})
	result := es[:0]
	for i, e := range es {
		if i+1 < len(es) && tag(es[i+1]) == tag(e) {
			continue
		}
		result = append(result, e)
	}
	return result
}

// isCanonicalTagged reports whether entries are sorted by tag with no
// duplicate tags.
func isCanonicalTagged[T any](entries []T, tag func(T) text.Tag) bool {
	for i := 1; i < len(entries); i++ {
		if tag(entries[i-1]) >= tag(entries[i]) {
			return false
		}
	}
	return true
}

// mergeTagged merges two canonical entry slices by tag, with b winning on a
// shared tag. The result is canonical. a or b is returned as is when it
// already equals the merged result.
func mergeTagged[T comparable](a, b []T, tag func(T) text.Tag) []T {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}

	// The merged result equals a when every b entry appears identically in
	// a, and equals b when every a tag appears in b.
	equalsA, equalsB := true, true
	{
		var i, j int
		for i < len(a) && j < len(b) {
			switch {
			case tag(a[i]) < tag(b[j]):
				equalsB = false
				i++
			case tag(a[i]) > tag(b[j]):
				equalsA = false
				j++
			default:
				if a[i] != b[j] {
					equalsA = false
				}
				i++
				j++
			}
		}
		if i < len(a) {
			equalsB = false
		}
		if j < len(b) {
			equalsA = false
		}
	}
	if equalsA {
		return a
	}
	if equalsB {
		return b
	}

	result := make([]T, 0, len(a)+len(b))
	var i, j int
	for i < len(a) && j < len(b) {
		switch {
		case tag(a[i]) < tag(b[j]):
			result = append(result, a[i])
			i++
		case tag(a[i]) > tag(b[j]):
			result = append(result, b[j])
			j++
		default:
			result = append(result, b[j])
			i++
			j++
		}
	}
	result = append(result, a[i:]...)
	result = append(result, b[j:]...)
	return result
}
