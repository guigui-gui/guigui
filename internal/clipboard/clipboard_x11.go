// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

//go:build unix && !android && !darwin

package clipboard

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"unsafe"
)

const (
	x11ReadTimeout = 2 * time.Second
	x11PropName    = "GUIGUI_CLIPBOARD"
	// x11IncrChunkSize is the per-chunk byte budget for INCR transfers, and the
	// threshold above which a single-shot ChangeProperty switches to INCR. It
	// stays within the X11 spec's guaranteed minimum maximum request length
	// (4096 4-byte units = 16 KiB) so a single ChangeProperty fits on any
	// server, less a small header margin.
	x11IncrChunkSize = 4096*4 - 64
	// x11IncrSendStaleAfter is the inactivity window after which an
	// in-progress INCR send is considered abandoned and dropped. A requestor
	// that never reads its property (window destroyed, process exited mid-read,
	// etc.) would otherwise keep the payload alive in incrSends indefinitely.
	x11IncrSendStaleAfter = 30 * time.Second
)

type x11Clipboard struct {
	display uintptr
	win     xID

	atomClipboard xAtom
	atomUTF8      xAtom
	atomString    xAtom
	atomTargets   xAtom
	atomProp      xAtom
	atomIncr      xAtom

	mu      sync.Mutex
	ownData []byte

	notifyCh   chan xSelectionEvent
	propertyCh chan xPropertyEvent

	incrSendsMu sync.Mutex
	incrSends   map[incrSendKey]*incrSend
}

type incrSendKey struct {
	requestor xID
	property  xAtom
}

type incrSend struct {
	target xAtom
	data   []byte
	offset int
	// terminated is set after the final empty chunk has been written. The next
	// PropertyDelete from the requestor confirms the receipt and the entry is
	// removed.
	terminated bool
	// lastActivity is refreshed on every chunk advance. cleanupTimer fires on
	// or after lastActivity + x11IncrSendStaleAfter; if the activity stamp has
	// been pushed forward in the meantime, the cleanup re-arms for the
	// remainder instead of dropping the transfer.
	lastActivity time.Time
	cleanupTimer *time.Timer
}

var (
	x11State    *x11Clipboard
	x11InitOnce sync.Once
)

// ensureX11 returns the lazily-initialized X11 clipboard, or nil if the X
// server is unavailable. The init error is logged once; the caller silently
// no-ops so background polling does not spam the log.
func ensureX11() *x11Clipboard {
	x11InitOnce.Do(func() {
		c, err := newX11Clipboard()
		if err != nil {
			slog.Error("clipboard: failed to initialize X11 clipboard", "error", err)
			return
		}
		x11State = c
		go c.eventLoop()
	})
	return x11State
}

func newX11Clipboard() (*x11Clipboard, error) {
	if err := loadX11(); err != nil {
		return nil, err
	}
	// XInitThreads enables Xlib's internal locking so the event goroutine and
	// the read/write callers can use the connection concurrently. It must
	// precede XOpenDisplay; redundant calls (e.g. when the windowing backend
	// already initialized threading) are harmless.
	xInitThreads()

	display := xOpenDisplay(0)
	if display == 0 {
		return nil, errors.New("clipboard: XOpenDisplay failed")
	}
	x11Display = display
	installX11ErrorHandler()

	win := xCreateSimpleWindow(display, xDefaultRootWindow(display), 0, 0, 1, 1, 0, 0, 0)
	// PropertyChangeMask on the owner window is required so the receive side of
	// the INCR protocol can observe new chunks landing on its own property.
	xSelectInput(display, win, xPropertyChangeMask)

	c := &x11Clipboard{
		display:    display,
		win:        win,
		atomString: xaString,
		notifyCh:   make(chan xSelectionEvent, 1),
		propertyCh: make(chan xPropertyEvent, 64),
		incrSends:  make(map[incrSendKey]*incrSend),
	}
	for _, a := range []struct {
		dst  *xAtom
		name string
	}{
		{&c.atomClipboard, "CLIPBOARD"},
		{&c.atomUTF8, "UTF8_STRING"},
		{&c.atomTargets, "TARGETS"},
		{&c.atomProp, x11PropName},
		{&c.atomIncr, "INCR"},
	} {
		atom := xInternAtom(display, a.name, false)
		if atom == xAtomNone {
			return nil, fmt.Errorf("clipboard: XInternAtom(%s) failed", a.name)
		}
		*a.dst = atom
	}
	return c, nil
}

func (c *x11Clipboard) eventLoop() {
	for {
		var ev xEvent
		xNextEvent(c.display, &ev)
		switch ev.kind() {
		case xSelectionRequest:
			c.handleSelectionRequest(*(*xSelectionRequestEvent)(unsafe.Pointer(&ev)))
		case xSelectionClear:
			c.setOwnData(nil)
		case xSelectionNotify:
			select {
			case c.notifyCh <- *(*xSelectionEvent)(unsafe.Pointer(&ev)):
			default:
			}
		case xPropertyNotify:
			c.handlePropertyNotify(*(*xPropertyEvent)(unsafe.Pointer(&ev)))
		}
	}
}

func (c *x11Clipboard) getOwnData() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ownData
}

func (c *x11Clipboard) setOwnData(data []byte) {
	var cp []byte
	if data != nil {
		cp = make([]byte, len(data))
		copy(cp, data)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ownData = cp
}

func (c *x11Clipboard) handleSelectionRequest(e xSelectionRequestEvent) {
	notify := xSelectionEvent{
		kind:      xSelectionNotify,
		time:      e.time,
		requestor: e.requestor,
		selection: e.selection,
		target:    e.target,
		property:  e.property,
	}
	// Per ICCCM, a None Property in the request means the requestor is
	// obsolete and the target atom should be used as the property.
	prop := e.property
	if prop == xAtomNone {
		prop = e.target
	}

	data := c.getOwnData()

	switch e.target {
	case c.atomTargets:
		// With format 32, Xlib reads the data as an array of C long, so the
		// atoms must be native words rather than packed 32-bit values.
		targets := []uint{uint(c.atomTargets), uint(c.atomUTF8), uint(c.atomString)}
		xChangeProperty(c.display, e.requestor, prop, xaAtom, 32, xPropModeReplace,
			unsafe.Pointer(&targets[0]), int32(len(targets)))
		notify.property = prop
	case c.atomUTF8, c.atomString:
		switch {
		case data == nil:
			notify.property = xAtomNone
		case len(data) <= x11IncrChunkSize:
			xChangeProperty(c.display, e.requestor, prop, e.target, 8, xPropModeReplace,
				unsafe.Pointer(unsafe.SliceData(data)), int32(len(data)))
			notify.property = prop
		default:
			c.startIncrSend(e.requestor, prop, e.target, data)
			notify.property = prop
		}
	default:
		notify.property = xAtomNone
	}

	xSendEvent(c.display, e.requestor, false, 0, (*xEvent)(unsafe.Pointer(&notify)))
	xFlush(c.display)
}

// startIncrSend kicks off an INCR transfer to the requestor by setting an
// INCR-typed property containing the total payload size. The actual payload
// chunks are pushed in advanceIncrSend as the requestor deletes the property
// after each read.
func (c *x11Clipboard) startIncrSend(requestor xID, property, target xAtom, data []byte) {
	// PropertyChangeMask on the requestor lets the send side observe the
	// requestor deleting the property after each chunk. Should the requestor
	// already be gone, the resulting BadWindow is reported by the error
	// handler and the transfer is dropped by the staleness timer.
	xSelectInput(c.display, requestor, xPropertyChangeMask)

	// The data slice is an internal copy returned by getOwnData, so retain it
	// directly without an extra copy.
	key := incrSendKey{requestor, property}
	s := &incrSend{
		target:       target,
		data:         data,
		lastActivity: time.Now(),
	}
	s.cleanupTimer = time.AfterFunc(x11IncrSendStaleAfter, func() {
		c.cleanupStaleIncrSend(key, s)
	})

	c.incrSendsMu.Lock()
	if old, ok := c.incrSends[key]; ok {
		old.cleanupTimer.Stop()
	}
	c.incrSends[key] = s
	c.incrSendsMu.Unlock()

	size := uint(len(data))
	xChangeProperty(c.display, requestor, property, c.atomIncr, 32, xPropModeReplace,
		unsafe.Pointer(&size), 1)
	xFlush(c.display)
}

// advanceIncrSend pushes the next chunk of an in-progress INCR transfer in
// response to the requestor deleting the property. After the entire payload
// has been delivered, a final zero-length write signals end-of-stream; the
// subsequent delete drops the entry from the map.
func (c *x11Clipboard) advanceIncrSend(requestor xID, property xAtom) {
	target, chunk, send, unsubscribe := c.nextIncrChunk(incrSendKey{requestor, property})
	if unsubscribe {
		c.unsubscribeRequestor(requestor)
	}
	if !send {
		return
	}
	xChangeProperty(c.display, requestor, property, target, 8, xPropModeReplace,
		unsafe.Pointer(unsafe.SliceData(chunk)), int32(len(chunk)))
	xFlush(c.display)
}

// nextIncrChunk returns the next chunk to write for an in-progress INCR
// transfer, advancing the transfer's offset under the lock. send is false
// when there is no chunk to write — either because the entry is unknown or
// because the prior call already wrote the terminating zero-length chunk —
// in which case unsubscribe indicates whether the caller should also clear
// the per-client event subscription on the requestor's window.
func (c *x11Clipboard) nextIncrChunk(key incrSendKey) (target xAtom, chunk []byte, send, unsubscribe bool) {
	c.incrSendsMu.Lock()
	defer c.incrSendsMu.Unlock()

	s, exists := c.incrSends[key]
	if !exists {
		return 0, nil, false, false
	}
	if s.terminated {
		unsubscribe = c.removeIncrSendLocked(key)
		return 0, nil, false, unsubscribe
	}

	if s.offset < len(s.data) {
		end := min(s.offset+x11IncrChunkSize, len(s.data))
		chunk = s.data[s.offset:end]
		s.offset = end
	} else {
		s.terminated = true
	}
	s.lastActivity = time.Now()
	return s.target, chunk, true, false
}

// cleanupStaleIncrSend is the AfterFunc body installed by startIncrSend. If
// activity has happened since the timer was scheduled, it re-arms for the
// remaining window; otherwise the entry is dropped so its payload can be
// freed even when no further INCR sends are started.
func (c *x11Clipboard) cleanupStaleIncrSend(key incrSendKey, s *incrSend) {
	unsubscribe := func() bool {
		c.incrSendsMu.Lock()
		defer c.incrSendsMu.Unlock()
		cur, ok := c.incrSends[key]
		if !ok || cur != s {
			return false
		}
		elapsed := time.Since(cur.lastActivity)
		if elapsed >= x11IncrSendStaleAfter {
			return c.removeIncrSendLocked(key)
		}
		cur.cleanupTimer.Reset(x11IncrSendStaleAfter - elapsed)
		return false
	}()
	if unsubscribe {
		c.unsubscribeRequestor(key.requestor)
	}
}

// removeIncrSendLocked drops the entry for key and stops its cleanup timer.
// It returns true when no other transfers remain to the same requestor, in
// which case the caller is expected to clear the per-client event mask on
// that window — outside the lock, since X requests may block on the display
// lock. Must be called with incrSendsMu held.
func (c *x11Clipboard) removeIncrSendLocked(key incrSendKey) (unsubscribe bool) {
	s, ok := c.incrSends[key]
	if !ok {
		return false
	}
	s.cleanupTimer.Stop()
	delete(c.incrSends, key)
	for k := range c.incrSends {
		if k.requestor == key.requestor {
			return false
		}
	}
	return true
}

// unsubscribeRequestor clears the PropertyChangeMask subscription this
// client installed on the requestor when starting an INCR send. Best-effort:
// the requestor's window may already be destroyed, in which case the
// resulting BadWindow surfaces through the error handler.
func (c *x11Clipboard) unsubscribeRequestor(requestor xID) {
	xSelectInput(c.display, requestor, xNoEventMask)
	xFlush(c.display)
}

func (c *x11Clipboard) handlePropertyNotify(e xPropertyEvent) {
	if e.window == c.win {
		// New chunk landed on the receive-side property during an INCR read.
		// The send is non-blocking on purpose: INCR is strictly serialized
		// (the sender writes the next chunk only after observing the previous
		// one being deleted), so at most one event is in flight at a time and
		// the buffer is far larger than that. A blocking send here would risk
		// stalling the entire event goroutine — and with it SelectionRequest
		// handling for outgoing transfers — if the buffer ever did fill from a
		// pathological producer.
		if e.state == xPropertyNewValue {
			select {
			case c.propertyCh <- e:
			default:
				slog.Warn("clipboard: dropped PropertyNewValue event; INCR receive may stall",
					"atom", e.atom)
			}
		}
		return
	}
	// Requestor deleted the property after consuming the previous chunk; push
	// the next one.
	if e.state == xPropertyDelete {
		c.advanceIncrSend(e.window, e.atom)
	}
}

func readAll() ([]byte, error) {
	c := ensureX11()
	if c == nil {
		return nil, nil
	}
	return c.read()
}

func writeAll(data []byte) error {
	c := ensureX11()
	if c == nil {
		return nil
	}
	return c.write(data)
}

func (c *x11Clipboard) read() ([]byte, error) {
	if data := c.getOwnData(); data != nil {
		out := make([]byte, len(data))
		copy(out, data)
		return out, nil
	}

	owner := xGetSelectionOwner(c.display, c.atomClipboard)
	if owner == xWindowNone {
		return nil, nil
	}

	// Drain any stray notifications before issuing the request so only this
	// reply is observed.
	for {
		select {
		case <-c.notifyCh:
			continue
		default:
		}
		break
	}
	for {
		select {
		case <-c.propertyCh:
			continue
		default:
		}
		break
	}

	xConvertSelection(c.display, c.atomClipboard, c.atomUTF8, c.atomProp, c.win, xCurrentTime)
	xFlush(c.display)

	var ev xSelectionEvent
	select {
	case ev = <-c.notifyCh:
	case <-time.After(x11ReadTimeout):
		return nil, errors.New("clipboard: read timeout")
	}
	if ev.property == xAtomNone {
		return nil, nil
	}

	value, typeAtom, err := c.readProperty(ev.property)
	if err != nil {
		return nil, err
	}
	if typeAtom == c.atomIncr {
		return c.readIncr(ev.property)
	}
	return value, nil
}

// readProperty reads the entire current value of a property on c.win,
// deleting it on completion. It loops on bytesAfter so a property whose value
// exceeds what a single GetWindowProperty reply can carry is reassembled
// correctly. Per X11, the server only deletes the property when the final
// reply has bytesAfter == 0, so passing delete=true on every call is safe.
func (c *x11Clipboard) readProperty(property xAtom) ([]byte, xAtom, error) {
	var value []byte
	var typeAtom xAtom
	var offset int
	for {
		var actualType xAtom
		var actualFormat int32
		var nitems uint
		var bytesAfter uint
		var prop uintptr
		if status := xGetWindowProperty(c.display, c.win, property,
			offset, 1<<20, true, xAnyPropertyType,
			&actualType, &actualFormat, &nitems, &bytesAfter, &prop); status != xSuccess {
			return nil, 0, fmt.Errorf("clipboard: XGetWindowProperty failed: status %d", status)
		}
		if offset == 0 {
			typeAtom = actualType
		}
		// Only byte data (format 8: the UTF8_STRING/STRING payload and INCR
		// chunks) is consumed. Non-8 formats — such as the format-32 INCR size
		// hint — are read solely to advance the loop and are ignored here.
		if prop != 0 {
			if actualFormat == 8 && nitems > 0 {
				value = append(value, unsafe.Slice((*byte)(unsafe.Pointer(prop)), nitems)...)
			}
			xFree(prop)
		}
		if bytesAfter == 0 {
			return value, typeAtom, nil
		}
		// Advance by the number of 32-bit units consumed.
		offset += int(nitems) * (int(actualFormat) / 8) / 4
	}
}

// readIncr collects an INCR-format selection by repeatedly waiting for the
// owner to write a new chunk to the receive property, reading and deleting
// it. A zero-length chunk signals end-of-stream.
func (c *x11Clipboard) readIncr(property xAtom) (out []byte, err error) {
	// Drain any stragglers in propertyCh on exit — successfully or not — so
	// they do not bleed into a subsequent read.
	defer func() {
		for {
			select {
			case <-c.propertyCh:
				continue
			default:
			}
			break
		}
	}()
	timer := time.NewTimer(x11ReadTimeout)
	defer timer.Stop()
	for {
		var ev xPropertyEvent
		select {
		case ev = <-c.propertyCh:
		case <-timer.C:
			return nil, errors.New("clipboard: INCR read timeout")
		}
		if ev.window != c.win || ev.atom != property || ev.state != xPropertyNewValue {
			continue
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(x11ReadTimeout)
		chunk, _, err := c.readProperty(property)
		if err != nil {
			return nil, err
		}
		if len(chunk) == 0 {
			return out, nil
		}
		out = append(out, chunk...)
	}
}

func (c *x11Clipboard) write(data []byte) error {
	c.setOwnData(data)

	xSetSelectionOwner(c.display, c.atomClipboard, c.win, xCurrentTime)
	xFlush(c.display)

	owner := xGetSelectionOwner(c.display, c.atomClipboard)
	if owner != c.win {
		c.setOwnData(nil)
		return errors.New("clipboard: failed to take selection ownership")
	}
	return nil
}
