// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textstyle

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// uniformValue returns the value a single style property takes over every
// byte of [start, end) and whether that value is uniform. A byte covered by
// a run whose style sets the property takes the run's value; every other
// byte takes fallback. value reports the property of a style. An empty
// range is uniform with fallback.
func uniformValue[T comparable](r *Runs, start, end int, fallback T, value func(Style) (T, bool)) (T, bool) {
	start = max(start, 0)
	if start >= end {
		return fallback, true
	}

	var result T
	var resultSet bool
	merge := func(v T) bool {
		if !resultSet {
			result = v
			resultSet = true
			return true
		}
		return result == v
	}

	// pos is the start of the part of [start, end) not yet merged.
	pos := start
	for _, run := range r.runs {
		if run.End <= start {
			continue
		}
		if run.Start >= end {
			break
		}
		if run.Start > pos && !merge(fallback) {
			var zero T
			return zero, false
		}
		v, ok := value(run.Style)
		if !ok {
			v = fallback
		}
		if !merge(v) {
			var zero T
			return zero, false
		}
		pos = min(run.End, end)
	}
	if pos < end && !merge(fallback) {
		var zero T
		return zero, false
	}
	return result, true
}

// UniformItalic returns the italic state every byte of [start, end) takes
// and whether the state is uniform. A byte with an italic override takes
// the override's value; every other byte takes fallback. An empty range is
// uniform with fallback.
func (r *Runs) UniformItalic(start, end int, fallback bool) (italic, uniform bool) {
	return uniformValue(r, start, end, fallback, func(s Style) (bool, bool) {
		return s.Italic()
	})
}

// UniformScale returns the font size multiplier every byte of [start, end)
// takes and whether the multiplier is uniform. A byte with a scale override
// takes the override's value; every other byte takes fallback. An empty
// range is uniform with fallback.
func (r *Runs) UniformScale(start, end int, fallback float64) (scale float64, uniform bool) {
	return uniformValue(r, start, end, fallback, func(s Style) (float64, bool) {
		return s.Scale()
	})
}

// UniformColor returns the text color every byte of [start, end) takes and
// whether the color is uniform. A byte with a color override takes the
// override's value; every other byte takes fallback. An empty range is
// uniform with fallback.
func (r *Runs) UniformColor(start, end int, fallback color.Color) (clr color.Color, uniform bool) {
	return uniformValue(r, start, end, fallback, func(s Style) (color.Color, bool) {
		return s.Color()
	})
}

// UniformUnderline returns the underline state every byte of [start, end)
// takes and whether the state is uniform. A byte with an underline override
// takes the override's value; every other byte takes fallback. An empty
// range is uniform with fallback.
func (r *Runs) UniformUnderline(start, end int, fallback bool) (underline, uniform bool) {
	return uniformValue(r, start, end, fallback, func(s Style) (bool, bool) {
		return s.Underline()
	})
}

// UniformStrikethrough returns the strikethrough state every byte of
// [start, end) takes and whether the state is uniform. A byte with a
// strikethrough override takes the override's value; every other byte takes
// fallback. An empty range is uniform with fallback.
func (r *Runs) UniformStrikethrough(start, end int, fallback bool) (strikethrough, uniform bool) {
	return uniformValue(r, start, end, fallback, func(s Style) (bool, bool) {
		return s.Strikethrough()
	})
}

// UniformVariation returns the value of the OpenType variation axis tag
// every byte of [start, end) takes and whether the value is uniform. A byte
// with an override of the axis takes the override's value; every other byte
// takes fallback. An empty range is uniform with fallback.
func (r *Runs) UniformVariation(start, end int, tag text.Tag, fallback float32) (value float32, uniform bool) {
	return uniformValue(r, start, end, fallback, func(s Style) (float32, bool) {
		return s.Variation(tag)
	})
}
