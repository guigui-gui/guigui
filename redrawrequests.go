// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package guigui

import (
	"fmt"
	"image"
	"log/slog"
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
	requestRedrawReasonUnknown requestRedrawReason = iota
	requestRedrawReasonRebuildWidget
	requestRedrawReasonRedrawWidget
	requestRedrawReasonLayout
	requestRedrawReasonScreenSize
	requestRedrawReasonScreenDeviceScale
	requestRedrawReasonAppScale
	requestRedrawReasonColorMode
	requestRedrawReasonLocale
)

func (r *redrawRequests) add(region image.Rectangle, reason requestRedrawReason, widget Widget) {
	r.region = r.region.Union(region)
	if theDebugMode.showRenderingRegions {
		switch reason {
		case requestRedrawReasonRebuildWidget:
			slog.Info("request redrawing", "reason", "rebuild widget", "requester", fmt.Sprintf("%T", widget), "at", widget.widgetState().rebuildRequestedAt, "region", region)
		case requestRedrawReasonRedrawWidget:
			slog.Info("request redrawing", "reason", "redraw widget", "requester", fmt.Sprintf("%T", widget), "at", widget.widgetState().redrawRequestedAt, "region", region)
		case requestRedrawReasonLayout:
			slog.Info("request redrawing", "reason", "layout", "region", region)
		case requestRedrawReasonScreenSize:
			slog.Info("request redrawing", "reason", "screen size", "region", region)
		case requestRedrawReasonScreenDeviceScale:
			slog.Info("request redrawing", "reason", "screen device scale", "region", region)
		case requestRedrawReasonAppScale:
			slog.Info("request redrawing", "reason", "app scale", "region", region)
		case requestRedrawReasonColorMode:
			slog.Info("request redrawing", "reason", "color mode", "region", region)
		case requestRedrawReasonLocale:
			slog.Info("request redrawing", "reason", "locale", "region", region)
		default:
			slog.Info("request redrawing", "reason", "unknown", "region", region)
		}
	}
}
