// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"image/color"

	"github.com/guigui-gui/guigui"
)

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
