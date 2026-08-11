// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"image"
	"io"
	"math"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/exp/textinput"

	"github.com/guigui-gui/guigui/basicwidget/internal/piecetable"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// maxComposerSurroundingBytes caps the bytes of surrounding text handed to
// [textinput.Composer] on either side of the selection, so a huge logical
// line is not round-tripped across the OS IME bridge at every session start.
const maxComposerSurroundingBytes = 1024

// textStore is the editable backing store behind [Text]. It wraps a
// [piecetable.PieceTable] for the committed buffer and a
// [textinput.Composer] for IME composition.
type textStore struct {
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
	// document coordinates relative to this region. With a non-empty
	// selection the region straddles the selection, which the surrounding
	// text handed to the IME excludes.
	imeTextStart int
	imeTextEnd   int

	err error

	generation int64

	// edits records the positional mutations of the committed text, newest
	// last, one per generation bump. Mutations without a positional record
	// (whole-value resets, undo, redo) leave a generation gap.
	edits []textEdit

	// readRangedState, when non-nil, reads the caller's current ranged text
	// state into its argument. It is registered once via
	// setRangedStateReadFunc and shares the owning widget's lifetime. The
	// store calls it just before a mutation leaves a history position — in
	// particular inside the composer callbacks, whose IME commits are not
	// observable from outside the store.
	readRangedState func(state *piecetable.RangedState)

	// textCommittedFunc, when non-nil, is invoked right after an IME commit
	// mutates the committed text, with the replaced byte range and the
	// length of the replacing text in bytes. Like readRangedState, it is
	// registered once via setTextCommittedFunc and shares the owning
	// widget's lifetime.
	textCommittedFunc func(startInBytes, endInBytes, newLenInBytes int)
}

// textEdit is a positional mutation of the committed text: a replacement of
// [start, end) with newLen bytes.
type textEdit struct {
	// generation is the store generation right after the mutation.
	generation int64

	// start is the inclusive start of the replaced range in bytes.
	start int

	// end is the exclusive end of the replaced range in bytes.
	end int

	// newLen is the length of the replacing text in bytes.
	newLen int
}

func (s *textStore) ensureComposerInited() {
	if s.composerInited {
		return
	}
	s.composer.OnNewSession = s.onNewIMESession
	s.composer.OnComposition = s.onIMEComposition
	s.composer.OnCommit = s.onIMECommit
	s.composerInited = true
}

func (s *textStore) onNewIMESession() *textinput.SessionOptions {
	before, after := s.lineAroundSelection()
	s.imeTextStart = s.selectionStartInBytes - len(before)
	s.imeTextEnd = s.selectionEndInBytes + len(after)
	return &textinput.SessionOptions{
		CaretBounds:     s.bounds,
		TextBeforeCaret: before,
		TextAfterCaret:  after,
	}
}

func (s *textStore) onIMEComposition(c *textinput.Composition) {
	selStart, selEnd := c.SelectionRangeInBytes()
	s.setComposition(c.Text(), selStart, selEnd)
}

// setComposition replaces the active composition with text, selected in
// [selStartInBytes, selEndInBytes) relative to text's start.
func (s *textStore) setComposition(text string, selStartInBytes, selEndInBytes int) {
	if s.composition == text && s.compositionSelStart == selStartInBytes && s.compositionSelEnd == selEndInBytes {
		return
	}
	s.composition = text
	s.compositionSelStart = selStartInBytes
	s.compositionSelEnd = selEndInBytes
	// The composition changes the rendering text only; the committed text
	// is untouched.
	s.bumpGenerationForEdit(0, 0, 0)
}

func (s *textStore) onIMECommit(c *textinput.Commit) {
	text := c.Text()
	beforeRepl, afterRepl := c.IsSurroundingTextReplaced()
	if !beforeRepl && !afterRepl {
		// Typical case: insert Text at the current selection.
		s.commitText(text)
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
	_, _ = s.pieceTable.WriteRangeTo(&sb, s.imeTextStart, s.imeTextEnd)
	oldContent := sb.String()

	prefixLen := commonPrefixLen(oldContent, newContent)
	suffixLen := commonSuffixLen(oldContent[prefixLen:], newContent[prefixLen:])
	insStart := s.imeTextStart + prefixLen
	insEnd := s.imeTextEnd - suffixLen
	insText := newContent[prefixLen : len(newContent)-suffixLen]

	s.recordCurrentRangedState()
	s.pieceTable.UpdateByIME(insText, insStart, insEnd)
	// Caret lands at the end of the IME's committed text within the new
	// joined content laid out at [imeTextStart, imeTextEnd).
	s.selectionStartInBytes = s.imeTextStart + len(newBefore) + len(text)
	s.selectionEndInBytes = s.selectionStartInBytes
	s.composition = ""
	s.compositionSelStart = 0
	s.compositionSelEnd = 0
	s.bumpGenerationForEdit(insStart, insEnd, len(insText))
	if s.textCommittedFunc != nil {
		s.textCommittedFunc(insStart, insEnd, len(insText))
	}
}

// commitText inserts text at the current selection as an IME commit, placing
// the caret after the inserted text.
func (s *textStore) commitText(text string) {
	start, end := s.selectionStartInBytes, s.selectionEndInBytes
	if start > end {
		start, end = end, start
	}
	s.recordCurrentRangedState()
	s.pieceTable.UpdateByIME(text, start, end)
	s.selectionStartInBytes = start + len(text)
	s.selectionEndInBytes = s.selectionStartInBytes
	s.composition = ""
	s.compositionSelStart = 0
	s.compositionSelEnd = 0
	s.bumpGenerationForEdit(start, end, len(text))
	if s.textCommittedFunc != nil {
		s.textCommittedFunc(start, end, len(text))
	}
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
// side of the selection, capped to [maxComposerSurroundingBytes] per side.
// Both halves combined form the surrounding text the IME uses for prediction
// and reconversion.
func (s *textStore) lineAroundSelection() (before, after string) {
	selStart, selEnd := s.selectionStartInBytes, s.selectionEndInBytes
	if selStart > selEnd {
		selStart, selEnd = selEnd, selStart
	}
	lineStart, lineEnd := s.pieceTable.FindLineBounds(selStart, selEnd)

	beforeStart := max(lineStart, selStart-maxComposerSurroundingBytes)
	afterEnd := min(lineEnd, selEnd+maxComposerSurroundingBytes)

	var sb strings.Builder
	_, _ = s.pieceTable.WriteRangeTo(&sb, beforeStart, selStart)
	before = sb.String()
	sb.Reset()
	_, _ = s.pieceTable.WriteRangeTo(&sb, selEnd, afterEnd)
	after = sb.String()

	// When the cap takes effect, the cut edge can land inside a multi-byte
	// UTF-8 sequence. Drop any partial bytes so the IME never sees half a
	// rune.
	if beforeStart > lineStart {
		before = textutil.TrimPartialUTF8Prefix(before)
	}
	if afterEnd < lineEnd {
		after = textutil.TrimPartialUTF8Suffix(after)
	}
	return before, after
}

// bumpGeneration advances the generation without recording a positional
// edit. Mutations that cannot be described as a single replacement (whole-
// value resets, undo, redo) call this directly; the resulting gap in the
// edit record makes [textStore.appendEditsSince] report non-coverage.
func (s *textStore) bumpGeneration() {
	s.generation++
}

// bumpGenerationForEdit advances the generation and records the mutation as
// a replacement of [start, end) with newLen bytes. A mutation that leaves
// the committed text unchanged records a zero-length replacement.
func (s *textStore) bumpGenerationForEdit(start, end, newLen int) {
	s.bumpGeneration()
	s.edits = append(s.edits, textEdit{
		generation: s.generation,
		start:      start,
		end:        end,
		newLen:     newLen,
	})
	// maxRecordedTextEdits caps the edit record; overflowing entries are
	// dropped, leaving a generation gap.
	const maxRecordedTextEdits = 256
	if len(s.edits) > maxRecordedTextEdits {
		s.edits = slices.Delete(s.edits, 0, len(s.edits)-maxRecordedTextEdits)
	}
}

// appendEditsSince appends the positional edits recorded after the
// generation sinceGen to dst, oldest first, and returns the extended slice.
// ok is false when the mutations since sinceGen are not fully covered by
// positional records, so the caller cannot replay them.
func (s *textStore) appendEditsSince(dst []textEdit, sinceGen int64) (edits []textEdit, ok bool) {
	// edits is ascending in generation; find the first entry past sinceGen.
	i, _ := slices.BinarySearchFunc(s.edits, sinceGen, func(e textEdit, gen int64) int {
		if e.generation > gen {
			return 1
		}
		return -1
	})
	tail := s.edits[i:]
	if int64(len(tail)) != s.generation-sinceGen {
		return dst, false
	}
	return append(dst, tail...), true
}

// Generation returns a counter that advances when the store's renderable
// content changes. Selection-only changes do not advance Generation.
func (s *textStore) Generation() int64 {
	return s.generation
}

// Selection returns the current selection range in bytes.
func (s *textStore) Selection() (startInBytes, endInBytes int) {
	return s.selectionStartInBytes, s.selectionEndInBytes
}

// IsFocused reports whether the store is focused.
func (s *textStore) IsFocused() bool {
	return s.focused
}

// Focus marks the store as focused. The Composer is driven only while the
// field is focused.
func (s *textStore) Focus() {
	if s.focused {
		return
	}
	s.focused = true
}

// Blur removes the focus from the store, ending any active IME session.
// [textinput.Composer.Finish] commits any in-progress composition.
func (s *textStore) Blur() {
	if !s.focused {
		return
	}
	s.focused = false
	s.composer.Confirm()
}

// SetBounds sets the bounds used for IME window positioning. The bounds are
// captured at the start of the next IME session.
func (s *textStore) SetBounds(bounds image.Rectangle) {
	s.bounds = bounds
}

// Update drives the IME composer for one tick. Call it once when the store
// gains focus, so the session is established before the next key event, and
// once per tick that has key input. handled reports whether the IME consumed
// input; the caller should suppress its own key handlers in that case.
func (s *textStore) Update() (handled bool, err error) {
	if s.err != nil {
		return false, s.err
	}
	if !s.focused {
		return false, nil
	}
	s.ensureComposerInited()
	handled, err = s.composer.Update()
	if err != nil {
		s.err = err
		return false, s.err
	}
	return handled, nil
}

// TextLengthInBytes returns the length of the current text in bytes.
func (s *textStore) TextLengthInBytes() int {
	return s.pieceTable.Len()
}

// UncommittedTextLengthInBytes returns the active composition length in
// bytes when the store is focused. Returns 0 otherwise.
func (s *textStore) UncommittedTextLengthInBytes() int {
	if s.focused {
		return len(s.composition)
	}
	return 0
}

// CompositionSelection returns the current composition selection as byte
// offsets relative to the composition text's start, with ok reporting whether
// an IME composition is in progress.
func (s *textStore) CompositionSelection() (startInBytes, endInBytes int, ok bool) {
	if s.focused && s.composition != "" {
		return s.compositionSelStart, s.compositionSelEnd, true
	}
	return 0, 0, false
}

// HasText reports whether the store has any committed text.
func (s *textStore) HasText() bool {
	return s.pieceTable.HasText()
}

// WriteTextTo writes the committed text to w.
func (s *textStore) WriteTextTo(w io.Writer) (int64, error) {
	return s.pieceTable.WriteRangeTo(w, 0, math.MaxInt)
}

// WriteTextRangeTo writes the committed text in [startInBytes, endInBytes)
// to w.
func (s *textStore) WriteTextRangeTo(w io.Writer, startInBytes, endInBytes int) (int64, error) {
	return s.pieceTable.WriteRangeTo(w, startInBytes, endInBytes)
}

// WriteTextForRenderingTo writes the rendering text — the committed text
// with the active IME composition spliced in at the selection — to w.
func (s *textStore) WriteTextForRenderingTo(w io.Writer) (int64, error) {
	if s.focused && s.composition != "" {
		return s.pieceTable.WriteRangeToWithInsertion(w, s.composition, s.selectionStartInBytes, s.selectionEndInBytes, 0, math.MaxInt)
	}
	return s.pieceTable.WriteRangeTo(w, 0, math.MaxInt)
}

// WriteTextForRenderingRangeTo writes the rendering text in [startInBytes,
// endInBytes) to w. Coordinates are in rendering space.
func (s *textStore) WriteTextForRenderingRangeTo(w io.Writer, startInBytes, endInBytes int) (int64, error) {
	if s.focused && s.composition != "" {
		return s.pieceTable.WriteRangeToWithInsertion(w, s.composition, s.selectionStartInBytes, s.selectionEndInBytes, startInBytes, endInBytes)
	}
	return s.pieceTable.WriteRangeTo(w, startInBytes, endInBytes)
}

// SetSelection sets the selection range, clamped to the current text length.
func (s *textStore) SetSelection(startInBytes, endInBytes int) {
	s.cleanUp()
	l := s.pieceTable.Len()
	newStart := min(max(startInBytes, 0), l)
	newEnd := min(max(endInBytes, 0), l)
	if newStart == s.selectionStartInBytes && newEnd == s.selectionEndInBytes {
		return
	}
	s.selectionStartInBytes = newStart
	s.selectionEndInBytes = newEnd
}

// ResetText resets the text and clears the undo history.
func (s *textStore) ResetText(text string) {
	s.cleanUp()
	s.pieceTable.Reset(text)
	s.selectionStartInBytes = 0
	s.selectionEndInBytes = 0
	s.bumpGeneration()
}

// ReadTextFrom resets the text by reading bytes from r until EOF and clears
// the undo history.
func (s *textStore) ReadTextFrom(r io.Reader) (int64, error) {
	s.cleanUp()
	n, err := s.pieceTable.ReadFrom(r)
	s.selectionStartInBytes = 0
	s.selectionEndInBytes = 0
	s.bumpGeneration()
	return n, err
}

// SetTextAndSelection sets the text and the selection range, recording the
// change in the undo history.
func (s *textStore) SetTextAndSelection(text string, selectionStartInBytes, selectionEndInBytes int) {
	s.cleanUp()
	l := s.pieceTable.Len()
	s.recordCurrentRangedState()
	s.pieceTable.Replace(text, 0, l)
	s.selectionStartInBytes = min(max(selectionStartInBytes, 0), len(text))
	s.selectionEndInBytes = min(max(selectionEndInBytes, 0), len(text))
	s.bumpGeneration()
}

// ReplaceText replaces the text at [startInBytes, endInBytes) and updates
// the selection to point past the inserted text. The change is recorded in
// the undo history.
func (s *textStore) ReplaceText(text string, startInBytes, endInBytes int) {
	s.cleanUp()
	if text == "" && startInBytes == endInBytes {
		return
	}
	s.recordCurrentRangedState()
	s.pieceTable.Replace(text, startInBytes, endInBytes)
	s.selectionStartInBytes = startInBytes + len(text)
	s.selectionEndInBytes = s.selectionStartInBytes
	s.bumpGenerationForEdit(startInBytes, endInBytes, len(text))
}

// setRangedStateReadFunc registers f to read the caller's current ranged
// text state for the undo history.
func (s *textStore) setRangedStateReadFunc(f func(state *piecetable.RangedState)) {
	s.readRangedState = f
}

// setTextCommittedFunc registers f to be invoked right after an IME commit
// mutates the committed text.
func (s *textStore) setTextCommittedFunc(f func(startInBytes, endInBytes, newLenInBytes int)) {
	s.textCommittedFunc = f
}

// recordRangedStateChange records the current ranged text state and appends
// a history entry for a change of that state alone: the text is unchanged,
// and undo and redo restore the selection to [startInBytes, endInBytes],
// clamped to the current text length.
func (s *textStore) recordRangedStateChange(startInBytes, endInBytes int) {
	s.recordCurrentRangedState()
	l := s.pieceTable.Len()
	startInBytes = min(max(startInBytes, 0), l)
	endInBytes = min(max(endInBytes, startInBytes), l)
	s.pieceTable.AppendHistoryEntry(startInBytes, endInBytes)
}

// recordCurrentRangedState records a snapshot of the caller's current ranged
// text state as the state of the current history position.
func (s *textStore) recordCurrentRangedState() {
	if s.readRangedState == nil {
		return
	}
	var state piecetable.RangedState
	s.readRangedState(&state)
	s.pieceTable.SetCurrentRangedState(&state)
}

// CanUndo reports whether the store can undo.
func (s *textStore) CanUndo() bool {
	return s.pieceTable.CanUndo()
}

// CanRedo reports whether the store can redo.
func (s *textStore) CanRedo() bool {
	return s.pieceTable.CanRedo()
}

// Undo undoes the last operation and returns the ranged state recorded for
// the position being returned to. The current state is recorded first, so a
// later Redo restores it. ok is false when there is nothing to undo.
func (s *textStore) Undo() (state *piecetable.RangedState, ok bool) {
	if !s.pieceTable.CanUndo() {
		return nil, false
	}
	s.recordCurrentRangedState()
	start, end, state, ok := s.pieceTable.Undo()
	if !ok {
		return nil, false
	}
	s.selectionStartInBytes = start
	s.selectionEndInBytes = end
	s.bumpGeneration()
	return state, true
}

// Redo redoes the last undone operation and returns the ranged state
// recorded for the position being returned to. ok is false when there is
// nothing to redo.
func (s *textStore) Redo() (state *piecetable.RangedState, ok bool) {
	start, end, state, ok := s.pieceTable.Redo()
	if !ok {
		return nil, false
	}
	s.selectionStartInBytes = start
	s.selectionEndInBytes = end
	s.bumpGeneration()
	return state, true
}

// cleanUp ends any active IME session before a programmatic mutation so a
// later commit cannot overwrite the new state. [textinput.Composer.Confirm]
// commits any in-progress composition.
func (s *textStore) cleanUp() {
	if s.err != nil {
		return
	}
	s.composer.Confirm()
}
