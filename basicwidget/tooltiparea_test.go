// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget_test

import (
	"testing"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

func TestTooltipAreaPopupInputPassesThrough(t *testing.T) {
	var area basicwidget.TooltipArea
	var context guigui.Context

	defer basicwidget.CleanupTooltipAreaForTest(&area)

	area.SetText("Tooltip")
	if err := area.Build(&context, &guigui.ChildAdder{}); err != nil {
		t.Fatal(err)
	}

	if !basicwidget.TooltipAreaPopupIsModelessForTest(&area) {
		t.Fatal("tooltip popup is modal; want modeless")
	}
	if !basicwidget.TooltipAreaPopupLayerPassthroughForTest(&context, &area) {
		t.Fatal("tooltip popup layer is not passthrough")
	}
	if !basicwidget.TooltipAreaPopupFramePassthroughForTest(&context, &area) {
		t.Fatal("tooltip popup frame is not passthrough")
	}
	if !basicwidget.TooltipAreaContentPassthroughForTest(&context, &area) {
		t.Fatal("tooltip content is not passthrough")
	}
}

func TestTooltipAreaBuildDropsPendingTooltipWhenHoveringStopped(t *testing.T) {
	var area basicwidget.TooltipArea
	var context guigui.Context

	defer basicwidget.CleanupTooltipAreaForTest(&area)

	basicwidget.SetTooltipAreaPendingForTest(&area, false, true)
	if err := area.Build(&context, &guigui.ChildAdder{}); err != nil {
		t.Fatal(err)
	}

	if basicwidget.TooltipAreaPendingForTest(&area) {
		t.Fatal("pending tooltip was not cleared")
	}
	if basicwidget.TooltipAreaPopupIsOpenForTest(&area) {
		t.Fatal("tooltip popup opened after hovering stopped")
	}
}
