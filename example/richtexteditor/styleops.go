// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// boolPropState describes a boolean style property over a toggle action's
// target: a selection, or the caret's typing style.
type boolPropState struct {
	// On reports whether the property is on. For a selection it is
	// meaningful only when Uniform is true.
	On bool

	// Uniform reports whether every byte of the target takes the same
	// property value. It is always true for a caret target.
	Uniform bool
}

// UniformlyOn reports whether the property is on across the whole target.
func (s boolPropState) UniformlyOn() bool {
	return s.Uniform && s.On
}

// WillToggleOn reports whether a toggle action will turn the property on
// across the target; false means it will override the property off
// explicitly.
func (s boolPropState) WillToggleOn() bool {
	return !s.UniformlyOn()
}

// isBoldWeight reports whether weight renders as bold.
func isBoldWeight(weight text.Weight) bool {
	return weight >= text.WeightSemibold
}

// scaleLadder holds the font size multipliers the scale buttons step
// through, in increasing order.
var scaleLadder = []float64{0.75, 1, 1.25, 1.5, 2}

// scaleEpsilon absorbs floating point noise when comparing a scale value
// against a ladder entry.
const scaleEpsilon = 1e-6

// scaleUp returns the smallest ladder entry greater than scale, or the
// largest entry when scale is at or beyond it. A non-uniform scale steps
// from 1.
func scaleUp(scale float64, uniform bool) float64 {
	if !uniform {
		scale = 1
	}
	for _, s := range scaleLadder {
		if s > scale+scaleEpsilon {
			return s
		}
	}
	return scaleLadder[len(scaleLadder)-1]
}

// scaleDown returns the largest ladder entry less than scale, or the
// smallest entry when scale is at or below it. A non-uniform scale steps
// from 1.
func scaleDown(scale float64, uniform bool) float64 {
	if !uniform {
		scale = 1
	}
	for i := len(scaleLadder) - 1; i >= 0; i-- {
		if s := scaleLadder[i]; s < scale-scaleEpsilon {
			return s
		}
	}
	return scaleLadder[0]
}
