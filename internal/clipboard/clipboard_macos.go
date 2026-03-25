// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

//go:build darwin && !ios

package clipboard

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var (
	class_NSAutoreleasePool objc.Class
	class_NSString          objc.Class
	class_NSArray           objc.Class
	class_NSPasteboard      objc.Class
)

var (
	sel_new                = objc.RegisterName("new")
	sel_release            = objc.RegisterName("release")
	sel_alloc              = objc.RegisterName("alloc")
	sel_initWithUTF8String = objc.RegisterName("initWithUTF8String:")
	sel_UTF8String         = objc.RegisterName("UTF8String")
	// Use lengthOfBytesUsingEncoding: with NSUTF8StringEncoding (4) for correct UTF-8 byte count.
	sel_lengthOfBytesUsingEncoding = objc.RegisterName("lengthOfBytesUsingEncoding:")

	sel_generalPasteboard  = objc.RegisterName("generalPasteboard")
	sel_declareTypes_owner = objc.RegisterName("declareTypes:owner:")
	sel_setString_forType  = objc.RegisterName("setString:forType:")
	sel_stringForType      = objc.RegisterName("stringForType:")
	sel_types              = objc.RegisterName("types")
	sel_containsObject     = objc.RegisterName("containsObject:")
	sel_arrayWithObject    = objc.RegisterName("arrayWithObject:")
)

// nsPasteboardTypeString is the UTI for plain text on the pasteboard.
var nsPasteboardTypeString objc.ID

func init() {
	if _, err := purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
		panic(fmt.Errorf("clipboard: failed to dlopen Foundation: %w", err))
	}
	if _, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
		panic(fmt.Errorf("clipboard: failed to dlopen AppKit: %w", err))
	}

	class_NSAutoreleasePool = objc.GetClass("NSAutoreleasePool")
	class_NSString = objc.GetClass("NSString")
	class_NSArray = objc.GetClass("NSArray")
	class_NSPasteboard = objc.GetClass("NSPasteboard")

	nsPasteboardTypeString = objc.ID(class_NSString).Send(sel_alloc).Send(sel_initWithUTF8String, "public.utf8-plain-text")
}

func readAll() ([]byte, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	pool := objc.ID(class_NSAutoreleasePool).Send(sel_new)
	defer pool.Send(sel_release)

	pasteboard := objc.ID(class_NSPasteboard).Send(sel_generalPasteboard)
	types := pasteboard.Send(sel_types)
	if !objc.Send[bool](types, sel_containsObject, nsPasteboardTypeString) {
		return nil, nil
	}

	strID := pasteboard.Send(sel_stringForType, nsPasteboardTypeString)
	if strID == 0 {
		return nil, nil
	}

	length := strID.Send(sel_lengthOfBytesUsingEncoding, 4)
	cstr := strID.Send(sel_UTF8String)
	return []byte(string(unsafe.Slice((*byte)(unsafe.Pointer(cstr)), length))), nil
}

func writeAll(text []byte) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	pool := objc.ID(class_NSAutoreleasePool).Send(sel_new)
	defer pool.Send(sel_release)

	pasteboard := objc.ID(class_NSPasteboard).Send(sel_generalPasteboard)
	types := objc.ID(class_NSArray).Send(sel_arrayWithObject, nsPasteboardTypeString)
	pasteboard.Send(sel_declareTypes_owner, types, 0)

	clipStr := objc.ID(class_NSString).Send(sel_alloc).Send(sel_initWithUTF8String, string(text))
	pasteboard.Send(sel_setString_forType, clipStr, nsPasteboardTypeString)
	clipStr.Send(sel_release)

	return nil
}
