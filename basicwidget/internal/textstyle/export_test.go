// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textstyle

import (
	"image/color"
)

func (s Style) WithScale(scale float64) Style {
	s.scale = opt(scale)
	return s
}

func (s Style) WithBackgroundColor(clr color.Color) Style {
	s.backgroundColor = opt(clr)
	return s
}

func (s Style) WithUnderline(underline bool) Style {
	s.underline = opt(underline)
	return s
}

func (s Style) WithStrikethrough(strikethrough bool) Style {
	s.strikethrough = opt(strikethrough)
	return s
}
