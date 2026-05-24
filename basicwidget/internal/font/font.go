// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package font

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"slices"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui"
)

//go:generate go run gen.go

//go:embed InterVariable.ttf.gz
var interVariableTTFGz []byte

var theDefaultFaceSource FaceSourceEntry

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

// Attributes is a comparable set of text rendering attributes: size, weight,
// ligatures, tabular numerals, and language. A face is resolved from
// Attributes together with a [Family]; the attributes alone do not identify a
// font.
type Attributes struct {
	Size   float64
	Weight text.Weight
	Liga   bool
	Tnum   bool
	Lang   language.Tag
}

// Face is a resolved text face.
type Face struct {
	face text.Face
}

// NewFace resolves the text face for attributes, rendering with fnt's face
// source entries followed by the registered fallback stack (unless fnt
// disables fallback). A nil fnt resolves using the registered fallback stack
// alone.
func NewFace(context *guigui.Context, fnt *Family, attributes Attributes) Face {
	return Face{
		face: resolveFace(context, fnt, attributes),
	}
}

// NewFaceForTest wraps face directly, bypassing font resolution. It is
// intended for tests that supply a specific [text.Face].
func NewFaceForTest(face text.Face) Face {
	return Face{
		face: face,
	}
}

// TextFace returns the resolved face.
func (f Face) TextFace() text.Face {
	return f.face
}

// UnicodeRange is an inclusive range of code points.
type UnicodeRange struct {
	Min rune
	Max rune
}

// FaceSourceEntry pairs a face source with the optional Unicode ranges it is
// limited to. An empty UnicodeRanges means the source is not range-limited.
type FaceSourceEntry struct {
	FaceSource    *text.GoTextFaceSource
	UnicodeRanges []UnicodeRange
}

// FamilyOptions controls how a [Family] resolves glyphs.
type FamilyOptions struct {
	// DisableFallback restricts rendering to the Family's own entries, skipping
	// the fallback stack.
	DisableFallback bool
}

// Family is an immutable ordered list of [FaceSourceEntry] values, optionally
// followed by the registered fallback stack. Size, weight, language, and
// OpenType features are not part of a Family; they are applied at render time.
type Family struct {
	id          uint64
	entries     []FaceSourceEntry
	useFallback bool
}

var theNextFamilyID atomic.Uint64

// NewFamily returns a Family that renders using entries. A nil opts is treated
// the same as the zero [FamilyOptions].
func NewFamily(entries []FaceSourceEntry, opts *FamilyOptions) *Family {
	var disableFallback bool
	if opts != nil {
		disableFallback = opts.DisableFallback
	}
	return &Family{
		id:          theNextFamilyID.Add(1),
		entries:     append([]FaceSourceEntry(nil), entries...),
		useFallback: !disableFallback,
	}
}

// ID returns the Family's process-unique identifier.
func (f *Family) ID() uint64 {
	return f.id
}

// cacheKey identifies a resolved face by family identity and render attributes.
type cacheKey struct {
	familyID   uint64
	attributes Attributes
}

var (
	theFaceCache map[cacheKey]text.Face
)

var (
	tmpFaceSourceEntries []FaceSourceEntry
)

var (
	tmpLocales  []language.Tag
	prevLocales []language.Tag
)

func resolveFace(context *guigui.Context, fnt *Family, attributes Attributes) text.Face {
	// As font entries registered by [RegisterFonts] might be affected by locales,
	// clear the cache when the locales change.
	tmpLocales = context.AppendLocales(tmpLocales[:0])
	if !slices.Equal(prevLocales, tmpLocales) {
		clear(theFaceCache)
		prevLocales = slices.Grow(prevLocales, len(tmpLocales))[:len(tmpLocales)]
		copy(prevLocales, tmpLocales)
	}

	var familyID uint64
	if fnt != nil {
		familyID = fnt.id
	}
	ck := cacheKey{
		familyID:   familyID,
		attributes: attributes,
	}
	if f, ok := theFaceCache[ck]; ok {
		return f
	}

	tmpFaceSourceEntries = slices.Delete(tmpFaceSourceEntries, 0, len(tmpFaceSourceEntries))
	if fnt != nil {
		tmpFaceSourceEntries = append(tmpFaceSourceEntries, fnt.entries...)
		if fnt.useFallback {
			tmpFaceSourceEntries = appendFontFaceEntries(tmpFaceSourceEntries, context)
		}
	} else {
		tmpFaceSourceEntries = appendFontFaceEntries(tmpFaceSourceEntries, context)
	}

	var fs []text.Face
	for _, entry := range tmpFaceSourceEntries {
		gtf := &text.GoTextFace{
			Source:   entry.FaceSource,
			Size:     attributes.Size,
			Language: attributes.Lang,
		}
		gtf.SetVariation(tagWght, float32(attributes.Weight))
		if attributes.Liga {
			gtf.SetFeature(tagLiga, 1)
		} else {
			gtf.SetFeature(tagLiga, 0)
		}
		if attributes.Tnum {
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
		theFaceCache = map[cacheKey]text.Face{}
	}
	theFaceCache[ck] = mf

	return mf
}

// DefaultFaceSourceEntry returns the entry for the bundled default face.
func DefaultFaceSourceEntry() FaceSourceEntry {
	return theDefaultFaceSource
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
	FontPriorityLow    FontPriority = 100
	FontPriorityNormal FontPriority = 200
	FontPriorityHigh   FontPriority = 300
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
