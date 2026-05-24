// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

// UnicodeRange is an inclusive range of code points.
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

// FaceSourceEntryPriority orders providers registered with
// [RegisterFaceSourceEntries]; a higher value resolves earlier.
type FaceSourceEntryPriority int

const (
	FaceSourceEntryPriorityLow    = FaceSourceEntryPriority(font.FaceSourceEntryPriorityLow)
	FaceSourceEntryPriorityNormal = FaceSourceEntryPriority(font.FaceSourceEntryPriorityNormal)
	FaceSourceEntryPriorityHigh   = FaceSourceEntryPriority(font.FaceSourceEntryPriorityHigh)
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

// FaceSourceEntryAdder collects face source entries for the fallback
// resolution stack.
type FaceSourceEntryAdder struct {
	adder *font.FaceSourceEntryAdder
}

// Add adds entry to the fallback resolution stack at the adder's current
// priority, which is [FaceSourceEntryPriorityNormal] until changed by
// SetPriority.
func (a *FaceSourceEntryAdder) Add(entry FaceSourceEntry) {
	a.adder.Add(toFontFaceSourceEntry(entry))
}

// SetPriority sets the priority applied to entries added afterward. A higher
// priority resolves earlier; entries of equal priority resolve in the order
// they are added.
func (a *FaceSourceEntryAdder) SetPriority(priority FaceSourceEntryPriority) {
	a.adder.SetPriority(font.FaceSourceEntryPriority(priority))
}

// RegisterFaceSourceEntries registers add as a contributor to the fallback
// resolution stack. add adds its face source entries through the provided
// [FaceSourceEntryAdder].
func RegisterFaceSourceEntries(add func(*guigui.Context, *FaceSourceEntryAdder)) {
	font.RegisterFaceSourceEntries(func(context *guigui.Context, a *font.FaceSourceEntryAdder) {
		add(context, &FaceSourceEntryAdder{adder: a})
	})
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
