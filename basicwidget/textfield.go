// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image"
	"io"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/exp/textinput"

	"github.com/guigui-gui/guigui/basicwidget/internal/piecetable"
)

// textField is the editable backing store behind [Text]. It wraps a
// [piecetable.PieceTable] for the committed buffer and a
// [textinput.Composer] for IME composition.
type textField struct {
	pieceTable            piecetable.PieceTable
	selectionStartInBytes int
	selectionEndInBytes   int

	bounds  image.Rectangle
	focused bool

	composer       textinput.Composer
	composerInited bool

	// composition holds the active preedit reported by the IME. composition
	// is the empty string when no composition is in progress.
	composition         string
	compositionSelStart int
	compositionSelEnd   int

	// imeTextStart and imeTextEnd are the absolute byte bounds of the region
	// that anchors the IME's view of the document for the active session.
	// onIMECommit translates the IME's replacement coordinates back to
	// document coordinates relative to this region.
	//
	// With a collapsed selection the region equals the joined
	// TextBeforeCaret + TextAfterCaret slice handed to the IME. With a
	// non-empty selection [lineAroundSelection] excludes the selection from
	// the surrounding text, so the region also straddles the selection;
	// [imeTextStart, imeTextEnd) is then larger than the joined slice by
	// exactly the selection's length. onIMECommit copes with that disjoint
	// mapping by diffing the IME's intended new content against the actual
	// bytes in [imeTextStart, imeTextEnd) rather than trusting len(newBefore)
	// / len(newAfter) to map directly to document offsets.
	imeTextStart int
	imeTextEnd   int

	err error

	generation int64
}

func (f *textField) ensureComposerInited() {
	if f.composerInited {
		return
	}
	f.composer.OnNewSession = f.onNewIMESession
	f.composer.OnComposition = f.onIMEComposition
	f.composer.OnCommit = f.onIMECommit
	f.composerInited = true
}

func (f *textField) onNewIMESession() *textinput.SessionOptions {
	before, after := f.lineAroundSelection()
	f.imeTextStart = f.selectionStartInBytes - len(before)
	f.imeTextEnd = f.selectionEndInBytes + len(after)
	return &textinput.SessionOptions{
		CaretBounds:     f.bounds,
		TextBeforeCaret: before,
		TextAfterCaret:  after,
	}
}

func (f *textField) onIMEComposition(c *textinput.Composition) {
	text := c.Text()
	selStart, selEnd := c.SelectionRangeInBytes()
	if f.composition == text && f.compositionSelStart == selStart && f.compositionSelEnd == selEnd {
		return
	}
	f.composition = text
	f.compositionSelStart = selStart
	f.compositionSelEnd = selEnd
	f.bumpGeneration()
}

func (f *textField) onIMECommit(c *textinput.Commit) {
	text := c.Text()
	beforeRepl, afterRepl := c.IsSurroundingTextReplaced()
	if !beforeRepl && !afterRepl {
		// Typical case: insert Text at the current selection.
		s, e := f.selectionStartInBytes, f.selectionEndInBytes
		if s > e {
			s, e = e, s
		}
		f.pieceTable.UpdateByIME(text, s, e)
		f.selectionStartInBytes = s + len(text)
		f.selectionEndInBytes = f.selectionStartInBytes
		f.composition = ""
		f.compositionSelStart = 0
		f.compositionSelEnd = 0
		f.bumpGeneration()
		return
	}

	// Surrounding-text replacement. The IME's intended new content for
	// [imeTextStart, imeTextEnd) is newBefore + Text + newAfter; diff it
	// against the actual bytes currently there to find the smallest edit.
	//
	// Diffing the full new content (rather than just c.Text()) handles
	// every uncommon span — bytes the IME pulled from one side of the
	// caret onto the other (newBefore can extend past the original
	// TextBeforeCaret, and likewise for newAfter), bytes that ended up in
	// the joined surrounding text but were never part of the slice handed
	// to the IME (e.g. a selection that lineAroundSelection excluded), and
	// any drift between the document and the IME's view. The common prefix
	// and suffix give the true unchanged span; the middle is what
	// UpdateByIME records, keeping the IME-merge undo entry tight.
	newBefore, newAfter := c.SurroundingText()
	newContent := newBefore + text + newAfter

	var sb strings.Builder
	_, _ = f.pieceTable.WriteRangeTo(&sb, f.imeTextStart, f.imeTextEnd)
	oldContent := sb.String()

	prefixLen := commonPrefixLen(oldContent, newContent)
	suffixLen := commonSuffixLen(oldContent[prefixLen:], newContent[prefixLen:])
	insStart := f.imeTextStart + prefixLen
	insEnd := f.imeTextEnd - suffixLen
	insText := newContent[prefixLen : len(newContent)-suffixLen]

	f.pieceTable.UpdateByIME(insText, insStart, insEnd)
	// Cursor lands at the end of the IME's committed text within the new
	// joined content laid out at [imeTextStart, imeTextEnd).
	f.selectionStartInBytes = f.imeTextStart + len(newBefore) + len(text)
	f.selectionEndInBytes = f.selectionStartInBytes
	f.composition = ""
	f.compositionSelStart = 0
	f.compositionSelEnd = 0
	f.bumpGeneration()
}

// commonPrefixLen returns the length in bytes of the longest common prefix
// of a and b.
func commonPrefixLen(a, b string) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// commonSuffixLen returns the length in bytes of the longest common suffix
// of a and b.
func commonSuffixLen(a, b string) int {
	la, lb := len(a), len(b)
	n := min(la, lb)
	for i := range n {
		if a[la-1-i] != b[lb-1-i] {
			return i
		}
	}
	return n
}

// lineAroundSelection returns the bytes of the current logical line on either
// side of the selection. Both halves combined form the surrounding text the
// IME uses for prediction and reconversion.
func (f *textField) lineAroundSelection() (before, after string) {
	selStart, selEnd := f.selectionStartInBytes, f.selectionEndInBytes
	if selStart > selEnd {
		selStart, selEnd = selEnd, selStart
	}
	lineStart, lineEnd := f.pieceTable.FindLineBounds(selStart, selEnd)

	var sb strings.Builder
	_, _ = f.pieceTable.WriteRangeTo(&sb, lineStart, selStart)
	before = sb.String()
	sb.Reset()
	_, _ = f.pieceTable.WriteRangeTo(&sb, selEnd, lineEnd)
	after = sb.String()
	return before, after
}

func (f *textField) bumpGeneration() {
	f.generation++
}

// Generation returns a counter that advances when the field's renderable
// content changes. Selection-only changes do not advance Generation.
func (f *textField) Generation() int64 {
	return f.generation
}

// Selection returns the current selection range in bytes.
func (f *textField) Selection() (startInBytes, endInBytes int) {
	return f.selectionStartInBytes, f.selectionEndInBytes
}

// IsFocused reports whether the field is focused.
func (f *textField) IsFocused() bool {
	return f.focused
}

// Focus marks the field as focused. The Composer is driven only while the
// field is focused.
func (f *textField) Focus() {
	if f.focused {
		return
	}
	f.focused = true
}

// Blur removes the focus from the field, cancelling any active IME
// session. [textinput.Composer.Cancel] fires [textinput.Composer.OnComposition]
// with an empty composition, which clears the composition state via
// [textField.onIMEComposition].
func (f *textField) Blur() {
	if !f.focused {
		return
	}
	f.focused = false
	f.composer.Cancel()
}

// SetBounds sets the bounds used for IME window positioning. The bounds are
// captured at the start of the next IME session.
func (f *textField) SetBounds(bounds image.Rectangle) {
	f.bounds = bounds
}

// Update drives the IME composer for one tick. Returns handled=true when the
// IME consumed input; the caller should suppress its own key handlers in
// that case.
func (f *textField) Update() (handled bool, err error) {
	if f.err != nil {
		return false, f.err
	}
	if !f.focused {
		return false, nil
	}
	f.ensureComposerInited()
	handled, err = f.composer.Update()
	if err != nil {
		f.err = err
		return false, f.err
	}
	return handled, nil
}

// TextLengthInBytes returns the length of the current text in bytes.
func (f *textField) TextLengthInBytes() int {
	return f.pieceTable.Len()
}

// UncommittedTextLengthInBytes returns the active composition length in
// bytes when the field is focused. Returns 0 otherwise.
func (f *textField) UncommittedTextLengthInBytes() int {
	if f.focused {
		return len(f.composition)
	}
	return 0
}

// CompositionSelection returns the current composition selection in bytes
// when an IME composition is in progress. The returned values indicate
// relative positions in bytes where the current composition text's start is
// 0.
func (f *textField) CompositionSelection() (startInBytes, endInBytes int, ok bool) {
	if f.focused && f.composition != "" {
		return f.compositionSelStart, f.compositionSelEnd, true
	}
	return 0, 0, false
}

// HasText reports whether the field has any committed text.
func (f *textField) HasText() bool {
	return f.pieceTable.HasText()
}

// WriteTextTo writes the committed text to w.
func (f *textField) WriteTextTo(w io.Writer) (int64, error) {
	return f.pieceTable.WriteRangeTo(w, 0, math.MaxInt)
}

// WriteTextRangeTo writes the committed text in [startInBytes, endInBytes)
// to w.
func (f *textField) WriteTextRangeTo(w io.Writer, startInBytes, endInBytes int) (int64, error) {
	return f.pieceTable.WriteRangeTo(w, startInBytes, endInBytes)
}

// WriteTextForRenderingTo writes the rendering text — the committed text
// with the active IME composition spliced in at the selection — to w.
func (f *textField) WriteTextForRenderingTo(w io.Writer) (int64, error) {
	if f.focused && f.composition != "" {
		return f.pieceTable.WriteRangeToWithInsertion(w, f.composition, f.selectionStartInBytes, f.selectionEndInBytes, 0, math.MaxInt)
	}
	return f.pieceTable.WriteRangeTo(w, 0, math.MaxInt)
}

// WriteTextForRenderingRangeTo writes the rendering text in [startInBytes,
// endInBytes) to w. Coordinates are in rendering space.
func (f *textField) WriteTextForRenderingRangeTo(w io.Writer, startInBytes, endInBytes int) (int64, error) {
	if f.focused && f.composition != "" {
		return f.pieceTable.WriteRangeToWithInsertion(w, f.composition, f.selectionStartInBytes, f.selectionEndInBytes, startInBytes, endInBytes)
	}
	return f.pieceTable.WriteRangeTo(w, startInBytes, endInBytes)
}

// SetSelection sets the selection range, clamped to the current text length.
func (f *textField) SetSelection(startInBytes, endInBytes int) {
	f.cleanUp()
	l := f.pieceTable.Len()
	newStart := min(max(startInBytes, 0), l)
	newEnd := min(max(endInBytes, 0), l)
	if newStart == f.selectionStartInBytes && newEnd == f.selectionEndInBytes {
		return
	}
	f.selectionStartInBytes = newStart
	f.selectionEndInBytes = newEnd
}

// ResetText resets the text and clears the undo history.
func (f *textField) ResetText(text string) {
	f.cleanUp()
	f.pieceTable.Reset(text)
	f.selectionStartInBytes = 0
	f.selectionEndInBytes = 0
	f.bumpGeneration()
}

// ReadTextFrom resets the text by reading bytes from r until EOF and clears
// the undo history.
func (f *textField) ReadTextFrom(r io.Reader) (int64, error) {
	f.cleanUp()
	n, err := f.pieceTable.ReadFrom(r)
	f.selectionStartInBytes = 0
	f.selectionEndInBytes = 0
	f.bumpGeneration()
	return n, err
}

// SetTextAndSelection sets the text and the selection range, recording the
// change in the undo history.
func (f *textField) SetTextAndSelection(text string, selectionStartInBytes, selectionEndInBytes int) {
	f.cleanUp()
	l := f.pieceTable.Len()
	f.pieceTable.Replace(text, 0, l)
	f.selectionStartInBytes = min(max(selectionStartInBytes, 0), len(text))
	f.selectionEndInBytes = min(max(selectionEndInBytes, 0), len(text))
	f.bumpGeneration()
}

// ReplaceText replaces the text at [startInBytes, endInBytes) and updates
// the selection to point past the inserted text. The change is recorded in
// the undo history.
func (f *textField) ReplaceText(text string, startInBytes, endInBytes int) {
	f.cleanUp()
	if text == "" && startInBytes == endInBytes {
		return
	}
	f.pieceTable.Replace(text, startInBytes, endInBytes)
	f.selectionStartInBytes = startInBytes + len(text)
	f.selectionEndInBytes = f.selectionStartInBytes
	f.bumpGeneration()
}

// CanUndo reports whether the field can undo.
func (f *textField) CanUndo() bool {
	return f.pieceTable.CanUndo()
}

// CanRedo reports whether the field can redo.
func (f *textField) CanRedo() bool {
	return f.pieceTable.CanRedo()
}

// Undo undoes the last operation.
func (f *textField) Undo() {
	start, end, ok := f.pieceTable.Undo()
	if !ok {
		return
	}
	f.selectionStartInBytes = start
	f.selectionEndInBytes = end
	f.bumpGeneration()
}

// Redo redoes the last undone operation.
func (f *textField) Redo() {
	start, end, ok := f.pieceTable.Redo()
	if !ok {
		return
	}
	f.selectionStartInBytes = start
	f.selectionEndInBytes = end
	f.bumpGeneration()
}

// cleanUp ends any active IME session before a programmatic mutation so the
// new state is not immediately overwritten by a pending commit.
// [textinput.Composer.Cancel] fires [textinput.Composer.OnComposition] with
// an empty composition, which clears the composition state via
// [textField.onIMEComposition].
func (f *textField) cleanUp() {
	if f.err != nil {
		return
	}
	f.composer.Cancel()
}
