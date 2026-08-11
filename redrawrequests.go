// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package guigui

import (
	"fmt"
	"image"
	"iter"
	"log/slog"
	"math/bits"

	"github.com/guigui-gui/guigui/internal/debugmode"
)

type redrawRequests struct {
	region image.Rectangle
}

func (r *redrawRequests) reset() {
	r.region = image.Rectangle{}
}

func (r *redrawRequests) empty() bool {
	return r.region.Empty()
}

func (r *redrawRequests) union(region image.Rectangle) image.Rectangle {
	return r.region.Union(region)
}

type requestRedrawReason int

const (
	// These reasons require only a redraw, not a rebuild.
	requestRedrawReasonExplicitRequest requestRedrawReason = iota
	requestRedrawReasonStateKeyChangedForDraw
	requestRedrawReasonTreeChanged

	// The remaining reasons require a rebuild in addition to a redraw.
	requestRedrawReasonStateKeyChangedForBuild
	requestRedrawReasonWidgetFocus
	requestRedrawReasonAppFocus
	requestRedrawReasonScreenSize
	requestRedrawReasonScreenDeviceScale
	requestRedrawReasonAppScale
	requestRedrawReasonColorMode
	requestRedrawReasonLocale
	requestRedrawReasonKeyBindingMode
)

func (r requestRedrawReason) String() string {
	switch r {
	case requestRedrawReasonExplicitRequest:
		return "explicit request"
	case requestRedrawReasonStateKeyChangedForDraw:
		return "state key changed for draw"
	case requestRedrawReasonTreeChanged:
		return "tree changed"
	case requestRedrawReasonStateKeyChangedForBuild:
		return "state key changed for build"
	case requestRedrawReasonWidgetFocus:
		return "widget focus"
	case requestRedrawReasonAppFocus:
		return "app focus"
	case requestRedrawReasonScreenSize:
		return "screen size"
	case requestRedrawReasonScreenDeviceScale:
		return "screen device scale"
	case requestRedrawReasonAppScale:
		return "app scale"
	case requestRedrawReasonColorMode:
		return "color mode"
	case requestRedrawReasonLocale:
		return "locale"
	case requestRedrawReasonKeyBindingMode:
		return "key binding mode"
	default:
		return "unknown"
	}
}

// requestRedrawReasons is a set of requestRedrawReason values, one bit per reason.
// The zero value is the empty set, meaning no redraw is pending.
type requestRedrawReasons uint32

// redrawReasonsOf returns a set containing the single given reason.
func redrawReasonsOf(reason requestRedrawReason) requestRedrawReasons {
	var r requestRedrawReasons
	r.add(reason)
	return r
}

func (r requestRedrawReasons) empty() bool {
	return r == 0
}

func (r requestRedrawReasons) has(reason requestRedrawReason) bool {
	return r&(1<<reason) != 0
}

// triggersRebuild reports whether the set contains any reason that forces a tree rebuild,
// i.e. any reason other than a redraw-only one.
func (r requestRedrawReasons) triggersRebuild() bool {
	const redrawOnly = 1<<requestRedrawReasonExplicitRequest |
		1<<requestRedrawReasonStateKeyChangedForDraw |
		1<<requestRedrawReasonTreeChanged
	return r&^redrawOnly != 0
}

// all returns an iterator over the reasons in the set, in ascending reason order.
func (r requestRedrawReasons) all() iter.Seq[requestRedrawReason] {
	return func(yield func(requestRedrawReason) bool) {
		for rs := r; rs != 0; rs &= rs - 1 {
			reason := requestRedrawReason(bits.TrailingZeros32(uint32(rs)))
			if !yield(reason) {
				return
			}
		}
	}
}

func (r *requestRedrawReasons) add(reason requestRedrawReason) {
	*r |= 1 << reason
}

func (r *requestRedrawReasons) clear() {
	*r = 0
}

func (r *redrawRequests) add(region image.Rectangle, reasons requestRedrawReasons, widget Widget) {
	r.region = r.region.Union(region)
	if !debugmode.ShowRenderingRegions() {
		return
	}
	for reason := range reasons.all() {
		switch reason {
		case requestRedrawReasonExplicitRequest:
			slog.Info("request redrawing", "reason", reason.String(), "requester", fmt.Sprintf("%T", widget), "at", widget.widgetState().redrawRequestedAt, "region", region)
		case requestRedrawReasonStateKeyChangedForDraw, requestRedrawReasonTreeChanged, requestRedrawReasonStateKeyChangedForBuild:
			slog.Info("request redrawing", "reason", reason.String(), "requester", fmt.Sprintf("%T", widget), "region", region)
		default:
			slog.Info("request redrawing", "reason", reason.String(), "region", region)
		}
	}
}
