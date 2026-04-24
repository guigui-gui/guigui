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

func readAll() ([]byte, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if ok, err := _IsClipboardFormatAvailable(_CF_UNICODETEXT); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	if err := openClipboard(); err != nil {
		return nil, err
	}
	defer func() {
		_ = _CloseClipboard()
	}()

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

func writeAll(text []byte) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// CF_UNICODETEXT is null-terminated, so an embedded NUL would silently
	// truncate the clipboard contents. Reject such input to avoid confusion.
	if bytes.IndexByte(text, 0) != -1 {
		return errors.New("clipboard: text contains a null byte")
	}

	// Decode UTF-8 from text and append UTF-16 units directly, avoiding the
	// string/rune/[]uint16 allocations that windows.UTF16FromString would make.
	u16 := make([]uint16, 0, len(text)+1)
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRune(text[i:])
		u16 = utf16.AppendRune(u16, r)
		i += size
	}
	u16 = append(u16, 0)
	// The buffer ownership transfers to the clipboard once SetClipboardData succeeds.
	byteLen := uintptr(len(u16)) * 2
	h, err := _GlobalAlloc(_GMEM_MOVEABLE, byteLen)
	if err != nil {
		return err
	}
	defer func() {
		if h != 0 {
			_ = _GlobalFree(h)
		}
	}()

	p, err := _GlobalLock(h)
	if err != nil {
		return err
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(u16))
	copy(dst, u16)
	if err := _GlobalUnlock(h); err != nil {
		return err
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

	if err := _SetClipboardData(_CF_UNICODETEXT, h); err != nil {
		return err
	}
	// Ownership of h transferred to the system; don't free it.
	h = 0

	return nil
}
