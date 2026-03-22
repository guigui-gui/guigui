// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

// WidgetWithTooltip is a widget wrapper that shows a balloon popup when the mouse cursor hovers over the wrapped widget.
// The tooltip appears above the cursor position and has a black background regardless of the color mode.
// The tooltip automatically disappears when the mouse cursor moves out of the content area.
type WidgetWithTooltip[T guigui.Widget] struct {
	guigui.DefaultWidget

	widget lazyWidget[T]
	layer  guigui.LayerWidget[*tooltipLayer]

	hovering        bool
	hoverTicks      int
	toShowTooltip   bool
	showTooltip     bool
	showPosition    image.Point
	contentMeasured image.Point
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

// Widget returns the wrapped widget.
func (t *WidgetWithTooltip[T]) Widget() T {
	return t.widget.Widget()
}

// SetTooltipContent sets a custom content widget for the tooltip balloon.
// [WidgetWithTooltip.SetTooltipContent] and [WidgetWithTooltip.SetTooltipText] are exclusive; [WidgetWithTooltip.SetTooltipContent] takes priority.
func (t *WidgetWithTooltip[T]) SetTooltipContent(widget guigui.Widget) {
	t.layer.Widget().setContent(widget)
}

// SetTooltipText sets the tooltip balloon text.
// [WidgetWithTooltip.SetTooltipContent] and [WidgetWithTooltip.SetTooltipText] are exclusive; [WidgetWithTooltip.SetTooltipContent] takes priority.
func (t *WidgetWithTooltip[T]) SetTooltipText(text string) {
	t.layer.Widget().setText(text)
}

// Build implements [guigui.Widget.Build].
func (t *WidgetWithTooltip[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(t.Widget())

	// Defer showing until Build so that Layout positions the tooltip correctly
	// before it becomes visible, avoiding a flash at a stale position.
	if t.toShowTooltip {
		t.toShowTooltip = false
		t.showTooltip = true
		t.layer.BringToFrontLayer(context)
	}
	if t.showTooltip {
		adder.AddWidget(&t.layer)
	}

	return nil
}

// Layout implements [guigui.Widget.Layout].
func (t *WidgetWithTooltip[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(t.Widget(), widgetBounds.Bounds())

	// Measure the tooltip content to position it.
	tooltipSize := t.layer.Widget().Measure(context, guigui.Constraints{})
	t.contentMeasured = tooltipSize

	// Position the tooltip above the widget bounds, centered horizontally on the cursor.
	wb := widgetBounds.Bounds()
	pos := t.showPosition
	u := UnitSize(context)
	gap := u / 8
	tooltipBounds := image.Rectangle{
		Min: image.Pt(pos.X-tooltipSize.X/2, wb.Min.Y-tooltipSize.Y-gap),
		Max: image.Pt(pos.X+tooltipSize.X/2+tooltipSize.X%2, wb.Min.Y-gap),
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
		// If no room above, show below the widget.
		tooltipBounds = image.Rectangle{
			Min: image.Pt(tooltipBounds.Min.X, wb.Max.Y+gap),
			Max: image.Pt(tooltipBounds.Max.X, wb.Max.Y+gap+tooltipSize.Y),
		}
	}

	layouter.LayoutWidget(&t.layer, tooltipBounds)
}

// Measure implements [guigui.Widget.Measure].
func (t *WidgetWithTooltip[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.Widget().Measure(context, constraints)
}

// HandlePointingInput implements [guigui.Widget.HandlePointingInput].
func (t *WidgetWithTooltip[T]) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if widgetBounds.IsHitAtCursor() {
		if !t.hovering {
			t.hovering = true
			t.hoverTicks = 0
		}
		// Only update position before the tooltip is shown, so it stays fixed once visible.
		if !t.toShowTooltip && !t.showTooltip {
			t.showPosition = image.Pt(ebiten.CursorPosition())
		}
	} else {
		if t.hovering {
			t.hovering = false
			t.hoverTicks = 0
			t.showTooltip = false
			guigui.RequestRebuild(t)
		}
	}
	return guigui.HandleInputResult{}
}

// Tick implements [guigui.Widget.Tick].
func (t *WidgetWithTooltip[T]) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if t.hovering {
		t.hoverTicks++
		if t.hoverTicks == tooltipShowDelay() {
			t.toShowTooltip = true
			guigui.RequestRebuild(t)
		}
	}
	return nil
}

// tooltipLayer is the internal layer widget that renders the tooltip.
type tooltipLayer struct {
	guigui.DefaultWidget

	shadow  tooltipShadow
	content guigui.Widget
	text    Text

	textContent string
}

func (t *tooltipLayer) setContent(content guigui.Widget) {
	t.content = content
}

func (t *tooltipLayer) setText(text string) {
	t.textContent = text
}

func (t *tooltipLayer) activeWidget() guigui.Widget {
	if t.content != nil {
		return t.content
	}
	return &t.text
}

// Build implements [guigui.Widget.Build].
func (t *tooltipLayer) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.shadow)
	adder.AddWidget(t.activeWidget())

	t.text.SetColor(color.White)
	t.text.SetMultiline(true)
	t.text.SetValue(t.textContent)

	return nil
}

func (t *tooltipLayer) layout(context *guigui.Context) guigui.LinearLayout {
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
func (t *tooltipLayer) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()
	s := context.Scale()
	t.layout(context).LayoutWidgets(context, bounds, layouter)
	// The shadow needs a larger area than the tooltip bounds.
	shadowBounds := bounds
	shadowBounds.Min.X -= int(16 * s)
	shadowBounds.Max.X += int(16 * s)
	shadowBounds.Min.Y -= int(8 * s)
	shadowBounds.Max.Y += int(16 * s)
	t.shadow.contentBounds = bounds
	layouter.LayoutWidget(&t.shadow, shadowBounds)
}

// Measure implements [guigui.Widget.Measure].
func (t *tooltipLayer) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.layout(context).Measure(context, constraints)
}

// Draw implements [guigui.Widget.Draw].
func (t *tooltipLayer) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	radius := RoundedCornerRadius(context)
	// Always draw a dark background regardless of color mode.
	basicwidgetdraw.DrawRoundedRect(context, dst, bounds, basicwidgetdraw.BackgroundColor(ebiten.ColorModeDark), radius)
	// Draw a border like a popup.
	clr1, clr2 := basicwidgetdraw.BorderColors(ebiten.ColorModeDark, basicwidgetdraw.RoundedRectBorderTypeOutset)
	width := float32(1 * context.Scale())
	basicwidgetdraw.DrawRoundedRectBorder(context, dst, bounds, clr1, clr2, radius, width, basicwidgetdraw.RoundedRectBorderTypeOutset)
}

// HandlePointingInput implements [guigui.Widget.HandlePointingInput].
func (t *tooltipLayer) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	return guigui.HandleInputResult{}
}

type tooltipShadow struct {
	guigui.DefaultWidget

	contentBounds image.Rectangle
}

func (t *tooltipShadow) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := t.contentBounds
	s := context.Scale()
	bounds.Min.X -= int(16 * s)
	bounds.Max.X += int(16 * s)
	bounds.Min.Y -= int(8 * s)
	bounds.Max.Y += int(16 * s)
	clr := draw.Color2(context.ResolvedColorMode(), draw.ColorTypeBase, 0, 0)
	clr = draw.ScaleAlpha(clr, 0.25)
	draw.DrawRoundedShadowRect(context, dst, bounds, clr, int(16*s)+RoundedCornerRadius(context))
}
