// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"strings"

	"github.com/go-text/typesetting/segmenter"
)

var theSegStack []segmenter.Segmenter

func pushSegmenter() *segmenter.Segmenter {
	if len(theSegStack) < cap(theSegStack) {
		theSegStack = theSegStack[:len(theSegStack)+1]
	} else {
		theSegStack = append(theSegStack, segmenter.Segmenter{})
	}
	return &theSegStack[len(theSegStack)-1]
}

func popSegmenter() {
	theSegStack = theSegStack[:len(theSegStack)-1]
}

// initSegmenterWithString initializes seg with str. If str is not valid UTF-8,
// it is sanitized first. The (possibly sanitized) string is returned so that
// callers can use it consistently with byte offsets reported by the segmenter,
// which would otherwise be mismatched against the original invalid-UTF-8 input.
func initSegmenterWithString(seg *segmenter.Segmenter, str string) string {
	if err := seg.InitWithString(str); err != nil {
		str = sanitizeUTF8(str)
		if err := seg.InitWithString(str); err != nil {
			panic("textutil: segmenter.InitWithString failed even after sanitizing: " + err.Error())
		}
	}
	return str
}

func sanitizeUTF8(s string) string {
	var b strings.Builder
	for _, r := range s {
		b.WriteRune(r)
	}
	return b.String()
}
