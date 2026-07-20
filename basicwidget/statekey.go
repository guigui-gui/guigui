// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"

	"github.com/guigui-gui/guigui"
)

// writePoint writes an image.Point into w.
func writePoint(w *guigui.StateKeyWriter, p image.Point) {
	w.WriteInt64(int64(p.X))
	w.WriteInt64(int64(p.Y))
}

// writeRectangle writes an image.Rectangle into w.
func writeRectangle(w *guigui.StateKeyWriter, r image.Rectangle) {
	writePoint(w, r.Min)
	writePoint(w, r.Max)
}

// writeColor writes a color.Color into w by its RGBA components.
// A nil color hashes distinctly from any concrete color.
func writeColor(w *guigui.StateKeyWriter, c color.Color) {
	if c == nil {
		w.WriteBool(false)
		return
	}
	w.WriteBool(true)
	r, g, b, a := c.RGBA()
	w.WriteUint16(uint16(r))
	w.WriteUint16(uint16(g))
	w.WriteUint16(uint16(b))
	w.WriteUint16(uint16(a))
}

// writePadding writes a guigui.Padding into w.
func writePadding(w *guigui.StateKeyWriter, p guigui.Padding) {
	w.WriteInt64(int64(p.Start))
	w.WriteInt64(int64(p.Top))
	w.WriteInt64(int64(p.End))
	w.WriteInt64(int64(p.Bottom))
}
