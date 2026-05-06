// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

// Package piecetable provides an editable text buffer with undo/redo support.
package piecetable

import (
	"io"
	"slices"
)

type opType int

const (
	opTypeIME opType = iota
	opTypeOneNewLine
	opTypeDelete
	opTypeOther
)

type lastOp struct {
	valid bool
	typ   opType
}

// PieceTable is an editable text buffer backed by a piece table. The zero
// value is empty and ready for use.
type PieceTable struct {
	table []byte

	history      []historyItem
	historyIndex int
	lastOp       lastOp
}

type historyItem struct {
	items []pieceTableItem

	undoSelectionStart int
	undoSelectionEnd   int
	redoSelectionStart int
	redoSelectionEnd   int
}

type pieceTableItem struct {
	start int
	end   int
}

func (p *PieceTable) items() []pieceTableItem {
	if len(p.history) == 0 {
		return nil
	}
	return p.history[p.historyIndex].items
}

// WriteRangeTo writes the bytes of the current text in [start, end) to w.
// start and end are clamped to [0, Len()]; if start >= end after clamping,
// nothing is written.
func (p *PieceTable) WriteRangeTo(w io.Writer, start, end int) (int64, error) {
	l := p.Len()
	start = max(start, 0)
	end = min(end, l)
	if start >= end {
		return 0, nil
	}

	var n int64
	var offset int
	items := p.items()
	for i := range items {
		item := &items[i]
		itemLen := item.end - item.start
		itemEnd := offset + itemLen

		if itemEnd <= start {
			offset = itemEnd
			continue
		}
		if offset >= end {
			break
		}

		readStart := item.start + max(start-offset, 0)
		readEnd := item.start + min(end-offset, itemLen)

		nn, err := w.Write(p.table[readStart:readEnd])
		n += int64(nn)
		if err != nil {
			return n, err
		}

		offset = itemEnd
	}
	return n, nil
}

// WriteRangeToWithInsertion writes the bytes of the rendering text in
// [rangeStart, rangeEnd) to w. The rendering text is the conceptual stream
//
//	committed[:insertStart] ++ text ++ committed[insertEnd:]
//
// where committed is the current piece-table content. rangeStart and rangeEnd
// are clamped to [0, renderingLength] where
// renderingLength = Len() - (insertEnd - insertStart) + len(text). If the
// clamped start is not less than the clamped end, nothing is written.
func (p *PieceTable) WriteRangeToWithInsertion(w io.Writer, text string, insertStart, insertEnd, rangeStart, rangeEnd int) (int64, error) {
	pl := p.Len()
	insertLen := len(text)
	selLen := insertEnd - insertStart
	totalLen := pl - selLen + insertLen

	rangeStart = max(rangeStart, 0)
	rangeEnd = min(rangeEnd, totalLen)
	if rangeStart >= rangeEnd {
		return 0, nil
	}

	var n int64

	// 1. Committed prefix overlap with [rangeStart, rangeEnd).
	if rangeStart < insertStart {
		prefixEnd := min(rangeEnd, insertStart)
		nn, err := p.WriteRangeTo(w, rangeStart, prefixEnd)
		n += nn
		if err != nil {
			return n, err
		}
	}

	// 2. Composition overlap with [rangeStart, rangeEnd).
	if ts, te := max(0, rangeStart-insertStart), min(insertLen, rangeEnd-insertStart); ts < te {
		nn, err := io.WriteString(w, text[ts:te])
		n += int64(nn)
		if err != nil {
			return n, err
		}
	}

	// 3. Committed suffix overlap with [rangeStart, rangeEnd).
	if rangeEnd > insertStart+insertLen {
		suffixRangeStart := max(rangeStart, insertStart+insertLen)
		ptStart := suffixRangeStart - insertLen + selLen
		ptEnd := rangeEnd - insertLen + selLen
		nn, err := p.WriteRangeTo(w, ptStart, ptEnd)
		n += nn
		if err != nil {
			return n, err
		}
	}

	return n, nil
}

// HasText reports whether the piece table has any text.
func (p *PieceTable) HasText() bool {
	for _, item := range p.items() {
		if item.start < item.end {
			return true
		}
	}
	return false
}

// Len returns the length of the current text in bytes.
func (p *PieceTable) Len() int {
	var n int
	items := p.items()
	for i := range items {
		item := &items[i]
		n += item.end - item.start
	}
	return n
}

// FindLineBounds returns the byte offsets bounding the line that contains
// the selection [selStart, selEnd]. lineStart is the position right after
// the previous line break (or 0 if none), and lineEnd is the position of
// the next line break (or Len() if none). The line break bytes themselves
// are excluded from both ends.
//
// Line breaks that fall within [selStart, selEnd) are ignored, so a
// selection crossing line breaks yields a single combined line view.
func (p *PieceTable) FindLineBounds(selStart, selEnd int) (lineStart, lineEnd int) {
	l := p.Len()
	selStart = min(max(selStart, 0), l)
	selEnd = min(max(selEnd, selStart), l)

	lineEnd = l

	items := p.items()

	// peekByte returns the byte at offset bytes after (pi, bi).
	peekByte := func(pi, bi, offset int) (byte, bool) {
		bi += offset
		for pi < len(items) {
			chunkLen := items[pi].end - items[pi].start
			if bi < chunkLen {
				return p.table[items[pi].start+bi], true
			}
			bi -= chunkLen
			pi++
		}
		return 0, false
	}

	// peekByteBack returns the byte at offset bytes before (pi, bi).
	peekByteBack := func(pi, bi, offset int) (byte, bool) {
		bi -= offset
		for bi < 0 {
			pi--
			if pi < 0 {
				return 0, false
			}
			bi += items[pi].end - items[pi].start
		}
		return p.table[items[pi].start+bi], true
	}

	// findPiece returns (pieceIdx, byteIdxWithinPiece) for absolute byte
	// position pos. Returns (len(items), 0) if pos is past the end.
	findPiece := func(pos int) (int, int) {
		var offset int
		for i, item := range items {
			chunkLen := item.end - item.start
			if pos < offset+chunkLen {
				return i, pos - offset
			}
			offset += chunkLen
		}
		return len(items), 0
	}

	// Scan backward from selStart-1 for the latest line break ending at or
	// before selStart. The first line break encountered going backward is by
	// definition the latest.
	if selStart > 0 {
		pi, bi := findPiece(selStart - 1)
		absPos := selStart - 1
		for pi >= 0 {
			b := p.table[items[pi].start+bi]
			var isLB bool
			switch b {
			case 0x0A, 0x0B, 0x0C, 0x0D: // LF, VT, FF, CR
				isLB = true
			case 0x85: // possible NEL last byte (0xC2 0x85)
				if prev, ok := peekByteBack(pi, bi, 1); ok && prev == 0xC2 {
					isLB = true
				}
			case 0xA8, 0xA9: // possible LS/PS last byte (0xE2 0x80 0xA8/0xA9)
				if b1, ok := peekByteBack(pi, bi, 1); ok && b1 == 0x80 {
					if b2, ok := peekByteBack(pi, bi, 2); ok && b2 == 0xE2 {
						isLB = true
					}
				}
			}
			if isLB {
				lineStart = absPos + 1
				break
			}
			// Step backward.
			if bi > 0 {
				bi--
			} else {
				pi--
				if pi < 0 {
					break
				}
				bi = items[pi].end - items[pi].start - 1
			}
			absPos--
		}
	}

	// Scan forward from selEnd for the earliest line break starting at or
	// after selEnd.
	pi, bi := findPiece(selEnd)
	absPos := selEnd
	for pi < len(items) {
		b := p.table[items[pi].start+bi]
		var isLB bool
		switch b {
		case 0x0A, 0x0B, 0x0C, 0x0D: // LF, VT, FF, CR (alone or as part of CRLF)
			isLB = true
		case 0xC2: // possible NEL first byte
			if next, ok := peekByte(pi, bi, 1); ok && next == 0x85 {
				isLB = true
			}
		case 0xE2: // possible LS/PS first byte
			if next1, ok := peekByte(pi, bi, 1); ok && next1 == 0x80 {
				if next2, ok := peekByte(pi, bi, 2); ok && (next2 == 0xA8 || next2 == 0xA9) {
					isLB = true
				}
			}
		}
		if isLB {
			lineEnd = absPos
			return
		}
		// Step forward.
		bi++
		if bi >= items[pi].end-items[pi].start {
			pi++
			bi = 0
		}
		absPos++
	}
	return
}

// Reset replaces the current text with text and clears the undo history.
func (p *PieceTable) Reset(text string) {
	p.table = p.table[:0]
	p.table = append(p.table, text...)
	p.resetHistory()
}

// ReadFrom resets the piece table by reading bytes from r until EOF.
// Unlike [bytes.Buffer.ReadFrom], ReadFrom does not append: any prior content
// is discarded.
//
// The return value is the number of bytes read. On non-EOF error, the piece
// table is left in an empty state and the error is returned.
func (p *PieceTable) ReadFrom(r io.Reader) (int64, error) {
	p.table = p.table[:0]
	var total int64
	const minRead = 512
	for {
		p.table = slices.Grow(p.table, minRead)
		n, err := r.Read(p.table[len(p.table):cap(p.table)])
		p.table = p.table[:len(p.table)+n]
		total += int64(n)
		if err == io.EOF {
			break
		}
		if err != nil {
			p.table = p.table[:0]
			p.resetHistory()
			return total, err
		}
	}
	p.resetHistory()
	return total, nil
}

func (p *PieceTable) resetHistory() {
	p.history = p.history[:0]
	p.history = append(p.history, historyItem{
		items: []pieceTableItem{
			{
				start: 0,
				end:   len(p.table),
			},
		},
		undoSelectionStart: 0,
		undoSelectionEnd:   len(p.table),
		redoSelectionStart: 0,
		redoSelectionEnd:   len(p.table),
	})
	p.historyIndex = 0
	p.lastOp = lastOp{}
}

// Replace replaces the bytes in [start, end) with text. The change is
// recorded in the undo history.
func (p *PieceTable) Replace(text string, start, end int) {
	p.maybeAppendHistory(text, start, end, false)
	p.doReplace(text, start, end)
}

func (p *PieceTable) doReplace(text string, start, end int) {
	items := p.history[p.historyIndex].items

	// Append the new text to the table.
	newTextStart := len(p.table)
	p.table = append(p.table, text...)
	newTextEnd := len(p.table)

	// Calculate the range of items to replace.
	var startItemIndex, endItemIndex int

	// Find the first intersecting item.
	var offset int
	for startItemIndex < len(items) {
		item := &items[startItemIndex]
		itemLen := item.end - item.start
		if offset+itemLen > start {
			break
		}
		offset += itemLen
		startItemIndex++
	}
	startItemOffset := offset

	// Find the last intersecting item.
	endItemIndex = startItemIndex
	for endItemIndex < len(items) {
		item := items[endItemIndex]
		itemLen := item.end - item.start
		if offset+itemLen >= end {
			break
		}
		offset += itemLen
		endItemIndex++
	}
	endItemOffset := offset

	// Prepare new items.
	var newItems [3]pieceTableItem
	var newItemsCount int

	// 1. Prefix of the first affected item.
	if startItemIndex < len(items) {
		if s := start - startItemOffset; s > 0 {
			item := &items[startItemIndex]
			newItems[newItemsCount] = pieceTableItem{
				start: item.start,
				end:   item.start + s,
			}
			newItemsCount++
		}
	}

	// 2. The new text.
	if newTextEnd > newTextStart {
		newItems[newItemsCount] = pieceTableItem{
			start: newTextStart,
			end:   newTextEnd,
		}
		newItemsCount++
	}

	// 3. Suffix of the last affected item.
	if endItemIndex < len(items) {
		item := &items[endItemIndex]
		if e := end - endItemOffset; e < item.end-item.start {
			newItems[newItemsCount] = pieceTableItem{
				start: item.start + e,
				end:   item.end,
			}
			newItemsCount++
		}
	}

	// Determine the number of items currently occupying the range to be replaced.
	var oldItemsCount int
	if endItemIndex < len(items) {
		oldItemsCount = endItemIndex - startItemIndex + 1
	} else {
		oldItemsCount = len(items) - startItemIndex
	}

	// Adjust the slice.
	newLen := len(items) - oldItemsCount + newItemsCount
	if newLen > cap(items) {
		newSlice := make([]pieceTableItem, newLen)
		copy(newSlice, items[:startItemIndex])
		copy(newSlice[startItemIndex+newItemsCount:], items[startItemIndex+oldItemsCount:])
		items = newSlice
	} else {
		if newLen > len(items) {
			items = items[:newLen]
		}
		copy(items[startItemIndex+newItemsCount:], items[startItemIndex+oldItemsCount:])
		if newLen < len(items) {
			items = items[:newLen]
		}
	}

	copy(items[startItemIndex:], newItems[:newItemsCount])
	p.history[p.historyIndex].items = items
}

// UpdateByIME replaces the bytes in [start, end) with text, recording the
// change in the undo history with IME-merge semantics — consecutive IME
// edits whose ranges touch the previous redo selection collapse into one
// undo entry.
func (p *PieceTable) UpdateByIME(text string, start, end int) {
	p.maybeAppendHistory(text, start, end, true)
	p.doReplace(text, start, end)
}

// CanUndo reports whether the piece table can undo.
func (p *PieceTable) CanUndo() bool {
	return p.historyIndex > 0
}

// CanRedo reports whether the piece table can redo.
func (p *PieceTable) CanRedo() bool {
	return p.historyIndex < len(p.history)-1
}

// Undo reverts the last operation and returns the selection to restore.
// The boolean is false when there is nothing to undo.
func (p *PieceTable) Undo() (int, int, bool) {
	if !p.CanUndo() {
		return 0, 0, false
	}
	item := p.history[p.historyIndex]
	p.historyIndex--
	p.lastOp.valid = false
	return item.undoSelectionStart, item.undoSelectionEnd, true
}

// Redo re-applies the last undone operation and returns the selection to
// restore. The boolean is false when there is nothing to redo.
func (p *PieceTable) Redo() (int, int, bool) {
	if !p.CanRedo() {
		return 0, 0, false
	}
	p.historyIndex++
	p.lastOp.valid = false
	item := p.history[p.historyIndex]
	return item.redoSelectionStart, item.redoSelectionEnd, true
}

func (p *PieceTable) maybeAppendHistory(text string, start, end int, fromIME bool) {
	// If the history is empty, initialize it.
	if p.history == nil {
		p.history = []historyItem{{}}
	}

	var opType opType
	switch {
	case text == "\n":
		opType = opTypeOneNewLine
	case fromIME:
		opType = opTypeIME
	case text == "":
		opType = opTypeDelete
	default:
		opType = opTypeOther
	}

	// Check if the piece table can merge this operation with the last one.
	var merge bool
	if len(p.history) > 0 &&
		p.lastOp.valid &&
		((p.lastOp.typ == opTypeIME && (opType == opTypeIME || opType == opTypeOneNewLine)) ||
			(p.lastOp.typ == opTypeDelete && opType == opTypeDelete)) {
		item := &p.history[p.historyIndex]
		if start == item.redoSelectionStart || start == item.redoSelectionEnd ||
			end == item.redoSelectionStart || end == item.redoSelectionEnd {
			merge = true
		}
	}

	p.lastOp.valid = true
	p.lastOp.typ = opType

	if !merge {
		p.appendHistory(start, end, start, start+len(text))
		return
	}

	item := &p.history[p.historyIndex]
	if opType == opTypeDelete {
		if end == item.redoSelectionStart {
			item.undoSelectionStart = start
		} else if start == item.redoSelectionStart {
			item.undoSelectionEnd += end - start
		}
	}

	item.redoSelectionStart = min(item.redoSelectionStart, start)
	if opType != opTypeDelete {
		item.redoSelectionEnd = max(item.redoSelectionEnd, start+len(text))
	} else {
		item.redoSelectionEnd = item.redoSelectionStart
	}
}

func (p *PieceTable) appendHistory(undoStart, undoEnd, redoStart, redoEnd int) {
	// Truncate the history.
	if p.historyIndex < len(p.history)-1 {
		p.history = p.history[:p.historyIndex+1]
	}

	// Append the current items (cloned) to the history.
	// As doReplace modifies the underlying array, duplicate the items here.
	newItems := append([]pieceTableItem(nil), p.history[p.historyIndex].items...)
	p.history = append(p.history, historyItem{
		items:              newItems,
		undoSelectionStart: undoStart,
		undoSelectionEnd:   undoEnd,
		redoSelectionStart: redoStart,
		redoSelectionEnd:   redoEnd,
	})
	p.historyIndex++
}
