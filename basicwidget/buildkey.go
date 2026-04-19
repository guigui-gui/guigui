// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"

	"github.com/guigui-gui/guigui"
)

// writePoint writes an image.Point into h.
func writePoint(h *guigui.BuildKeyHasher, p image.Point) {
	h.WriteInt64(int64(p.X))
	h.WriteInt64(int64(p.Y))
}

// writeRectangle writes an image.Rectangle into h.
func writeRectangle(h *guigui.BuildKeyHasher, r image.Rectangle) {
	writePoint(h, r.Min)
	writePoint(h, r.Max)
}

// writeRGBA64 writes a color.RGBA64 into h.
func writeRGBA64(h *guigui.BuildKeyHasher, c color.RGBA64) {
	h.WriteUint64(uint64(c.R))
	h.WriteUint64(uint64(c.G))
	h.WriteUint64(uint64(c.B))
	h.WriteUint64(uint64(c.A))
}

// writeColor writes a color.Color into h by its RGBA components.
// A nil color hashes distinctly from any concrete color.
func writeColor(h *guigui.BuildKeyHasher, c color.Color) {
	if c == nil {
		h.WriteBool(false)
		return
	}
	h.WriteBool(true)
	r, g, b, a := c.RGBA()
	h.WriteUint32(r)
	h.WriteUint32(g)
	h.WriteUint32(b)
	h.WriteUint32(a)
}

// writePadding writes a guigui.Padding into h.
func writePadding(h *guigui.BuildKeyHasher, p guigui.Padding) {
	h.WriteInt64(int64(p.Start))
	h.WriteInt64(int64(p.Top))
	h.WriteInt64(int64(p.End))
	h.WriteInt64(int64(p.Bottom))
}
