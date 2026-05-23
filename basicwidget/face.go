// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"slices"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

//go:generate go run gen.go

//go:embed InterVariable.ttf.gz
var interVariableTTFGz []byte

var theDefaultFaceSource FaceSourceEntry

type UnicodeRange struct {
	Min rune
	Max rune
}

type FaceSourceEntry struct {
	FaceSource    *text.GoTextFaceSource
	UnicodeRanges []UnicodeRange
}

func init() {
	r, err := gzip.NewReader(bytes.NewReader(interVariableTTFGz))
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = r.Close()
	}()
	f, err := text.NewGoTextFaceSource(r)
	if err != nil {
		panic(err)
	}
	e := FaceSourceEntry{
		FaceSource: f,
	}
	theDefaultFaceSource = e
}

var (
	tagWght = text.MustParseTag("wght")
	tagLiga = text.MustParseTag("liga")
	tagTnum = text.MustParseTag("tnum")
)

// FontOptions controls how a [Font] resolves glyphs.
type FontOptions struct {
	// DisableFallback disables the fallback chain (custom face sources set
	// by [SetFaceSources], entries registered by [RegisterFonts], and the
	// bundled default face). A Font with DisableFallback=true renders only
	// using its own entries.
	DisableFallback bool
}

// Font is an immutable ordered list of [FaceSourceEntry] values, optionally
// followed by the registered fallback stack. Size, weight, language, and
// OpenType features are not part of a Font; they are applied at render time
// by the consuming widget. A Text renders with a specific Font via
// [Text.SetFont].
type Font struct {
	id          uint64
	entries     []FaceSourceEntry
	useFallback bool
}

var theNextFontID atomic.Uint64

// NewFont returns a Font that renders using entries. A nil opts is treated
// the same as the zero [FontOptions].
func NewFont(entries []FaceSourceEntry, opts *FontOptions) *Font {
	var disableFallback bool
	if opts != nil {
		disableFallback = opts.DisableFallback
	}
	return &Font{
		id:          theNextFontID.Add(1),
		entries:     append([]FaceSourceEntry(nil), entries...),
		useFallback: !disableFallback,
	}
}

// ID returns a unique identifier for the Font. The identifier is stable
// across the Font's lifetime and distinct from all other Fonts in the
// same process.
func (f *Font) ID() uint64 {
	return f.id
}

var (
	theFaceCache map[textutil.FaceKey]text.Face
)

var (
	tmpFaceSourceEntries []FaceSourceEntry
)

var (
	tmpLocales  []language.Tag
	prevLocales []language.Tag
)

func fontFace(context *guigui.Context, key textutil.FaceKey, font *Font) text.Face {
	// As font entires registered by [RegisterFonts] might be affected by locales,
	// clear the cache when the locales change.
	tmpLocales = context.AppendLocales(tmpLocales[:0])
	if !slices.Equal(prevLocales, tmpLocales) {
		clear(theFaceCache)
		prevLocales = slices.Grow(prevLocales, len(tmpLocales))[:len(tmpLocales)]
		copy(prevLocales, tmpLocales)
	}

	if f, ok := theFaceCache[key]; ok {
		return f
	}

	tmpFaceSourceEntries = slices.Delete(tmpFaceSourceEntries, 0, len(tmpFaceSourceEntries))
	if font != nil {
		tmpFaceSourceEntries = append(tmpFaceSourceEntries, font.entries...)
		if font.useFallback {
			tmpFaceSourceEntries = appendFontFaceEntries(tmpFaceSourceEntries, context)
		}
	} else {
		tmpFaceSourceEntries = appendFontFaceEntries(tmpFaceSourceEntries, context)
	}

	var fs []text.Face
	for _, entry := range tmpFaceSourceEntries {
		gtf := &text.GoTextFace{
			Source:   entry.FaceSource,
			Size:     key.Size,
			Language: key.Lang,
		}
		gtf.SetVariation(tagWght, float32(key.Weight))
		if key.Liga {
			gtf.SetFeature(tagLiga, 1)
		} else {
			gtf.SetFeature(tagLiga, 0)
		}
		if key.Tnum {
			gtf.SetFeature(tagTnum, 1)
		} else {
			gtf.SetFeature(tagTnum, 0)
		}

		var f text.Face
		if len(entry.UnicodeRanges) > 0 {
			lf := text.NewLimitedFace(gtf)
			for _, r := range entry.UnicodeRanges {
				lf.AddUnicodeRange(r.Min, r.Max)
			}
			f = lf
		} else {
			f = gtf
		}
		fs = append(fs, f)
	}
	mf, err := text.NewMultiFace(fs...)
	if err != nil {
		panic(err)
	}

	if theFaceCache == nil {
		theFaceCache = map[textutil.FaceKey]text.Face{}
	}
	theFaceCache[key] = mf

	return mf
}

func DefaultFaceSourceEntry() FaceSourceEntry {
	return theDefaultFaceSource
}

func areFaceSourceEntriesEqual(a, b []FaceSourceEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].FaceSource != b[i].FaceSource {
			return false
		}
		if !slices.Equal(a[i].UnicodeRanges, b[i].UnicodeRanges) {
			return false
		}
	}
	return true
}

var (
	theCustomFaceSourceEntries []FaceSourceEntry
)

// SetFaceSources sets the face sources.
func SetFaceSources(entries []FaceSourceEntry) {
	if areFaceSourceEntriesEqual(theCustomFaceSourceEntries, entries) {
		return
	}

	if len(theCustomFaceSourceEntries) < len(entries) {
		theCustomFaceSourceEntries = slices.Grow(theCustomFaceSourceEntries, len(entries))[:len(entries)]
	} else if len(theCustomFaceSourceEntries) > len(entries) {
		theCustomFaceSourceEntries = slices.Delete(theCustomFaceSourceEntries, len(entries), len(theCustomFaceSourceEntries))
	}
	copy(theCustomFaceSourceEntries, entries)

	clear(theFaceCache)
}

type appendFunc struct {
	f         func([]FaceSourceEntry, *guigui.Context) []FaceSourceEntry
	priority1 FontPriority
	priority2 int
}

var (
	theAppendFuncs []appendFunc
)

// FontPriority is used to determine the order of the fonts for [RegisterFonts].
type FontPriority int

const (
	FontPriorityLow    = 100
	FontPriorityNormal = 200
	FontPriorityHigh   = 300
)

// RegisterFonts registers the fonts.
//
// priority is used to determine the order of the fonts.
// The order of the fonts is determined by the priority.
// The bigger priority value, the higher priority.
// If the priority is the same, the order of the fonts is determined by the order of registration.
func RegisterFonts(appendEntries func([]FaceSourceEntry, *guigui.Context) []FaceSourceEntry, priority FontPriority) {
	theAppendFuncs = append(theAppendFuncs, appendFunc{
		f:         appendEntries,
		priority1: priority,
		priority2: -len(theAppendFuncs),
	})
}

func appendFontFaceEntries(entries []FaceSourceEntry, context *guigui.Context) []FaceSourceEntry {
	entries = append(entries, theCustomFaceSourceEntries...)

	slices.SortFunc(theAppendFuncs, func(a, b appendFunc) int {
		if a.priority1 != b.priority1 {
			return int(b.priority1 - a.priority1)
		}
		return b.priority2 - a.priority2
	})
	for _, f := range theAppendFuncs {
		entries = f.f(entries, context)
	}
	return append(entries, theDefaultFaceSource)
}
