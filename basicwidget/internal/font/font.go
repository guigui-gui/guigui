// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package font

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"
)

// Key is a comparable fingerprint identifying a resolved text face. The
// font is identified by FontID rather than a pointer so that the key carries
// no references to font objects.
type Key struct {
	FontID uint64
	Size   float64
	Weight text.Weight
	Liga   bool
	Tnum   bool
	Lang   language.Tag
}

// Face pairs a resolved text face with the Key that identifies it.
type Face struct {
	key  Key
	face text.Face
}

// NewFace pairs a resolved face with the key it was built from. The caller is
// responsible for passing a face that key identifies.
//
// TODO: Ideally NewFace would resolve the face from key itself, so an
// inconsistent key/face pair would be impossible to construct. Resolution
// currently lives in the basicwidget package (it needs the registered fonts
// and the locale context), which this package cannot import.
func NewFace(key Key, face text.Face) Face {
	return Face{
		key:  key,
		face: face,
	}
}

// Key returns the key the face was built from.
func (f Face) Key() Key {
	return f.key
}

// TextFace returns the resolved face.
func (f Face) TextFace() text.Face {
	return f.face
}
