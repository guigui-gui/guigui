// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textstyle

import (
	"image/color"
)

// Writer receives the serialized style properties of [Runs.WriteStateKey]
// and [Runs.WriteMetricStateKey]. guigui.StateKeyWriter satisfies it.
type Writer interface {
	WriteBool(v bool)
	WriteInt(v int)
	WriteUint16(v uint16)
	WriteUint32(v uint32)
	WriteFloat64(v float64)
}

// WriteStateKey writes the runs into w.
func (r *Runs) WriteStateKey(w Writer) {
	for _, run := range r.runs {
		w.WriteInt(run.Start)
		w.WriteInt(run.End)
		run.Style.writeStateKey(w)
	}
}

// WriteMetricStateKey writes the runs' metric-affecting properties (the
// variation and feature settings and the italic face selection) into w. Runs
// without any contribute nothing.
func (r *Runs) WriteMetricStateKey(w Writer) {
	for _, run := range r.runs {
		if !run.Style.HasMetricProperties() {
			continue
		}
		w.WriteInt(run.Start)
		w.WriteInt(run.End)
		run.Style.writeMetricProperties(w)
	}
}

// HasMetricProperties reports whether any run overrides a metric-affecting
// property.
func (r *Runs) HasMetricProperties() bool {
	for _, run := range r.runs {
		if run.Style.HasMetricProperties() {
			return true
		}
	}
	return false
}

// HasMetricProperties reports whether the style overrides a metric-affecting
// property (a variation or feature setting, or the italic face selection).
func (s Style) HasMetricProperties() bool {
	return len(s.variations) > 0 || len(s.features) > 0 || s.italic.set
}

// writeStateKey writes the style's consumed properties into w.
func (s Style) writeStateKey(w Writer) {
	s.writeMetricProperties(w)
	clr, ok := s.Color()
	w.WriteBool(ok)
	if ok {
		writeColor(w, clr)
	}
	clr, ok = s.BackgroundColor()
	w.WriteBool(ok)
	if ok {
		writeColor(w, clr)
	}
	underline, ok := s.Underline()
	w.WriteBool(ok)
	w.WriteBool(underline)
	strikethrough, ok := s.Strikethrough()
	w.WriteBool(ok)
	w.WriteBool(strikethrough)
}

// writeMetricProperties writes the metric-affecting properties (the
// variation and feature settings and the italic face selection) into w.
func (s Style) writeMetricProperties(w Writer) {
	italic, ok := s.Italic()
	w.WriteBool(ok)
	w.WriteBool(italic)
	variations := s.Variations()
	w.WriteInt(len(variations))
	for _, v := range variations {
		w.WriteUint32(uint32(v.Tag))
		w.WriteFloat64(float64(v.Value))
	}
	features := s.Features()
	w.WriteInt(len(features))
	for _, f := range features {
		w.WriteUint32(uint32(f.Tag))
		w.WriteUint32(f.Value)
	}
}

func writeColor(w Writer, c color.Color) {
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
