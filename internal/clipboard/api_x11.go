// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

//go:build unix && !android && !darwin

package clipboard

import (
	"fmt"
	"log/slog"
	"unsafe"

	"github.com/ebitengine/purego"
)

// This file binds the handful of libX11 functions the X11 clipboard needs via
// purego.
//
// Type mapping: C int/Bool -> int32, C long/unsigned long (and the XID, Atom,
// and Time families, all unsigned long) -> int/uint, C pointers -> uintptr.
// C long and unsigned long are pointer-sized on every supported Unix ABI, as
// are Go's int and uint.
//
// Pointers passed IN to Xlib that reference Go memory are typed pointers or
// unsafe.Pointer; purego keeps them alive for the duration of the call.
// Pointers received FROM Xlib are C-owned memory represented as uintptr and
// must be released with xFree where the Xlib contract requires it.
type (
	xID   = uint
	xAtom = uint
	xTime = uint
)

const (
	xPropModeReplace = 0

	xPropertyChangeMask = 1 << 22
	xNoEventMask        = 0

	// Predefined atoms and sentinels with fixed protocol values.
	xAtomNone        xAtom = 0
	xAnyPropertyType xAtom = 0
	xaAtom           xAtom = 4  // XA_ATOM
	xaString         xAtom = 31 // XA_STRING

	xCurrentTime xTime = 0
	xWindowNone  xID   = 0

	xSuccess = 0

	xPropertyNotify   = 28
	xSelectionClear   = 29
	xSelectionRequest = 30
	xSelectionNotify  = 31

	// State values of a PropertyNotify event.
	xPropertyNewValue = 0
	xPropertyDelete   = 1
)

// xEvent is the XEvent union. Its largest member is a [24]long pad array, so
// the union occupies 24 pointer-sized words.
type xEvent struct {
	data [24]int
}

// kind returns the event type, which is the first int of every XEvent member.
func (e *xEvent) kind() int32 {
	return *(*int32)(unsafe.Pointer(e))
}

type xSelectionRequestEvent struct {
	kind      int32
	serial    uint
	sendEvent int32
	display   uintptr
	owner     xID
	requestor xID
	selection xAtom
	target    xAtom
	property  xAtom
	time      xTime
}

type xSelectionEvent struct {
	kind      int32
	serial    uint
	sendEvent int32
	display   uintptr
	requestor xID
	selection xAtom
	target    xAtom
	property  xAtom
	time      xTime
}

type xPropertyEvent struct {
	kind      int32
	serial    uint
	sendEvent int32
	display   uintptr
	window    xID
	atom      xAtom
	time      xTime
	state     int32
}

type xErrorEvent struct {
	kind        int32
	display     uintptr
	resourceID  xID
	serial      uint
	errorCode   uint8
	requestCode uint8
	minorCode   uint8
}

var (
	xInitThreads        func() int32
	xOpenDisplay        func(displayName uintptr) uintptr
	xDefaultRootWindow  func(display uintptr) xID
	xCreateSimpleWindow func(display uintptr, parent xID, x, y int32, width, height, borderWidth uint32, border, background uint) xID
	xSelectInput        func(display uintptr, w xID, eventMask int) int32
	xInternAtom         func(display uintptr, atomName string, onlyIfExists bool) xAtom
	xChangeProperty     func(display uintptr, w xID, property, typ xAtom, format, mode int32, data unsafe.Pointer, nelements int32) int32
	xGetWindowProperty  func(display uintptr, w xID, property xAtom, longOffset, longLength int, delete bool, reqType xAtom, actualTypeReturn *xAtom, actualFormatReturn *int32, nitemsReturn, bytesAfterReturn *uint, propReturn *uintptr) int32
	xConvertSelection   func(display uintptr, selection, target, property xAtom, requestor xID, time xTime) int32
	xSetSelectionOwner  func(display uintptr, selection xAtom, owner xID, time xTime) int32
	xGetSelectionOwner  func(display uintptr, selection xAtom) xID
	xSendEvent          func(display uintptr, w xID, propagate bool, eventMask int, eventSend *xEvent) int32
	xNextEvent          func(display uintptr, eventReturn *xEvent) int32
	xFlush              func(display uintptr) int32
	xFree               func(data uintptr) int32
	xSetErrorHandler    func(handler uintptr) uintptr
)

var libX11 uintptr

// loadX11 dlopens libX11 and binds the functions used by the clipboard. It is
// safe to call repeatedly; binding happens only once.
func loadX11() error {
	if libX11 != 0 {
		return nil
	}
	lib, err := openX11Library("libX11.so.6", "libX11.so")
	if err != nil {
		return err
	}
	purego.RegisterLibFunc(&xInitThreads, lib, "XInitThreads")
	purego.RegisterLibFunc(&xOpenDisplay, lib, "XOpenDisplay")
	purego.RegisterLibFunc(&xDefaultRootWindow, lib, "XDefaultRootWindow")
	purego.RegisterLibFunc(&xCreateSimpleWindow, lib, "XCreateSimpleWindow")
	purego.RegisterLibFunc(&xSelectInput, lib, "XSelectInput")
	purego.RegisterLibFunc(&xInternAtom, lib, "XInternAtom")
	purego.RegisterLibFunc(&xChangeProperty, lib, "XChangeProperty")
	purego.RegisterLibFunc(&xGetWindowProperty, lib, "XGetWindowProperty")
	purego.RegisterLibFunc(&xConvertSelection, lib, "XConvertSelection")
	purego.RegisterLibFunc(&xSetSelectionOwner, lib, "XSetSelectionOwner")
	purego.RegisterLibFunc(&xGetSelectionOwner, lib, "XGetSelectionOwner")
	purego.RegisterLibFunc(&xSendEvent, lib, "XSendEvent")
	purego.RegisterLibFunc(&xNextEvent, lib, "XNextEvent")
	purego.RegisterLibFunc(&xFlush, lib, "XFlush")
	purego.RegisterLibFunc(&xFree, lib, "XFree")
	purego.RegisterLibFunc(&xSetErrorHandler, lib, "XSetErrorHandler")
	libX11 = lib
	return nil
}

func openX11Library(names ...string) (uintptr, error) {
	var firstErr error
	for _, name := range names {
		lib, err := purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil {
			return lib, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return 0, fmt.Errorf("clipboard: failed to load libX11: %w", firstErr)
}

var (
	// x11Display is the clipboard's display, used to filter errors meant for
	// this connection from those belonging to other Xlib users in the process
	// (e.g. the windowing backend), which share the process-global handler.
	x11Display uintptr
	// x11PrevErrorHandlerFn is the error handler that was installed before the
	// clipboard's, invoked for errors on other displays so the clipboard does
	// not swallow them.
	x11PrevErrorHandlerFn func(display, event uintptr) uintptr
)

// x11ErrorHandler is the process-global Xlib error handler. Async protocol
// errors (e.g. a BadWindow when a requestor window is destroyed mid-transfer)
// are reported here rather than at the call site.
func x11ErrorHandler(display, event uintptr) uintptr {
	if display == x11Display {
		e := (*xErrorEvent)(unsafe.Pointer(event))
		slog.Error("clipboard: X11 protocol error",
			"error_code", e.errorCode,
			"request_code", e.requestCode,
			"minor_code", e.minorCode)
		return 0
	}
	if x11PrevErrorHandlerFn != nil {
		return x11PrevErrorHandlerFn(display, event)
	}
	return 0
}

// installX11ErrorHandler installs x11ErrorHandler as the process-global Xlib
// error handler, retaining the previous one for errors on other displays.
func installX11ErrorHandler() {
	prev := xSetErrorHandler(purego.NewCallback(x11ErrorHandler))
	if prev != 0 {
		purego.RegisterFunc(&x11PrevErrorHandlerFn, prev)
	}
}
