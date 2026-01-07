// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

//go:build !darwin

package basicwidget

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

func isDarwin() bool {
	return darwin
}
