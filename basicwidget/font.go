// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

type UnicodeRange struct {
	Min rune
	Max rune
}

type FaceSourceEntry struct {
	FaceSource    *text.GoTextFaceSource
	UnicodeRanges []UnicodeRange
}

// FontFamilyOptions controls how a [FontFamily] resolves glyphs.
type FontFamilyOptions struct {
	// DisableFallback restricts rendering to the FontFamily's own entries,
	// skipping the fallback stack.
	DisableFallback bool
}

// FontFamily is an immutable ordered list of [FaceSourceEntry] values,
// optionally followed by the registered fallback stack. Size, weight,
// language, and OpenType features are not part of a FontFamily; they are
// applied at render time.
type FontFamily struct {
	f *font.Family
}

// FontPriority is used to determine the order of the fonts for [RegisterFonts].
type FontPriority int

const (
	FontPriorityLow    = FontPriority(font.FontPriorityLow)
	FontPriorityNormal = FontPriority(font.FontPriorityNormal)
	FontPriorityHigh   = FontPriority(font.FontPriorityHigh)
)

// NewFontFamily returns a FontFamily that renders using entries. A nil opts is
// treated the same as the zero [FontFamilyOptions].
func NewFontFamily(entries []FaceSourceEntry, opts *FontFamilyOptions) *FontFamily {
	var familyOpts *font.FamilyOptions
	if opts != nil {
		familyOpts = &font.FamilyOptions{
			DisableFallback: opts.DisableFallback,
		}
	}
	return &FontFamily{
		f: font.NewFamily(toFontFaceSourceEntries(entries), familyOpts),
	}
}

// DefaultFaceSourceEntry returns the entry for the bundled default face.
func DefaultFaceSourceEntry() FaceSourceEntry {
	return fromFontFaceSourceEntry(font.DefaultFaceSourceEntry())
}

// SetFaceSources sets the face sources.
func SetFaceSources(entries []FaceSourceEntry) {
	font.SetFaceSources(toFontFaceSourceEntries(entries))
}

// RegisterFonts registers the fonts.
//
// priority is used to determine the order of the fonts.
// The order of the fonts is determined by the priority.
// The bigger priority value, the higher priority.
// If the priority is the same, the order of the fonts is determined by the order of registration.
func RegisterFonts(appendEntries func([]FaceSourceEntry, *guigui.Context) []FaceSourceEntry, priority FontPriority) {
	font.RegisterFonts(func(entries []font.FaceSourceEntry, context *guigui.Context) []font.FaceSourceEntry {
		converted := make([]FaceSourceEntry, len(entries))
		for i, e := range entries {
			converted[i] = fromFontFaceSourceEntry(e)
		}
		converted = appendEntries(converted, context)
		return toFontFaceSourceEntries(converted)
	}, font.FontPriority(priority))
}

func toFontFaceSourceEntries(entries []FaceSourceEntry) []font.FaceSourceEntry {
	if entries == nil {
		return nil
	}
	result := make([]font.FaceSourceEntry, len(entries))
	for i, e := range entries {
		result[i] = toFontFaceSourceEntry(e)
	}
	return result
}

func toFontFaceSourceEntry(e FaceSourceEntry) font.FaceSourceEntry {
	var ranges []font.UnicodeRange
	if e.UnicodeRanges != nil {
		ranges = make([]font.UnicodeRange, len(e.UnicodeRanges))
		for i, r := range e.UnicodeRanges {
			ranges[i] = font.UnicodeRange{
				Min: r.Min,
				Max: r.Max,
			}
		}
	}
	return font.FaceSourceEntry{
		FaceSource:    e.FaceSource,
		UnicodeRanges: ranges,
	}
}

func fromFontFaceSourceEntry(e font.FaceSourceEntry) FaceSourceEntry {
	var ranges []UnicodeRange
	if e.UnicodeRanges != nil {
		ranges = make([]UnicodeRange, len(e.UnicodeRanges))
		for i, r := range e.UnicodeRanges {
			ranges[i] = UnicodeRange{
				Min: r.Min,
				Max: r.Max,
			}
		}
	}
	return FaceSourceEntry{
		FaceSource:    e.FaceSource,
		UnicodeRanges: ranges,
	}
}
