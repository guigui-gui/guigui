// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package keybindingmode

import (
	"regexp"
	"syscall/js"
)

var (
	isMacintosh = regexp.MustCompile(`\bMacintosh\b`)
	isIPhone    = regexp.MustCompile(`\biPhone\b`)
	isIPad      = regexp.MustCompile(`\biPad\b`)
)

var mode = ControlDefault

func init() {
	ua := js.Global().Get("navigator").Get("userAgent").String()
	if isMacintosh.MatchString(ua) || isIPhone.MatchString(ua) || isIPad.MatchString(ua) {
		mode = Command
	}
}

func systemMode() Mode {
	return mode
}
