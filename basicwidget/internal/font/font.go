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
	face       text.Face
	family     *Family
	attributes Attributes
	id         uint64
}

// NewFace resolves the text face for attributes, rendering with fnt's face
// source entries followed by the registered fallback stack (unless fnt
// disables fallback). A nil fnt resolves using the registered fallback stack
// alone.
func NewFace(context *guigui.Context, fnt *Family, attributes Attributes) Face {
	face, id := resolveFace(context, fnt, attributes)
	return Face{
		face:       face,
		family:     fnt,
		attributes: attributes,
		id:         id,
	}
}

// NewFaceForTest wraps face directly with attributes, bypassing font
// resolution. It is intended for tests that supply a specific [text.Face].
func NewFaceForTest(face text.Face, attributes Attributes) Face {
	return Face{
		face:       face,
		attributes: attributes,
		id:         theNextFaceID.Add(1),
	}
}

// TextFace returns the resolved face.
func (f Face) TextFace() text.Face {
	return f.face
}

// Family returns the [Family] f was resolved from, or nil when f was resolved
// without a family.
func (f Face) Family() *Family {
	return f.family
}

// Attributes returns the render attributes f was resolved from.
func (f Face) Attributes() Attributes {
	return f.attributes
}

// ID returns a process-unique identifier of this resolved face. Faces that
// resolve identically share an ID, and re-resolving after a locale change
// yields a new one; the zero Face has ID 0. It identifies a face for caching
// results that depend on the face's metrics.
func (f Face) ID() uint64 {
	return f.id
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

// cachedFace is a resolved face together with its process-unique id.
type cachedFace struct {
	face text.Face
	id   uint64
}

var (
	theFaceCache  map[cacheKey]cachedFace
	theNextFaceID atomic.Uint64
)

var (
	tmpFaceSourceEntries []FaceSourceEntry
)

var (
	tmpLocales  []language.Tag
	prevLocales []language.Tag
)

func resolveFace(context *guigui.Context, fnt *Family, attributes Attributes) (text.Face, uint64) {
	// As face source entries registered by [RegisterFaceSourceEntries] might be
	// affected by locales, clear the cache when the locales change. Dropping the
	// entries re-resolves faces, which assigns them fresh ids so cache entries
	// keyed on a face id fall out of use naturally.
	tmpLocales = context.AppendLocales(tmpLocales[:0])
	if !slices.Equal(prevLocales, tmpLocales) {
		clear(theFaceCache)
		prevLocales = append(prevLocales[:0], tmpLocales...)
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
		return f.face, f.id
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
		theFaceCache = map[cacheKey]cachedFace{}
	}
	id := theNextFaceID.Add(1)
	theFaceCache[ck] = cachedFace{face: mf, id: id}

	return mf, id
}

// DefaultFaceSourceEntry returns the entry for the bundled default face.
func DefaultFaceSourceEntry() FaceSourceEntry {
	return theDefaultFaceSource
}

// FaceSourceEntryPriority orders entries in the fallback resolution stack; a
// higher value resolves earlier.
type FaceSourceEntryPriority int

const (
	FaceSourceEntryPriorityLow    FaceSourceEntryPriority = 100
	FaceSourceEntryPriorityNormal FaceSourceEntryPriority = 200
	FaceSourceEntryPriorityHigh   FaceSourceEntryPriority = 300
)

type prioritizedFaceSourceEntry struct {
	entry    FaceSourceEntry
	priority FaceSourceEntryPriority
}

// FaceSourceEntryAdder collects face source entries for the fallback
// resolution stack.
type FaceSourceEntryAdder struct {
	entries  []prioritizedFaceSourceEntry
	priority FaceSourceEntryPriority
}

// Add adds entry to the fallback resolution stack at the adder's current
// priority, which is [FaceSourceEntryPriorityNormal] until changed by
// SetPriority.
func (a *FaceSourceEntryAdder) Add(entry FaceSourceEntry) {
	a.entries = append(a.entries, prioritizedFaceSourceEntry{
		entry:    entry,
		priority: a.priority,
	})
}

// SetPriority sets the priority applied to entries added afterward. A higher
// priority resolves earlier; entries of equal priority resolve in the order
// they are added.
func (a *FaceSourceEntryAdder) SetPriority(priority FaceSourceEntryPriority) {
	a.priority = priority
}

var (
	theAddFuncs []func(*guigui.Context, *FaceSourceEntryAdder)
)

// RegisterFaceSourceEntries registers add as a contributor to the fallback
// resolution stack. add adds its face source entries through the provided
// [FaceSourceEntryAdder].
func RegisterFaceSourceEntries(add func(*guigui.Context, *FaceSourceEntryAdder)) {
	theAddFuncs = append(theAddFuncs, add)
}

func appendFontFaceEntries(entries []FaceSourceEntry, context *guigui.Context) []FaceSourceEntry {
	var adder FaceSourceEntryAdder
	for _, f := range theAddFuncs {
		adder.priority = FaceSourceEntryPriorityNormal
		f(context, &adder)
	}
	slices.SortStableFunc(adder.entries, func(a, b prioritizedFaceSourceEntry) int {
		return int(b.priority) - int(a.priority)
	})
	for _, pe := range adder.entries {
		entries = append(entries, pe.entry)
	}
	return append(entries, theDefaultFaceSource)
}
