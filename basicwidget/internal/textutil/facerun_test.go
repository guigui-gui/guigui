// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil_test

import (
	"bytes"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

func testFaces(t *testing.T) (small, large font.Face) {
	t.Helper()
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	smallFace := &text.GoTextFace{Source: source, Size: 16}
	largeFace := &text.GoTextFace{Source: source, Size: 32}
	return font.NewFaceForTest(smallFace, font.Attributes{Size: 16}),
		font.NewFaceForTest(largeFace, font.Attributes{Size: 32})
}

func advanceOf(str string, face font.Face) float64 {
	return text.AdvanceAt(str, len(str), face.TextFace())
}

func TestFaceAt(t *testing.T) {
	small, large := testFaces(t)
	faceRuns := []textutil.FaceRun{
		{Start: 2, End: 5, Face: large},
		{Start: 8, End: 10, Face: large},
	}

	tests := []struct {
		offset     int
		wantFace   font.Face
		wantChange int
	}{
		{offset: 0, wantFace: small, wantChange: 2},
		{offset: 1, wantFace: small, wantChange: 2},
		{offset: 2, wantFace: large, wantChange: 5},
		{offset: 4, wantFace: large, wantChange: 5},
		{offset: 5, wantFace: small, wantChange: 8},
		{offset: 7, wantFace: small, wantChange: 8},
		{offset: 8, wantFace: large, wantChange: 10},
		{offset: 10, wantFace: small, wantChange: math.MaxInt},
	}

	for _, tt := range tests {
		face, change := textutil.FaceAt(faceRuns, small, tt.offset)
		if face != tt.wantFace || change != tt.wantChange {
			t.Errorf("FaceAt(%d): got: %v, %d, want: %v, %d", tt.offset, face.Attributes().Size, change, tt.wantFace.Attributes().Size, tt.wantChange)
		}
	}
}

func TestAdvanceWithFaces(t *testing.T) {
	small, large := testFaces(t)

	t.Run("no runs matches single face", func(t *testing.T) {
		str := "Hello, world"
		got := textutil.AdvanceWithFaces(str, 0, len(str), small, nil, 0, false)
		want := advanceOf(str, small)
		if got != want {
			t.Errorf("got: %f, want: %f", got, want)
		}
	})

	t.Run("segments measured standalone per face", func(t *testing.T) {
		// "abcdef" with "cd" in the large face.
		faceRuns := []textutil.FaceRun{
			{Start: 2, End: 4, Face: large},
		}
		got := textutil.AdvanceWithFaces("abcdef", 0, 6, small, faceRuns, 0, false)
		want := advanceOf("ab", small) + advanceOf("cd", large) + advanceOf("ef", small)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("got: %f, want: %f", got, want)
		}
	})

	t.Run("base rebases run offsets", func(t *testing.T) {
		// The same text as above, but str starts at offset 2 within the
		// run coordinate space.
		faceRuns := []textutil.FaceRun{
			{Start: 2, End: 4, Face: large},
		}
		got := textutil.AdvanceWithFaces("cdef", 2, 4, small, faceRuns, 0, false)
		want := advanceOf("cd", large) + advanceOf("ef", small)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("got: %f, want: %f", got, want)
		}
	})

	t.Run("prefix measurement stops at index", func(t *testing.T) {
		faceRuns := []textutil.FaceRun{
			{Start: 2, End: 4, Face: large},
		}
		got := textutil.AdvanceWithFaces("abcdef", 0, 3, small, faceRuns, 0, false)
		want := advanceOf("ab", small) + advanceOf("c", large)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("got: %f, want: %f", got, want)
		}
	})

	t.Run("tab snaps to the next stop between faces", func(t *testing.T) {
		const tabWidth = 100
		faceRuns := []textutil.FaceRun{
			{Start: 3, End: 5, Face: large},
		}
		got := textutil.AdvanceWithFaces("ab\tcd", 0, 5, small, faceRuns, tabWidth, false)
		abWidth := advanceOf("ab", small)
		afterTab := float64(int(abWidth/tabWidth)+1) * tabWidth
		want := afterTab + advanceOf("cd", large)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("got: %f, want: %f", got, want)
		}
	})

	t.Run("trailing spaces are trimmed", func(t *testing.T) {
		faceRuns := []textutil.FaceRun{
			{Start: 0, End: 2, Face: large},
		}
		got := textutil.AdvanceWithFaces("ab  ", 0, 4, small, faceRuns, 0, false)
		want := advanceOf("ab", large)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("got: %f, want: %f", got, want)
		}
	})
}
