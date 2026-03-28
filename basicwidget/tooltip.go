// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
)

// Tooltip is a standalone widget that shows a balloon popup when the mouse cursor hovers
// over the area specified by [Tooltip.SetHoverBounds].
// The tooltip appears above the hover bounds and has a black background regardless of the color mode.
// The tooltip automatically disappears when the mouse cursor moves out of the hover bounds.
//
// Tooltip is a modeless popup: it does not prevent user interactions with other widgets.
type Tooltip struct {
	guigui.DefaultWidget

	popup          Popup
	tooltipContent tooltipContent

	hoverBounds   image.Rectangle
	hovering      bool
	hoverTicks    int
	toShowTooltip bool
	showPosition  image.Point
}

// SetContent sets a custom content widget for the tooltip balloon.
// [Tooltip.SetContent] and [Tooltip.SetText] are exclusive; [Tooltip.SetContent] takes priority.
func (t *Tooltip) SetContent(widget guigui.Widget) {
	t.tooltipContent.content = widget
}

// SetText sets the tooltip balloon text.
// [Tooltip.SetContent] and [Tooltip.SetText] are exclusive; [Tooltip.SetContent] takes priority.
func (t *Tooltip) SetText(text string) {
	t.tooltipContent.textContent = text
}

// SetHoverBounds sets the area that triggers the tooltip on hover.
// Typically called during Layout with the bounds of the associated widget.
func (t *Tooltip) SetHoverBounds(bounds image.Rectangle) {
	t.hoverBounds = bounds
}

// Build implements [guigui.Widget.Build].
func (t *Tooltip) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.popup)

	t.popup.SetContent(&t.tooltipContent)
	t.popup.SetModal(false)

	// Defer showing until Build so that Layout positions the tooltip correctly
	// before it becomes visible, avoiding a flash at a stale position.
	if t.toShowTooltip {
		t.toShowTooltip = false
		t.popup.SetOpen(true)
	}

	return nil
}

// Layout implements [guigui.Widget.Layout].
func (t *Tooltip) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	// Measure the tooltip content to position it.
	tooltipSize := t.tooltipContent.Measure(context, guigui.Constraints{})

	// Position the tooltip above the hover bounds, centered horizontally on the cursor.
	hb := t.hoverBounds
	pos := t.showPosition
	u := UnitSize(context)
	gap := u / 8
	tooltipBounds := image.Rectangle{
		Min: image.Pt(pos.X-tooltipSize.X/2, hb.Min.Y-tooltipSize.Y-gap),
		Max: image.Pt(pos.X+tooltipSize.X/2+tooltipSize.X%2, hb.Min.Y-gap),
	}

	// Clamp to app bounds so it doesn't go off screen.
	appBounds := context.AppBounds()
	if tooltipBounds.Min.X < appBounds.Min.X {
		tooltipBounds = tooltipBounds.Add(image.Pt(appBounds.Min.X-tooltipBounds.Min.X, 0))
	}
	if tooltipBounds.Max.X > appBounds.Max.X {
		tooltipBounds = tooltipBounds.Add(image.Pt(appBounds.Max.X-tooltipBounds.Max.X, 0))
	}
	if tooltipBounds.Min.Y < appBounds.Min.Y {
		// If no room above, show below the hover bounds.
		tooltipBounds = image.Rectangle{
			Min: image.Pt(tooltipBounds.Min.X, hb.Max.Y+gap),
			Max: image.Pt(tooltipBounds.Max.X, hb.Max.Y+gap+tooltipSize.Y),
		}
	}

	layouter.LayoutWidget(&t.popup, tooltipBounds)
}

// HandlePointingInput implements [guigui.Widget.HandlePointingInput].
func (t *Tooltip) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	cursorPos := image.Pt(ebiten.CursorPosition())
	if cursorPos.In(t.hoverBounds) {
		if !t.hovering {
			t.hovering = true
			t.hoverTicks = 0
		}
		// Only update position before the tooltip is shown, so it stays fixed once visible.
		if !t.toShowTooltip && !t.popup.IsOpen() {
			t.showPosition = cursorPos
		}
	} else {
		if t.hovering {
			t.hovering = false
			t.hoverTicks = 0
			if t.popup.IsOpen() {
				t.popup.SetOpen(false)
			}
		}
	}
	return guigui.HandleInputResult{}
}

// Tick implements [guigui.Widget.Tick].
func (t *Tooltip) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if t.hovering {
		t.hoverTicks++
		if t.hoverTicks == tooltipShowDelay() {
			t.toShowTooltip = true
			guigui.RequestRebuild(t)
		}
	}
	return nil
}

func tooltipShowDelay() int {
	return ebiten.TPS() / 2
}

// TooltipTextPadding returns the padding for tooltip text content.
func TooltipTextPadding(context *guigui.Context) guigui.Padding {
	u := UnitSize(context)
	return guigui.Padding{
		Start:  u / 2,
		Top:    u / 4,
		End:    u / 2,
		Bottom: u / 4,
	}
}

// tooltipContent is the content widget rendered inside the tooltip popup.
// It draws a dark background and border regardless of the color mode.
type tooltipContent struct {
	guigui.DefaultWidget

	content guigui.Widget
	text    Text

	textContent string
}

func (t *tooltipContent) activeWidget() guigui.Widget {
	if t.content != nil {
		return t.content
	}
	return &t.text
}

// Build implements [guigui.Widget.Build].
func (t *tooltipContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(t.activeWidget())

	t.text.SetColor(color.White)
	t.text.SetMultiline(true)
	t.text.SetValue(t.textContent)

	return nil
}

func (t *tooltipContent) layout(context *guigui.Context) guigui.LinearLayout {
	var padding guigui.Padding
	if t.content == nil {
		padding = TooltipTextPadding(context)
	}
	return guigui.LinearLayout{
		Items: []guigui.LinearLayoutItem{
			{
				Widget: t.activeWidget(),
				Size:   guigui.FlexibleSize(1),
			},
		},
		Padding: padding,
	}
}

// Layout implements [guigui.Widget.Layout].
func (t *tooltipContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	t.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

// Measure implements [guigui.Widget.Measure].
func (t *tooltipContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.layout(context).Measure(context, constraints)
}

// Draw implements [guigui.Widget.Draw].
func (t *tooltipContent) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	radius := RoundedCornerRadius(context)
	// Always draw a dark background regardless of color mode.
	basicwidgetdraw.DrawRoundedRect(context, dst, bounds, basicwidgetdraw.BackgroundColor(ebiten.ColorModeDark), radius)
	// Draw a border like a popup.
	clr1, clr2 := basicwidgetdraw.BorderColors(ebiten.ColorModeDark, basicwidgetdraw.RoundedRectBorderTypeOutset)
	width := float32(1 * context.Scale())
	basicwidgetdraw.DrawRoundedRectBorder(context, dst, bounds, clr1, clr2, radius, width, basicwidgetdraw.RoundedRectBorderTypeOutset)
}
