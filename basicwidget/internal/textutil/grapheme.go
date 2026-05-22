// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package textutil

import "unicode/utf8"

func PrevPositionOnGraphemes(str string, position int) int {
	if !utf8.ValidString(str) {
		// Invalid UTF-8: byte offsets from segmentation would be into a
		// sanitized string and would not match the input position.
		return position
	}
	var start int
	for end := range theSegmentCache.graphemeBoundaries(str) {
		if position <= end {
			return start
		}
		start = end
	}
	return position
}

func NextPositionOnGraphemes(str string, position int) int {
	if !utf8.ValidString(str) {
		// Invalid UTF-8: byte offsets from segmentation would be into a
		// sanitized string and would not match the input position.
		return position
	}
	var start int
	for end := range theSegmentCache.graphemeBoundaries(str) {
		if position <= start {
			return end
		}
		start = end
	}
	return position
}
