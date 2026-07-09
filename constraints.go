// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package guigui

type constraintsType int

const (
	constraintsTypeNone constraintsType = iota
	constraintsTypeFixedWidth
	constraintsTypeFixedHeight
)

// Constraints specifies at most one fixed dimension for measuring a widget.
// The zero value leaves both dimensions unconstrained.
type Constraints struct {
	typ  constraintsType
	size int
}

// FixedWidth returns the fixed width, if one is specified.
func (c *Constraints) FixedWidth() (int, bool) {
	if c.typ != constraintsTypeFixedWidth {
		return 0, false
	}
	return c.size, true
}

// FixedHeight returns the fixed height, if one is specified.
func (c *Constraints) FixedHeight() (int, bool) {
	if c.typ != constraintsTypeFixedHeight {
		return 0, false
	}
	return c.size, true
}

// FixedWidthConstraints returns constraints with a fixed width.
func FixedWidthConstraints(w int) Constraints {
	return Constraints{
		typ:  constraintsTypeFixedWidth,
		size: w,
	}
}

// FixedHeightConstraints returns constraints with a fixed height.
func FixedHeightConstraints(h int) Constraints {
	return Constraints{
		typ:  constraintsTypeFixedHeight,
		size: h,
	}
}
