// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package cjkfont

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui/basicwidget"
)

//go:generate go run gen.go

//go:embed NotoSansCJK-VF.otf.ttc.gz
var notoSansCJKVFOTFTTCGz []byte

var (
	theFaceSourceSC *text.GoTextFaceSource
	theFaceSourceTC *text.GoTextFaceSource
	theFaceSourceHK *text.GoTextFaceSource
	theFaceSourceJP *text.GoTextFaceSource
	theFaceSourceKR *text.GoTextFaceSource
)

func init() {
	r, err := gzip.NewReader(bytes.NewReader(notoSansCJKVFOTFTTCGz))
	if err != nil {
		panic(err)
	}
	fs, err := text.NewGoTextFaceSourcesFromCollection(r)
	if err != nil {
		panic(err)
	}
	var (
		faceSC *text.GoTextFaceSource
		faceTC *text.GoTextFaceSource
		faceHK *text.GoTextFaceSource
		faceJP *text.GoTextFaceSource
		faceKR *text.GoTextFaceSource
	)
	for _, f := range fs {
		switch f.Metadata().Family {
		case "Noto Sans CJK SC":
			faceSC = f
		case "Noto Sans CJK TC":
			faceTC = f
		case "Noto Sans CJK HK":
			faceHK = f
		case "Noto Sans CJK JP":
			faceJP = f
		case "Noto Sans CJK KR":
			faceKR = f
		default:
			panic(fmt.Sprintf("cjkfont: unknown family: %s", f.Metadata().Family))
		}
	}

	theFaceSourceSC = faceSC
	theFaceSourceTC = faceTC
	theFaceSourceHK = faceHK
	theFaceSourceJP = faceJP
	theFaceSourceKR = faceKR
}

func FaceSourceSC() *text.GoTextFaceSource {
	return theFaceSourceSC
}

func FaceSourceTC() *text.GoTextFaceSource {
	return theFaceSourceTC
}

func FaceSourceHK() *text.GoTextFaceSource {
	return theFaceSourceHK
}

func FaceSourceJP() *text.GoTextFaceSource {
	return theFaceSourceJP
}

func FaceSourceKR() *text.GoTextFaceSource {
	return theFaceSourceKR
}

var (
	ja = language.MustParseBase("ja")
	ko = language.MustParseBase("ko")
	zh = language.MustParseBase("zh")
	cn = language.MustParseRegion("CN")
	tw = language.MustParseRegion("TW")
	mo = language.MustParseRegion("MO")
	hk = language.MustParseRegion("HK")
)

func faceSourceFromLocale(locale language.Tag) *text.GoTextFaceSource {
	if locale == language.Und {
		return nil
	}

	base, conf := locale.Base()
	if conf == language.No {
		return nil
	}
	switch base {
	case ja:
		return theFaceSourceJP
	case ko:
		return theFaceSourceKR
	case zh:
		region, conf := locale.Region()
		if conf == language.No {
			return nil
		}
		switch region {
		case cn:
			return theFaceSourceSC
		case tw, mo:
			return theFaceSourceTC
		case hk:
			return theFaceSourceHK
		}
	}
	return nil
}

func AppendRecommendedFaceSourceEntries(faceSourceEntries []basicwidget.FaceSourceEntry, locales []language.Tag) []basicwidget.FaceSourceEntry {
	var isCJKPrimary bool
	var cjkFaceSource *text.GoTextFaceSource
	for i, locale := range locales {
		fs := faceSourceFromLocale(locale)
		if fs == nil {
			continue
		}
		cjkFaceSource = fs
		isCJKPrimary = i == 0
		break
	}

	if cjkFaceSource == nil {
		// Set a Chinese font as a fallback.
		cjkFaceSource = FaceSourceSC()
	}

	if isCJKPrimary {
		faceSourceEntries = append(faceSourceEntries,
			// There are ambiguous glyphs that are different between CJK and Western fonts.
			// Prefer CJK fonts for such glyphs.
			// See https://www.unicode.org/L2/L2014/14006-sv-western-vs-cjk.pdf
			basicwidget.FaceSourceEntry{
				FaceSource: basicwidget.DefaultFaceSourceEntry().FaceSource,
				UnicodeRanges: []basicwidget.UnicodeRange{
					{
						Min: 0x0000,
						Max: 0x2013,
					},
					// U+2014 EM DASH
					// U+2015 HORIZONTAL BAR
					{
						Min: 0x2016,
						Max: 0x2017,
					},
					// U+2018 LEFT SINGLE QUOTATION MARK
					// U+2019 RIGHT SINGLE QUOTATION MARK
					{
						Min: 0x201a,
						Max: 0x201b,
					},
					// U+201c LEFT DOUBLE QUOTATION MARK
					// U+201d RIGHT DOUBLE QUOTATION MARK
					{
						Min: 0x201e,
						Max: 0x2025,
					},
					// U+2026 HORIZONTAL ELLIPSIS
					{
						Min: 0x2027,
						Max: 0x2e39,
					},
					// U+2e3a TWO-EM DASH
					// U+2e3b THREE-EM DASH
					{
						Min: 0x2e3c,
						Max: 0x7fffffff,
					},
				},
			},
			basicwidget.FaceSourceEntry{
				FaceSource: cjkFaceSource,
			},
			basicwidget.DefaultFaceSourceEntry())
	} else {
		faceSourceEntries = append(faceSourceEntries,
			basicwidget.DefaultFaceSourceEntry(),
			basicwidget.FaceSourceEntry{
				FaceSource: cjkFaceSource,
			})
	}

	return faceSourceEntries
}
