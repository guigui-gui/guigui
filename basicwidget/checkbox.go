package basicwidget

import (
	"image"
	"image/color"

	gui "github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type check_mark_box struct {
	gui.DefaultWidget
	icon        Image
	is_checked  bool
	is_hovering bool
}

func (c *check_mark_box) Draw(ctx *gui.Context, widgetBounds *gui.WidgetBounds, dst *ebiten.Image) {
	var background_color color.Color
	color_mod := ctx.ColorMode()
	if c.is_checked {
		background_color = draw.Color(color_mod, draw.ColorTypeAccent, 0.5)
	} else if c.is_hovering {
		background_color = draw.Color(color_mod, draw.ColorTypeBase, 0.8)
	} else {
		background_color = draw.Color(color_mod, draw.ColorTypeBase, 0.88)
	}

	basicwidgetdraw.DrawRoundedRect(ctx, dst, widgetBounds.Bounds(), background_color, 6)
}

func (c *check_mark_box) Build(ctx *gui.Context, adder *gui.ChildAdder) error {
	if c.is_checked {
		img, err := theResourceImages.Get("check", ctx.ColorMode())
		if err != nil {
			return err
		}
		c.icon.SetImage(img)
		adder.AddChild(&c.icon)
	}
	return nil
}

func (c *check_mark_box) Layout(ctx *gui.Context, widgetBounds *gui.WidgetBounds, layouter *gui.ChildLayouter) {
	layouter.LayoutWidget(&c.icon, widgetBounds.Bounds())
	layout := gui.LinearLayout{
		Items: []gui.LinearLayoutItem{
			{
				Size: gui.FlexibleSize(2),
			},
			{
				Widget: &c.icon,
				Size:   gui.FixedSize(UnitSize(ctx) / 2),
			},
			{
				Size: gui.FlexibleSize(1),
			},
		},
	}
	layout.LayoutWidgets(ctx, widgetBounds.Bounds(), layouter)
}

func (*check_mark_box) Measure(ctx *gui.Context, constraints gui.Constraints) image.Point {
	size := (UnitSize(ctx) / 3) * 2
	return image.Pt(size, size)
}

type Checkbox struct {
	gui.DefaultWidget
	box    check_mark_box
	widget gui.Widget
}

func (c *Checkbox) IsChecked() bool {
	return c.box.is_checked
}

func (c *Checkbox) SetWidget(widget gui.Widget) {
	c.widget = widget
}

func (c *Checkbox) SetText(text string) {
	var text_widget Text
	text_widget.setText(text)
	text_widget.SetVerticalAlign(VerticalAlignMiddle)
	c.widget = &text_widget
}

func (c *Checkbox) Build(ctx *gui.Context, adder *gui.ChildAdder) error {
	adder.AddChild(&c.box)
	if c.widget != nil {
		adder.AddChild(c.widget)
	}
	return nil
}

func (c *Checkbox) Measure(ctx *gui.Context, constraints gui.Constraints) image.Point {
	if h, ok := constraints.FixedHeight(); ok {
		return image.Pt(UnitSize(ctx)*10, h)
	} else if w, ok := constraints.FixedWidth(); ok {
		return image.Pt(w, UnitSize(ctx))
	}

	return image.Pt(0, 0)
}

func (c *Checkbox) Layout(ctx *gui.Context, widgetBounds *gui.WidgetBounds, layouter *gui.ChildLayouter) {
	layout := gui.LinearLayout{
		Direction: gui.LayoutDirectionHorizontal,
		Gap:       (UnitSize(ctx) / 4),
		Items: []gui.LinearLayoutItem{
			{
				Layout: gui.LinearLayout{
					Direction: gui.LayoutDirectionVertical,
					Items: []gui.LinearLayoutItem{
						{Size: gui.FlexibleSize(1)},
						{
							Widget: &c.box,
						},
						{Size: gui.FlexibleSize(1)},
					},
				},
			},
			{
				Widget: c.widget,
			},
		},
	}
	layout.LayoutWidgets(ctx, widgetBounds.Bounds(), layouter)
}

const (
	Checkbox_event_mouseover = "checkbox widget mouse over"
	Checkbox_event_click     = "checkbox widget mouse click"
)

func (c *Checkbox) HandlePointingInput(ctx *gui.Context, widgetBounds *gui.WidgetBounds) gui.HandleInputResult {
	if widgetBounds.IsHitAtCursor() {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
			c.box.is_checked = !c.box.is_checked
			gui.DispatchEvent(c, Checkbox_event_click)
		} else {
			c.box.is_hovering = true
			gui.DispatchEvent(c, Checkbox_event_mouseover)
		}
	} else {
		c.box.is_hovering = false
	}
	return gui.HandleInputResult{}
}
