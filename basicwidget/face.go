// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"slices"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui"
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

var (
	theFaceCache         map[faceCacheKey]text.Face
	theFaceSourceEntries []FaceSourceEntry
)

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
	theFaceSourceEntries = []FaceSourceEntry{e}
}

var (
	tagLiga = text.MustParseTag("liga")
	tagTnum = text.MustParseTag("tnum")
)

func fontFace(size float64, weight text.Weight, liga bool, tnum bool, lang language.Tag) text.Face {
	key := faceCacheKey{
		size:   size,
		weight: weight,
		liga:   liga,
		tnum:   tnum,
		lang:   lang,
	}
	if f, ok := theFaceCache[key]; ok {
		return f
	}

	var fs []text.Face
	for _, entry := range theFaceSourceEntries {
		gtf := &text.GoTextFace{
			Source:   entry.FaceSource,
			Size:     size,
			Language: lang,
		}
		gtf.SetVariation(text.MustParseTag("wght"), float32(weight))
		if liga {
			gtf.SetFeature(tagLiga, 1)
		} else {
			gtf.SetFeature(tagLiga, 0)
		}
		if tnum {
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
		theFaceCache = map[faceCacheKey]text.Face{}
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

// SetFaceSources sets the face sources explicitly.
//
// SetFaceSources and [SetAutoFaceSources] are exclusive.
func SetFaceSources(entries []FaceSourceEntry) {
	if len(entries) == 0 {
		entries = []FaceSourceEntry{theDefaultFaceSource}
	}

	if areFaceSourceEntriesEqual(theFaceSourceEntries, entries) {
		return
	}

	if len(theFaceSourceEntries) < len(entries) {
		theFaceSourceEntries = slices.Grow(theFaceSourceEntries, len(entries))[:len(entries)]
	} else if len(theFaceSourceEntries) > len(entries) {
		theFaceSourceEntries = slices.Delete(theFaceSourceEntries, len(entries), len(theFaceSourceEntries))
	}
	copy(theFaceSourceEntries, entries)

	clear(theFaceCache)
}

type faceCacheKey struct {
	size   float64
	weight text.Weight
	liga   bool
	tnum   bool
	lang   language.Tag
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

var (
	theFontFaceEntries []FaceSourceEntry
)

// SetAutoFaceSources sets the face sources based on the registered fonts by [RegisterFonts].
//
// SetAutoFaceSources should be called every Build in case the locales are changed.
//
// SetAutoFaceSources and [SetFaceSources] are exclusive.
func SetAutoFaceSources(context *guigui.Context) {
	theFontFaceEntries = slices.Delete(theFontFaceEntries, 0, len(theFontFaceEntries))
	slices.SortFunc(theAppendFuncs, func(a, b appendFunc) int {
		if a.priority1 != b.priority1 {
			return int(b.priority1 - a.priority1)
		}
		return b.priority2 - a.priority2
	})
	for _, f := range theAppendFuncs {
		theFontFaceEntries = f.f(theFontFaceEntries, context)
	}
	theFontFaceEntries = append(theFontFaceEntries, theDefaultFaceSource)
	SetFaceSources(theFontFaceEntries)
}
