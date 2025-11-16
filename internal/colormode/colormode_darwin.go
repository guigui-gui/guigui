// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package colormode

import (
	"bytes"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

var (
	idNSApplication = objc.ID(objc.GetClass("NSApplication"))

	selEffectiveAppearance = objc.RegisterName("effectiveAppearance")
	selLength              = objc.RegisterName("length")
	selName                = objc.RegisterName("name")
	selSharedApplication   = objc.RegisterName("sharedApplication")
	selUTF8String          = objc.RegisterName("UTF8String")
)

var (
	bytesDark = []byte("Dark")
)

func systemColorMode() ColorMode {
	// "effectiveAppearance" works from macOS 10.14. As Go 1.23 supports macOS 11, it's OK to use it.
	//
	// See also:
	// * https://developer.apple.com/documentation/appkit/nsapplication/effectiveappearance?language=objc
	// * https://go.dev/wiki/MinimumRequirements
	objcName := idNSApplication.Send(selSharedApplication).Send(selEffectiveAppearance).Send(selName)
	name := unsafe.Slice((*byte)(unsafe.Pointer(objcName.Send(selUTF8String))), objcName.Send(selLength))
	// https://developer.apple.com/documentation/appkit/nsappearance/name-swift.struct?language=objc
	if bytes.Contains(name, bytesDark) {
		return Dark
	}
	return Light
}
