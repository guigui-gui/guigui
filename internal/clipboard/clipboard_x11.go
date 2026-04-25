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
)

type x11Clipboard struct {
	conn *xgb.Conn
	win  xproto.Window

	atomClipboard xproto.Atom
	atomUTF8      xproto.Atom
	atomString    xproto.Atom
	atomTargets   xproto.Atom
	atomProp      xproto.Atom

	mu      sync.Mutex
	ownData []byte

	notifyCh chan xproto.SelectionNotifyEvent
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
	screen := xproto.Setup(conn).DefaultScreen(conn)

	wid, err := xproto.NewWindowId(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("clipboard: NewWindowId failed: %w", err)
	}
	if err := xproto.CreateWindowChecked(conn, screen.RootDepth, wid, screen.Root,
		0, 0, 1, 1, 0,
		xproto.WindowClassInputOutput, screen.RootVisual, 0, nil).Check(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("clipboard: CreateWindow failed: %w", err)
	}

	c := &x11Clipboard{
		conn:       conn,
		win:        wid,
		atomString: xproto.AtomString,
		notifyCh:   make(chan xproto.SelectionNotifyEvent, 1),
	}
	for _, a := range []struct {
		dst  *xproto.Atom
		name string
	}{
		{&c.atomClipboard, "CLIPBOARD"},
		{&c.atomUTF8, "UTF8_STRING"},
		{&c.atomTargets, "TARGETS"},
		{&c.atomProp, x11PropName},
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
	// obsolete and we should fall back to using the target atom.
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
		if data == nil {
			notify.Property = xproto.AtomNone
		} else {
			xproto.ChangeProperty(c.conn, xproto.PropModeReplace, e.Requestor, prop,
				e.Target, 8, uint32(len(data)), data)
			notify.Property = prop
		}
	default:
		notify.Property = xproto.AtomNone
	}

	xproto.SendEvent(c.conn, false, e.Requestor, 0, string(notify.Bytes()))
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

	// Drain any stray notify before issuing the request so we only see our reply.
	select {
	case <-c.notifyCh:
	default:
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

	reply, err := xproto.GetProperty(c.conn, true, c.win, ev.Property,
		xproto.AtomAny, 0, 1<<24).Reply()
	if err != nil {
		return nil, fmt.Errorf("clipboard: GetProperty failed: %w", err)
	}
	if reply == nil {
		return nil, nil
	}
	out := make([]byte, len(reply.Value))
	copy(out, reply.Value)
	return out, nil
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
