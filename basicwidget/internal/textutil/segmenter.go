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

func initSegmenterWithString(seg *segmenter.Segmenter, str string) {
	if err := seg.InitWithString(str); err != nil {
		str = sanitizeUTF8(str)
		if err := seg.InitWithString(str); err != nil {
			panic("textutil: segmenter.InitWithString failed even after sanitizing: " + err.Error())
		}
	}
}

func sanitizeUTF8(s string) string {
	var b strings.Builder
	for _, r := range s {
		b.WriteRune(r)
	}
	return b.String()
}
