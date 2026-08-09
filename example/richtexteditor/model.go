// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/guigui-gui/guigui/basicwidget"
)

// Model owns the document: the text value and its ranged style overrides.
type Model struct {
	value  string
	styles basicwidget.TextStyles
	inited bool
}

func (m *Model) ensureInit() {
	if m.inited {
		return
	}
	m.Reset()
}

// Reset replaces the document with the styled sample.
func (m *Model) Reset() {
	m.value = sampleText
	m.styles = sampleStyles()
	m.inited = true
}

func (m *Model) Value() string {
	m.ensureInit()
	return m.value
}

func (m *Model) SetValue(value string) {
	m.ensureInit()
	m.value = value
}

func (m *Model) Styles() basicwidget.TextStyles {
	m.ensureInit()
	return m.styles
}

func (m *Model) SetStyles(styles basicwidget.TextStyles) {
	m.ensureInit()
	m.styles = styles
}

const sampleText = "Rich Text Editor\n" +
	"Select a range and use the toolbar to make it bold, italic, underlined, or struck through, change its color or highlight, and scale it up or down.\n" +
	"With no selection, a toggled style applies to the text typed next."

// styleSampleRange calls f with the byte range of sub's first occurrence in
// [sampleText].
func styleSampleRange(sub string, f func(start, end int)) {
	idx := strings.Index(sampleText, sub)
	if idx < 0 {
		return
	}
	f(idx, idx+len(sub))
}

// sampleStyles returns the initial ranged styles of [sampleText]; the byte
// ranges refer to it.
func sampleStyles() basicwidget.TextStyles {
	var styles basicwidget.TextStyles
	styleSampleRange("Rich Text Editor", func(start, end int) {
		styles.SetWeightInRange(start, end, text.WeightBold)
		styles.SetScaleInRange(start, end, 1.5)
	})
	styleSampleRange("bold", func(start, end int) {
		styles.SetWeightInRange(start, end, text.WeightBold)
	})
	styleSampleRange("italic", func(start, end int) {
		styles.SetItalicInRange(start, end, true)
	})
	styleSampleRange("underlined", func(start, end int) {
		styles.SetUnderlineInRange(start, end, true)
	})
	styleSampleRange("struck through", func(start, end int) {
		styles.SetStrikethroughInRange(start, end, true)
	})
	styleSampleRange("color", func(start, end int) {
		styles.SetColorInRange(start, end, color.RGBA{R: 0xff, G: 0x4b, B: 0x00, A: 0xff})
	})
	styleSampleRange("highlight", func(start, end int) {
		styles.SetBackgroundColorInRange(start, end, color.NRGBA{R: 0xf6, G: 0xaa, B: 0x00, A: 0x40})
	})
	styleSampleRange("scale it up or down", func(start, end int) {
		styles.SetScaleInRange(start, end, 1.25)
	})
	return styles
}
