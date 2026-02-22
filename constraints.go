// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package guigui

type constraintsType int

const (
	constraintsTypeNone constraintsType = iota
	constraintsTypeFixedWidth
	constraintsTypeFixedHeight
)

type Constraints struct {
	typ  constraintsType
	size int
}

func (c *Constraints) FixedWidth() (int, bool) {
	if c.typ != constraintsTypeFixedWidth {
		return 0, false
	}
	return c.size, true
}

func (c *Constraints) FixedHeight() (int, bool) {
	if c.typ != constraintsTypeFixedHeight {
		return 0, false
	}
	return c.size, true
}

func FixedWidthConstraints(w int) Constraints {
	return Constraints{
		typ:  constraintsTypeFixedWidth,
		size: w,
	}
}

func FixedHeightConstraints(h int) Constraints {
	return Constraints{
		typ:  constraintsTypeFixedHeight,
		size: h,
	}
}
