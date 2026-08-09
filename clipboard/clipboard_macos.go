// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

//go:build darwin && !ios

package clipboard

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var (
	class_NSAutoreleasePool objc.Class
	class_NSString          objc.Class
	class_NSData            objc.Class
	class_NSPasteboard      objc.Class
)

var (
	sel_new                = objc.RegisterName("new")
	sel_release            = objc.RegisterName("release")
	sel_alloc              = objc.RegisterName("alloc")
	sel_initWithUTF8String = objc.RegisterName("initWithUTF8String:")

	sel_generalPasteboard = objc.RegisterName("generalPasteboard")
	sel_changeCount       = objc.RegisterName("changeCount")
	sel_clearContents     = objc.RegisterName("clearContents")
	sel_dataForType       = objc.RegisterName("dataForType:")
	sel_setData_forType   = objc.RegisterName("setData:forType:")
	sel_types             = objc.RegisterName("types")
	sel_containsObject    = objc.RegisterName("containsObject:")

	sel_dataWithBytes_length = objc.RegisterName("dataWithBytes:length:")
	sel_length               = objc.RegisterName("length")
	sel_bytes                = objc.RegisterName("bytes")
)

// NSPasteboardType UTI strings.
var (
	nsPasteboardTypeText objc.ID // public.utf8-plain-text
	nsPasteboardTypeHTML objc.ID // public.html
	nsPasteboardTypePNG  objc.ID // public.png
)

func init() {
	if _, err := purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
		panic(fmt.Errorf("clipboard: failed to dlopen Foundation: %w", err))
	}
	if _, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
		panic(fmt.Errorf("clipboard: failed to dlopen AppKit: %w", err))
	}

	class_NSAutoreleasePool = objc.GetClass("NSAutoreleasePool")
	class_NSString = objc.GetClass("NSString")
	class_NSData = objc.GetClass("NSData")
	class_NSPasteboard = objc.GetClass("NSPasteboard")

	nsPasteboardTypeText = objc.ID(class_NSString).Send(sel_alloc).Send(sel_initWithUTF8String, "public.utf8-plain-text")
	nsPasteboardTypeHTML = objc.ID(class_NSString).Send(sel_alloc).Send(sel_initWithUTF8String, "public.html")
	nsPasteboardTypePNG = objc.ID(class_NSString).Send(sel_alloc).Send(sel_initWithUTF8String, "public.png")
}

// Change-detection state. Accessed only from the clipboard goroutine.
var (
	lastChangeCount = -1
	lastContents    Contents
)

// readPasteboardData returns the data for the given pasteboard type, or nil
// when the type is not on the pasteboard. types is the pasteboard's current
// type array.
func readPasteboardData(pasteboard, types, typ objc.ID) []byte {
	if !objc.Send[bool](types, sel_containsObject, typ) {
		return nil
	}
	nsData := pasteboard.Send(sel_dataForType, typ)
	if nsData == 0 {
		return nil
	}
	length := objc.Send[int](nsData, sel_length)
	data := make([]byte, length)
	if length > 0 {
		p := nsData.Send(sel_bytes)
		copy(data, unsafe.Slice((*byte)(unsafe.Pointer(p)), length))
	}
	return data
}

func readContents() (Contents, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	pool := objc.ID(class_NSAutoreleasePool).Send(sel_new)
	defer pool.Send(sel_release)

	pasteboard := objc.ID(class_NSPasteboard).Send(sel_generalPasteboard)

	// changeCount increments whenever the pasteboard contents are replaced;
	// skip re-reading unchanged contents.
	changeCount := objc.Send[int](pasteboard, sel_changeCount)
	if changeCount == lastChangeCount {
		return lastContents, nil
	}

	types := pasteboard.Send(sel_types)
	contents := Contents{
		Text:  readPasteboardData(pasteboard, types, nsPasteboardTypeText),
		HTML:  readPasteboardData(pasteboard, types, nsPasteboardTypeHTML),
		Image: readPasteboardData(pasteboard, types, nsPasteboardTypePNG),
	}

	lastChangeCount = changeCount
	lastContents = contents
	return contents, nil
}

// writePasteboardData sets the data for the given pasteboard type. The
// pasteboard contents must have been cleared by the same owner beforehand.
func writePasteboardData(pasteboard, typ objc.ID, data []byte) error {
	if data == nil {
		return nil
	}
	var p unsafe.Pointer
	if len(data) > 0 {
		p = unsafe.Pointer(unsafe.SliceData(data))
	}
	nsData := objc.ID(class_NSData).Send(sel_dataWithBytes_length, p, len(data))
	if !objc.Send[bool](pasteboard, sel_setData_forType, nsData, typ) {
		return errors.New("clipboard: NSPasteboard setData:forType: failed")
	}
	return nil
}

func writeContents(contents Contents) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	pool := objc.ID(class_NSAutoreleasePool).Send(sel_new)
	defer pool.Send(sel_release)

	pasteboard := objc.ID(class_NSPasteboard).Send(sel_generalPasteboard)
	pasteboard.Send(sel_clearContents)

	if err := writePasteboardData(pasteboard, nsPasteboardTypeText, contents.Text); err != nil {
		return err
	}
	if err := writePasteboardData(pasteboard, nsPasteboardTypeHTML, contents.HTML); err != nil {
		return err
	}
	if err := writePasteboardData(pasteboard, nsPasteboardTypePNG, contents.Image); err != nil {
		return err
	}
	return nil
}
