// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"log/slog"

	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
	"github.com/guigui-gui/guigui/internal/clipboard"
)

// clipboardRead and clipboardWrite access the OS clipboard. They are
// variables so tests can substitute an in-process fake.
var (
	clipboardRead  = clipboard.Read
	clipboardWrite = clipboard.Write
)

// theRichClipboard is the process-global rich-text clipboard slot: the plain
// text of the last copy or cut together with its style overrides. A rich
// paste consumes the slot only while its text equals the OS clipboard's
// plain text, so a clipboard write by another application invalidates the
// slot implicitly.
var theRichClipboard struct {
	// text is the plain text of the last copy or cut, or empty when the
	// slot is unfilled.
	text string

	// styleRuns holds the copied range's style overrides, rebased to byte
	// offset 0.
	styleRuns textstyle.Runs
}

// copyRangeToClipboard writes the text in [start, end) to the OS clipboard
// and fills the rich clipboard slot with the range's style overrides. A
// styled range also gets an HTML flavor on the OS clipboard.
func (t *Text) copyRangeToClipboard(start, end int) bool {
	text := t.stringValueWithRange(start, end)
	theRichClipboard.text = text
	theRichClipboard.styleRuns.CopyRangeFrom(t.ensureStyleRuns(), start, end)

	contents := clipboard.Contents{
		Text: []byte(text),
	}
	if !theRichClipboard.styleRuns.IsEmpty() {
		contents.HTML = textstyle.AppendHTML(nil, text, &theRichClipboard.styleRuns)
	}
	if err := clipboardWrite(contents); err != nil {
		slog.Error(err.Error())
		return false
	}
	return true
}

// canPasteRichText reports whether pasting text should apply the rich
// clipboard slot's style overrides.
func (t *Text) canPasteRichText(text string) bool {
	if !t.richTextEditable || t.masking() {
		return false
	}
	if text == "" || theRichClipboard.text != text {
		return false
	}
	if !t.IsMultiline() {
		// A single-line widget replaces line breaks with spaces on
		// insertion, shifting the slot's byte offsets; paste plain instead.
		if replaced, _, _ := replaceNewLinesWithSpace(text, 0, 0); replaced != text {
			return false
		}
	}
	return true
}
