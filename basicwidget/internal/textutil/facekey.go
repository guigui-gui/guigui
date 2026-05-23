// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"
)

// FaceKey is a comparable fingerprint identifying a resolved text face. The
// font is identified by FontID rather than a pointer so that the key carries
// no references to font objects.
type FaceKey struct {
	FontID uint64
	Size   float64
	Weight text.Weight
	Liga   bool
	Tnum   bool
	Lang   language.Tag
}
