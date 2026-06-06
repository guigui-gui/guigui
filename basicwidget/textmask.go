// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"strings"

	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// maskMapping maps byte offsets between a source string and its masked form.
// A masked text draws one mask rune per grapheme cluster, so a source byte
// offset at cluster boundary k maps to k*len(maskRune) in the masked string,
// and a masked byte offset maps back to the k-th cluster boundary.
type maskMapping struct {
	// maskStr is the display string: the mask rune repeated once per grapheme
	// cluster of the source.
	maskStr string

	// runeLen is the byte length of a single mask rune.
	runeLen int

	// boundaries holds the source byte offsets at grapheme cluster boundaries,
	// including the leading 0 and the trailing source length.
	boundaries []int
}

// reset rebuilds the mapping for src masked with maskRune, reusing the existing
// boundaries capacity.
func (m *maskMapping) reset(src string, maskRune rune) {
	runeStr := string(maskRune)
	m.runeLen = len(runeStr)
	m.boundaries = append(m.boundaries[:0], 0)
	for pos := 0; pos < len(src); {
		next := textutil.NextPositionOnGraphemes(src, pos)
		if next <= pos {
			next = pos + 1
		}
		pos = next
		m.boundaries = append(m.boundaries, pos)
	}
	m.maskStr = strings.Repeat(runeStr, len(m.boundaries)-1)
}

// offsetToMasked maps a source byte offset to the corresponding masked byte
// offset, snapping to the enclosing cluster boundary.
func (m *maskMapping) offsetToMasked(srcOffset int) int {
	var idx int
	for idx+1 < len(m.boundaries) && m.boundaries[idx+1] <= srcOffset {
		idx++
	}
	return idx * m.runeLen
}

// offsetFromMasked maps a masked byte offset back to a source byte offset at a
// cluster boundary.
func (m *maskMapping) offsetFromMasked(maskedOffset int) int {
	idx := min(max(maskedOffset/m.runeLen, 0), len(m.boundaries)-1)
	return m.boundaries[idx]
}
