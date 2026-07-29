// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package clipboard

import (
	"bytes"
	"errors"
	"runtime"
	"time"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

func openClipboard() error {
	// Another process may temporarily hold the clipboard; retry briefly.
	deadline := time.Now().Add(time.Second)
	var lastErr error
	for {
		err := _OpenClipboard()
		if err == nil {
			return nil
		}
		lastErr = err
		if time.Now().After(deadline) {
			return lastErr
		}
		time.Sleep(time.Millisecond)
	}
}

// Registered clipboard format IDs, resolved lazily. Accessed only from the
// clipboard goroutine.
var (
	htmlClipboardFormat  uint32
	pngClipboardFormat   uint32
	clipboardFormatsErr  error
	clipboardFormatsDone bool
)

func ensureClipboardFormats() error {
	if clipboardFormatsDone {
		return clipboardFormatsErr
	}
	clipboardFormatsDone = true
	// "HTML Format" carries a CF_HTML payload; "PNG" carries a raw PNG stream.
	// Both names are registered by browsers and office applications.
	htmlClipboardFormat, clipboardFormatsErr = _RegisterClipboardFormatW("HTML Format")
	if clipboardFormatsErr != nil {
		return clipboardFormatsErr
	}
	pngClipboardFormat, clipboardFormatsErr = _RegisterClipboardFormatW("PNG")
	return clipboardFormatsErr
}

// Change-detection state. Accessed only from the clipboard goroutine.
var (
	lastSequenceNumber    uint32
	hasLastSequenceNumber bool
	lastContents          Contents
)

func readContents() (Contents, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ensureClipboardFormats(); err != nil {
		return Contents{}, err
	}

	// The sequence number changes whenever the clipboard contents change;
	// skip re-reading unchanged contents. It is captured before reading so a
	// concurrent change is observed by the next poll at the latest.
	sequenceNumber := _GetClipboardSequenceNumber()
	if hasLastSequenceNumber && sequenceNumber == lastSequenceNumber {
		return lastContents, nil
	}

	if err := openClipboard(); err != nil {
		return Contents{}, err
	}
	defer func() {
		_ = _CloseClipboard()
	}()

	var contents Contents

	text, err := readClipboardText()
	if err != nil {
		return Contents{}, err
	}
	contents.Text = text

	if payload, err := readClipboardBytes(htmlClipboardFormat); err != nil {
		return Contents{}, err
	} else if payload != nil {
		// A malformed CF_HTML payload only drops the HTML representation;
		// the other formats are still served.
		if fragment, err := decodeCFHTML(bytes.TrimRight(payload, "\x00")); err == nil {
			contents.HTML = fragment
		}
	}

	png, err := readClipboardBytes(pngClipboardFormat)
	if err != nil {
		return Contents{}, err
	}
	contents.PNG = png

	lastSequenceNumber = sequenceNumber
	hasLastSequenceNumber = true
	lastContents = contents
	return contents, nil
}

// readClipboardText reads CF_UNICODETEXT and decodes it to UTF-8. It returns
// nil without an error when the format is not on the clipboard. The clipboard
// must be open.
func readClipboardText() ([]byte, error) {
	if ok, err := _IsClipboardFormatAvailable(_CF_UNICODETEXT); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	h, err := _GetClipboardData(_CF_UNICODETEXT)
	if err != nil {
		return nil, err
	}
	if h == 0 {
		return nil, nil
	}

	p, err := _GlobalLock(h)
	if err != nil {
		return nil, err
	}
	// Walk the null-terminated UTF-16 buffer to determine its length, then
	// decode straight into a []byte to avoid the intermediate string that
	// windows.UTF16PtrToString would allocate.
	var n int
	for ptr := unsafe.Pointer(p); *(*uint16)(ptr) != 0; n++ {
		ptr = unsafe.Add(ptr, 2)
	}
	runes := utf16.Decode(unsafe.Slice((*uint16)(unsafe.Pointer(p)), n))
	b := make([]byte, 0, n)
	for _, r := range runes {
		b = utf8.AppendRune(b, r)
	}
	if err := _GlobalUnlock(h); err != nil {
		return nil, err
	}
	return b, nil
}

// readClipboardBytes reads the raw bytes of the given clipboard format. It
// returns nil without an error when the format is not on the clipboard. The
// clipboard must be open.
func readClipboardBytes(format uint32) ([]byte, error) {
	if ok, err := _IsClipboardFormatAvailable(format); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	h, err := _GetClipboardData(format)
	if err != nil {
		return nil, err
	}
	if h == 0 {
		return nil, nil
	}

	p, err := _GlobalLock(h)
	if err != nil {
		return nil, err
	}
	size, err := _GlobalSize(h)
	if err != nil {
		_ = _GlobalUnlock(h)
		return nil, err
	}
	b := make([]byte, size)
	copy(b, unsafe.Slice((*byte)(unsafe.Pointer(p)), size))
	if err := _GlobalUnlock(h); err != nil {
		return nil, err
	}
	return b, nil
}

// encodeClipboardText encodes UTF-8 text as null-terminated UTF-16LE bytes
// for CF_UNICODETEXT.
func encodeClipboardText(text []byte) ([]byte, error) {
	// CF_UNICODETEXT is null-terminated, so an embedded NUL would silently
	// truncate the clipboard contents. Reject such input to avoid confusion.
	if bytes.IndexByte(text, 0) != -1 {
		return nil, errors.New("clipboard: text contains a null byte")
	}

	u16 := make([]uint16, 0, len(text)+1)
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRune(text[i:])
		u16 = utf16.AppendRune(u16, r)
		i += size
	}
	u16 = append(u16, 0)

	b := make([]byte, 0, len(u16)*2)
	for _, u := range u16 {
		b = append(b, byte(u), byte(u>>8))
	}
	return b, nil
}

// allocGlobalBytes copies data into a new GMEM_MOVEABLE global allocation
// suitable for SetClipboardData.
func allocGlobalBytes(data []byte) (uintptr, error) {
	h, err := _GlobalAlloc(_GMEM_MOVEABLE, uintptr(len(data)))
	if err != nil {
		return 0, err
	}
	p, err := _GlobalLock(h)
	if err != nil {
		_ = _GlobalFree(h)
		return 0, err
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(p)), len(data)), data)
	if err := _GlobalUnlock(h); err != nil {
		_ = _GlobalFree(h)
		return 0, err
	}
	return h, nil
}

func writeContents(contents Contents) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ensureClipboardFormats(); err != nil {
		return err
	}

	// Prepare all allocations before opening the clipboard to keep the
	// open-empty-set sequence short.
	type clipboardEntry struct {
		format uint32
		h      uintptr
	}
	var entries []clipboardEntry
	defer func() {
		for _, entry := range entries {
			if entry.h != 0 {
				_ = _GlobalFree(entry.h)
			}
		}
	}()
	appendEntry := func(format uint32, data []byte) error {
		h, err := allocGlobalBytes(data)
		if err != nil {
			return err
		}
		entries = append(entries, clipboardEntry{
			format: format,
			h:      h,
		})
		return nil
	}
	if contents.Text != nil {
		text, err := encodeClipboardText(contents.Text)
		if err != nil {
			return err
		}
		if err := appendEntry(_CF_UNICODETEXT, text); err != nil {
			return err
		}
	}
	if contents.HTML != nil {
		// The trailing NUL is conventional for CF_HTML consumers and is not
		// counted by the header offsets.
		if err := appendEntry(htmlClipboardFormat, append(encodeCFHTML(contents.HTML), 0)); err != nil {
			return err
		}
	}
	if contents.PNG != nil {
		if err := appendEntry(pngClipboardFormat, contents.PNG); err != nil {
			return err
		}
	}

	if err := openClipboard(); err != nil {
		return err
	}
	defer func() {
		_ = _CloseClipboard()
	}()

	if err := _EmptyClipboard(); err != nil {
		return err
	}

	for i := range entries {
		if err := _SetClipboardData(entries[i].format, entries[i].h); err != nil {
			return err
		}
		// Ownership of the allocation transferred to the system.
		entries[i].h = 0
	}
	return nil
}
