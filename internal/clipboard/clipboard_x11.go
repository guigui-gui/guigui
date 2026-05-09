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

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

const (
	x11ReadTimeout = 2 * time.Second
	x11PropName    = "GUIGUI_CLIPBOARD"
	// x11IncrChunkSizeCap upper-bounds the per-chunk byte budget for INCR
	// transfers. The actual chunk size is derived from the server's advertised
	// MaximumRequestLength at connection time, but the cap keeps in-flight
	// memory small even on servers that would allow much larger requests.
	x11IncrChunkSizeCap = 64 * 1024
	// x11IncrSendStaleAfter is the inactivity window after which an
	// in-progress INCR send is considered abandoned and dropped. A requestor
	// that never reads its property (window destroyed, process exited mid-read,
	// etc.) would otherwise keep the payload alive in incrSends indefinitely.
	x11IncrSendStaleAfter = 30 * time.Second
)

type x11Clipboard struct {
	conn *xgb.Conn
	win  xproto.Window

	atomClipboard xproto.Atom
	atomUTF8      xproto.Atom
	atomString    xproto.Atom
	atomTargets   xproto.Atom
	atomProp      xproto.Atom
	atomIncr      xproto.Atom

	// incrChunkSize is the per-chunk byte budget for INCR transfers, derived
	// from the server's MaximumRequestLength at connection time and capped at
	// x11IncrChunkSizeCap. It also doubles as the threshold for switching from
	// a single-shot ChangeProperty to INCR.
	incrChunkSize int

	mu      sync.Mutex
	ownData []byte

	notifyCh   chan xproto.SelectionNotifyEvent
	propertyCh chan xproto.PropertyNotifyEvent

	incrSendsMu sync.Mutex
	incrSends   map[incrSendKey]*incrSend
}

type incrSendKey struct {
	requestor xproto.Window
	property  xproto.Atom
}

type incrSend struct {
	target xproto.Atom
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
		go func() {
			for {
				ev, err := c.conn.WaitForEvent()
				if ev == nil && err == nil {
					return
				}
				if err != nil {
					slog.Error("clipboard: X event error", "error", err)
					continue
				}
				switch e := ev.(type) {
				case xproto.SelectionRequestEvent:
					c.handleSelectionRequest(e)
				case xproto.SelectionClearEvent:
					c.setOwnData(nil)
				case xproto.SelectionNotifyEvent:
					select {
					case c.notifyCh <- e:
					default:
					}
				case xproto.PropertyNotifyEvent:
					c.handlePropertyNotify(e)
				}
			}
		}()
	})
	return x11State
}

func newX11Clipboard() (*x11Clipboard, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("clipboard: NewConn failed: %w", err)
	}
	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)

	wid, err := xproto.NewWindowId(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("clipboard: NewWindowId failed: %w", err)
	}
	// PropertyChangeMask on the owner window is required so the receive side of
	// the INCR protocol can observe new chunks landing on its own property.
	if err := xproto.CreateWindowChecked(conn, screen.RootDepth, wid, screen.Root,
		0, 0, 1, 1, 0,
		xproto.WindowClassInputOutput, screen.RootVisual,
		xproto.CwEventMask, []uint32{xproto.EventMaskPropertyChange}).Check(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("clipboard: CreateWindow failed: %w", err)
	}

	c := &x11Clipboard{
		conn:          conn,
		win:           wid,
		atomString:    xproto.AtomString,
		incrChunkSize: deriveIncrChunkSize(setup.MaximumRequestLength),
		notifyCh:      make(chan xproto.SelectionNotifyEvent, 1),
		propertyCh:    make(chan xproto.PropertyNotifyEvent, 64),
		incrSends:     make(map[incrSendKey]*incrSend),
	}
	for _, a := range []struct {
		dst  *xproto.Atom
		name string
	}{
		{&c.atomClipboard, "CLIPBOARD"},
		{&c.atomUTF8, "UTF8_STRING"},
		{&c.atomTargets, "TARGETS"},
		{&c.atomProp, x11PropName},
		{&c.atomIncr, "INCR"},
	} {
		atom, err := internAtom(conn, a.name)
		if err != nil {
			conn.Close()
			return nil, err
		}
		*a.dst = atom
	}
	return c, nil
}

// deriveIncrChunkSize picks a per-chunk byte budget that respects the
// server's advertised MaximumRequestLength. The X11 spec mandates a minimum
// of 4096 4-byte units = 16 KiB; ChangeProperty has a 24-byte header, so
// drop a small safety margin and cap at x11IncrChunkSizeCap to keep
// in-flight memory predictable.
func deriveIncrChunkSize(maxRequestLength uint16) int {
	const headerSafety = 64
	return min(max(int(maxRequestLength)*4-headerSafety, 1024), x11IncrChunkSizeCap)
}

func internAtom(conn *xgb.Conn, name string) (xproto.Atom, error) {
	reply, err := xproto.InternAtom(conn, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0, fmt.Errorf("clipboard: InternAtom(%s) failed: %w", name, err)
	}
	return reply.Atom, nil
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

func (c *x11Clipboard) handleSelectionRequest(e xproto.SelectionRequestEvent) {
	notify := xproto.SelectionNotifyEvent{
		Time:      e.Time,
		Requestor: e.Requestor,
		Selection: e.Selection,
		Target:    e.Target,
		Property:  e.Property,
	}
	// Per ICCCM, a None Property in the request means the requestor is
	// obsolete and the target atom should be used as the property.
	prop := e.Property
	if prop == xproto.AtomNone {
		prop = e.Target
	}

	data := c.getOwnData()

	switch e.Target {
	case c.atomTargets:
		buf := make([]byte, 12)
		xgb.Put32(buf[0:], uint32(c.atomTargets))
		xgb.Put32(buf[4:], uint32(c.atomUTF8))
		xgb.Put32(buf[8:], uint32(c.atomString))
		xproto.ChangeProperty(c.conn, xproto.PropModeReplace, e.Requestor, prop,
			xproto.AtomAtom, 32, 3, buf)
		notify.Property = prop
	case c.atomUTF8, c.atomString:
		switch {
		case data == nil:
			notify.Property = xproto.AtomNone
		case len(data) <= c.incrChunkSize:
			xproto.ChangeProperty(c.conn, xproto.PropModeReplace, e.Requestor, prop,
				e.Target, 8, uint32(len(data)), data)
			notify.Property = prop
		default:
			if err := c.startIncrSend(e.Requestor, prop, e.Target, data); err != nil {
				slog.Error("clipboard: failed to start INCR send", "error", err)
				notify.Property = xproto.AtomNone
			} else {
				notify.Property = prop
			}
		}
	default:
		notify.Property = xproto.AtomNone
	}

	xproto.SendEvent(c.conn, false, e.Requestor, 0, string(notify.Bytes()))
}

// startIncrSend kicks off an INCR transfer to the requestor by setting an
// INCR-typed property containing the total payload size. The actual payload
// chunks are pushed in advanceIncrSend as the requestor deletes the property
// after each read.
func (c *x11Clipboard) startIncrSend(requestor xproto.Window, property, target xproto.Atom, data []byte) error {
	if err := xproto.ChangeWindowAttributesChecked(c.conn, requestor, xproto.CwEventMask,
		[]uint32{xproto.EventMaskPropertyChange}).Check(); err != nil {
		return fmt.Errorf("clipboard: ChangeWindowAttributes(PropertyChange) on requestor failed: %w", err)
	}

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

	sizeBuf := make([]byte, 4)
	xgb.Put32(sizeBuf, uint32(len(data)))
	xproto.ChangeProperty(c.conn, xproto.PropModeReplace, requestor, property,
		c.atomIncr, 32, 1, sizeBuf)
	return nil
}

// advanceIncrSend pushes the next chunk of an in-progress INCR transfer in
// response to the requestor deleting the property. After the entire payload
// has been delivered, a final zero-length write signals end-of-stream; the
// subsequent delete drops the entry from the map.
func (c *x11Clipboard) advanceIncrSend(requestor xproto.Window, property xproto.Atom) {
	target, chunk, send, unsubscribe := c.nextIncrChunk(incrSendKey{requestor, property})
	if unsubscribe {
		c.unsubscribeRequestor(requestor)
	}
	if !send {
		return
	}
	xproto.ChangeProperty(c.conn, xproto.PropModeReplace, requestor, property,
		target, 8, uint32(len(chunk)), chunk)
}

// nextIncrChunk returns the next chunk to write for an in-progress INCR
// transfer, advancing the transfer's offset under the lock. send is false
// when there is no chunk to write — either because the entry is unknown or
// because the prior call already wrote the terminating zero-length chunk —
// in which case unsubscribe indicates whether the caller should also clear
// the per-client event subscription on the requestor's window.
func (c *x11Clipboard) nextIncrChunk(key incrSendKey) (target xproto.Atom, chunk []byte, send, unsubscribe bool) {
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
		end := min(s.offset+c.incrChunkSize, len(s.data))
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
// that window — outside the lock, since X requests may block on xgb's
// internal request queue. Must be called with incrSendsMu held.
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
// resulting BadWindow surfaces through the event goroutine's error log.
func (c *x11Clipboard) unsubscribeRequestor(requestor xproto.Window) {
	xproto.ChangeWindowAttributes(c.conn, requestor, xproto.CwEventMask,
		[]uint32{xproto.EventMaskNoEvent})
}

func (c *x11Clipboard) handlePropertyNotify(e xproto.PropertyNotifyEvent) {
	if e.Window == c.win {
		// New chunk landed on the receive-side property during an INCR read.
		// The send is non-blocking on purpose: INCR is strictly serialized
		// (the sender writes the next chunk only after observing the previous
		// one being deleted), so at most one event is in flight at a time and
		// the buffer is far larger than that. A blocking send here would risk
		// stalling the entire event goroutine — and with it SelectionRequest
		// handling for our outgoing transfers — if the buffer ever did fill
		// from a pathological producer.
		if e.State == xproto.PropertyNewValue {
			select {
			case c.propertyCh <- e:
			default:
				slog.Warn("clipboard: dropped PropertyNewValue event; INCR receive may stall",
					"atom", e.Atom)
			}
		}
		return
	}
	// Requestor deleted the property after consuming the previous chunk; push
	// the next one.
	if e.State == xproto.PropertyDelete {
		c.advanceIncrSend(e.Window, e.Atom)
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

	owner, err := xproto.GetSelectionOwner(c.conn, c.atomClipboard).Reply()
	if err != nil {
		return nil, fmt.Errorf("clipboard: GetSelectionOwner failed: %w", err)
	}
	if owner.Owner == xproto.WindowNone {
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

	if err := xproto.ConvertSelectionChecked(c.conn, c.win, c.atomClipboard,
		c.atomUTF8, c.atomProp, xproto.TimeCurrentTime).Check(); err != nil {
		return nil, fmt.Errorf("clipboard: ConvertSelection failed: %w", err)
	}

	var ev xproto.SelectionNotifyEvent
	select {
	case ev = <-c.notifyCh:
	case <-time.After(x11ReadTimeout):
		return nil, errors.New("clipboard: read timeout")
	}
	if ev.Property == xproto.AtomNone {
		return nil, nil
	}

	value, typeAtom, err := c.readProperty(ev.Property)
	if err != nil {
		return nil, err
	}
	if typeAtom == c.atomIncr {
		return c.readIncr(ev.Property)
	}
	return value, nil
}

// readProperty reads the entire current value of a property on c.win,
// deleting it on completion. It loops on BytesAfter so a property whose
// value exceeds what a single GetProperty reply can carry is reassembled
// correctly. Per X11, the server only deletes the property when the final
// reply has BytesAfter == 0, so passing delete=true on every call is safe.
func (c *x11Clipboard) readProperty(property xproto.Atom) ([]byte, xproto.Atom, error) {
	var value []byte
	var typeAtom xproto.Atom
	var offset uint32
	for {
		reply, err := xproto.GetProperty(c.conn, true, c.win, property,
			xproto.AtomAny, offset, 1<<20).Reply()
		if err != nil {
			return nil, 0, fmt.Errorf("clipboard: GetProperty failed: %w", err)
		}
		if reply == nil {
			return nil, 0, errors.New("clipboard: nil GetProperty reply")
		}
		if offset == 0 {
			typeAtom = reply.Type
		}
		value = append(value, reply.Value...)
		if reply.BytesAfter == 0 {
			return value, typeAtom, nil
		}
		offset += uint32(len(reply.Value)) / 4
	}
}

// readIncr collects an INCR-format selection by repeatedly waiting for the
// owner to write a new chunk to the receive property, reading and deleting
// it. A zero-length chunk signals end-of-stream.
func (c *x11Clipboard) readIncr(property xproto.Atom) (out []byte, err error) {
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
		var ev xproto.PropertyNotifyEvent
		select {
		case ev = <-c.propertyCh:
		case <-timer.C:
			return nil, errors.New("clipboard: INCR read timeout")
		}
		if ev.Window != c.win || ev.Atom != property || ev.State != xproto.PropertyNewValue {
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

	if err := xproto.SetSelectionOwnerChecked(c.conn, c.win, c.atomClipboard,
		xproto.TimeCurrentTime).Check(); err != nil {
		c.setOwnData(nil)
		return fmt.Errorf("clipboard: SetSelectionOwner failed: %w", err)
	}
	owner, err := xproto.GetSelectionOwner(c.conn, c.atomClipboard).Reply()
	if err != nil {
		return fmt.Errorf("clipboard: GetSelectionOwner failed: %w", err)
	}
	if owner.Owner != c.win {
		return errors.New("clipboard: failed to take selection ownership")
	}
	return nil
}
