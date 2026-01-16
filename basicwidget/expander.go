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

const (
	expanderEventExpansionChanged = "expansionChanged"
	expanderToggleEventDown       = "down"
)

type Expander struct {
	guigui.DefaultWidget

	toggle        expanderToggle
	headerWidget  guigui.Widget
	contentWidget guigui.Widget

	expanded bool

	layoutItems []guigui.LinearLayoutItem
	onDown      func(context *guigui.Context)
}

func (e *Expander) SetOnExpansionChanged(callback func(context *guigui.Context, expanded bool)) {
	guigui.SetEventHandler(e, expanderEventExpansionChanged, callback)
}

func (e *Expander) SetHeaderWidget(w guigui.Widget) {
	if e.headerWidget == w {
		return
	}
	e.headerWidget = w
	guigui.RequestRebuild(e)
}

func (e *Expander) SetContentWidget(w guigui.Widget) {
	if e.contentWidget == w {
		return
	}
	e.contentWidget = w
	guigui.RequestRebuild(e)
}

func (e *Expander) SetExpanded(expanded bool) {
	if e.expanded == expanded {
		return
	}
	e.expanded = expanded
	e.toggle.setExpanded(e.expanded)
	guigui.DispatchEvent(e, expanderEventExpansionChanged, e.expanded)
	guigui.RequestRebuild(e)
}

func (e *Expander) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&e.toggle)
	if e.headerWidget != nil {
		adder.AddChild(e.headerWidget)
	}
	if e.expanded && e.contentWidget != nil {
		adder.AddChild(e.contentWidget)
	}

	if e.onDown == nil {
		e.onDown = func(context *guigui.Context) {
			e.SetExpanded(!e.expanded)
		}
	}
	e.toggle.setOnDown(e.onDown)

	return nil
}

func (e *Expander) layout(context *guigui.Context) guigui.LinearLayout {
	e.layoutItems = slices.Delete(e.layoutItems, 0, len(e.layoutItems))
	u := UnitSize(context)
	e.layoutItems = append(e.layoutItems, guigui.LinearLayoutItem{
		Layout: guigui.LinearLayout{
			Direction: guigui.LayoutDirectionHorizontal,
			Gap:       u / 4,
			Items: []guigui.LinearLayoutItem{
				{
					Widget: &e.toggle,
					Size:   guigui.FixedSize(defaultIconSize(context)),
				},
				{
					Widget: e.headerWidget,
					Size:   guigui.FlexibleSize(1),
				},
			},
		},
	})
	if e.expanded {
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
	return e.layout(context).Measure(context, constraints)
}

type expanderToggle struct {
	guigui.DefaultWidget

	image    Image
	expanded bool
}

func (e *expanderToggle) setOnDown(callback func(context *guigui.Context)) {
	guigui.SetEventHandler(e, expanderToggleEventDown, callback)
}

func (e *expanderToggle) setExpanded(expanded bool) {
	if e.expanded == expanded {
		return
	}
	e.expanded = expanded
	guigui.RequestRebuild(e)
}

func (e *expanderToggle) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&e.image)

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

func (e *expanderToggle) layout(context *guigui.Context) guigui.LinearLayout {
	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items: []guigui.LinearLayoutItem{
			{
				Size: guigui.FlexibleSize(1),
			},
			{
				Widget: &e.image,
				Size:   guigui.FixedSize(defaultIconSize(context)),
			},
			{
				Size: guigui.FlexibleSize(1),
			},
		},
	}
}

func (e *expanderToggle) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	e.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (e *expanderToggle) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return e.layout(context).Measure(context, constraints)
}

func (e *expanderToggle) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if widgetBounds.IsHitAtCursor() {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			guigui.DispatchEvent(e, expanderToggleEventDown)
			return guigui.HandleInputByWidget(e)
		}
	}
	return guigui.HandleInputResult{}
}

func (e *expanderToggle) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	return ebiten.CursorShapePointer, true
}
