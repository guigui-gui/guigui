// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image"
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	expanderEventExpansionChanged guigui.EventKey = guigui.GenerateEventKey()
	expanderHeaderEventDown       guigui.EventKey = guigui.GenerateEventKey()
)

type Expander struct {
	guigui.DefaultWidget

	header        expanderHeader
	headerWidget  guigui.Widget
	contentWidget guigui.Widget

	expanded     bool
	onceRendered bool
	count        int

	layoutItems []guigui.LinearLayoutItem
	onDown      func(context *guigui.Context)
}

func (e *Expander) OnExpansionChanged(callback func(context *guigui.Context, expanded bool)) {
	guigui.SetEventHandler(e, expanderEventExpansionChanged, callback)
}

func (e *Expander) SetHeaderWidget(w guigui.Widget) {
	e.headerWidget = w
}

func (e *Expander) SetContentWidget(w guigui.Widget) {
	e.contentWidget = w
}

// expandCollapseMaxCount returns the number of ticks for an expand/collapse animation.
// This is shared between Expander and List.
func expandCollapseMaxCount() int {
	return ebiten.TPS() / 20
}

func (e *Expander) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteBool(e.expanded)
	w.WriteInt(e.count)
	w.WriteWidget(e.headerWidget)
	w.WriteWidget(e.contentWidget)
}

func (e *Expander) SetExpanded(expanded bool) {
	if e.expanded == expanded {
		return
	}
	e.expanded = expanded
	e.header.setExpanded(e.expanded)
	if e.onceRendered {
		e.count = expandCollapseMaxCount() - e.count
	}
	guigui.DispatchEvent(e, expanderEventExpansionChanged, e.expanded)
}

func (e *Expander) isContentVisible() bool {
	return e.expanded || e.animating()
}

func (e *Expander) animating() bool {
	return e.count > 0
}

func (e *Expander) animationRate() float64 {
	if !e.animating() {
		if e.expanded {
			return 1
		}
		return 0
	}
	rate := 1 - float64(e.count)/float64(expandCollapseMaxCount())
	if !e.expanded {
		rate = 1 - rate
	}
	return rate
}

func (e *Expander) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&e.header)
	if e.isContentVisible() && e.contentWidget != nil {
		adder.AddWidget(e.contentWidget)
	}

	e.header.setWidget(e.headerWidget)
	if e.onDown == nil {
		e.onDown = func(context *guigui.Context) {
			e.SetExpanded(!e.expanded)
		}
	}
	e.header.setOnDown(e.onDown)

	context.SetClipChildren(e, e.animating())

	return nil
}

func (e *Expander) layout(context *guigui.Context) guigui.LinearLayout {
	e.layoutItems = slices.Delete(e.layoutItems, 0, len(e.layoutItems))
	u := UnitSize(context)
	e.layoutItems = append(e.layoutItems, guigui.LinearLayoutItem{
		Widget: &e.header,
		Size:   guigui.FixedSize(defaultIconSize(context)),
	})
	if e.isContentVisible() {
		e.layoutItems = append(e.layoutItems, guigui.LinearLayoutItem{
			Widget: e.contentWidget,
			Size:   guigui.FlexibleSize(1),
		})
	}

	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Gap:       u / 4,
		Items:     e.layoutItems,
	}
}

func (e *Expander) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	e.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (e *Expander) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	s := e.layout(context).Measure(context, constraints)
	if e.animating() {
		u := UnitSize(context)
		headerHeight := defaultIconSize(context)
		gap := u / 4
		contentHeight := s.Y - headerHeight - gap
		s.Y = headerHeight + gap + int(float64(contentHeight)*e.animationRate())
	}
	return s
}

func (e *Expander) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if e.count > 0 {
		e.count--
		guigui.RequestRedraw(e)
	}
	return nil
}

func (e *Expander) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	e.onceRendered = true
}

type expanderHeader struct {
	guigui.DefaultWidget

	image  Image
	widget guigui.Widget

	expanded   bool
	iconLayout guigui.LinearLayout

	iconLayoutItems []guigui.LinearLayoutItem
	layoutItems     []guigui.LinearLayoutItem
}

func (e *expanderHeader) setOnDown(callback func(context *guigui.Context)) {
	guigui.SetEventHandler(e, expanderHeaderEventDown, callback)
}

func (e *expanderHeader) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteBool(e.expanded)
	w.WriteWidget(e.widget)
}

func (e *expanderHeader) setExpanded(expanded bool) {
	e.expanded = expanded
}

func (e *expanderHeader) setWidget(w guigui.Widget) {
	e.widget = w
}

func (e *expanderHeader) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&e.image)
	if e.widget != nil {
		adder.AddWidget(e.widget)
	}

	var iconName string
	if e.expanded {
		iconName = "keyboard_arrow_down"
	} else {
		iconName = "keyboard_arrow_right"
	}
	icon, err := theResourceImages.Get(iconName, context.ColorMode())
	if err != nil {
		return err
	}
	e.image.SetImage(icon)

	return nil
}

func (e *expanderHeader) layout(context *guigui.Context) guigui.LinearLayout {
	u := UnitSize(context)

	e.iconLayoutItems = slices.Delete(e.iconLayoutItems, 0, len(e.iconLayoutItems))
	e.iconLayoutItems = append(e.iconLayoutItems,
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Size: guigui.FixedSize(UnitSize(context) / 32),
		},
		guigui.LinearLayoutItem{
			Widget: &e.image,
			Size:   guigui.FixedSize(defaultIconSize(context)),
		},
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		})
	e.iconLayout = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     e.iconLayoutItems,
	}

	e.layoutItems = slices.Delete(e.layoutItems, 0, len(e.layoutItems))
	e.layoutItems = append(e.layoutItems,
		guigui.LinearLayoutItem{
			Layout: &e.iconLayout,
			Size:   guigui.FixedSize(defaultIconSize(context)),
		},
		guigui.LinearLayoutItem{
			Widget: e.widget,
			Size:   guigui.FlexibleSize(1),
		})

	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Gap:       u / 4,
		Items:     e.layoutItems,
	}
}

func (e *expanderHeader) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	e.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (e *expanderHeader) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return e.layout(context).Measure(context, constraints)
}

func (e *expanderHeader) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if widgetBounds.IsHitAtCursor() {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			guigui.DispatchEvent(e, expanderHeaderEventDown)
			return guigui.HandleInputByWidget(e)
		}
	}
	return guigui.HandleInputResult{}
}

func (e *expanderHeader) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	return ebiten.CursorShapePointer, true
}
