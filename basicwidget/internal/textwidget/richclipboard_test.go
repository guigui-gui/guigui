// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget_test

import (
	"image/color"
	"slices"
	"strings"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
	"github.com/guigui-gui/guigui/basicwidget/internal/textwidget"
	"github.com/guigui-gui/guigui/internal/clipboard"
)

// setUpFakeClipboard substitutes the OS clipboard with an in-process fake
// and empties the rich clipboard slot. The fake and the slot are restored
// and reset when the test finishes.
func setUpFakeClipboard(t *testing.T) *clipboard.Contents {
	t.Helper()
	var contents clipboard.Contents
	restore := textwidget.SetClipboardForTest(
		func() (clipboard.Contents, error) {
			return contents, nil
		},
		func(c clipboard.Contents) error {
			contents = c
			return nil
		},
	)
	textwidget.ResetRichClipboardForTest()
	t.Cleanup(func() {
		restore()
		textwidget.ResetRichClipboardForTest()
	})
	return &contents
}

func newRichEditableText(value string) *textwidget.Text {
	var txt textwidget.Text
	txt.SetMultiline(true)
	txt.SetValue(value)
	txt.SetEditable(true)
	txt.SetRichTextEditable(true)
	return &txt
}

func TestRichClipboardCopyPasteRoundTrip(t *testing.T) {
	contents := setUpFakeClipboard(t)
	red := color.RGBA{R: 0xff, A: 0xff}

	txt := newRichEditableText("Hello world")
	txt.SetColorInRange(0, 5, red)
	txt.SetUnderlineInRange(6, 11, true)

	txt.SetSelection(0, 5)
	if !txt.Copy() {
		t.Fatal("Copy failed")
	}
	if got, want := string(contents.Text), "Hello"; got != want {
		t.Errorf("clipboard text: got %q, want %q", got, want)
	}
	if got, want := string(contents.HTML), `<span style="color:#ff0000">Hello</span>`; got != want {
		t.Errorf("clipboard HTML: got %q, want %q", got, want)
	}

	txt.SetSelection(11, 11)
	if !txt.Paste() {
		t.Fatal("Paste failed")
	}
	if got, want := txt.Value(), "Hello worldHello"; got != want {
		t.Errorf("value: got %q, want %q", got, want)
	}
	var wantRuns textstyle.Runs
	wantRuns.SetColor(0, 5, red)
	wantRuns.SetUnderline(6, 11, true)
	wantRuns.SetColor(11, 16, red)
	if got, want := txt.StyleRuns(), slices.Collect(wantRuns.All()); !equalStyleRuns(got, want) {
		t.Errorf("style runs: got %+v, want %+v", got, want)
	}
}

func TestRichClipboardPasteReplacesAdoptedStyles(t *testing.T) {
	setUpFakeClipboard(t)
	red := color.RGBA{R: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}

	txt := newRichEditableText("red plain")
	txt.SetColorInRange(0, 3, red)

	// Copy the partially styled range [2, 6): "d pl" with "d" red.
	txt.SetSelection(2, 6)
	if !txt.Copy() {
		t.Fatal("Copy failed")
	}

	// Paste inside a differently styled range; the pasted span must carry
	// the copied styles, not the styles adopted from the surrounding text.
	txt.SetSelection(9, 9)
	txt.SetColorInRange(4, 9, blue)
	if !txt.Paste() {
		t.Fatal("Paste failed")
	}
	if got, want := txt.Value(), "red plaind pl"; got != want {
		t.Errorf("value: got %q, want %q", got, want)
	}
	var wantRuns textstyle.Runs
	wantRuns.SetColor(0, 3, red)
	wantRuns.SetColor(4, 9, blue)
	wantRuns.SetColor(9, 10, red)
	if got, want := txt.StyleRuns(), slices.Collect(wantRuns.All()); !equalStyleRuns(got, want) {
		t.Errorf("style runs: got %+v, want %+v", got, want)
	}
}

func TestRichClipboardCutKeepsStyles(t *testing.T) {
	contents := setUpFakeClipboard(t)
	red := color.RGBA{R: 0xff, A: 0xff}

	txt := newRichEditableText("abcdef")
	txt.SetColorInRange(2, 4, red)

	txt.SetSelection(1, 5)
	if !txt.Cut() {
		t.Fatal("Cut failed")
	}
	if got, want := txt.Value(), "af"; got != want {
		t.Errorf("value: got %q, want %q", got, want)
	}
	if got, want := string(contents.Text), "bcde"; got != want {
		t.Errorf("clipboard text: got %q, want %q", got, want)
	}

	txt.SetSelection(1, 1)
	if !txt.Paste() {
		t.Fatal("Paste failed")
	}
	if got, want := txt.Value(), "abcdef"; got != want {
		t.Errorf("value: got %q, want %q", got, want)
	}
	var wantRuns textstyle.Runs
	wantRuns.SetColor(2, 4, red)
	if got, want := txt.StyleRuns(), slices.Collect(wantRuns.All()); !equalStyleRuns(got, want) {
		t.Errorf("style runs: got %+v, want %+v", got, want)
	}
}

func TestRichClipboardCrossWidgetPaste(t *testing.T) {
	setUpFakeClipboard(t)
	red := color.RGBA{R: 0xff, A: 0xff}

	src := newRichEditableText("styled")
	src.SetColorInRange(0, 6, red)
	src.SetSelection(0, 6)
	if !src.Copy() {
		t.Fatal("Copy failed")
	}

	dst := newRichEditableText("target: ")
	dst.SetSelection(8, 8)
	if !dst.Paste() {
		t.Fatal("Paste failed")
	}
	if got, want := dst.Value(), "target: styled"; got != want {
		t.Errorf("value: got %q, want %q", got, want)
	}
	var wantRuns textstyle.Runs
	wantRuns.SetColor(8, 14, red)
	if got, want := dst.StyleRuns(), slices.Collect(wantRuns.All()); !equalStyleRuns(got, want) {
		t.Errorf("style runs: got %+v, want %+v", got, want)
	}
}

func TestRichClipboardExternalOverwriteFallsBackToPlain(t *testing.T) {
	contents := setUpFakeClipboard(t)
	red := color.RGBA{R: 0xff, A: 0xff}

	txt := newRichEditableText("styled")
	txt.SetColorInRange(0, 6, red)
	txt.SetSelection(0, 6)
	if !txt.Copy() {
		t.Fatal("Copy failed")
	}

	// Simulate another application replacing the clipboard contents.
	*contents = clipboard.Contents{Text: []byte("other")}

	dst := newRichEditableText("")
	dst.SetSelection(0, 0)
	if !dst.Paste() {
		t.Fatal("Paste failed")
	}
	if got, want := dst.Value(), "other"; got != want {
		t.Errorf("value: got %q, want %q", got, want)
	}
	if got := dst.StyleRuns(); len(got) != 0 {
		t.Errorf("style runs: got %+v, want none", got)
	}
}

func TestRichClipboardPasteDisabledByDefault(t *testing.T) {
	setUpFakeClipboard(t)
	red := color.RGBA{R: 0xff, A: 0xff}

	src := newRichEditableText("styled")
	src.SetColorInRange(0, 6, red)
	src.SetSelection(0, 6)
	if !src.Copy() {
		t.Fatal("Copy failed")
	}

	var dst textwidget.Text
	dst.SetMultiline(true)
	dst.SetValue("")
	dst.SetEditable(true)
	dst.SetSelection(0, 0)
	if !dst.Paste() {
		t.Fatal("Paste failed")
	}
	if got, want := dst.Value(), "styled"; got != want {
		t.Errorf("value: got %q, want %q", got, want)
	}
	if got := dst.StyleRuns(); len(got) != 0 {
		t.Errorf("style runs: got %+v, want none", got)
	}
}

func TestRichClipboardPasteWithoutStyles(t *testing.T) {
	setUpFakeClipboard(t)
	red := color.RGBA{R: 0xff, A: 0xff}

	src := newRichEditableText("styled")
	src.SetColorInRange(0, 6, red)
	src.SetSelection(0, 6)
	if !src.Copy() {
		t.Fatal("Copy failed")
	}

	// The destination is rich text editable and the slot matches the
	// clipboard, yet PasteWithoutStyles must not apply the copied styles.
	dst := newRichEditableText("")
	dst.SetSelection(0, 0)
	if !dst.PasteWithoutStyles() {
		t.Fatal("PasteWithoutStyles failed")
	}
	if got, want := dst.Value(), "styled"; got != want {
		t.Errorf("value: got %q, want %q", got, want)
	}
	if got := dst.StyleRuns(); len(got) != 0 {
		t.Errorf("style runs: got %+v, want none", got)
	}
}

func TestRichClipboardUnstyledCopyClearsSlot(t *testing.T) {
	setUpFakeClipboard(t)
	red := color.RGBA{R: 0xff, A: 0xff}

	txt := newRichEditableText("abc abc")
	txt.SetColorInRange(0, 3, red)

	// Copy the styled "abc", then the unstyled "abc". The second copy must
	// supersede the first even though the plain texts are equal.
	txt.SetSelection(0, 3)
	if !txt.Copy() {
		t.Fatal("Copy failed")
	}
	txt.SetSelection(4, 7)
	if !txt.Copy() {
		t.Fatal("Copy failed")
	}

	txt.SetSelection(7, 7)
	if !txt.Paste() {
		t.Fatal("Paste failed")
	}
	if got, want := txt.Value(), "abc abcabc"; got != want {
		t.Errorf("value: got %q, want %q", got, want)
	}
	var wantRuns textstyle.Runs
	wantRuns.SetColor(0, 3, red)
	if got, want := txt.StyleRuns(), slices.Collect(wantRuns.All()); !equalStyleRuns(got, want) {
		t.Errorf("style runs: got %+v, want %+v", got, want)
	}
}

func TestRichClipboardSingleLinePasteOfMultiLineCopyFallsBackToPlain(t *testing.T) {
	setUpFakeClipboard(t)
	red := color.RGBA{R: 0xff, A: 0xff}

	src := newRichEditableText("one\ntwo")
	src.SetColorInRange(4, 7, red)
	src.SetSelection(0, 7)
	if !src.Copy() {
		t.Fatal("Copy failed")
	}

	var dst textwidget.Text
	dst.SetValue("")
	dst.SetEditable(true)
	dst.SetRichTextEditable(true)
	dst.SetSelection(0, 0)
	if !dst.Paste() {
		t.Fatal("Paste failed")
	}
	if got, want := dst.Value(), "one two"; got != want {
		t.Errorf("value: got %q, want %q", got, want)
	}
	if got := dst.StyleRuns(); len(got) != 0 {
		t.Errorf("style runs: got %+v, want none", got)
	}
}

func TestRichClipboardCopyWithoutStylesHasNoHTML(t *testing.T) {
	contents := setUpFakeClipboard(t)

	txt := newRichEditableText("plain <text>")
	txt.SetSelection(0, 12)
	if !txt.Copy() {
		t.Fatal("Copy failed")
	}
	if got, want := string(contents.Text), "plain <text>"; got != want {
		t.Errorf("clipboard text: got %q, want %q", got, want)
	}
	if contents.HTML != nil {
		t.Errorf("clipboard HTML: got %q, want nil", contents.HTML)
	}
}

func TestRichClipboardUndoRestoresPrePasteStyles(t *testing.T) {
	setUpFakeClipboard(t)
	red := color.RGBA{R: 0xff, A: 0xff}

	txt := newRichEditableText("styled plain")
	txt.SetColorInRange(0, 6, red)
	txt.SetSelection(0, 6)
	if !txt.Copy() {
		t.Fatal("Copy failed")
	}

	txt.SetSelection(12, 12)
	if !txt.Paste() {
		t.Fatal("Paste failed")
	}
	if got, want := txt.Value(), "styled plainstyled"; got != want {
		t.Fatalf("value: got %q, want %q", got, want)
	}

	if !txt.Undo() {
		t.Fatal("Undo failed")
	}
	if got, want := txt.Value(), "styled plain"; got != want {
		t.Errorf("value after undo: got %q, want %q", got, want)
	}
	var wantRuns textstyle.Runs
	wantRuns.SetColor(0, 6, red)
	if got, want := txt.StyleRuns(), slices.Collect(wantRuns.All()); !equalStyleRuns(got, want) {
		t.Errorf("style runs after undo: got %+v, want %+v", got, want)
	}

	if !txt.Redo() {
		t.Fatal("Redo failed")
	}
	if got, want := txt.Value(), "styled plainstyled"; got != want {
		t.Errorf("value after redo: got %q, want %q", got, want)
	}
	wantRuns.SetColor(12, 18, red)
	if got, want := txt.StyleRuns(), slices.Collect(wantRuns.All()); !equalStyleRuns(got, want) {
		t.Errorf("style runs after redo: got %+v, want %+v", got, want)
	}
}

func TestRichClipboardHTMLEscapesCopiedText(t *testing.T) {
	contents := setUpFakeClipboard(t)
	red := color.RGBA{R: 0xff, A: 0xff}

	txt := newRichEditableText("a<b>&c")
	txt.SetColorInRange(0, 6, red)
	txt.SetSelection(0, 6)
	if !txt.Copy() {
		t.Fatal("Copy failed")
	}
	html := string(contents.HTML)
	if strings.Contains(html, "<b>") {
		t.Errorf("clipboard HTML contains unescaped markup: %q", html)
	}
	if want := "a&lt;b&gt;&amp;c"; !strings.Contains(html, want) {
		t.Errorf("clipboard HTML: got %q, want it to contain %q", html, want)
	}
}
