// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"regexp"
	"syscall/js"
)

var (
	isMacintosh = regexp.MustCompile(`\bMacintosh\b`)
	isIPhone    = regexp.MustCompile(`\biPhone\b`)
	isIPad      = regexp.MustCompile(`\biPad\b`)
)

var darwin bool

func init() {
	ua := js.Global().Get("navigator").Get("userAgent").String()
	if isMacintosh.MatchString(ua) {
		darwin = true
		return
	}
	if isIPhone.MatchString(ua) {
		darwin = true
		return
	}
	if isIPad.MatchString(ua) {
		darwin = true
		return
	}
}

// IsDarwin reports whether the app runs on a Darwin-based platform, where
// text-editing key bindings follow the macOS conventions.
func IsDarwin() bool {
	return darwin
}
