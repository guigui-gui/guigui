// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textstyle

import (
	"image/color"

	"golang.org/x/text/language"
)

// Writer receives the serialized style properties of [Runs.WriteStateKey]
// and [Runs.WriteMetricStateKey]. guigui.StateKeyWriter satisfies it.
type Writer interface {
	WriteBool(v bool)
	WriteInt(v int)
	WriteUint16(v uint16)
	WriteUint32(v uint32)
	WriteUint64(v uint64)
	WriteFloat64(v float64)
	WriteString(v string)
}

// WriteStateKey writes the runs into w.
func (r *Runs) WriteStateKey(w Writer) {
	for _, run := range r.runs {
		w.WriteInt(run.Start)
		w.WriteInt(run.End)
		run.Style.writeStateKey(w)
	}
}

// WriteMetricStateKey writes the runs' metric-affecting properties into w.
// Runs without any contribute nothing.
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
// property, such as a variation setting or the font family.
func (s Style) HasMetricProperties() bool {
	return len(s.variations) > 0 || len(s.features) > 0 || s.italic.set || s.family.set || s.lang.set
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

// writeMetricProperties writes the metric-affecting properties into w.
func (s Style) writeMetricProperties(w Writer) {
	family, ok := s.Family()
	w.WriteBool(ok)
	if ok {
		var id uint64
		if family != nil {
			id = family.ID()
		}
		w.WriteUint64(id)
	}
	italic, ok := s.Italic()
	w.WriteBool(ok)
	w.WriteBool(italic)
	lang, ok := s.Lang()
	w.WriteBool(ok)
	if ok {
		w.WriteString(langString(lang))
	}
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

// langStrings caches the string forms of language tags, so writing a state
// key does not re-resolve them.
var langStrings = map[language.Tag]string{}

// langString returns lang's string form.
func langString(lang language.Tag) string {
	s, ok := langStrings[lang]
	if !ok {
		s = lang.String()
		langStrings[lang] = s
	}
	return s
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
