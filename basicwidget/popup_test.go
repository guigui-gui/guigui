// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget_test

import (
	"testing"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

func advancePopup(t *testing.T, p *basicwidget.PopupState, ticks int) {
	t.Helper()
	for range ticks {
		if err := p.Advance(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPopupRepeatedSetOpen(t *testing.T) {
	var p basicwidget.PopupState
	var reasons []basicwidget.PopupCloseReason
	p.OnClose(func(_ *guigui.Context, reason basicwidget.PopupCloseReason) {
		reasons = append(reasons, reason)
	})
	for range 120 {
		p.SetOpen(true)
		advancePopup(t, &p, 1)
		if !p.IsOpen() || p.Passthrough() {
			t.Fatal("popup stopped receiving input while requested open")
		}
	}
	if len(reasons) != 0 {
		t.Fatalf("unexpected close events: %v", reasons)
	}
	for range 120 {
		p.SetOpen(false)
		advancePopup(t, &p, 1)
	}
	if p.IsOpen() {
		t.Fatal("popup remained open")
	}
	if len(reasons) != 1 || reasons[0] != basicwidget.PopupCloseReasonFuncCall {
		t.Fatalf("close events = %v, want one function-call close", reasons)
	}
}

func TestPopupCancelPendingClose(t *testing.T) {
	var p basicwidget.PopupState
	p.SetOpen(true)
	advancePopup(t, &p, 30)
	p.OnClose(func(_ *guigui.Context, reason basicwidget.PopupCloseReason) {
		t.Errorf("unexpected close event: %v", reason)
	})
	p.SetOpen(false)
	p.SetOpen(true)
	advancePopup(t, &p, 30)
	if !p.IsOpen() || p.Passthrough() {
		t.Fatal("canceling a pending close did not keep the popup open")
	}
	p.OnClose(func(*guigui.Context, basicwidget.PopupCloseReason) {})
	p.SetOpen(false)
	advancePopup(t, &p, 30)
}

func TestPopupReopenWhileClosing(t *testing.T) {
	for _, tc := range []struct {
		name   string
		cancel bool
	}{{name: "reopen"}, {name: "cancel reopen", cancel: true}} {
		t.Run(tc.name, func(t *testing.T) {
			cancel := tc.cancel
			var p basicwidget.PopupState
			p.SetOpen(true)
			advancePopup(t, &p, 30)
			var reasons []basicwidget.PopupCloseReason
			p.OnClose(func(_ *guigui.Context, reason basicwidget.PopupCloseReason) {
				reasons = append(reasons, reason)
			})
			p.CloseByClickingOutside()
			p.SetOpen(true)
			advancePopup(t, &p, 1)
			if cancel {
				p.SetOpen(false)
			}
			advancePopup(t, &p, 30)
			if p.IsOpen() != !cancel {
				t.Fatalf("IsOpen() = %v, want %v", p.IsOpen(), !cancel)
			}
			if len(reasons) != 1 || reasons[0] != basicwidget.PopupCloseReasonReopen {
				t.Fatalf("close events = %v, want one reopen close", reasons)
			}
			p.OnClose(func(*guigui.Context, basicwidget.PopupCloseReason) {})
			p.SetOpen(false)
			advancePopup(t, &p, 30)
		})
	}
}
