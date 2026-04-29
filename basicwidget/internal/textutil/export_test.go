// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package textutil

import (
	"iter"
)

type VisualLine struct {
	Pos int
	Str string
}

func VisualLines(width int, str string, autoWrap bool, advance func(str string) float64) iter.Seq[VisualLine] {
	return func(yield func(VisualLine) bool) {
		for l := range visualLines(width, str, autoWrap, advance) {
			if !yield(VisualLine{
				Pos: l.pos,
				Str: l.str,
			}) {
				return
			}
		}
	}
}

func NextIndentPosition(position float64, indentWidth float64) float64 {
	return nextIndentPosition(position, indentWidth)
}
