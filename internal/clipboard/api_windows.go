// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package clipboard

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

const (
	_CF_UNICODETEXT = 13
	_GMEM_MOVEABLE  = 0x0002
)

var (
	user32                         = windows.NewLazySystemDLL("user32.dll")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procEmptyClipboard             = user32.NewProc("EmptyClipboard")
	procGetClipboardData           = user32.NewProc("GetClipboardData")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procSetClipboardData           = user32.NewProc("SetClipboardData")

	kernel32         = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	procGlobalFree   = kernel32.NewProc("GlobalFree")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")
)

func _CloseClipboard() error {
	r, _, err := procCloseClipboard.Call()
	if r == 0 {
		if err != nil && !errors.Is(err, windows.ERROR_SUCCESS) {
			return fmt.Errorf("clipboard: CloseClipboard failed: %w", err)
		}
		return fmt.Errorf("clipboard: CloseClipboard failed: returned 0")
	}
	return nil
}

func _EmptyClipboard() error {
	r, _, err := procEmptyClipboard.Call()
	if r == 0 {
		if err != nil && !errors.Is(err, windows.ERROR_SUCCESS) {
			return fmt.Errorf("clipboard: EmptyClipboard failed: %w", err)
		}
		return fmt.Errorf("clipboard: EmptyClipboard failed: returned 0")
	}
	return nil
}

func _GetClipboardData(format uint32) (uintptr, error) {
	h, _, err := procGetClipboardData.Call(uintptr(format))
	if err != nil && !errors.Is(err, windows.ERROR_SUCCESS) {
		return 0, fmt.Errorf("clipboard: GetClipboardData failed: %w", err)
	}
	return h, nil
}

func _IsClipboardFormatAvailable(format uint32) (bool, error) {
	r, _, err := procIsClipboardFormatAvailable.Call(uintptr(format))
	if r == 0 && err != nil && !errors.Is(err, windows.ERROR_SUCCESS) {
		return false, fmt.Errorf("clipboard: IsClipboardFormatAvailable failed: %w", err)
	}
	return r != 0, nil
}

func _OpenClipboard() error {
	r, _, err := procOpenClipboard.Call(0)
	if r == 0 {
		if err != nil && !errors.Is(err, windows.ERROR_SUCCESS) {
			return fmt.Errorf("clipboard: OpenClipboard failed: %w", err)
		}
		return fmt.Errorf("clipboard: OpenClipboard failed: returned 0")
	}
	return nil
}

func _SetClipboardData(format uint32, h uintptr) error {
	r, _, err := procSetClipboardData.Call(uintptr(format), h)
	if r == 0 {
		if err != nil && !errors.Is(err, windows.ERROR_SUCCESS) {
			return fmt.Errorf("clipboard: SetClipboardData failed: %w", err)
		}
		return fmt.Errorf("clipboard: SetClipboardData failed: returned 0")
	}
	return nil
}

func _GlobalAlloc(flags uint32, size uintptr) (uintptr, error) {
	h, _, err := procGlobalAlloc.Call(uintptr(flags), size)
	if h == 0 {
		if err != nil && !errors.Is(err, windows.ERROR_SUCCESS) {
			return 0, fmt.Errorf("clipboard: GlobalAlloc failed: %w", err)
		}
		return 0, fmt.Errorf("clipboard: GlobalAlloc failed: returned 0")
	}
	return h, nil
}

func _GlobalFree(h uintptr) error {
	r, _, err := procGlobalFree.Call(h)
	if r != 0 {
		if err != nil && !errors.Is(err, windows.ERROR_SUCCESS) {
			return fmt.Errorf("clipboard: GlobalFree failed: %w", err)
		}
		return fmt.Errorf("clipboard: GlobalFree failed: returned non-zero")
	}
	return nil
}

func _GlobalLock(h uintptr) (uintptr, error) {
	p, _, err := procGlobalLock.Call(h)
	if p == 0 {
		if err != nil && !errors.Is(err, windows.ERROR_SUCCESS) {
			return 0, fmt.Errorf("clipboard: GlobalLock failed: %w", err)
		}
		return 0, fmt.Errorf("clipboard: GlobalLock failed: returned 0")
	}
	return p, nil
}

func _GlobalUnlock(h uintptr) error {
	r, _, err := procGlobalUnlock.Call(h)
	// GlobalUnlock returns 0 both on success (when the lock count hits 0)
	// and on failure, so check GetLastError.
	if r == 0 && err != nil && !errors.Is(err, windows.ERROR_SUCCESS) {
		return fmt.Errorf("clipboard: GlobalUnlock failed: %w", err)
	}
	return nil
}
